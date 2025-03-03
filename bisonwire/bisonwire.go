package bisonwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
	"unicode/utf8"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
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

// WriteMessageWithEncodingN writes a bitcoin Message to w including the
// necessary header information and returns the number of bytes written.
// This function is the same as WriteMessageN except it also allows the caller
// to specify the message encoding format to be used when serializing wire
// messages.
func WriteMessageWithEncodingN(w io.Writer, msg wire.Message, pver uint32,
	btcnet wire.BitcoinNet, encoding wire.MessageEncoding) (int, error) {

	totalBytes := 0

	// Enforce max command size.
	var command [wire.CommandSize]byte
	cmd := msg.Command()
	if len(cmd) > wire.CommandSize {
		str := fmt.Sprintf("command [%s] is too long [max %v]",
			cmd, wire.CommandSize)
		return totalBytes, messageError("WriteMessage", str)
	}
	copy(command[:], []byte(cmd))

	// Encode the message payload.
	var bw bytes.Buffer
	err := msg.BtcEncode(&bw, pver, encoding)
	if err != nil {
		return totalBytes, err
	}
	payload := bw.Bytes()
	lenp := len(payload)

	// Enforce maximum overall message payload.
	if lenp > wire.MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - encoded "+
			"%d bytes, but maximum message payload is %d bytes",
			lenp, wire.MaxMessagePayload)
		return totalBytes, messageError("WriteMessage", str)
	}

	// Enforce maximum message payload based on the message type.
	mpl := msg.MaxPayloadLength(pver)
	if uint32(lenp) > mpl {
		str := fmt.Sprintf("message payload is too large - encoded "+
			"%d bytes, but maximum message payload size for "+
			"messages of type [%s] is %d.", lenp, cmd, mpl)
		return totalBytes, messageError("WriteMessage", str)
	}

	// Create header for the message.
	hdr := messageHeader{}
	hdr.magic = btcnet
	hdr.command = cmd
	hdr.length = uint32(lenp)
	copy(hdr.checksum[:], chainhash.DoubleHashB(payload)[0:4])

	// Encode the header for the message.  This is done to a buffer
	// rather than directly to the writer since writeElements doesn't
	// return the number of bytes written.
	hw := bytes.NewBuffer(make([]byte, 0, wire.MessageHeaderSize))
	writeElements(hw, hdr.magic, command, hdr.length, hdr.checksum)

	// Write header.
	n, err := w.Write(hw.Bytes())
	totalBytes += n
	if err != nil {
		return totalBytes, err
	}

	// Only write the payload if there is one, e.g., verack messages don't
	// have one.
	if len(payload) > 0 {
		n, err = w.Write(payload)
		totalBytes += n
	}

	return totalBytes, err
}

