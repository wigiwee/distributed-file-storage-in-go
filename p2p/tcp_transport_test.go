package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTcpTransport(t *testing.T) {

	listenAddr := ":3000"
	tr := NewTCPTransport(listenAddr)
	assert.Equal(t, tr.listenAddr, listenAddr)

	//server

	assert.Nil(t, tr.ListenAndAccept())

}
