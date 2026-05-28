package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/phumulock/cemetarypiss/internal/handlers"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", handlers.DemoCMLogoPage)
	r.Get("/classic", handlers.Home) // previous animated/audio landing page
	r.Get("/demo", handlers.DemoPage)
	r.Get("/demo/logo", handlers.DemoLogoPage)
	r.Get("/demo/logo/updates", handlers.DemoLogoUpdates)
	r.Get("/demo/cmlogo", handlers.DemoCMLogoPage)
	r.Get("/demo/cmlogo/updates", handlers.DemoCMLogoUpdates)
	r.Get("/demo/cmlogo/fire", handlers.DemoCMFire)

	r.Post("/demo/bump", handlers.DemoBump)
	r.Post("/demo/time", handlers.DemoTime)
	r.Post("/demo/note", handlers.DemoNote)
	r.Post("/demo/dismiss", handlers.DemoDismiss)
	r.Post("/demo/greet", handlers.DemoGreet)

	staticFS := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", staticFS))

	audioFS := http.FileServer(http.Dir("./static/audio"))
	r.Handle("/audio/*", http.StripPrefix("/audio/", audioFS))

	addr := ":8080"
	log.Printf("listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
