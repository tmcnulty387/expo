package ui

import (
	"context"
	"image"
	"image/color"
	"log"
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
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// stroke represents a drawn line
type stroke struct {
	points []f32.Point
	col    color.NRGBA
}

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
	drawColor   = Black
	strokes     []stroke
)

func Loop(ctx context.Context) error {
	window := new(app.Window)

	var toggleSessionBtn widget.Clickable
	var sessionCodeInput widget.Editor
	sessionCodeInput.SingleLine = true
	sessionCodeInput.Submit = true
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	// colour palette setup vars (needs to be persistent across frames)
	colorChoices := []color.NRGBA{Black, Red, Green, Blue, Yellow, Cyan, Magenta, Orange}
	var colorBtns = make([]widget.Clickable, len(colorChoices))

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

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					draw(gtx.Ops, gtx.Source, gtx.Constraints.Max)
					return layout.Dimensions{Size: gtx.Constraints.Max}
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if inSession {
									btn := material.Button(th, &toggleSessionBtn, "Stop Session")
									btn.TextSize = unit.Sp(14)
									btn.Background = Red
									return btn.Layout(gtx)
								} else {
									btn := material.Button(th, &toggleSessionBtn, "Start Session")
									btn.TextSize = unit.Sp(14)
									return btn.Layout(gtx)
								}
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								if inSession {
									textValue := sessionCode
									if textValue == "" {
										textValue = "(not set)"
									}
									lbl := material.Body1(th, "Session: "+textValue)
									return layout.UniformInset(unit.Dp(9)).Layout(gtx, lbl.Layout)
								}

								borderColor := Gray
								if gtx.Source.Focused(&sessionCodeInput) {
									borderColor = Blue
								}

								border := widget.Border{
									Color:        borderColor,
									Width:        unit.Dp(1),
									CornerRadius: unit.Dp(6),
								}

								gtx.Constraints.Min.X = gtx.Dp(180)

								return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(9)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										return material.Editor(th, &sessionCodeInput, "Session Code").Layout(gtx)
									})
								})
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								// colour picker tool: label + colour swatches
								return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									children := make([]layout.FlexChild, 0, len(colorChoices)+1)
									children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										lbl := material.Body1(th, "Color:")
										return layout.UniformInset(unit.Dp(6)).Layout(gtx, lbl.Layout)
									}))
									for i := range colorChoices {
										ci := i
										children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											btn := &colorBtns[ci]
											for btn.Clicked(gtx) {
												drawColor = colorChoices[ci]
											}
											return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
												mbtn := material.Button(th, btn, "")
												mbtn.Background = colorChoices[ci]
												return mbtn.Layout(gtx)
											})
										}))
									}
									return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, children...)
								})
							}),
						)
					})
				}),
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
		// TODO: I think we block here -- do we need to propogate context to
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
				drawing = true
				log.Println("Started Drawing")
				// start new stroke with current drawing colour
				strokes = append(strokes, stroke{points: []f32.Point{e.Position}, col: drawColor})
			case pointer.Drag:
				if drawing {
					s := &strokes[len(strokes)-1]
					s.points = append(s.points, e.Position)
				}
			case pointer.Release:
				if drawing {
					s := &strokes[len(strokes)-1]
					s.points = append(s.points, e.Position)
					drawing = false
					log.Println("Stopped Drawing")
				}
			case pointer.Cancel:
				if drawing {
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
				Width: 4,
			}.Op())
	}
}
