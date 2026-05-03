package ui

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

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
