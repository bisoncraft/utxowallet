package bisonwire

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

type Chain string

const (
	ChainBTC Chain = "btc"
	ChainLTC Chain = "ltc"
)

var KnownChains = []Chain{
	ChainBTC,
	ChainLTC,
}

func ChainFromString(s string) (Chain, error) {
	sc := Chain(s)
	for _, c := range KnownChains {
		if c == sc {
			return c, nil
		}
	}
	return "", fmt.Errorf("no known chain identified by %q", s)
}

// makeEmptyMessage creates a message of the appropriate concrete type based
// on the command.
func makeEmptyMessage(chain Chain, command string) (wire.Message, error) {
	var msg wire.Message
	switch command {
	// Bisonwire variations
	case wire.CmdGetBlocks:
		msg = &wire.MsgGetBlocks{}

	case wire.CmdBlock:
		msg = &Block{Chain: string(chain)}

	case wire.CmdHeaders:
		msg = &wire.MsgHeaders{}

	case wire.CmdTx:
		msg = &Tx{Chain: string(chain)}

	// Standard BTC wire types
	case wire.CmdVersion:
		msg = &wire.MsgVersion{}

	case wire.CmdVerAck:
		msg = &wire.MsgVerAck{}

	case wire.CmdSendAddrV2:
		msg = &wire.MsgSendAddrV2{}

	case wire.CmdGetAddr:
		msg = &wire.MsgGetAddr{}

	case wire.CmdAddr:
		msg = &wire.MsgAddr{}

	case wire.CmdAddrV2:
		msg = &wire.MsgAddrV2{}

	case wire.CmdInv:
		msg = &wire.MsgInv{}

	case wire.CmdGetData:
		msg = &wire.MsgGetData{}

	case wire.CmdNotFound:
		msg = &wire.MsgNotFound{}

	case wire.CmdPing:
		msg = &wire.MsgPing{}

	case wire.CmdPong:
		msg = &wire.MsgPong{}

	case wire.CmdGetHeaders:
		msg = &wire.MsgGetHeaders{}

	case wire.CmdAlert:
		msg = &wire.MsgAlert{}

	case wire.CmdMemPool:
		msg = &wire.MsgMemPool{}

	case wire.CmdFilterAdd:
		msg = &wire.MsgFilterAdd{}

	case wire.CmdFilterClear:
		msg = &wire.MsgFilterClear{}

	case wire.CmdFilterLoad:
		msg = &wire.MsgFilterLoad{}

	case wire.CmdMerkleBlock:
		msg = &wire.MsgMerkleBlock{}

	case wire.CmdReject:
		msg = &wire.MsgReject{}

	case wire.CmdSendHeaders:
		msg = &wire.MsgSendHeaders{}

	case wire.CmdFeeFilter:
		msg = &wire.MsgFeeFilter{}

	case wire.CmdGetCFilters:
		msg = &wire.MsgGetCFilters{}

	case wire.CmdGetCFHeaders:
		msg = &wire.MsgGetCFHeaders{}

	case wire.CmdGetCFCheckpt:
		msg = &wire.MsgGetCFCheckpt{}

	case wire.CmdCFilter:
		msg = &wire.MsgCFilter{}

	case wire.CmdCFHeaders:
		msg = &wire.MsgCFHeaders{}

	case wire.CmdCFCheckpt:
		msg = &wire.MsgCFCheckpt{}

	default:
		return nil, fmt.Errorf("%w: %s", wire.ErrUnknownMessage, command)
	}
	return msg, nil
}
