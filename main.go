package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"dfs/p2p"
)

func OnPeer(peer p2p.Peer) error {
	peer.Close()
	// return fmt.Errorf("failed the onpeer function")
	return nil
}

func makeServer(listenAddr, root string, nodes ...string) *FileServer {

	tcpOpts := &p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandShakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(tcpOpts)

	FileServerOpts := &FileServerOpts{
		EncKey:            newEncryptionKey(),
		storageRoot:       root + string(os.PathSeparator) + "network_" + listenAddr,
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}
	s := NewFileServer(FileServerOpts)
	tcpTransport.OnPeer = s.OnPeer

	return s

}
func main() {
	s1 := makeServer(":3000", "./")
	s2 := makeServer(":4000", "./", ":3000")
	s3 := makeServer(":5000", "./", ":3000", ":4000")
	// go func() {
	// 	log.Fatal(s1.Start())
	// 	// time.Sleep(20 * time.Millisecond)
	// }()

	// go s2.Start()

	go s1.Start()
	go s2.Start()
	time.Sleep(2 * time.Second)
	go s3.Start()
	time.Sleep(2 * time.Second)

	// for i := 0; i < 1; i++ {
	// 	bufBytes := make([]byte, rand.Intn(50)+10)
	// 	rand.Read(bufBytes)
	// 	data := bytes.NewReader(bufBytes)
	// 	go s3.Store(fmt.Sprintf("myPrivateData_%d", i), data)
	// 	// time.Sleep(3 * time.Millisecond)
	// }

	s2.Store("myPrivateData", bytes.NewReader([]byte("this is some big data")))
	s2.store.Delete("myPrivateData")

	r, err := s2.Get("myPrivateData")
	if err != nil {
		log.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))

	s3.Store("data", bytes.NewReader([]byte("this is data")))
	s3.store.Delete("data")
	r, _ = s3.Get("data")
	b, _ = io.ReadAll(r)
	fmt.Println(string(b))

	// select {}
}
