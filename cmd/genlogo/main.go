// Converts logo.png into the ASCII grid consumed by handlers/cmlogo.go.
// Usage: go run ./cmd/genlogo -in cmd/genlogo/logo.png -out internal/views/cmlogo.txt -cols 480
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"strings"
)

// Same ramp as the fire (cold -> hot). Background pixels (black) map to ' ',
// fully-lit pixels (white) map to '@', so the logo glyphs render with the same
// palette the fire strips use — at a much higher column count the per-glyph
// size drops into the same ~3-5px range as the fire, giving matched texture.
var ramp = []rune(" .:-=+*#%@")

func main() {
	in := flag.String("in", "cmd/genlogo/logo.png", "input PNG (white-on-black logo)")
	out := flag.String("out", "internal/views/cmlogo.txt", "output ASCII grid")
	cols := flag.Int("cols", 480, "target columns")
	cellAspect := flag.Float64("cell-aspect", 2.1, "char cell height/width (monospace ~2.0-2.2)")
	flag.Parse()

	f, err := os.Open(*in)
	if err != nil {
		die(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		die(err)
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()

	rows := max(int(float64(*cols)*float64(ih)/float64(iw)/ *cellAspect), 1)

	// Box-filter each cell: average luminance across the source pixels it covers.
	var sb strings.Builder
	sb.Grow((*cols + 1) * rows)
	for ry := range rows {
		if ry > 0 {
			sb.WriteByte('\n')
		}
		y0 := b.Min.Y + ih*ry/rows
		y1 := b.Min.Y + ih*(ry+1)/rows
		if y1 == y0 {
			y1 = y0 + 1
		}
		for cx := range *cols {
			x0 := b.Min.X + iw*cx/(*cols)
			x1 := b.Min.X + iw*(cx+1)/(*cols)
			if x1 == x0 {
				x1 = x0 + 1
			}
			var sum, n uint64
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					r, g, bl, _ := img.At(x, y).RGBA()
					// Rec. 601 luma on 16-bit channels.
					sum += uint64(299*r+587*g+114*bl) / 1000
					n++
				}
			}
			lum := sum / n // 0..65535
			idx := int(lum) * (len(ramp) - 1) / 65535
			sb.WriteRune(ramp[idx])
		}
	}
	sb.WriteByte('\n')

	if err := os.WriteFile(*out, []byte(sb.String()), 0o644); err != nil {
		die(err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d cols x %d rows\n", *out, *cols, rows)
}

func die(err error) {
	fmt.Fprintln(os.Stderr, "genlogo:", err)
	os.Exit(1)
}
