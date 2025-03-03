package chain

import (
	"errors"
)

var (
	// ErrBitcoindClientShuttingDown is an error returned when we attempt
	// to receive a notification for a specific item and the bitcoind client
	// is in the middle of shutting down.
	ErrBitcoindClientShuttingDown = errors.New("client is shutting down")
)
