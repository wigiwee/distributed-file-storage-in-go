package p2p

import (
	"fmt"
	"log"
	"net"
)

// TCPPeer represents the remote node over a TCP established connection.
type TCPPeer struct {

	//conn is underlying connection of the peer
	conn net.Conn
	// If we dial the connection => outbound => true
	// If we accept the incoming connection => outbound => false
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

// Close implements the Peer interface
func (tp *TCPPeer) Close() error {
	return tp.conn.Close()
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

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

// Consume implements the Transport interface, which will return read only channel
// for reading the incoming message received from another peer in the network
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
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
		if err != nil {
			fmt.Printf("tcp accept error: %s", err)
		}

		log.Printf("New incomming connection %+v\n", conn)

		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	var err error
	defer func() {
		log.Printf("dropping peer connection : %s\n", err)
		conn.Close()

	}()
	peer := NewTCPPeer(conn, true)
	if err = t.HandShakeFunc(peer); err != nil {
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}
	//readloop
	rpc := &RPC{}

	for {
		err := t.Decoder.Decode(conn, rpc)

		if err != nil {

			fmt.Printf("TCP read error: %s\n", err)
			return
		}

		rpc.from = conn.RemoteAddr()
		t.rpcch <- *rpc
	}
}