// ReadMessageWithEncodingN reads, validates, and parses the next bitcoin Message
// from r for the provided protocol version and bitcoin network.  It returns the
// number of bytes read in addition to the parsed Message and raw bytes which
// comprise the message.  This function is the same as ReadMessageN except it
// allows the caller to specify which message encoding is to consult when
// decoding wire messages.
func ReadMessageWithEncodingN(
	r io.Reader,
	pver uint32,
	chain Chain,
	btcnet wire.BitcoinNet,
	enc wire.MessageEncoding,
) (int, wire.Message, []byte, error) {

	totalBytes := 0
	n, hdr, err := readMessageHeader(r)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

	// Enforce maximum message payload.
	if hdr.length > wire.MaxMessagePayload {
		str := fmt.Sprintf("message payload is too large - header "+
			"indicates %d bytes, but max message payload is %d "+
			"bytes.", hdr.length, wire.MaxMessagePayload)
		return totalBytes, nil, nil, messageError("ReadMessage", str)

	}

	// Check for messages from the wrong bitcoin network.
	if hdr.magic != btcnet {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("message from other network [%v]", hdr.magic)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Check for malformed commands.
	command := hdr.command
	if !utf8.ValidString(command) {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("invalid command %v", []byte(command))
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Create struct of appropriate message type based on the command.
	msg, err := makeEmptyMessage(chain, command)
	if err != nil {
		// makeEmptyMessage can only return ErrUnknownMessage and it is
		// important that we bubble it up to the caller.
		discardInput(r, hdr.length)
		return totalBytes, nil, nil, err
	}

	// Check for maximum length based on the message type as a malicious client
	// could otherwise create a well-formed header and set the length to max
	// numbers in order to exhaust the machine's memory.
	mpl := msg.MaxPayloadLength(pver)
	if hdr.length > mpl {
		discardInput(r, hdr.length)
		str := fmt.Sprintf("payload exceeds max length - header "+
			"indicates %v bytes, but max payload size for "+
			"messages of type [%v] is %v.", hdr.length, command, mpl)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Read payload.
	payload := make([]byte, hdr.length)
	n, err = io.ReadFull(r, payload)
	totalBytes += n
	if err != nil {
		return totalBytes, nil, nil, err
	}

	// Test checksum.
	checksum := chainhash.DoubleHashB(payload)[0:4]
	if !bytes.Equal(checksum, hdr.checksum[:]) {
		str := fmt.Sprintf("payload checksum failed - header "+
			"indicates %v, but actual checksum is %v.",
			hdr.checksum, checksum)
		return totalBytes, nil, nil, messageError("ReadMessage", str)
	}

	// Unmarshal message.  NOTE: This must be a *bytes.Buffer since the
	// MsgVersion BtcDecode function requires it.
	pr := bytes.NewBuffer(payload)
	err = msg.BtcDecode(pr, pver, enc)
	if err != nil {
		return totalBytes, nil, nil, err
	}

	return totalBytes, msg, payload, nil
}

// ReadMessageN reads, validates, and parses the next bitcoin Message from r for
// the provided protocol version and bitcoin network.  It returns the number of
// bytes read in addition to the parsed Message and raw bytes which comprise the
// message.  This function is the same as ReadMessage except it also returns the
// number of bytes read.
func ReadMessageN(r io.Reader, pver uint32, chain Chain, btcnet wire.BitcoinNet) (int, wire.Message, []byte, error) {
	return ReadMessageWithEncodingN(r, pver, chain, btcnet, wire.BaseEncoding)
}

// ReadMessage reads, validates, and parses the next bitcoin Message from r for
// the provided protocol version and bitcoin network.  It returns the parsed
// Message and raw bytes which comprise the message.  This function only differs
// from ReadMessageN in that it doesn't return the number of bytes read.  This
// function is mainly provided for backwards compatibility with the original
// API, but it's also useful for callers that don't care about byte counts.
func ReadMessage(r io.Reader, pver uint32, chain Chain, btcnet wire.BitcoinNet) (wire.Message, []byte, error) {
	_, msg, buf, err := ReadMessageN(r, pver, chain, btcnet)
	return msg, buf, err
}

// messageError creates an error for the given function and description.
func messageError(f string, desc string) *wire.MessageError {
	return &wire.MessageError{Func: f, Description: desc}
}

// messageHeader defines the header structure for all bitcoin protocol messages.
type messageHeader struct {
	magic    wire.BitcoinNet // 4 bytes
	command  string          // 12 bytes
	length   uint32          // 4 bytes
	checksum [4]byte         // 4 bytes
}

// readMessageHeader reads a bitcoin message header from r.
func readMessageHeader(r io.Reader) (int, *messageHeader, error) {
	// Since readElements doesn't return the amount of bytes read, attempt
	// to read the entire header into a buffer first in case there is a
	// short read so the proper amount of read bytes are known.  This works
	// since the header is a fixed size.
	var headerBytes [wire.MessageHeaderSize]byte
	n, err := io.ReadFull(r, headerBytes[:])
	if err != nil {
		return n, nil, err
	}
	hr := bytes.NewReader(headerBytes[:])

	// Create and populate a messageHeader struct from the raw header bytes.
	hdr := messageHeader{}
	var command [wire.CommandSize]byte
	readElements(hr, &hdr.magic, &command, &hdr.length, &hdr.checksum)

	// Strip trailing zeros from command string.
	hdr.command = string(bytes.TrimRight(command[:], "\x00"))

	return n, &hdr, nil
}

// discardInput reads n bytes from reader r in chunks and discards the read
// bytes.  This is used to skip payloads when various errors occur and helps
// prevent rogue nodes from causing massive memory allocation through forging
// header length.
func discardInput(r io.Reader, n uint32) {
	maxSize := uint32(10 * 1024) // 10k at a time
	numReads := n / maxSize
	bytesRemaining := n % maxSize
	if n > 0 {
		buf := make([]byte, maxSize)
		for i := uint32(0); i < numReads; i++ {
			io.ReadFull(r, buf)
		}
	}
	if bytesRemaining > 0 {
		buf := make([]byte, bytesRemaining)
		io.ReadFull(r, buf)
	}
}

// readElements reads multiple items from r.  It is equivalent to multiple
// calls to readElement.
func readElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := readElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeElement writes the little endian representation of element to w.
func writeElement(w io.Writer, element interface{}) error {
	// Attempt to write the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case int32:
		err := binarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case uint32:
		err := binarySerializer.PutUint32(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case int64:
		err := binarySerializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case uint64:
		err := binarySerializer.PutUint64(w, littleEndian, e)
		if err != nil {
			return err
		}
		return nil

	case bool:
		var err error
		if e {
			err = binarySerializer.PutUint8(w, 0x01)
		} else {
			err = binarySerializer.PutUint8(w, 0x00)
		}
		if err != nil {
			return err
		}
		return nil

	// Message header checksum.
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case [wire.CommandSize]uint8:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *chainhash.Hash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case wire.ServiceFlag:
		err := binarySerializer.PutUint64(w, littleEndian, uint64(e))
		if err != nil {
			return err
		}
		return nil

	case wire.InvType:
		err := binarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case wire.BitcoinNet:
		err := binarySerializer.PutUint32(w, littleEndian, uint32(e))
		if err != nil {
			return err
		}
		return nil

	case wire.BloomUpdateType:
		err := binarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil

	case wire.RejectCode:
		err := binarySerializer.PutUint8(w, uint8(e))
		if err != nil {
			return err
		}
		return nil
	}

	// Fall back to the slower binary.Write if a fast path was not available
	// above.
	return binary.Write(w, littleEndian, element)
}

// writeElements writes multiple items to w.  It is equivalent to multiple
// calls to writeElement.
func writeElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := writeElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// // uint32Time represents a unix timestamp encoded with a uint32.  It is used as
// // a way to signal the readElement function how to decode a timestamp into a Go
// // time.Time since it is otherwise ambiguous.
// type uint32Time time.Time

// // int64Time represents a unix timestamp encoded with an int64.  It is used as
// // a way to signal the readElement function how to decode a timestamp into a Go
// // time.Time since it is otherwise ambiguous.
// type int64Time time.Time

// readElement reads the next sequence of bytes from r using little endian
// depending on the concrete type of element pointed to.
func readElement(r io.Reader, element interface{}) error {
	// Attempt to read the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case *int32:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int32(rv)
		return nil

	case *uint32:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *int64:
		rv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = int64(rv)
		return nil

	case *uint64:
		rv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = rv
		return nil

	case *bool:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		if rv == 0x00 {
			*e = false
		} else {
			*e = true
		}
		return nil

	// Unix timestamp encoded as a uint32.
	case *uint32Time:
		rv, err := binarySerializer.Uint32(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = uint32Time(time.Unix(int64(rv), 0))
		return nil

	// Unix timestamp encoded as an int64.
	case *int64Time:
		rv, err := binarySerializer.Uint64(r, binary.LittleEndian)
		if err != nil {
			return err
		}
		*e = int64Time(time.Unix(int64(rv), 0))
		return nil

	// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case *[wire.CommandSize]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case *[16]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *chainhash.Hash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *wire.ServiceFlag:
		rv, err := binarySerializer.Uint64(r, littleEndian)
		if err != nil {
			return err
		}
		*e = wire.ServiceFlag(rv)
		return nil

	case *wire.InvType:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = wire.InvType(rv)
		return nil

	case *wire.BitcoinNet:
		rv, err := binarySerializer.Uint32(r, littleEndian)
		if err != nil {
			return err
		}
		*e = wire.BitcoinNet(rv)
		return nil

	case *wire.BloomUpdateType:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = wire.BloomUpdateType(rv)
		return nil

	case *wire.RejectCode:
		rv, err := binarySerializer.Uint8(r)
		if err != nil {
			return err
		}
		*e = wire.RejectCode(rv)
		return nil
	}

	// Fall back to the slower binary.Read if a fast path was not available
	// above.
	return binary.Read(r, littleEndian, element)
}

var (
	// littleEndian is a convenience variable since binary.LittleEndian is
	// quite long.
	littleEndian = binary.LittleEndian

	// bigEndian is a convenience variable since binary.BigEndian is quite
	// long.
	bigEndian = binary.BigEndian
)

// binaryFreeList defines a concurrent safe free list of byte slices (up to the
// maximum number defined by the binaryFreeListMaxItems constant) that have a
// cap of 8 (thus it supports up to a uint64).  It is used to provide temporary
// buffers for serializing and deserializing primitive numbers to and from their
// binary encoding in order to greatly reduce the number of allocations
// required.
//
// For convenience, functions are provided for each of the primitive unsigned
// integers that automatically obtain a buffer from the free list, perform the
// necessary binary conversion, read from or write to the given io.Reader or
// io.Writer, and return the buffer to the free list.
type binaryFreeList chan []byte

// Borrow returns a byte slice from the free list with a length of 8.  A new
// buffer is allocated if there are not any available on the free list.
func (l binaryFreeList) Borrow() []byte {
	var buf []byte
	select {
	case buf = <-l:
	default:
		buf = make([]byte, 8)
	}
	return buf[:8]
}

// Return puts the provided byte slice back on the free list.  The buffer MUST
// have been obtained via the Borrow function and therefore have a cap of 8.
func (l binaryFreeList) Return(buf []byte) {
	select {
	case l <- buf:
	default:
		// Let it go to the garbage collector.
	}
}

// Uint8 reads a single byte from the provided reader using a buffer from the
// free list and returns it as a uint8.
func (l binaryFreeList) Uint8(r io.Reader) (uint8, error) {
	buf := l.Borrow()[:1]
	defer l.Return(buf)

	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	rv := buf[0]

	return rv, nil
}

// Uint16 reads two bytes from the provided reader using a buffer from the
// free list, converts it to a number using the provided byte order, and returns
// the resulting uint16.
func (l binaryFreeList) Uint16(r io.Reader, byteOrder binary.ByteOrder) (uint16, error) {
	buf := l.Borrow()[:2]
	defer l.Return(buf)

	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	rv := byteOrder.Uint16(buf)

	return rv, nil
}

// Uint32 reads four bytes from the provided reader using a buffer from the
// free list, converts it to a number using the provided byte order, and returns
// the resulting uint32.
func (l binaryFreeList) Uint32(r io.Reader, byteOrder binary.ByteOrder) (uint32, error) {
	buf := l.Borrow()[:4]
	defer l.Return(buf)

	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	rv := byteOrder.Uint32(buf)

	return rv, nil
}

// Uint64 reads eight bytes from the provided reader using a buffer from the
// free list, converts it to a number using the provided byte order, and returns
// the resulting uint64.
func (l binaryFreeList) Uint64(r io.Reader, byteOrder binary.ByteOrder) (uint64, error) {
	buf := l.Borrow()[:8]
	defer l.Return(buf)

	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	rv := byteOrder.Uint64(buf)

	return rv, nil
}

// PutUint8 copies the provided uint8 into a buffer from the free list and
// writes the resulting byte to the given writer.
func (l binaryFreeList) PutUint8(w io.Writer, val uint8) error {
	buf := l.Borrow()[:1]
	defer l.Return(buf)

	buf[0] = val
	_, err := w.Write(buf)

	return err
}

// PutUint16 serializes the provided uint16 using the given byte order into a
// buffer from the free list and writes the resulting two bytes to the given
// writer.
func (l binaryFreeList) PutUint16(w io.Writer, byteOrder binary.ByteOrder, val uint16) error {
	buf := l.Borrow()[:2]
	defer l.Return(buf)

	byteOrder.PutUint16(buf, val)
	_, err := w.Write(buf)

	return err
}

// PutUint32 serializes the provided uint32 using the given byte order into a
// buffer from the free list and writes the resulting four bytes to the given
// writer.
func (l binaryFreeList) PutUint32(w io.Writer, byteOrder binary.ByteOrder, val uint32) error {
	buf := l.Borrow()[:4]
	defer l.Return(buf)

	byteOrder.PutUint32(buf, val)
	_, err := w.Write(buf)

	return err
}

// PutUint64 serializes the provided uint64 using the given byte order into a
// buffer from the free list and writes the resulting eight bytes to the given
// writer.
func (l binaryFreeList) PutUint64(w io.Writer, byteOrder binary.ByteOrder, val uint64) error {
	buf := l.Borrow()[:8]
	defer l.Return(buf)

	byteOrder.PutUint64(buf, val)
	_, err := w.Write(buf)

	return err
}

const (
	// binaryFreeListMaxItems is the number of buffers to keep in the free
	// list to use for binary serialization and deserialization.
	binaryFreeListMaxItems = 1024
)

// binarySerializer provides a free list of buffers to use for serializing and
// deserializing primitive integer values to and from io.Readers and io.Writers.
var binarySerializer binaryFreeList = make(chan []byte, binaryFreeListMaxItems)

// errNonCanonicalVarInt is the common format string used for non-canonically
// encoded variable length integer errors.
var errNonCanonicalVarInt = "non-canonical varint %x - discriminant %x must " +
	"encode a value greater than %x"

// uint32Time represents a unix timestamp encoded with a uint32.  It is used as
// a way to signal the readElement function how to decode a timestamp into a Go
// time.Time since it is otherwise ambiguous.
type uint32Time time.Time

// int64Time represents a unix timestamp encoded with an int64.  It is used as
// a way to signal the readElement function how to decode a timestamp into a Go
// time.Time since it is otherwise ambiguous.
type int64Time time.Time

// makeEmptyMessage creates a message of the appropriate concrete type based
// on the command.
func makeEmptyMessage(chain Chain, command string) (wire.Message, error) {
	var msg wire.Message
	switch command {
	// Bisonwire variations
	case wire.CmdGetBlocks:
		msg = &wire.MsgGetBlocks{}

	case wire.CmdBlock:
		msg = &wire.MsgBlock{}

	case wire.CmdHeaders:
		msg = &wire.MsgHeaders{}

	case wire.CmdTx:
		msg = &wire.MsgTx{}

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
		return nil, wire.ErrUnknownMessage
	}
	return msg, nil
}
