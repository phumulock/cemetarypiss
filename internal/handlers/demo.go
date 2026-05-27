package handlers

import (
	"fmt"
	"html"
	"net/http"
	"time"

	"github.com/starfederation/datastar-go/datastar"
)

type bumpStore struct {
	Count int `json:"count"`
}

func DemoBump(w http.ResponseWriter, r *http.Request) {
	var s bumpStore
	if err := datastar.ReadSignals(r, &s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sse := datastar.NewSSE(w, r)
	sse.MarshalAndPatchSignals(map[string]any{"count": s.Count + 1})
}

func DemoTime(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sse.PatchElements(fmt.Sprintf(
		`<div id="server-time">%s</div>`,
		html.EscapeString(time.Now().Format(time.RFC1123)),
	))
}

func DemoNote(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sse.PatchElements(
		fmt.Sprintf(`<li>note added at %s</li>`, html.EscapeString(time.Now().Format(time.Kitchen))),
		datastar.WithSelector("#notes"),
		datastar.WithModeAppend(),
	)
}

func DemoDismiss(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sse.RemoveElement("#banner")
}

func DemoGreet(w http.ResponseWriter, r *http.Request) {
	sse := datastar.NewSSE(w, r)
	sse.ExecuteScript(`alert("hello from server")`)
}
