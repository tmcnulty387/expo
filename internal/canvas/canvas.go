// Package canvas stores the serializable state of the shared whiteboard.
package canvas

import (
	"sync"

	"github.com/tmcnulty387/expo/internal/client/message"
)

var (
	mu        sync.Mutex
	strokes   []message.Stroke
	textboxes []message.Textbox
)

func UpsertStroke(stroke *message.Stroke) {
	if stroke == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	copy := cloneStroke(*stroke)
	for i := range strokes {
		if strokes[i].StrokeID == copy.StrokeID {
			strokes[i] = copy
			return
		}
	}
	strokes = append(strokes, copy)
}

func EraseStroke(strokeID int64) {
	mu.Lock()
	defer mu.Unlock()

	for i := 0; i < len(strokes); i++ {
		if strokes[i].StrokeID == strokeID {
			strokes = append(strokes[:i], strokes[i+1:]...)
			i--
		}
	}
}

func UpsertTextbox(textbox *message.Textbox) {
	if textbox == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	copy := *textbox
	for i := range textboxes {
		if textboxes[i].TextboxID == copy.TextboxID {
			textboxes[i] = copy
			return
		}
	}
	textboxes = append(textboxes, copy)
}

func EraseTextbox(textboxID int64) {
	mu.Lock()
	defer mu.Unlock()

	for i := 0; i < len(textboxes); i++ {
		if textboxes[i].TextboxID == textboxID {
			textboxes = append(textboxes[:i], textboxes[i+1:]...)
			i--
		}
	}
}

func Snapshot() []message.Message {
	mu.Lock()
	defer mu.Unlock()

	objects := make([]message.Message, 0, len(strokes)+len(textboxes))
	for _, s := range strokes {
		copy := cloneStroke(s)
		objects = append(objects, &copy)
	}
	for _, t := range textboxes {
		copy := t
		objects = append(objects, &copy)
	}
	return objects
}

func Clear() {
	mu.Lock()
	defer mu.Unlock()

	strokes = nil
	textboxes = nil
}

func cloneStroke(stroke message.Stroke) message.Stroke {
	stroke.Points = append([]message.Point(nil), stroke.Points...)
	return stroke
}
