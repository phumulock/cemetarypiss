package handlers

import (
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/phumulock/cemetarypiss/internal/views"
	"github.com/starfederation/datastar-go/datastar"
)

func DemoLogoPage(w http.ResponseWriter, r *http.Request) {
	views.DemoLogo().Render(r.Context(), w)
}

const (
	logoCols       = 80
	logoRows       = 28
	logoFPS        = 30
	logoLoopSecs   = 60
	logoFrameCount = logoFPS * logoLoopSecs
)

var (
	logoPalette = []rune(" .·:-=+*#%@")
	logoFrames  []string
)

func init() {
	start := time.Now()
	logoFrames = make([]string, logoFrameCount)
	for i := 0; i < logoFrameCount; i++ {
		t := float64(i) / float64(logoFPS)
		logoFrames[i] = renderLogoFrame(t)
	}
	log.Printf("logo: precomputed %d frames in %s", logoFrameCount, time.Since(start))
}

func DemoLogoUpdates(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	ticker := time.NewTicker(time.Second / logoFPS)
	defer ticker.Stop()

	ctx := r.Context()
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			elapsed := int(now.Sub(start).Seconds() * float64(logoFPS))
			i := elapsed % logoFrameCount
			pct := float64(i) / float64(logoFrameCount) * 100

			if err := sse.MarshalAndPatchSignals(map[string]any{
				"_contents":   logoFrames[i],
				"_percentage": pct,
			}); err != nil {
				return
			}
		}
	}
}

func renderLogoFrame(t float64) string {
	var b strings.Builder
	b.Grow((logoCols*3 + 1) * logoRows)
	n := float64(len(logoPalette) - 1)
	for y := 0; y < logoRows; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := 0; x < logoCols; x++ {
			v := math.Sin(float64(x)*0.18+float64(y)*0.12+t*2.0) +
				math.Sin(float64(x)*0.07-float64(y)*0.21+t*1.3)
			idx := int(math.Round((v + 2) / 4 * n))
			if idx < 0 {
				idx = 0
			} else if idx >= len(logoPalette) {
				idx = len(logoPalette) - 1
			}
			b.WriteRune(logoPalette[idx])
		}
	}
	return b.String()
}
