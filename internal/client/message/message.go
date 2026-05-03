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
	Kind() uint32
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

const (
	EchoKind uint32 = iota
)

type Echo struct {
	Text string
}

func (_ *Echo) Kind() uint32 { return EchoKind }
func (m *Echo) MarshalBinary() (data []byte, err error) {
	return []byte(m.Text), nil
}
func (m *Echo) UnmarshalBinary(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("Invalid UTF-8")
	}
	m.Text = string(data)
	return nil
}

type Header struct {
	Kind   uint32
	Length uint32
}

// Writes the message to the provided writer.
// First, a [Header] for the message is written.
// Then, the binary encoding of the message is written.
func Write(w io.Writer, message Message) error {
	marshalled, err := message.MarshalBinary()
	if err != nil {
		return err
	}
	header := Header{Kind: message.Kind(), Length: uint32(len(marshalled))}
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

// Reads the message from the provided reader.
// TODO: What should this look like? How does writing to a generic work?
func Read(r io.Reader, message Message) error {
	// TODO: implement
	return errors.New("Unimplemented")
}
