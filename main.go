package main

import (
	"log"
	"time"

	"dfs/p2p"
)

func OnPeer(peer p2p.Peer) error {
	peer.Close()
	// return fmt.Errorf("failed the onpeer function")
	return nil
}
func main() {

	tcpOpts := p2p.TCPTransportOpts{
		ListenAddr:    ":3000",
		HandShakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		//todo : onpeer func
	}

	tcpTransport := p2p.NewTCPTransport(tcpOpts)

	FileServerOpts := FileServerOpts{
		storageRoot:       "/home/happypotter/dfs",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
	}

	s := NewFileServer(FileServerOpts)

	go func() {
		time.Sleep(time.Second * 3)
		s.Stop()
	}()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}

}
