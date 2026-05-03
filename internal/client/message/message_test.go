package message

import (
	"bytes"
	"testing"
)

func testMessage(t *testing.T, message Message) {
	t.Logf("testing: %+v", message)
	buf := new(bytes.Buffer)
	err := Write(buf, message)
	if err != nil {
		t.Error(err)
	}
	b := buf.Bytes()
	t.Log("buffer:", b)

	r := bytes.NewReader(b)

	message2, err := Read(r)
	if err != nil {
		t.Error(err)
	}

	if !message.Equals(message2) {
		t.Errorf("message != message2: %+v", message2)
	}
}

func TestMessagesWriteRead(t *testing.T) {
	// XXX: Update this when adding new message types
	for _, message := range []Message{
		&Echo{Text: "Hello, World!"},
		&Echo{Text: ""},
		&Echo{},
	} {
		testMessage(t, message)
	}
}
