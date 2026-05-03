package ui

import (
	"fmt"
	"image/color"
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
