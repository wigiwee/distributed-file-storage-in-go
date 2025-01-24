package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

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
	key string
}

func (s *FileServer) broadcast(msg *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	log.Printf("broadcasting %+v", msg)
	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(msg)
}

func (fs *FileServer) StoreData(key string, r io.Reader) error {
	// 1. store this file to disk
	// 2. store this file to all the known peers in the network

	buf := new(bytes.Buffer)

	msg := &Message{
		Payload: MessageStoreFile{
			key: key,
		},
	}
	fmt.Println("i came here")
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	fmt.Println("i came here")
	for _, peer := range fs.peers {
		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}
	}
	// time.Sleep(2 * time.Second)

	// payload := []byte("this is large file")
	// for _, peer := range fs.peers {
	// 	if err := peer.Send(payload); err != nil {
	// 		return err
	// 	}
	// }
	return nil
	// buf := new(bytes.Buffer)
	// tee := io.TeeReader(r, buf)

	// if err := fs.store.Write(key, tee); err != nil {
	// 	return err
	// }
	// p := &DataMessage{
	// 	Key:  key,
	// 	Data: buf.Bytes(),
	// }

	// return fs.broadcast(&Message{
	// 	From:    "Todo",
	// 	Payload: p.Data,
	// })
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

// func (s *FileServer) handleMessage(m *Message) error {
// 	switch v := m.Payload.(type) {
// 	case *DataMessage:
// 		log.Printf("received data of type dataMsg %+v", v)
// 	}
// 	return nil
// }

func (fs *FileServer) loop() {
	defer func() {
		log.Println("File server stopped due to user quit action")
		fs.Transport.Close()
	}()
	for {
		select {
		case rpc := <-fs.Transport.Consume():
			var msg Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Println(err)
			}
			log.Printf("received key: %+v\n", msg.Payload)

			peer, ok := fs.peers[rpc.From]
			if !ok {
				panic("peer not found in peer map")
			}

			b := make([]byte, 1000)
			if _, err := peer.Read(b); err != nil {
				panic(err)
			}

			log.Printf("received data : %+v\n", string(b))
			// if err := fs.handleMessage(&m); err != nil {
			// 	log.Println(err)
			// }

			peer.(*p2p.TCPPeer).Wg.Done()
		case <-fs.quitch:
			return
		}
	}
}

func init() {
	gob.Register(MessageStoreFile{})
}
