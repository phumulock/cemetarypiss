package handlers

import (
	"net/http"

	"github.com/phumulock/cemetarypiss/internal/views"
)

func Home(w http.ResponseWriter, r *http.Request) {
	views.Home().Render(r.Context(), w)
}

func DemoPage(w http.ResponseWriter, r *http.Request) {
	views.Demo().Render(r.Context(), w)
}
