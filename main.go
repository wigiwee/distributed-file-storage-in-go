package main

import (
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
	go func() {
		log.Fatal(s1.Start())
	}()

	go s2.Start()
	time.Sleep(2 * time.Second)
	// data := bytes.NewReader([]byte("my big data file here"))
	// s2.Store("myPrivateData", data)

	r, err := s2.Get("myPrivateData")
	if err != nil {
		log.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
	select {}
}
