// Package message defines the binary message format used to communicate
// between Expo clients.
package message

import (
	"encoding"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"unicode/utf8"
)

var ByteOrder = binary.BigEndian

type Message interface {
	Kind() int32
	// TODO: Equals is convenient for testing but is it necessary for
	// application functionality?
	// Could be useful for ensuring duplicate messages are not processed, might
	// as well keep it
	Equals(m Message) bool
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

// XXX: Update this when adding new message types
const (
	EchoKind int32 = iota
	StrokeKind
	EraseKind
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

// Point represents a 2D point with float32 coordinates.
type Point struct {
	X float32
	Y float32
}

// Color represents an RGBA color.
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// Stroke represents a drawing stroke (freehand or straight line).
type Stroke struct {
	StrokeID int64   // Unique identifier for this stroke
	Points   []Point // Array of points making up the stroke
	Color    Color   // RGBA color
	Width    float32 // Stroke width
}

func (_ *Stroke) Kind() int32 { return StrokeKind }

func (s *Stroke) Equals(m Message) bool {
	switch m := m.(type) {
	case *Stroke:
		if s.StrokeID != m.StrokeID || s.Width != m.Width {
			return false
		}
		if s.Color != m.Color {
			return false
		}
		if len(s.Points) != len(m.Points) {
			return false
		}
		for i := range s.Points {
			if s.Points[i] != m.Points[i] {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (s *Stroke) MarshalBinary() (data []byte, err error) {
	// Calculate size: 8 (StrokeID) + 4 (NumPoints) + 8*len(Points) + 4 (Color) + 4 (Width)
	size := 8 + 4 + 8*len(s.Points) + 4 + 4
	data = make([]byte, size)

	offset := 0

	// Write StrokeID
	ByteOrder.PutUint64(data[offset:], uint64(s.StrokeID))
	offset += 8

	// Write number of points
	ByteOrder.PutUint32(data[offset:], uint32(len(s.Points)))
	offset += 4

	// Write each point
	for _, p := range s.Points {
		ByteOrder.PutUint32(data[offset:], math.Float32bits(p.X))
		offset += 4
		ByteOrder.PutUint32(data[offset:], math.Float32bits(p.Y))
		offset += 4
	}

	// Write color
	data[offset] = s.Color.R
	offset++
	data[offset] = s.Color.G
	offset++
	data[offset] = s.Color.B
	offset++
	data[offset] = s.Color.A
	offset++

	// Write width
	ByteOrder.PutUint32(data[offset:], math.Float32bits(s.Width))

	return data, nil
}

func (s *Stroke) UnmarshalBinary(data []byte) error {
	if len(data) < 8+4+4+4 {
		return errors.New("stroke data too short")
	}

	offset := 0

	// Read StrokeID
	s.StrokeID = int64(ByteOrder.Uint64(data[offset:]))
	offset += 8

	// Read number of points
	numPoints := ByteOrder.Uint32(data[offset:])
	offset += 4

	// Validate remaining data
	expectedSize := 8 + 4 + 8*int(numPoints) + 4 + 4
	if len(data) != expectedSize {
		return errors.New("stroke data size mismatch")
	}

	// Read points
	s.Points = make([]Point, numPoints)
	for i := range s.Points {
		s.Points[i].X = math.Float32frombits(ByteOrder.Uint32(data[offset:]))
		offset += 4
		s.Points[i].Y = math.Float32frombits(ByteOrder.Uint32(data[offset:]))
		offset += 4
	}

	// Read color
	s.Color.R = data[offset]
	offset++
	s.Color.G = data[offset]
	offset++
	s.Color.B = data[offset]
	offset++
	s.Color.A = data[offset]
	offset++

	// Read width
	s.Width = math.Float32frombits(ByteOrder.Uint32(data[offset:]))

	return nil
}

// Erase represents a stroke erasure operation.
type Erase struct {
	StrokeID int64 // ID of stroke to erase
}

func (_ *Erase) Kind() int32 { return EraseKind }

func (e *Erase) Equals(m Message) bool {
	switch m := m.(type) {
	case *Erase:
		return e.StrokeID == m.StrokeID
	default:
		return false
	}
}

func (e *Erase) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 8)
	ByteOrder.PutUint64(data, uint64(e.StrokeID))
	return data, nil
}

func (e *Erase) UnmarshalBinary(data []byte) error {
	if len(data) != 8 {
		return errors.New("erase data must be 8 bytes")
	}
	e.StrokeID = int64(ByteOrder.Uint64(data))
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
	case StrokeKind:
		message = &Stroke{}
	case EraseKind:
		message = &Erase{}
	default:
		return nil, errors.New("Unknown message kind")
	}
	err = readMessage(r, header.Length, message)
	if err != nil {
		return nil, err
	}

	return message, nil
}
