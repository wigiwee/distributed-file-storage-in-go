package p2p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// TCPPeer represents the remote node over a TCP established connection.
type TCPPeer struct {
	//the underlying connection of the peer
	//which in this case is a tcp connection
	net.Conn
	// If we dial the connection => outbound => true
	// If we accept the incoming connection => outbound => false
	outbound bool

	wg *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

// Send implements the Peer interface
func (p *TCPPeer) Send(b []byte) error {
	_, err := p.Conn.Write(b)
	return err
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandShakeFunc HandShakeFunc
	Decoder       Decoder
	OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener
	rpcch    chan RPC
}

func NewTCPTransport(opts *TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: *opts,
		rpcch:            make(chan RPC, 1024),
	}
}

// Addr implements the Transport returning the address
// the transport is accepting connection
func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

// func (t *TCPTransport) ListenAddr() string {
// 	return t.TCPTransportOpts.ListenAddr
// }

// Dial implements the Transport interface
func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)

	return nil

}

// Consume implements the Transport interface, which will return read only channel
// for reading the incoming message received from another peer in the network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

// Close implements the transport interface
func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.startAcceptLoop()

	log.Printf("TCP transport listening on port: %s\n", t.ListenAddr)

	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			fmt.Printf("tcp accept error: %s", err)
		}

		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	defer func() {
		log.Printf("dropping peer connection : %s\n", err)
		conn.Close()

	}()
	peer := NewTCPPeer(conn, outbound)
	if err = t.HandShakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}
	//readloop

	for {
		rpc := &RPC{}
		err := t.Decoder.Decode(conn, rpc)

		if err != nil {
			fmt.Printf("TCP read error: %s\n", err)
			return
		}

		rpc.From = conn.RemoteAddr().String()

		if rpc.Stream {
			peer.wg.Add(1)
			log.Printf("[%s] incoming [%s] waiting \n", t.Addr(), rpc.From)
			peer.wg.Wait()
			log.Printf("[%s] stream [%s] closed \n", t.Addr(), rpc.From)
			continue
		}

		// peer.Wg.Add(1)
		// fmt.Println("waiting till stream is done")
		t.rpcch <- *rpc
		// peer.Wg.Wait()
		// fmt.Println("stream done continuing normal read loop")

	}
}
