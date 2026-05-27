package handlers

import (
	_ "embed"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/phumulock/cemetarypiss/internal/views"
	"github.com/starfederation/datastar-go/datastar"
)

//go:embed cmlogo.txt
var cmlogoRaw string

const (
	cmFPS      = 20
	cmSideCols = 14 // width of each lightning gutter
	cmGapCols  = 4  // blank columns between a gutter and the logo
	cmMaxBolts = 3  // concurrent bolts per side
)

var (
	cmLogo      [][]rune
	cmLogoW     int
	cmLogoH     int
	cmColOffset int
	cmTotalCols int
	cmTotalRows int
)

func init() {
	lines := strings.Split(strings.TrimRight(cmlogoRaw, "\n"), "\n")
	cmLogoH = len(lines)
	for _, ln := range lines {
		if len(ln) > cmLogoW {
			cmLogoW = len(ln)
		}
	}
	cmLogo = make([][]rune, cmLogoH)
	for i, ln := range lines {
		row := []rune(ln)
		for len(row) < cmLogoW {
			row = append(row, ' ')
		}
		cmLogo[i] = row
	}
	cmColOffset = cmSideCols + cmGapCols
	cmTotalCols = cmSideCols + cmGapCols + cmLogoW + cmGapCols + cmSideCols
	cmTotalRows = cmLogoH
}

func DemoCMLogoPage(w http.ResponseWriter, r *http.Request) {
	views.DemoCMLogo(cmTotalCols, cmTotalRows).Render(r.Context(), w)
}

// cmBolt is a jagged lightning streak that is revealed top-to-bottom (so it
// reads as "shooting down"), then flickers for a few frames before vanishing.
type cmBolt struct {
	path  []int    // absolute column for each row [0..cmTotalRows)
	forks []cmFork // optional diagonal branches
	head  float64  // rows revealed so far
	speed float64  // rows revealed per frame
	hold  int      // frames to flicker after fully revealed
}

type cmFork struct {
	startRow int
	cols     []int // columns for rows startRow+1, startRow+2, ...
}

func newBolt(rng *rand.Rand, lo, hi int) *cmBolt {
	x := lo + rng.Intn(hi-lo)
	path := make([]int, cmTotalRows)
	path[0] = x
	for r := 1; r < cmTotalRows; r++ {
		switch rng.Intn(4) { // bias toward vertical (2 of 4 keep column)
		case 0:
			x--
		case 1:
			x++
		}
		if x < lo {
			x = lo
		} else if x >= hi {
			x = hi - 1
		}
		path[r] = x
	}

	b := &cmBolt{
		path:  path,
		speed: float64(cmTotalRows) / float64(6+rng.Intn(6)), // full reveal in 6-11 frames
		hold:  3 + rng.Intn(5),
	}

	if cmTotalRows > 10 && rng.Intn(2) == 0 { // sometimes branch
		start := 4 + rng.Intn(cmTotalRows-8)
		n := 3 + rng.Intn(5)
		dir := 1
		if rng.Intn(2) == 0 {
			dir = -1
		}
		fx := path[start]
		cols := make([]int, 0, n)
		for i := 0; i < n; i++ {
			fx += dir
			if fx < lo || fx >= hi {
				break
			}
			cols = append(cols, fx)
		}
		if len(cols) > 0 {
			b.forks = append(b.forks, cmFork{startRow: start, cols: cols})
		}
	}
	return b
}

func boltChar(dx int) rune {
	switch {
	case dx > 0:
		return '\\'
	case dx < 0:
		return '/'
	default:
		return '|'
	}
}

func (b *cmBolt) draw(grid [][]rune, rng *rand.Rand) {
	revealed := int(b.head)
	if revealed > cmTotalRows {
		revealed = cmTotalRows
	}
	fully := revealed >= cmTotalRows
	if fully && rng.Intn(2) == 0 {
		return // flicker the held bolt off this frame
	}

	for r := 0; r < revealed; r++ {
		dx := 0
		if r > 0 {
			dx = b.path[r] - b.path[r-1]
		}
		ch := boltChar(dx)
		if !fully && r == revealed-1 {
			ch = '*' // bright leading head
		}
		grid[r][b.path[r]] = ch
	}

	for _, f := range b.forks {
		if f.startRow >= revealed {
			continue
		}
		prev := b.path[f.startRow]
		for i, col := range f.cols {
			r := f.startRow + 1 + i
			if r >= revealed {
				break
			}
			grid[r][col] = boltChar(col - prev)
			prev = col
		}
	}
}

func cmAdvance(bolts []*cmBolt) []*cmBolt {
	out := bolts[:0]
	for _, b := range bolts {
		if b.head < float64(cmTotalRows) {
			b.head += b.speed
		} else {
			b.hold--
			if b.hold <= 0 {
				continue // fully faded; drop
			}
		}
		out = append(out, b)
	}
	return out
}

func DemoCMLogoUpdates(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	ticker := time.NewTicker(time.Second / cmFPS)
	defer ticker.Stop()

	ctx := r.Context()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	leftLo, leftHi := 0, cmSideCols
	rightLo, rightHi := cmTotalCols-cmSideCols, cmTotalCols

	var left, right []*cmBolt

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(left) < cmMaxBolts && rng.Float64() < 0.18 {
				left = append(left, newBolt(rng, leftLo, leftHi))
			}
			if len(right) < cmMaxBolts && rng.Float64() < 0.18 {
				right = append(right, newBolt(rng, rightLo, rightHi))
			}
			left = cmAdvance(left)
			right = cmAdvance(right)

			grid := make([][]rune, cmTotalRows)
			for i := range grid {
				row := make([]rune, cmTotalCols)
				for j := range row {
					row[j] = ' '
				}
				copy(row[cmColOffset:cmColOffset+cmLogoW], cmLogo[i])
				grid[i] = row
			}
			for _, b := range left {
				b.draw(grid, rng)
			}
			for _, b := range right {
				b.draw(grid, rng)
			}

			var sb strings.Builder
			sb.Grow((cmTotalCols + 1) * cmTotalRows)
			for i, row := range grid {
				if i > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString(string(row))
			}

			if err := sse.MarshalAndPatchSignals(map[string]any{"_contents": sb.String()}); err != nil {
				return
			}
		}
	}
}
