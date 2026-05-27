package handlers

import (
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

// Server-side take on the pixelated "Doom fire" from static/fire.js (the
// homepage fire strips), rendered to ASCII and streamed as its own signals so
// it layers over the logo/lightning stream as independent data.
//
// Unlike the homepage's uniformly-hot base, the source row here is a smoothed
// per-column heat field that drifts over time — giving tall peaks and deep
// valleys. It starts cold and climbs to the baseline, so the fire visibly
// "builds in" when a client first connects.
const (
	// fireW must be large enough that fireW * (font cap) * (char aspect ~0.6)
	// exceeds the widest viewport we care about. Otherwise the strip's chars
	// stop at the cap-width and leave black space at the right edge of wide
	// windows. 1200 * 5.5 * 0.6 = 3960px covers everything up to 4K.
	fireW   = 1200 // columns; CSS spans width via 100vw/(fireW*0.6)=100vw/720
	fireH   = 28  // rows tall — peaks need vertical headroom to tower past baseline
	fireFPS = 16
	fireMax = 36 // matches the 37-entry palette in fire.js (indices 0..36)

	// Baseline is deliberately set so an average column dies well before row 0
	// (baseline / avg-decay ≈ rows traveled). Without this, every column would
	// carry heat all the way to the top edge and the silhouette would read as a
	// flat horizontal cut. Hot bursts (the big-swing branch in stoke) still hit
	// fireMax and reach all the way up, so peaks tower over the dead field.
	fireBaseline    = 8.0
	fireCtrlSpacing = 38  // columns between source-heat control points (broad hills)
	fireCtrlDrift   = 1.6 // per-frame random-walk magnitude of a control point
)

// Intensity ramp: 0 = coolest (space) .. len-1 = hottest (dense).
var fireRamp = []rune(" .:-=+*#%@")

type doomFire struct {
	buf  []uint8   // palette indices, fireW*fireH
	ctrl []float64 // sparse source-heat control points (one per ~fireCtrlSpacing cols)
	rng  *rand.Rand
}

func newDoomFire(rng *rand.Rand) *doomFire {
	return &doomFire{
		buf:  make([]uint8, fireW*fireH),
		ctrl: make([]float64, fireW/fireCtrlSpacing+2), // start cold -> builds in on connect
		rng:  rng,
	}
}

// stoke advances the sparse heat control points (which swing widely for deep
// hills and valleys), interpolates them smoothly across the bottom row, and
// adds per-cell flicker for the light/dark texture.
func (f *doomFire) stoke() {
	for i := range f.ctrl {
		f.ctrl[i] += f.rng.Float64()*2*fireCtrlDrift - fireCtrlDrift // random walk
		f.ctrl[i] += (fireBaseline - f.ctrl[i]) * 0.012              // weak pull => lingers at extremes
		if f.rng.Float64() < 0.08 {
			f.ctrl[i] += (f.rng.Float64()*2 - 1) * 26 // big swing: dead valley or towering peak
		}
		f.ctrl[i] = clampHeat(f.ctrl[i])
	}

	for x := 0; x < fireW; x++ {
		t := float64(x) / fireCtrlSpacing
		i := int(t)
		m := (1 - math.Cos((t-float64(i))*math.Pi)) / 2 // cosine ease between control points
		h := f.ctrl[i]*(1-m) + f.ctrl[i+1]*m
		v := clampHeat(h + f.rng.Float64()*5 - 2.5) // per-cell flicker -> texture
		f.buf[(fireH-1)*fireW+x] = uint8(v)
	}
}

func clampHeat(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > fireMax {
		return fireMax
	}
	return v
}

// spread propagates one cell upward. Horizontal jitter is biased toward 0 (so
// columns stay coherent and the hill/valley silhouette survives) and per-row
// cooling is gentle (so hot columns tower while cold ones die low).
func (f *doomFire) spread(srcIdx int) {
	off := 0
	switch f.rng.Intn(4) { // ~half the time stay vertical, else drift +/-1
	case 0:
		off = -1
	case 1:
		off = 1
	}
	dst := srcIdx - fireW + off
	if dst < 0 || dst >= len(f.buf) {
		return
	}
	decay := uint8(f.rng.Intn(2)) // 0 or 1
	if srcIdx/fireW < fireH/2 {
		decay++ // upper half cools a little faster
	}
	if v := f.buf[srcIdx]; v > decay {
		f.buf[dst] = v - decay
	} else {
		f.buf[dst] = 0
	}
}

func (f *doomFire) update() {
	for x := 0; x < fireW; x++ {
		for y := 1; y < fireH; y++ {
			f.spread(y*fireW + x)
		}
	}
}

func fireChar(v uint8) rune {
	return fireRamp[int(v)*(len(fireRamp)-1)/fireMax]
}

// render walks rows top-to-bottom. flip=true emits the buffer hot-edge-first so
// the top strip's flames hang downward (the homepage uses scaleY(-1); we reverse
// row order instead, which keeps the glyphs upright).
func (f *doomFire) render(flip bool) string {
	var sb strings.Builder
	sb.Grow((fireW + 1) * fireH)
	for row := 0; row < fireH; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		y := row
		if flip {
			y = fireH - 1 - row
		}
		base := y * fireW
		for x := 0; x < fireW; x++ {
			sb.WriteRune(fireChar(f.buf[base+x]))
		}
	}
	return sb.String()
}

func DemoCMFire(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	ticker := time.NewTicker(time.Second / fireFPS)
	defer ticker.Stop()

	ctx := r.Context()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	top := newDoomFire(rng)
	bottom := newDoomFire(rng)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			top.stoke()
			top.update()
			bottom.stoke()
			bottom.update()

			if err := sse.MarshalAndPatchSignals(map[string]any{
				"_fireTop":    top.render(true),
				"_fireBottom": bottom.render(false),
			}); err != nil {
				return
			}
		}
	}
}
