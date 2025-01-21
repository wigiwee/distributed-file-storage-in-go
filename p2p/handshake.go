package p2p

import "errors"

// ErrorInvalidHandshake is returned if the handshake between the local
// and the remote node could not be established
var ErrInvalidHandshake = errors.New("invalid handshake")

// HandshakeFunc is executed
type HandShakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error { return nil }
