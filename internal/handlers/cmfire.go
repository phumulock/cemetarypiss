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
// The base row is uniformly hot (a bright flame source), but each column cools
// at its own rate — a smooth control field of per-column cooling. Strongly
// cooled columns burn out low (valleys, with empty sky above); weakly cooled
// columns climb to the top (peaks) and fade rather than hard-clip. So flames
// reach genuinely different heights instead of all maxing out. The whole field
// ignites from cold on connect, so the fire visibly "builds in".
const (
	// fireW is intentionally generous: the CSS clamps font-size, so on wide
	// screens the (capped) glyphs still need enough columns to span the page.
	fireW   = 600 // columns; CSS spans width via clamp(.., 100vw/(fireW*0.6), ..)
	fireH   = 16  // rows tall
	fireFPS = 16
	fireMax = 36 // matches the 37-entry palette in fire.js (indices 0..36)

	fireCtrlSpacing = 38   // columns between cooling control points (broad hills)
	fireCoolMin     = 2.6  // weakest cooling => tallest peaks (fade near the top row)
	fireCoolMax     = 8.0  // strongest cooling => shortest valleys (burn out low)
	fireCoolDrift   = 0.25 // per-frame random-walk magnitude of a cooling control point
	fireIgnite      = 0.06 // per-frame build-in ramp (0 -> full over ~17 frames)
)

// Intensity ramp: 0 = coolest (space) .. len-1 = hottest (dense).
var fireRamp = []rune(" .:-=+*#%@")

type doomFire struct {
	buf      []uint8   // palette indices, fireW*fireH
	coolCtrl []float64 // sparse per-column cooling control points
	cool     []float64 // cooling interpolated to every column
	ignite   float64   // 0..1 global build-in ramp
	rng      *rand.Rand
}

func newDoomFire(rng *rand.Rand) *doomFire {
	coolCtrl := make([]float64, fireW/fireCtrlSpacing+2)
	mid := (fireCoolMin + fireCoolMax) / 2
	for i := range coolCtrl {
		coolCtrl[i] = mid
	}
	return &doomFire{
		buf:      make([]uint8, fireW*fireH),
		coolCtrl: coolCtrl,
		cool:     make([]float64, fireW),
		rng:      rng,
	}
}

// stoke drifts the cooling control points (which shape the peaks/valleys),
// interpolates them across every column, advances the ignite ramp, and seeds
// the bright hot base row.
func (f *doomFire) stoke() {
	for i := range f.coolCtrl {
		f.coolCtrl[i] += f.rng.Float64()*2*fireCoolDrift - fireCoolDrift
		if f.rng.Float64() < 0.06 { // occasionally jump toward an extreme
			if f.rng.Float64() < 0.5 {
				f.coolCtrl[i] = fireCoolMin + f.rng.Float64()*1.5 // tall peak
			} else {
				f.coolCtrl[i] = fireCoolMax - f.rng.Float64()*1.5 // short valley
			}
		}
		f.coolCtrl[i] = clampF(f.coolCtrl[i], fireCoolMin, fireCoolMax)
	}
	for x := 0; x < fireW; x++ {
		t := float64(x) / fireCtrlSpacing
		i := int(t)
		m := (1 - math.Cos((t-float64(i))*math.Pi)) / 2 // cosine ease between control points
		f.cool[x] = f.coolCtrl[i]*(1-m) + f.coolCtrl[i+1]*m
	}

	if f.ignite < 1 {
		f.ignite = math.Min(1, f.ignite+fireIgnite)
	}
	for x := 0; x < fireW; x++ {
		v := f.ignite * fireMax * (0.9 + 0.1*f.rng.Float64()) // bright base + slight flicker
		f.buf[(fireH-1)*fireW+x] = uint8(clampF(v, 0, fireMax))
	}
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// spread propagates one cell upward, cooling by that column's rate. Horizontal
// jitter is biased toward 0 so columns stay coherent and the silhouette holds;
// the cooling rate (not the source heat) decides how high each column reaches.
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
	d := f.cool[srcIdx%fireW]
	decay := uint8(d)
	if f.rng.Float64() < d-float64(decay) { // probabilistic rounding => avg decay == d
		decay++
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
