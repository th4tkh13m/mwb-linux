//go:build linux

package tray

import "image"
import "image/color"

type iconPixmap struct {
	W, H int32
	Data []byte
}

// drawIcon renders the two-screens-and-arrow logo as an ARGB32 (network byte
// order) pixmap, so the tray icon shows even when the icon theme lacks "mwb".
func drawIcon(size int, dim bool) iconPixmap {
	blue := color.NRGBA{0x3d, 0xae, 0xe9, 0xff}
	orange := color.NRGBA{0xf6, 0x74, 0x00, 0xff}
	if dim {
		blue = color.NRGBA{0x8a, 0x8a, 0x8a, 0xff}
		orange = blue
	}
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	s := float64(size) / 64
	rect := func(x0, y0, x1, y1 float64, c color.NRGBA) {
		for y := int(y0 * s); y < int(y1*s); y++ {
			for x := int(x0 * s); x < int(x1*s); x++ {
				img.SetNRGBA(x, y, c)
			}
		}
	}
	frame := func(x, y, w, h float64) {
		t := 4.0
		rect(x, y, x+w, y+t, blue)
		rect(x, y+h-t, x+w, y+h, blue)
		rect(x, y, x+t, y+h, blue)
		rect(x+w-t, y, x+w, y+h, blue)
	}
	frame(3, 12, 26, 20)
	frame(35, 32, 26, 20)
	rect(30, 20, 44, 24, orange)
	rect(40, 20, 44, 30, orange)
	rect(37, 28, 47, 32, orange)
	data := make([]byte, 0, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(x, y)
			data = append(data, c.A, c.R, c.G, c.B)
		}
	}
	return iconPixmap{int32(size), int32(size), data}
}
