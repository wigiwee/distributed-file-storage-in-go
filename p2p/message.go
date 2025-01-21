package p2p

import "net"

// RCP holds any arbitrary data that is begin sent over the
// each transport between two nodes in the network
type RPC struct {
	from    net.Addr
	Payload []byte
}
