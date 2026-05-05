package ui

/*
Author: Rina Peshori
*/

import (
	"context"
	"image"
	"image/color"
	"log"
	"math"
	"strings"
	"unicode/utf8"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// stroke represents a drawn line
type stroke struct {
	points []f32.Point
	col    color.NRGBA
	width  float32
}

type textbox struct {
	text     *widget.Editor
	theme    material.Theme
	pos      f32.Point // top-left position of the textbox (relative to the drawing area)
	size     image.Point
	dragging bool
}

// for window title display
const appTitle = "EXPO"

var (
	Red         = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	Green       = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	Blue        = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	Yellow      = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	Cyan        = color.NRGBA{R: 0, G: 255, B: 255, A: 255}
	Magenta     = color.NRGBA{R: 255, G: 0, B: 255, A: 255}
	Black       = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	White       = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	Gray        = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	Orange      = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	tag         = new(int)
	drawing     = false
	inSession   = false
	sessionCode string
	drawColor           = Black
	strokeWidth float32 = 4
	strokes     []stroke
	textboxes   []textbox

	activeTextbox       = -1
	tbDragOffset        f32.Point
	insertingText       = false
	drawMode            = true // defaults to draw state
	lineMode            = false
	previewActive       = false
	lineStart           f32.Point
	previewEnd          f32.Point
	eraserMode          = false
	eraserPreviewActive = false
	eraserPos           f32.Point
	eraserSize          float32 = 12
	textMode                    = false
	fontSize            float32 = 12
)

func Loop(ctx context.Context) error {
	window := new(app.Window)
	window.Option(app.Title(appTitle))

	var toggleSessionBtn widget.Clickable
	var sessionCodeInput widget.Editor
	sessionCodeInput.SingleLine = true
	sessionCodeInput.Submit = true
	var customColorInput widget.Editor
	customColorInput.SingleLine = true
	customColorInput.Submit = true
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	inactiveTh := material.NewTheme() // colour theme for "inactive" buttons
	inactiveTh.Shaper = th.Shaper
	inactiveTh.Palette.ContrastBg = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
	textTh := material.NewTheme() // currently selected theme for user-editable text
	textTh.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	// colour palette setup vars (needs to be persistent across frames)
	colorChoices := []color.NRGBA{Black, Red, Green, Blue, Yellow, Cyan, Magenta, Orange}
	var colorBtns = make([]widget.Clickable, len(colorChoices))
	// buttons for decreasing or increasing stroke width
	var decWidth widget.Clickable
	var incWidth widget.Clickable
	// toggle for freehand draw mode
	var drawBtn widget.Clickable
	// toggle for line mode
	var lineBtn widget.Clickable
	// eraser controls
	var eraserBtn widget.Clickable
	var decEraser widget.Clickable
	var incEraser widget.Clickable
	// text mode controls
	var textBtn widget.Clickable
	var textInput widget.Editor
	textInput.SingleLine = false // allow for multi-line text input
	var decFont widget.Clickable
	var incFont widget.Clickable
	var textPreview widget.Editor
	textPreview.SingleLine = false
	textPreview.ReadOnly = true
	var insertTextBtn widget.Clickable

	var ops op.Ops
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			for toggleSessionBtn.Clicked(gtx) {
				if inSession {
					log.Println("Stopping Session")
				} else {
					log.Println("Starting Session")
				}
				inSession = !inSession
			}

			for {
				ev, ok := sessionCodeInput.Update(gtx)
				if !ok {
					break
				}
				if sub, ok := ev.(widget.SubmitEvent); ok {
					sessionCode = strings.TrimSpace(sub.Text)
					log.Println("Submitted Session Code: ", sessionCode)
				}
			}

			for {
				ev, ok := customColorInput.Update(gtx)
				if !ok {
					break
				}
				if sub, ok := ev.(widget.SubmitEvent); ok {
					hex := strings.TrimSpace(sub.Text)
					if c, err := parseHexColor(hex); err == nil {
						drawColor = c
						log.Println("Set custom draw color:", hex)
					} else {
						log.Println("Invalid hex color:", hex, err)
					}
				}
			}

			for {
				ev, ok := textInput.Update(gtx)
				if !ok {
					break
				}
				if _, ok := ev.(widget.ChangeEvent); ok {
					// first clear all characters from preview
					textPreview.Delete(-textPreview.Len())
					// now update preview
					customText := strings.TrimSpace(textInput.Text())
					numAffected := textPreview.Insert(customText)
					log.Println("Set custom text:", customText, " with affected characters: ", numAffected)
				}
			}

			for insertTextBtn.Clicked(gtx) {
				insertingText = true
			}

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(TopToolbar(th, inactiveTh, &drawBtn, &lineBtn, &eraserBtn, &textBtn)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(Sidebar(th, textTh, colorChoices, colorBtns, &customColorInput, &textInput, &textPreview, &decWidth, &incWidth, &decEraser, &incEraser, &decFont, &incFont, &insertTextBtn)),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							draw(gtx, textTh, &textPreview)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						}),
					)
				}),
				layout.Rigid(BottomControls(th, &toggleSessionBtn, &sessionCodeInput)),
			)
			e.Frame(gtx.Ops)

			// reset necessary variables every frame
			insertingText = false
			textTh.Fg = drawColor
			textTh.TextSize = unit.Sp(fontSize)
		}
	}
}

