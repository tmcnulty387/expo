package main

import (
	"image"
	"image/color"
	"log"
	"os"
	"strings"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gioui.org/op"
)

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
	strokes     [][]f32.Point
)

func loop(window *app.Window) error {
	var toggleSessionBtn widget.Clickable
	var sessionCodeInput widget.Editor
	sessionCodeInput.SingleLine = true
	sessionCodeInput.Submit = true
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	var ops op.Ops
	for {
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
				strokes = append(strokes, []f32.Point{e.Position})
			case pointer.Drag:
				if drawing {
					strokes[len(strokes)-1] = append(strokes[len(strokes)-1], e.Position)
				}
			case pointer.Release:
				if drawing {
					strokes[len(strokes)-1] = append(strokes[len(strokes)-1], e.Position)
					drawing = false
					log.Println("Stopped Drawing")
				}
			case pointer.Cancel:
				if drawing {
					if len(strokes[len(strokes)-1]) == 1 {
						strokes = strokes[:len(strokes)-1]
					}
					drawing = false
					log.Println("Cancelled Drawing")
				}
			}
			log.Println("Event: ", e)
		}
	}
	for _, stroke := range strokes {
		var path clip.Path
		path.Begin(ops)

		path.MoveTo(stroke[0])
		for _, p := range stroke[1:] {
			path.LineTo(p)
		}
		paint.FillShape(ops, Black,
			clip.Stroke{
				Path:  path.End(),
				Width: 4,
			}.Op())
	}
}

func sessionFooter(gtx layout.Context, th *material.Theme, code string) layout.Dimensions {
	if code == "" {
		return layout.Dimensions{}
	}

	height := gtx.Dp(unit.Dp(40))

	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = height
			gtx.Constraints.Max.Y = height

			// light footer background
			paint.FillShape(gtx.Ops, color.NRGBA{R: 245, G: 245, B: 245, A: 255},
				clip.Rect{Max: gtx.Constraints.Min}.Op())

			return layout.Dimensions{Size: gtx.Constraints.Min}
		}),
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.Y = height
			gtx.Constraints.Max.Y = height

			return layout.Inset{
				Left:   unit.Dp(12),
				Right:  unit.Dp(12),
				Top:    unit.Dp(10),
				Bottom: unit.Dp(10),
			}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, "Session Code: "+code)
				lbl.Color = color.NRGBA{R: 30, G: 30, B: 30, A: 255}
				return lbl.Layout(gtx)
			})
		}),
	)
}

func startApp() {
	go func() {
		window := new(app.Window)
		err := loop(window)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
