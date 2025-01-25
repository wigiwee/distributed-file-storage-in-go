package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"dfs/p2p"
)

type FileServerOpts struct {
	storageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store  *Store
	quitch chan struct{}
}

func NewFileServer(opts *FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.storageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}
	return &FileServer{
		FileServerOpts: *opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

type Message struct {
	// From    string
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

func (s *FileServer) stream(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	log.Printf("broadcasting %+v", msg)
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) broadcast(msg *Message) error {
	msgBuf := new(bytes.Buffer)
	if err := gob.NewEncoder(msgBuf).Encode(msg); err != nil {
		return err
	}
	for _, peer := range fs.peers {
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		return fs.store.Read(key)
	}
	log.Println("dont have file locally looking over network...")
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}
	select {}
	return nil, nil
}
func (fs *FileServer) Store(key string, r io.Reader) error {
	var (
		fileData = new(bytes.Buffer)
		tee      = io.TeeReader(r, fileData)
	)

	size, err := fs.store.Write(key, tee)

	if err != nil {
		return err
	}

	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: size,
		},
	}
	if err := fs.broadcast(&msg); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	//do multiwriter
	for _, peer := range fs.peers {
		n, err := io.Copy(peer, fileData)
		if err != nil {
			return err
		}
		fmt.Println("received an written to disk", n)
	}
	return nil

}

func (fs *FileServer) Stop() {
	close(fs.quitch)
}

func (fs *FileServer) Start() error {
	if err := fs.Transport.ListenAndAccept(); err != nil {
		return err
	}

	fs.bootstrapNetwork()

	fs.loop()

	return nil

}

func (fs *FileServer) OnPeer(p p2p.Peer) error {
	fs.peerLock.Lock()
	defer fs.peerLock.Unlock()

	fs.peers[p.RemoteAddr().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddr())
	return nil
}

func (fs *FileServer) bootstrapNetwork() error {
	for _, addr := range fs.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			log.Println("attempting to connect with remote:", addr)
			if err := fs.Transport.Dial(addr); err != nil {
				log.Println("dial error:", err)
				panic(err)
			}
		}(addr)
	}
	return nil
}

func (s *FileServer) handleMessage(from string, m *Message) error {
	switch v := m.Payload.(type) {
	case MessageStoreFile:
		return s.handleMesssageStoreFile(from, v)
	case MessageGetFile:
		return s.handleMessageGetFile(from, v)
	}
	return nil
}

func (fs *FileServer) handleMessageGetFile(from string, msg MessageGetFile) error {

	fmt.Println("need to get file from disk and send it over the network")
	if !fs.store.Has(msg.Key) {
		return fmt.Errorf("file not present on this disk %s ", msg.Key)
	}

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	r, err := fs.store.Read(msg.Key)
	if err != nil {
		return err
	}

	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}
	fmt.Println("written %d bytes to network", n)
	return nil
}

func (fs *FileServer) handleMesssageStoreFile(from string, msg MessageStoreFile) error {
	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer (%s) could not be found in peer list", from)
	}

	n, err := fs.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}

	log.Printf("written %d bytes to disk\n", n)
	peer.(*p2p.TCPPeer).Wg.Done()

	return nil
}

func (fs *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to error or user quit action")
		fs.Transport.Close()
	}()
	for {
		select {
		case rpc := <-fs.Transport.Consume():
			var msg Message

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println("decoding error:", err)
			}

			if err := fs.handleMessage(rpc.From, &msg); err != nil {
				fmt.Println("handle message error:", err)
			}

		case <-fs.quitch:
			return
		}
	}
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