func draw(gtx layout.Context, textTh *material.Theme, textPreview *widget.Editor) {
	ops := gtx.Ops
	source := gtx.Source
	size := gtx.Constraints.Max

	// Confine the area of interest to the whole drawing area ("whiteboard").
	defer clip.Rect{Max: size}.Push(ops).Pop()

	// Declare `tag` as being one of the targets.
	event.Op(ops, tag)

	// Process events that arrived between the last frame and this one.
	for {
		ev, ok := source.Event(pointer.Filter{
			Target: tag,
			Kinds:  pointer.Move | pointer.Press | pointer.Drag | pointer.Release | pointer.Leave,
		})
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Move:
				if eraserMode { // eraser preview
					eraserPreviewActive = true
					eraserPos = e.Position
					log.Println("Eraser preview started")
				} else {
					eraserPreviewActive = false
				}
				hit := false
				if textMode {
					// if hovering over a textbox, switch cursor to indicate draggable
					idx := textboxHit(e.Position.X, e.Position.Y)
					if idx != -1 {
						pointer.CursorGrab.Add(ops)
						hit = true
					}
				}
				if !hit {
					pointer.CursorDefault.Add(ops)
				}
			case pointer.Press:
				if eraserMode { // eraser
					eraserPreviewActive = true
					eraserPos = e.Position
					log.Println("Started Erasing")
					eraseAt()
				} else if lineMode { // straight line
					// start line preview
					previewActive = true
					lineStart = e.Position
					previewEnd = e.Position
					log.Println("Line preview started")
				} else if drawMode { // freehand drawing
					drawing = true
					log.Println("Started Drawing")
					// start new stroke with current drawing colour and width
					strokes = append(strokes, stroke{points: []f32.Point{e.Position}, col: drawColor, width: strokeWidth})
				} else if textMode { // text edit
					// check if the press hits any textbox
					// (topmost textbox will be hit first)
					idx := textboxHit(e.Position.X, e.Position.Y)
					if idx != -1 {
						// update textboxes so this one is now topmost
						t := textboxes[idx]
						textboxes = append(textboxes[:idx], textboxes[idx+1:]...)
						textboxes = append(textboxes, t)
						activeTextbox = len(textboxes) - 1
						tbDragOffset = f32.Point{X: e.Position.X - t.pos.X, Y: e.Position.Y - t.pos.Y}
						textboxes[activeTextbox].dragging = true
						pointer.CursorGrabbing.Add(ops)
						log.Println("Started dragging textbox")
					}
				}
			case pointer.Drag:
				if eraserMode {
					eraseAt()
					eraserPos = e.Position
				} else if lineMode && previewActive {
					previewEnd = e.Position
				} else if drawing {
					s := &strokes[len(strokes)-1]
					s.points = append(s.points, e.Position)
				} else if textMode && activeTextbox != -1 { // dragging a textbox
					pointer.CursorGrabbing.Add(ops)
					tb := &textboxes[activeTextbox]
					tb.pos = f32.Point{X: e.Position.X - tbDragOffset.X, Y: e.Position.Y - tbDragOffset.Y}
				}
			case pointer.Release:
				if eraserMode {
					log.Println("Stopped Erasing")
				} else if lineMode && previewActive {
					// commit straight line as a two-point stroke
					strokes = append(strokes, stroke{points: []f32.Point{lineStart, e.Position}, col: drawColor, width: strokeWidth})
					previewActive = false
					// TODO: send Message
					log.Println("Committed straight line")
				} else if drawing {
					s := &strokes[len(strokes)-1]
					s.points = append(s.points, e.Position)
					drawing = false
					log.Println("Stopped Drawing")
				} else if textMode && activeTextbox != -1 {
					// TODO: send Message
					tb := &textboxes[activeTextbox]
					tb.dragging = false
					activeTextbox = -1
					pointer.CursorDefault.Add(ops)
					log.Println("Stopped dragging textbox")
				}
			case pointer.Leave:
				if eraserMode {
					eraserPreviewActive = false
				}
			}
			log.Println("Event: ", e)
		}
	}

	// draw committed strokes
	for _, s := range strokes {
		if len(s.points) == 0 {
			continue
		}
		var path clip.Path
		path.Begin(ops)

		path.MoveTo(s.points[0])
		for _, p := range s.points[1:] {
			path.LineTo(p)
		}
		paint.FillShape(ops, s.col,
			clip.Stroke{
				Path:  path.End(),
				Width: s.width,
			}.Op())
	}

	// render preview line (for straight-line mode)
	if previewActive {
		var p clip.Path
		p.Begin(ops)
		p.MoveTo(lineStart)
		p.LineTo(previewEnd)
		c := drawColor
		c.A = 128 // preview at half opacity
		paint.FillShape(ops, c,
			clip.Stroke{
				Path:  p.End(),
				Width: strokeWidth,
			}.Op())
	}

	// preview for eraser mode
	if eraserPreviewActive {
		// get circle data
		circle, circleColor := previewErase()
		// render preview circle
		paint.FillShape(ops, circleColor, circle.Op(ops))
	}

	// insert any new textboxes
	if insertingText {
		previewText := textPreview.Text()
		if previewText != "" {
			newTB := new(widget.Editor)
			newTB.SingleLine = false
			newTB.ReadOnly = true
			newTB.Insert(previewText)
			// place roughly in drawing area's center
			pos := f32.Point{X: float32(size.X) / 2, Y: float32(size.Y) / 2}
			tb := textbox{text: newTB, theme: *textTh, pos: pos}
			// give textbox a size (heuristics-calculated)
			computedWidth, computedHeight := getTextboxSize(&gtx, &tb)
			tb.size = image.Point{X: computedWidth, Y: computedHeight}
			// TODO: send Message
			textboxes = append(textboxes, tb)
			log.Println("Inserted textbox with text: ", previewText, " and size: ", tb.size.X, ", ", tb.size.Y)
		}
	}

	// render textboxes (basic rendering using material.Editor at stored positions)
	for i := range textboxes {
		tb := &textboxes[i]
		// push the textbox insert operation to the ops stack, with a position offset of the textbox's pos
		call := op.Offset(image.Point{X: tb.pos.Round().X, Y: tb.pos.Round().Y}).Push(ops)
		material.Editor(&tb.theme, tb.text, "").Layout(gtx) // get dimensions of the textbox
		call.Pop()
	}

	// must refresh event space constantly to allow for nice dragging
	event.Op(ops, tag)
}

