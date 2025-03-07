package p2p

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

// RCP holds any arbitrary data that is begin sent over the
// each transport between two nodes in the network
type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}
