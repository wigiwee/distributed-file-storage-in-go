package p2p

// RCP holds any arbitrary data that is begin sent over the
// each transport between two nodes in the network
type RPC struct {
	From    string
	Payload []byte
}