// getTextboxSize computes a natural width & height for a given textbox (heuristics-based)
func getTextboxSize(gtx *layout.Context, tb *textbox) (int, int) {
	maxAllowedWidth := min(gtx.Constraints.Max.X, 400)

	approxCharWidth := float32(fontSize) * 0.6
	paddingX := 8
	paddingY := 8

	var textStr string
	if tb.text != nil {
		textStr = tb.text.Text()
	}
	lines := strings.Split(textStr, "\n")

	// determine the longest line in runes to estimate a natural width
	longestRunes := 0
	for _, ln := range lines {
		rc := utf8.RuneCountInString(ln)
		if rc > longestRunes {
			longestRunes = rc
		}
	}
	if longestRunes < 1 {
		longestRunes = 1
	}

	initialWidth := max(int(float32(longestRunes)*approxCharWidth)+paddingX, 20)
	computedWidth := min(initialWidth, maxAllowedWidth)

	// now compute wrapped lines using the computed width
	charsPerLine := int(math.Max(1, math.Floor(float64(computedWidth)/float64(approxCharWidth))))
	totalLines := 0
	for _, ln := range lines {
		runeCount := utf8.RuneCountInString(ln)
		if runeCount == 0 {
			totalLines += 1
			continue
		}
		wraps := int(math.Ceil(float64(runeCount) / float64(charsPerLine)))
		totalLines += wraps
	}
	if totalLines < 1 {
		totalLines = 1
	}
	lineHeight := int(float32(fontSize) * 1.4)
	computedHeight := min(max(totalLines*lineHeight+paddingY, 20), gtx.Constraints.Max.Y)

	return computedWidth, computedHeight
}

