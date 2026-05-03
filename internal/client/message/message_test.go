package message

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestWrite(t *testing.T) {
	buf := new(bytes.Buffer)
	message := Echo{Text: "Hello, World!"}
	err := Write(buf, &message)
	if err != nil {
		t.Error(err)
	}
	b := buf.Bytes()
	t.Log("buffer:", b)

	r := bytes.NewReader(b)

	var header Header
	err = binary.Read(r, ByteOrder, &header)
	if err != nil {
		t.Error(err)
	}
	if header.Kind != EchoKind {
		t.Error("header.Kind != EchoKind")
	}
	if header.Length != uint32(len(message.Text)) {
		t.Error("header.Length != len(message.Text)")
	}

	payloadBytes, err := io.ReadAll(r)
	if err != nil {
		t.Error(err)
	}
	var message2 Echo
	err = message2.UnmarshalBinary(payloadBytes)
	if err != nil {
		t.Error(err)
	}
	if message != message2 {
		t.Errorf("message != message2: %+v", message2)
	}
}
