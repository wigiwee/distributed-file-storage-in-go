package main

import (
	"bytes"
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

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandShakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(tcpOpts)

	FileServerOpts := FileServerOpts{
		storageRoot:       root + string(os.PathSeparator) + "network_" + listenAddr,
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(FileServerOpts)
	tcpOpts.OnPeer = s.OnPeer
	return s

}
func main() {
	s1 := makeServer(":3000", "/home/happypotter/dfs")
	s2 := makeServer(":4000", "/home/happypotter/dfs", ":3000")
	go func() {
		log.Fatal(s1.Start())
	}()

	go s2.Start()
	time.Sleep(2 * time.Second)
	data := bytes.NewReader([]byte("lkjsdfj"))
	s2.StoreData("myPrivateData", data)

	select {}
}
