package ui

/*
Author: Rina Peshori
*/

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/tmcnulty387/expo/internal/client/message"
)

func strokeToMessage(s stroke) *message.Stroke {
	points := make([]message.Point, len(s.points))
	for i, p := range s.points {
		points[i] = message.Point{X: p.X, Y: p.Y}
	}
	return &message.Stroke{
		StrokeID: s.id,
		Points:   points,
		Color:    message.Color{R: s.col.R, G: s.col.G, B: s.col.B, A: s.col.A},
		Width:    s.width,
	}
}

func strokeFromMessage(m message.Stroke) stroke {
	points := make([]f32.Point, len(m.Points))
	for i, p := range m.Points {
		points[i] = f32.Point{X: p.X, Y: p.Y}
	}
	return stroke{
		id:     m.StrokeID,
		points: points,
		col:    color.NRGBA{R: m.Color.R, G: m.Color.G, B: m.Color.B, A: m.Color.A},
		width:  m.Width,
	}
}

func textboxToMessage(t textbox) *message.Textbox {
	return &message.Textbox{
		TextboxID: t.id,
		X:         t.pos.X,
		Y:         t.pos.Y,
		FontSize:  float32(t.theme.TextSize),
		Color:     message.Color{R: t.theme.Fg.R, G: t.theme.Fg.G, B: t.theme.Fg.B, A: t.theme.Fg.A},
		Text:      t.text.Text(),
	}
}

func textboxFromMessage(m message.Textbox, th material.Theme) textbox {
	editor := new(widget.Editor)
	editor.SingleLine = false
	editor.ReadOnly = true
	editor.Insert(m.Text)
	th.TextSize = unit.Sp(m.FontSize)
	th.Fg = color.NRGBA{R: m.Color.R, G: m.Color.G, B: m.Color.B, A: m.Color.A}
	return textbox{
		id:    m.TextboxID,
		text:  editor,
		theme: th,
		pos:   f32.Point{X: m.X, Y: m.Y},
	}
}

// disableAllModes disables all modes (draw, line, eraser...)
// this is in one function so as to improve maintainability (little closer to single responsibility)
// caller functions are responsible for turning back on the appropriate mode
func disableAllModes() {
	drawMode = false
	lineMode = false
	eraserMode = false
	textMode = false
}

// parseHexColor parses 6- or 8-digit hex color strings like "#RRGGBB" or "RRGGBBAA"
func parseHexColor(s string) (color.NRGBA, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 && len(s) != 8 {
		return color.NRGBA{}, fmt.Errorf("invalid hex length")
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.NRGBA{}, err
	}
	if len(s) == 6 {
		r := uint8(v >> 16)
		g := uint8((v >> 8) & 0xFF)
		b := uint8(v & 0xFF)
		return color.NRGBA{R: r, G: g, B: b, A: 255}, nil
	}
	r := uint8(v >> 24)
	g := uint8((v >> 16) & 0xFF)
	b := uint8((v >> 8) & 0xFF)
	a := uint8(v & 0xFF)
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

// isErasableLine returns a bool value representing whether or not the given stroke, s, is an erasable freehand line
// based on the current position of the eraser, its radius squared, and a given stroke
func isErasableFreehand(s stroke, r2 float32) bool {
	for _, p := range s.points {
		dx := p.X - eraserPos.X
		dy := p.Y - eraserPos.Y
		if dx*dx+dy*dy <= r2 {
			return true
		}
	}
	// else...
	return false
}

// isErasableLine returns a bool value representing whether or not the given stroke, s, is an erasable straight line
// based on the current position of the eraser, its radius squared, and a given stroke
func isErasableLine(s stroke, r2 float32) bool {
	// https://www.geeksforgeeks.org/dsa/minimum-distance-from-a-point-to-the-line-segment-using-vectors/
	a, b := s.points[0], s.points[1]
	// define vector AB
	abx := b.X - a.X
	aby := b.Y - a.Y
	// define vector AC (a -> eraserPos)
	acx := eraserPos.X - a.X
	acy := eraserPos.Y - a.Y
	// define vector BC (b -> eraserPos)
	bcx := eraserPos.X - b.X
	bcy := eraserPos.Y - b.Y
	// calculate dot products
	ab_bc := abx*bcx + aby*bcy
	ab_ac := abx*acx + aby*acy
	// minimum distance from point to line segment, squared
	distanceSqrd := float32(0)
	if ab_bc > 0 { // Case 1: B is closest point
		y := eraserPos.Y - b.Y
		x := eraserPos.X - b.X
		distanceSqrd = x*x + y*y
	} else if ab_ac < 0 { // Case 2: A is closest point
		y := eraserPos.Y - a.Y
		x := eraserPos.X - a.X
		distanceSqrd = x*x + y*y
	} else { // Case 3: C is perpendicular to AB, must calculate closest point
		mod := math.Sqrt(float64(abx*abx + aby*aby))
		distToClosest := float32(math.Abs(float64(abx*acy-aby*acx)) / mod)
		distanceSqrd = distToClosest * distToClosest
	}
	// check distance against eraser size
	if distanceSqrd <= r2 {
		return true
	}
	// else...
	return false
}

// isErasableTextbox returns a bool value representing whether or not the given textbox, t, is currently erasable
// note that only the TOPMOST textbox hit will be erased for any one erase operation
// since operations occur so quickly (within ms) this will not be visually apparent to users
// based on the current position of the eraser and its radius squared
func isErasableTextbox(t textbox, r2 float32) bool {
	padding := math.Sqrt(float64(r2)) // padding should be of size = radius
	hitTextboxIdx := textboxHit(eraserPos.X, eraserPos.Y, float32(padding))
	if hitTextboxIdx == -1 { // make sure we don't panic
		return false
	}
	return textboxes[hitTextboxIdx].id == t.id
}
