package main

import (
	"bytes"
	"encoding/binary"
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

func (fs *FileServer) Get(key string) (io.Reader, error) {
	if fs.store.Has(key) {
		log.Printf("[%s] serving file with key %s from local disk", fs.Transport.Addr(), key)
		_, r, err := fs.store.Read(key)
		return r, err
	}
	log.Printf("[%s] dosen't have %s locally, looking over the network", fs.Transport.Addr(), key)
	msg := Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}

	if err := fs.broadcast(&msg); err != nil {
		return nil, err
	}

	time.Sleep(time.Millisecond * 30)

	for _, peer := range fs.peers {
		//read the file size so we can limit the amount of bytes we read from
		//the connection
		var filesize int64
		binary.Read(peer, binary.LittleEndian, &filesize)
		if filesize == 0 {
			return nil, fmt.Errorf("[%s] dosen't have file %s", peer.RemoteAddr(), key)
		}
		n, err := fs.store.Write(key, io.LimitReader(peer, filesize))
		if err != nil {
			return nil, err
		}

		log.Printf("[%s] received %d bytes from %s:", fs.Transport.Addr(), n, peer.RemoteAddr())

		peer.CloseStream()
	}
	_, r, err := fs.store.Read(key)
	return r, err
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

	time.Sleep(10 * time.Millisecond)

	//do multiwriter
	for _, peer := range fs.peers {
		peer.Send([]byte{p2p.IncomingStream})
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
		peer.Send([]byte{p2p.IncomingMessage})
		if err := peer.Send(msgBuf.Bytes()); err != nil {
			return err
		}
	}

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

	peer, ok := fs.peers[from]
	if !ok {
		return fmt.Errorf("peer %s not in map", from)
	}

	if !fs.store.Has(msg.Key) {
		peer.Send([]byte{p2p.IncomingStream})
		binary.Write(peer, binary.LittleEndian, int64(0))
		return fmt.Errorf("[%s] file not present on disk %s\n ", fs.Transport.Addr(), msg.Key)
	}

	log.Printf("[%s] serving file %s over the network\n", fs.Transport.Addr(), msg.Key)

	filesize, r, err := fs.store.Read(msg.Key)
	if err != nil {
		peer.Send([]byte{p2p.IncomingStream})
		binary.Write(peer, binary.LittleEndian, int64(0))
		return err
	}

	//first send incoming stream byte to the peer and then we can send the file size
	// as an int64
	peer.Send([]byte{p2p.IncomingStream})

	binary.Write(peer, binary.LittleEndian, filesize)
	n, err := io.Copy(peer, r)
	if err != nil {
		return err
	}

	log.Printf("[%s] written %d bytes to %s\n", fs.Transport.Addr(), n, from)
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

	log.Printf("%s written %d bytes to disk\n", fs.Transport.Addr(), n)

	peer.CloseStream()
	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
}
