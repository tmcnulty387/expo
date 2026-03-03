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
	Red        = color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	Green      = color.NRGBA{R: 0, G: 255, B: 0, A: 255}
	Blue       = color.NRGBA{R: 0, G: 0, B: 255, A: 255}
	Yellow     = color.NRGBA{R: 255, G: 255, B: 0, A: 255}
	Cyan       = color.NRGBA{R: 0, G: 255, B: 255, A: 255}
	Magenta    = color.NRGBA{R: 255, G: 0, B: 255, A: 255}
	Black      = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	White      = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	Gray       = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
	Orange     = color.NRGBA{R: 255, G: 165, B: 0, A: 255}
	tag        = new(int)
	drawing    = false
	current    []f32.Point
	strokes    [][]f32.Point
	beginCount = 0
	endCount   = 0
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
			Kinds:  pointer.Press | pointer.Drag | pointer.Release,
		})
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Press | pointer.Drag:
				if !drawing {
					current = current[:0]
					drawing = true
				}
				current = append(current, e.Position)
			case pointer.Drag:
				if !drawing {
					current = current[:0]
					drawing = true
				}
				current = current[:0]
				current = append(current, e.Position)
			case pointer.Release:
				current = append(current, e.Position)
				strokes = append(strokes, current)
			}
			log.Println("Event: ", e)
		}
	}
	for _, stroke := range strokes {
		var path clip.Path
		path.Begin(ops)
		beginCount++

		path.MoveTo(stroke[0])
		for _, p := range stroke[1:] {
			path.LineTo(p)
		}
		log.Println("End stroke")
		path.Close()
		endCount++
		paint.FillShape(ops, Black,
			clip.Stroke{
				Path:  path.End(),
				Width: 4,
			}.Op())
	}
	// Draw the button.
	// var c color.NRGBA
	// if pressed {
	// 	c = color.NRGBA{R: 0xFF, A: 0xFF}
	// } else {
	// 	c = color.NRGBA{G: 0xFF, A: 0xFF}
	// }
	// paint.ColorOp{Color: Black}.Add(ops)
	// paint.PaintOp{}.Add(ops)
}

// func strokeTriangle(ops *op.Ops) {
// var path clip.Path
// path.Begin(ops)
// path.MoveTo(f32.Pt(30, 30))
// path.LineTo(f32.Pt(70, 30))
// path.LineTo(f32.Pt(50, 70))
// path.Close()

// 	paint.FillShape(ops, Green,
// 		clip.Stroke{
// 			Path:  path.End(),
// 			Width: 4,
// 		}.Op())
// }