// textboxHit checks if the current cursor position is above a textbox, returns index of first hit
func textboxHit(currX, currY float32) int {
	for i := len(textboxes) - 1; i >= 0; i-- {
		tb := &textboxes[i]
		w, h := float32(tb.size.X), float32(tb.size.Y)
		if w <= 0 || h <= 0 {
			// size not measured yet, skip this textbox
			continue
		}
		if currX >= tb.pos.X && currX <= tb.pos.X+w &&
			currY >= tb.pos.Y && currY <= tb.pos.Y+h {
			return i
		}
	}
	return -1
}

// previewErase returns data for a semi-transparent preview circle on the screen where the user's eraser is placed
func previewErase() (clip.Ellipse, color.NRGBA) {
	// generate preview circle (for eraser mode)
	topLeft := image.Pt(eraserPos.Round().X-int(eraserSize), eraserPos.Round().Y-int(eraserSize))
	bottomRight := image.Pt(eraserPos.Round().X+int(eraserSize), eraserPos.Round().Y+int(eraserSize))
	circle := clip.Ellipse{Min: topLeft, Max: bottomRight}
	circleColor := color.NRGBA{R: 222, G: 222, B: 222, A: 128} // light gray, semi-transparent
	return circle, circleColor
}

// eraseAt removes any stroke that has a point within eraserSize of pos
func eraseAt() {
	r2 := eraserSize * eraserSize
	updatedStrokes := strokes[:0] // stores any strokes that weren't erased
	for _, s := range strokes {
		// first check for freehand line in range of eraserPos
		hit := isErasableFreehand(s, r2)
		// else check if it's a straight line in range of eraserPos
		if !hit && len(s.points) == 2 {
			hit = isErasableLine(s, r2)
		}
		if !hit {
			// if this stroke should not be erased, add it back to the updated list of strokes
			updatedStrokes = append(updatedStrokes, s)
		}
	}
	// TODO: send Message
	strokes = updatedStrokes
}
