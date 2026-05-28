package handlers

import (
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/phumulock/cemetarypiss/internal/views"
	"github.com/starfederation/datastar-go/datastar"
)

// Lightning grid is independent of the logo: it fills the entire band between
// navbar and footer (CSS scales the pre to the band rectangle), and bolts can
// strike anywhere across the width. The logo sits on top as a separate pre.
const (
	cmFPS       = 20
	cmLightCols = 720
	cmLightRows = 180
	// Density tuned for portrait phones: only ~30% of the cols are on-screen
	// there (the rest overflows and is clipped by .logo-stage), so a generous
	// max keeps the visible center busy without flooding widescreen viewers.
	cmMaxBolts = 10

	// Flash kicks when 2+ bolts strike in the same frame (a "burst"). The
	// signal is the overlay opacity; decay is per-frame at cmFPS so the fade
	// lasts ~200-300ms — long enough to read as a flash, short enough that it
	// doesn't wash the scene.
	cmBurstChance  = 0.06 // per frame, on top of the normal single-bolt spawn
	cmFlashImpulse = 0.16 // opacity added per burst
	cmFlashCap     = 0.22 // hard cap keeps it subtle even on rapid bursts
	cmFlashDecay   = 0.74 // multiplier each frame
)

func DemoCMLogoPage(w http.ResponseWriter, r *http.Request) {
	views.DemoCMLogo().Render(r.Context(), w)
}

// cmBolt is a jagged lightning streak revealed top-to-bottom, then flickering
// for a few frames before vanishing.
type cmBolt struct {
	path  []int
	forks []cmFork
	head  float64
	speed float64
	hold  int
}

type cmFork struct {
	startRow int
	cols     []int
}

func newBolt(rng *rand.Rand) *cmBolt {
	x := rng.Intn(cmLightCols)
	path := make([]int, cmLightRows)
	path[0] = x
	for r := 1; r < cmLightRows; r++ {
		switch rng.Intn(4) { // bias toward vertical (2 of 4 keep column)
		case 0:
			x--
		case 1:
			x++
		}
		if x < 0 {
			x = 0
		} else if x >= cmLightCols {
			x = cmLightCols - 1
		}
		path[r] = x
	}

	b := &cmBolt{
		path: path,
		// Fast reveal + long held-flicker so each bolt spends most of its life
		// drawn full-height. Without this the population is dominated by
		// half-revealed bolts and lightning visually bunches at the top.
		speed: float64(cmLightRows) / float64(3+rng.Intn(4)), // full reveal in 3-6 frames
		hold:  8 + rng.Intn(10),                              // flicker for 8-17 frames
	}

	if cmLightRows > 10 && rng.Intn(2) == 0 {
		start := 4 + rng.Intn(cmLightRows-8)
		n := 3 + rng.Intn(6)
		dir := 1
		if rng.Intn(2) == 0 {
			dir = -1
		}
		fx := path[start]
		cols := make([]int, 0, n)
		for i := 0; i < n; i++ {
			fx += dir
			if fx < 0 || fx >= cmLightCols {
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

func boltChar(dx int) byte {
	switch {
	case dx > 0:
		return '\\'
	case dx < 0:
		return '/'
	default:
		return '|'
	}
}

func (b *cmBolt) draw(grid [][]byte, rng *rand.Rand) {
	revealed := int(b.head)
	if revealed > cmLightRows {
		revealed = cmLightRows
	}
	fully := revealed >= cmLightRows
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
		if b.head < float64(cmLightRows) {
			b.head += b.speed
		} else {
			b.hold--
			if b.hold <= 0 {
				continue
			}
		}
		out = append(out, b)
	}
	return out
}

// lightningFrame is one rendered lightning grid + the matching flash level,
// broadcast to every connected client. The RNG-driven bolts look identical to
// every viewer so there's no reason to recompute per-connection.
type lightningFrame struct {
	grid  string
	flash float64
}

// lightningHub fans one producer's frames out to N SSE subscribers. The
// producer starts on first subscribe and exits when the last subscriber leaves.
type lightningHub struct {
	mu      sync.Mutex
	subs    map[chan lightningFrame]struct{}
	running bool
}

var globalLightningHub = &lightningHub{subs: make(map[chan lightningFrame]struct{})}

func (h *lightningHub) subscribe() chan lightningFrame {
	ch := make(chan lightningFrame, 2)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	start := !h.running
	if start {
		h.running = true
	}
	h.mu.Unlock()
	if start {
		go h.run()
	}
	return ch
}

func (h *lightningHub) unsubscribe(ch chan lightningFrame) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *lightningHub) run() {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	ticker := time.NewTicker(time.Second / cmFPS)
	defer ticker.Stop()

	var bolts []*cmBolt
	flash := 0.0

	// Reused across frames: avoids the per-frame [][]rune allocation +
	// strings.Builder churn from the old per-client loop.
	grid := make([][]byte, cmLightRows)
	for i := range grid {
		grid[i] = make([]byte, cmLightCols)
	}
	var sb strings.Builder

	for range ticker.C {
		flash *= cmFlashDecay
		if flash < 0.005 {
			flash = 0
		}

		if len(bolts) < cmMaxBolts && rng.Float64() < 0.18 {
			bolts = append(bolts, newBolt(rng))
		}
		if len(bolts) < cmMaxBolts-2 && rng.Float64() < cmBurstChance {
			extra := 1 + rng.Intn(2)
			for i := 0; i < extra; i++ {
				bolts = append(bolts, newBolt(rng))
			}
			flash += cmFlashImpulse
			if flash > cmFlashCap {
				flash = cmFlashCap
			}
		}
		bolts = cmAdvance(bolts)

		for i := range grid {
			row := grid[i]
			for j := range row {
				row[j] = ' '
			}
		}
		for _, b := range bolts {
			b.draw(grid, rng)
		}

		sb.Reset()
		sb.Grow((cmLightCols + 1) * cmLightRows)
		for i, row := range grid {
			if i > 0 {
				sb.WriteByte('\n')
			}
			sb.Write(row)
		}
		frame := lightningFrame{grid: sb.String(), flash: flash}

		h.mu.Lock()
		if len(h.subs) == 0 {
			h.running = false
			h.mu.Unlock()
			return
		}
		for ch := range h.subs {
			select {
			case ch <- frame:
			default:
			}
		}
		h.mu.Unlock()
	}
}

func DemoCMLogoUpdates(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r, datastar.WithCompression())
	ctx := r.Context()

	ch := globalLightningHub.subscribe()
	defer globalLightningHub.unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-ch:
			if err := sse.MarshalAndPatchSignals(map[string]any{
				"_lightning": frame.grid,
				"_flash":     frame.flash,
			}); err != nil {
				return
			}
		}
	}
}
