package ui

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// TopToolbar returns a widget that renders the top toolbar with mode toggles (will later be tool selection bar).
func TopToolbar(th *material.Theme, lineModeBtn, eraserBtn *widget.Clickable) func(gtx layout.Context) layout.Dimensions {
	return func(gtx layout.Context) layout.Dimensions {
		// Record the content ops (operators - buttons, etc.) so we can draw background/border behind it
		rec := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					for lineModeBtn.Clicked(gtx) {
						lineMode = !lineMode
						if !lineMode {
							previewActive = false
						}
						if lineMode {
							eraserMode = false
						}
					}
					label := "Freehand"
					if lineMode {
						label = "Line"
					}
					btn := material.Button(th, lineModeBtn, label)
					btn.TextSize = unit.Sp(12)
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					for eraserBtn.Clicked(gtx) {
						eraserMode = !eraserMode
						if eraserMode {
							lineMode = false
							previewActive = false
							drawing = false
						}
					}
					label := "Eraser"
					if eraserMode {
						label = "Eraser On"
					}
					btn := material.Button(th, eraserBtn, label)
					btn.TextSize = unit.Sp(12)
					return btn.Layout(gtx)
				}),
			)
		})
		call := rec.Stop()

		// Draw light gray background and border with rounded corners
		bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
		borderCol := color.NRGBA{R: 220, G: 220, B: 220, A: 255}
		radius := gtx.Dp(unit.Dp(6))
		borderWidth := gtx.Dp(unit.Dp(1))

		rect := image.Rectangle{Max: dims.Size}
		rr := clip.UniformRRect(rect, radius)
		paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
		if borderWidth > 0 {
			paint.FillShape(gtx.Ops, borderCol, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(borderWidth)}.Op())
		}

		call.Add(gtx.Ops)
		return dims
	}
}

