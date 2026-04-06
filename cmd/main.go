package main

import (
	"image"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"gioui.org/op"
)

var (
	Red     = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	Green   = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	Blue    = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	Yellow  = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	Cyan    = color.NRGBA{R: 0, G: 255, B: 255, A: 255}
	Magenta = color.NRGBA{R: 255, G: 0, B: 255, A: 255}
	Black   = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	White   = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	Gray    = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	Orange  = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	tag     = new(int)
	drawing = false
	strokes [][]f32.Point
)

func main() {
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

func loop(window *app.Window) error {
	// theme := material.NewTheme()
	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			draw(gtx.Ops, gtx.Source, gtx.Constraints.Max)
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
