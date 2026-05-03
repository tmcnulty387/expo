// Definitions of message info.

package message

import (
	"encoding"
	"encoding/binary"
	"errors"
	"io"
	"unicode/utf8"
)

var ByteOrder = binary.BigEndian

type Message interface {
	Kind() int32
	// TODO: Equals is convenient for testing but is it necessary for
	// application functionality?
	Equals(m Message) bool
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// XXX: Update this when adding new message types
const (
	EchoKind int32 = iota
)

// Test message type.
type Echo struct {
	Text string
}

func (_ *Echo) Kind() int32 { return EchoKind }
func (echo *Echo) Equals(m Message) bool {
	switch m := m.(type) {
	case *Echo:
		return *echo == *m
	default:
		return false
	}
}
func (echo *Echo) MarshalBinary() (data []byte, err error) {
	return []byte(echo.Text), nil
}
func (echo *Echo) UnmarshalBinary(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("Invalid UTF-8")
	}
	echo.Text = string(data)
	return nil
}

type Header struct {
	Kind   int32
	Length int32
}

// Writes the message to the provided writer.
// First, a [Header] for the message is written.
// Then, the binary encoding of the message is written.
func Write(w io.Writer, message Message) error {
	marshalled, err := message.MarshalBinary()
	if err != nil {
		return err
	}
	header := Header{Kind: message.Kind(), Length: int32(len(marshalled))}
	err = binary.Write(w, ByteOrder, header)
	if err != nil {
		return err
	}
	n, err := w.Write(marshalled)
	if err != nil {
		return err
	} else if n < len(marshalled) {
		return io.ErrShortWrite
	}
	return nil
}

// Reads a [Message] of the provided length from the provided reader.
// Assumes that a header has already been read from [ReadHeader].
func readMessage(r io.Reader, length int32, message Message) error {
	buffer := make([]byte, length)
	_, err := io.ReadFull(r, buffer)
	if err != nil {
		return err
	}
	err = message.UnmarshalBinary(buffer)
	if err != nil {
		return err
	}
	return nil
}

// Reads a [Message] from the provided reader.
// Use a type switch to determine the concrete type.
// message is guaranteed to be non-nil if and only if err is nil.
func Read(r io.Reader) (message Message, err error) {
	var header Header
	err = binary.Read(r, ByteOrder, &header)
	if err != nil {
		return nil, err
	}

	// XXX: Update this when adding new message types
	switch header.Kind {
	case EchoKind:
		message = &Echo{}
	default:
		return nil, errors.New("Unknown message kind")
	}
	err = readMessage(r, header.Length, message)
	if err != nil {
		return nil, err
	}

	return message, nil
}