// Sidebar renders a vertical palette on the left and updates the global drawColor.
func Sidebar(th *material.Theme, palette []color.NRGBA, colorBtns []widget.Clickable, customEditor *widget.Editor, decWidth, incWidth, decEraser, incEraser, eraserBtn *widget.Clickable) func(gtx layout.Context) layout.Dimensions {
	return func(gtx layout.Context) layout.Dimensions {
		// first define some standard values for gap measurements
		sectionGapDp := 8
		// compact grid: target 4 swatches per row
		swatchDp := 20
		gapDp := 4
		insetDp := 6
		itemsPerRow := 4
		totalDp := swatchDp*itemsPerRow + gapDp*(itemsPerRow-1) + insetDp*2
		gtx.Constraints.Min.X = gtx.Dp(unit.Dp(totalDp))

		// Record child ops (operators) so background can be drawn behind them
		rec := op.Record(gtx.Ops)
		dims := layout.UniformInset(unit.Dp(insetDp)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			children := make([]layout.FlexChild, 0)

			// Starting with the colour picker
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, "Color:")
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, lbl.Layout)
			}))
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(gapDp / 2)}.Layout))

			// build rows of up to itemsPerRow
			for i := 0; i < len(palette); i += itemsPerRow {
				end := i + itemsPerRow
				if end > len(palette) {
					end = len(palette)
				}
				start := i
				// row widget
				children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					rowChildren := make([]layout.FlexChild, 0)
					for j := start; j < end; j++ {
						idx := j
						rowChildren = append(rowChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							for colorBtns[idx].Clicked(gtx) {
								drawColor = palette[idx]
							}
							gtx.Constraints.Min.X = gtx.Dp(unit.Dp(swatchDp))
							gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(swatchDp))
							btn := material.Button(th, &colorBtns[idx], "")
							btn.Background = palette[idx]
							btn.TextSize = unit.Sp(0)
							return btn.Layout(gtx)
						}))
						if j < end-1 {
							rowChildren = append(rowChildren, layout.Rigid(layout.Spacer{Width: unit.Dp(gapDp)}.Layout))
						}
					}
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, rowChildren...)
				}))
				// spacer between rows
				children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(gapDp)}.Layout))
			}

			// Add custom color editor and preview below grid
			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(gapDp)}.Layout))

			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body2(th, "Or Custom:")
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, lbl.Layout)
			}))
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						borderColor := Gray
						if gtx.Source.Focused(customEditor) {
							borderColor = Blue
						}
						border := widget.Border{
							Color:        borderColor,
							Width:        unit.Dp(1),
							CornerRadius: unit.Dp(6),
						}
						return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(2)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return material.Editor(th, customEditor, "#RRGGBB").Layout(gtx)
							})
						})
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(gapDp)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(swatchDp))
						gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(swatchDp))
						var preview widget.Clickable
						btn := material.Button(th, &preview, "")
						btn.Background = drawColor
						btn.TextSize = unit.Sp(0)
						return btn.Layout(gtx)
					}),
				)
			}))

			// display Stroke Width selector if needed for current tool

			children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(sectionGapDp)}.Layout))

			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Body1(th, "Stroke Width:")
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, lbl.Layout)
			}))

			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						for decWidth.Clicked(gtx) {
							if strokeWidth > 1 {
								strokeWidth -= 1
							}
						}
						btn := material.Button(th, decWidth, "-")
						btn.TextSize = unit.Sp(8)
						return btn.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, fmt.Sprintf("%.0f", float32(strokeWidth)))
						return layout.UniformInset(unit.Dp(6)).Layout(gtx, lbl.Layout)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						for incWidth.Clicked(gtx) {
							if strokeWidth < 64 {
								strokeWidth += 1
							}
						}
						btn := material.Button(th, incWidth, "+")
						btn.TextSize = unit.Sp(8)
						return btn.Layout(gtx)
					}),
				)
			}))

			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if !eraserMode {
					return layout.Dimensions{}
				}
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						for decEraser.Clicked(gtx) {
							if eraserSize > 2 {
								eraserSize -= 2
							}
						}
						btn := material.Button(th, decEraser, "-")
						btn.TextSize = unit.Sp(14)
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Body1(th, fmt.Sprintf("E:%.0f", float32(eraserSize)))
						return layout.UniformInset(unit.Dp(6)).Layout(gtx, lbl.Layout)
					}),
					layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						for incEraser.Clicked(gtx) {
							if eraserSize < 256 {
								eraserSize += 2
							}
						}
						btn := material.Button(th, incEraser, "+")
						btn.TextSize = unit.Sp(14)
						return btn.Layout(gtx)
					}),
				)
			}))

			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx, children...)
		})
		call := rec.Stop()

		// Draw light gray background and border with rounded corners
		bg := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
		borderCol := color.NRGBA{R: 220, G: 220, B: 220, A: 255}
		radius := gtx.Dp(unit.Dp(6))
		borderWidth := gtx.Dp(unit.Dp(1))

		rect := image.Rectangle{Max: dims.Size}
		rr := clip.UniformRRect(rect, radius)
		paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))
		if borderWidth > 0 {
			paint.FillShape(gtx.Ops, borderCol, clip.Stroke{Path: rr.Path(gtx.Ops), Width: float32(borderWidth)}.Op())
		}

		call.Add(gtx.Ops)
		return dims
	}
}

// BottomControls returns the row of session/session code editor and width/eraser controls (will be moved to sidebar later).
// TODO: Move all tool controls to sidebar - Rina
// TODO: Give this a slightly darker background, same style as top/sidebars - Rina
func BottomControls(th *material.Theme, toggleSessionBtn *widget.Clickable, sessionCodeInput *widget.Editor) func(gtx layout.Context) layout.Dimensions {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if inSession {
						btn := material.Button(th, toggleSessionBtn, "Stop Session")
						btn.TextSize = unit.Sp(14)
						btn.Background = Red
						return btn.Layout(gtx)
					}
					btn := material.Button(th, toggleSessionBtn, "Start Session")
					btn.TextSize = unit.Sp(14)
					return btn.Layout(gtx)
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
					if gtx.Source.Focused(sessionCodeInput) {
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
							return material.Editor(th, sessionCodeInput, "Session Code").Layout(gtx)
						})
					})
				}),
			)
		})
	}
}
