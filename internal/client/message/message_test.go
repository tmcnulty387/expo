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
		&Stroke{
			StrokeID: 12345,
			Points: []Point{
				{X: 10.5, Y: 20.3},
				{X: 15.7, Y: 25.9},
				{X: 30.1, Y: 40.8},
			},
			Color: Color{R: 255, G: 128, B: 64, A: 255},
			Width: 4.5,
		},
		&Stroke{
			StrokeID: 67890,
			Points: []Point{
				{X: 0.0, Y: 0.0},
				{X: 100.0, Y: 100.0},
			},
			Color: Color{R: 0, G: 0, B: 0, A: 255},
			Width: 1.0,
		},
		&Stroke{
			StrokeID: 99999,
			Points:   []Point{{X: 50.0, Y: 50.0}},
			Color:    Color{R: 255, G: 0, B: 0, A: 255},
			Width:    10.0,
		},
		&Erase{StrokeID: 12345},
		&Erase{StrokeID: 0},
		&Erase{StrokeID: -1},
		&Textbox{TextboxID: 1, X: 100.0, Y: 200.5, FontSize: 14.0, Color: Color{R: 255, G: 128, B: 64, A: 255}, Text: "Hello, World!"},
		&Textbox{TextboxID: 2, X: 0.0, Y: 0.0, FontSize: 12.0, Color: Color{R: 0, G: 0, B: 0, A: 255}, Text: ""},
		&Textbox{TextboxID: 3, X: 50.0, Y: 75.0, FontSize: 24.0, Color: Color{R: 255, G: 0, B: 0, A: 255}, Text: "multi\nline\ntext"},
		&PeerAnnounce{Addr: "/ip4/1.2.3.4/tcp/4001/p2p/QmYyQSo1c1Ym7orWxLYvCrzRX5All4KXFwxR4XB5Ce4ony"},
		&PeerAnnounce{Addr: ""},
		&PeerIntroduction{Addrs: []string{"/ip4/1.2.3.4/tcp/4001/p2p/QmYyQSo1c1Ym7orWxLYvCrzRX5All4KXFwxR4XB5Ce4ony"}},
		&PeerIntroduction{Addrs: []string{}},
		&PeerIntroduction{Addrs: []string{"/ip4/1.2.3.4/tcp/4001/p2p/QmA", "/ip4/5.6.7.8/tcp/4001/p2p/QmB"}},
		&PeerList{Addrs: []string{"/ip4/1.2.3.4/tcp/4001/p2p/QmYyQSo1c1Ym7orWxLYvCrzRX5All4KXFwxR4XB5Ce4ony"}},
		&PeerList{Addrs: []string{}},
		&PeerList{Addrs: []string{"/ip4/1.2.3.4/tcp/4001/p2p/QmA", "/ip4/5.6.7.8/tcp/4001/p2p/QmB"}},
	} {
		testMessage(t, message)
	}
}
