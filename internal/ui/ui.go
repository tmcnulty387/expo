package ui

import (
	"context"
	"image"
	"image/color"
	"log"
	"math"
	"strings"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// stroke represents a drawn line
type stroke struct {
	points []f32.Point
	col    color.NRGBA
	width  float32
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
	// straight line vs free draw mode state
	lineMode      = false
	previewActive = false
	lineStart     f32.Point
	previewEnd    f32.Point
	eraserMode            = false
	eraserSize    float32 = 12
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
	// colour palette setup vars (needs to be persistent across frames)
	colorChoices := []color.NRGBA{Black, Red, Green, Blue, Yellow, Cyan, Magenta, Orange}
	var colorBtns = make([]widget.Clickable, len(colorChoices))
	// buttons for decreasing or increasing stroke width
	var decWidth widget.Clickable
	var incWidth widget.Clickable
	// toggle for line mode
	var lineModeBtn widget.Clickable
	// eraser controls
	var eraserBtn widget.Clickable
	var decEraser widget.Clickable
	var incEraser widget.Clickable

	var ops op.Ops
	for {
		select {
		case <-ctx.Done():
			// TODO: Is any other shutdown logic required here?
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

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(TopToolbar(th, &lineModeBtn, &eraserBtn)),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
						layout.Rigid(Sidebar(th, colorChoices, colorBtns, &customColorInput, &decWidth, &incWidth)),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							draw(gtx.Ops, gtx.Source, gtx.Constraints.Max)
							return layout.Dimensions{Size: gtx.Constraints.Max}
						}),
					)
				}),
				layout.Rigid(BottomControls(th, &toggleSessionBtn, &sessionCodeInput, &decEraser, &incEraser, &lineModeBtn, &eraserBtn)),
			)
			e.Frame(gtx.Ops)
		}
	}
}

func draw(ops *op.Ops, source input.Source, size image.Point) {
	// Confine the area of interest to a 100x100 rectangle.
	defer clip.Rect{Max: size}.Push(ops).Pop()

	// Declare `tag` as being one of the targets.
	event.Op(ops, tag)

	// Process events that arrived between the last frame and this one.
	for {
		// TODO: I think we block here -- do we need to propagate context to
		// this function as well?
		ev, ok := source.Event(pointer.Filter{
			Target: tag,
			Kinds:  pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
		})
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Press:
				if eraserMode { // eraser
					eraseAt(e.Position)
				} else if lineMode { // straight line
					// start line preview
					previewActive = true
					lineStart = e.Position
					previewEnd = e.Position
					log.Println("Line preview started")
				} else { // freehand drawing
					drawing = true
					log.Println("Started Drawing")
					// start new stroke with current drawing colour and width
					strokes = append(strokes, stroke{points: []f32.Point{e.Position}, col: drawColor, width: strokeWidth})
				}
			case pointer.Drag:
				if eraserMode {
					eraseAt(e.Position)
				} else if lineMode && previewActive {
					previewEnd = e.Position
				} else if drawing {
					s := &strokes[len(strokes)-1]
					s.points = append(s.points, e.Position)
				}
			case pointer.Release:
				if eraserMode {
					// nothing special on release for eraser
				} else if lineMode && previewActive {
					// commit straight line as a two-point stroke
					strokes = append(strokes, stroke{points: []f32.Point{lineStart, e.Position}, col: drawColor, width: strokeWidth})
					previewActive = false
					log.Println("Committed straight line")
				} else if drawing {
					s := &strokes[len(strokes)-1]
					s.points = append(s.points, e.Position)
					drawing = false
					log.Println("Stopped Drawing")
				}
			case pointer.Cancel:
				if eraserMode {
					// nothing specific for eraser
				} else if lineMode && previewActive {
					previewActive = false
					log.Println("Cancelled Line Preview")
				} else if drawing {
					if len(strokes[len(strokes)-1].points) == 1 {
						strokes = strokes[:len(strokes)-1]
					}
					drawing = false
					log.Println("Cancelled Drawing")
				}
			}
			log.Println("Event: ", e)
		}
	}
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
}

// eraseAt removes any stroke that has a point within eraserSize of pos
func eraseAt(eraserPos f32.Point) {
	r2 := eraserSize * eraserSize
	updatedStrokes := strokes[:0] // stores any strokes that weren't erased
	for _, s := range strokes {
		hit := false
		// first check for freehand line in range of eraserPos
		for _, p := range s.points {
			dx := p.X - eraserPos.X
			dy := p.Y - eraserPos.Y
			if dx*dx+dy*dy <= r2 {
				hit = true
				break
			}
		}
		// else check if it's a straight line in range of eraserPos
		// for now, just checking for eraserPos being exactly on the line segment
		if !hit && len(s.points) == 2 {
			// cross product magic
			a, b := s.points[0], s.points[1]
			dxc := eraserPos.X - a.X
			dyc := eraserPos.Y - b.Y
			dxl := b.X - a.X
			dyl := b.Y - a.Y
			cross := dxc*dyl - dyc*dxl
			if cross == 0 { // lies on the line
				// now check that it's between the line endpoints
				if math.Abs(float64(dxl)) >= math.Abs(float64(dyl)) {
					// line is "more horizontal than vertical"
					// so compare x coords
					if dxl > 0 {
						hit = a.X <= eraserPos.X && eraserPos.X <= b.X
					} else {
						hit = b.X <= eraserPos.X && eraserPos.X <= a.X
					}
				} else {
					// line is "more vertical than horizontal"
					// so compare y coords
					if dyl > 0 {
						hit = a.Y <= eraserPos.Y && eraserPos.Y <= b.Y
					} else {
						hit = b.Y <= eraserPos.Y && eraserPos.Y <= a.Y
					}
				}
				// after all that, hit should only be true if the eraser is on the line
			}
		}
		if !hit {
			// if this stroke should not be erased, add it back to the updated list of strokes
			updatedStrokes = append(updatedStrokes, s)
		}
	}
	strokes = updatedStrokes
}
