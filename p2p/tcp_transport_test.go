package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTcpTransport(t *testing.T) {

	listenAddr := ":3000"
	opts := TCPTransportOpts{
		ListenAddr:    listenAddr,
		HandShakeFunc: NOPHandshakeFunc,
		Decoder:       DefaultDecoder{},
	}
	tr := NewTCPTransport(opts)
	assert.Equal(t, tr.ListenAddr, listenAddr)

	assert.Nil(t, tr.ListenAndAccept())

}
