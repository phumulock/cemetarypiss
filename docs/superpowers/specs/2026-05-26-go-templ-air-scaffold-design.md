# Go + templ + air backend scaffold

Date: 2026-05-26
Status: Approved (pending user spec review)

## Summary

Stand up a Go backend for the `cemetarypiss` project using **chi** (router), **templ** (typed Go templates), **air** (live-reload), and **Taskfile** (named entry points). The new server serves a templ port of the existing site at `/`, preserves the original `index.html` on disk untouched as a reference (never served by Go), and adds a `/demo` page that exercises every main method of the Datastar Go SDK as a smoke test.

## Goals

- A working Go binary that serves the existing site visually unchanged at `/`.
- A dev loop where saving any `.go` or `.templ` file regenerates and restarts the server automatically.
- A smoke-test page that proves the whole Datastar stack (frontend signals → SSE → Go SDK → DOM/signal patches) works end-to-end.
- Project structure that scales naturally if the site grows beyond a few pages.

## Non-goals

- Persisting any state (the smoke-test counter lives in client signals; the server is stateless).
- Authentication, sessions, or any user accounts.
- Database, ORM, migrations.
- Tests beyond manual smoke verification (the Datastar skill's `evals/` covers that side).
- Containerization, deployment config, CI workflows.
- Modifying `index.html` in any way.

## Constraints

- `index.html` is preserved exactly as-is on disk. Not served by Go. Not edited.
- The two MP3s (`01 - Intro.mp3`, `04 - Terrorvoid.mp3`) stay at the project root, so opening `index.html` directly via `file://` still works as a reference.

## Architecture

### Project layout

```
cemetarypiss/
├── index.html                ← original, untouched, NOT served
├── 01 - Intro.mp3            ← served at /audio/01-intro.mp3
├── 04 - Terrorvoid.mp3       ← served at /audio/04-terrorvoid.mp3
├── go.mod                    ← module github.com/phumulock/cemetarypiss
├── Taskfile.yml              ← named commands (dev, build, run, fmt, clean)
├── .air.toml                 ← air config (watches .go + .templ, runs templ generate)
├── .gitignore                ← tmp/, bin/, *_templ.go
├── cmd/
│   └── server/
│       └── main.go           ← chi mux setup, route registration, ListenAndServe
├── internal/
│   ├── handlers/
│   │   ├── home.go           ← GET /
│   │   └── demo.go           ← GET /demo + 5 POST endpoints
│   └── views/
│       ├── layout.templ      ← base <html>, CSS, Datastar <script>, content slot
│       ├── home.templ        ← port of index.html body
│       └── demo.templ        ← Datastar SDK sandbox page
└── static/                   ← future assets (CSS, images); empty at scaffold time
```

### Module path

`github.com/phumulock/cemetarypiss` (matches the git remote at `https://github.com/phumulock/cemetarypiss.git`).

### Routes

| Method | Path | Handler | Purpose |
|---|---|---|---|
| GET | `/` | `handlers.Home` | Render `home.templ` inside `layout.templ` |
| GET | `/demo` | `handlers.DemoPage` | Render `demo.templ` (Datastar smoke test) |
| POST | `/demo/bump` | `handlers.DemoBump` | Read `$count`, patch back `$count + 1` |
| POST | `/demo/time` | `handlers.DemoTime` | `PatchElements` morphs `#server-time` |
| POST | `/demo/note` | `handlers.DemoNote` | `PatchElements` appends `<li>` into `#notes` |
| POST | `/demo/dismiss` | `handlers.DemoDismiss` | `RemoveElement("#banner")` |
| POST | `/demo/greet` | `handlers.DemoGreet` | `ExecuteScript` runs `alert(...)` |
| GET | `/audio/01-intro.mp3` | static file | Serves project-root MP3 |
| GET | `/audio/04-terrorvoid.mp3` | static file | Serves project-root MP3 |
| GET | `/static/*` | static file | Serves `./static/...` (future assets) |

## Component design

### `cmd/server/main.go`

- Creates a chi router.
- Registers middleware: `middleware.Logger`, `middleware.Recoverer`.
- Wires routes (handler functions live in `internal/handlers`).
- Wires the three static file paths.
- Starts `http.ListenAndServe(":8080", r)`.

Port `:8080` is the default; can be overridden later via env if needed (not in scope here).

### `internal/views/layout.templ`

Templ component that takes a child component as a parameter and renders the full HTML shell:

- `<!DOCTYPE html>`, `<html>`, `<head>`
- Meta tags from the original (`charset`, `viewport`)
- `<title>` parameterized so each page can set its own
- The full `<style>` block copied verbatim from the original `index.html` (preserves the fire-strip CSS, navbar/footer styling, fonts, etc.)
- `<script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.1/bundles/datastar.js"></script>` in `<head>`
- `<body>` with the child component rendered inside

This is the only place Datastar is loaded, so all pages get it.

### `internal/views/home.templ`

Templ component rendering the body of `index.html` verbatim — fire strips, canvas, navbar, footer, audio elements — inside `layout.templ`. Audio elements reference the new `/audio/01-intro.mp3` and `/audio/04-terrorvoid.mp3` paths.

The canvas-driven fire effect's JavaScript is preserved as an inline `<script>` block in `home.templ`. (Could be extracted to `static/fire.js` later; out of scope here.)

### `internal/views/demo.templ`

Five labeled sections, each demonstrating one SDK capability:

```
1. Counter:    [span data-text=$count]  [button @post('/demo/bump')]
2. Time:       [div #server-time]       [button @post('/demo/time')]
3. Notes:      [ul #notes]              [button @post('/demo/note')]
4. Banner:     [section #banner]        [button @post('/demo/dismiss')]
5. Script:                              [button @post('/demo/greet')]
6. Debug:      [pre data-json-signals]
```

The page initializes signals with `data-signals="{count: 0}"`. Everything else either uses signals or pure DOM patches.

### `internal/handlers/demo.go`

One handler per route. Pattern:

```go
type bumpStore struct{ Count int `json:"count"` }

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
        html.EscapeString(time.Now().Format(time.RFC1123))))
}

// ... and so on for /note, /dismiss, /greet
```

`html.EscapeString` is used wherever any content reaches the page, even when it's currently server-controlled — sets the right habit.

### `internal/handlers/home.go`

```go
func Home(w http.ResponseWriter, r *http.Request) {
    views.Home().Render(r.Context(), w)
}

func DemoPage(w http.ResponseWriter, r *http.Request) {
    views.Demo().Render(r.Context(), w)
}
```

Plain — just renders the templ component to the response writer.

## Dev loop

### `Taskfile.yml`

| Task | Behavior |
|---|---|
| `task install` | `go install` templ + air; `go mod tidy` |
| `task generate` | `templ generate ./...` |
| `task dev` | `air` (which runs templ generate, builds, restarts on change) |
| `task build` | `templ generate ./...` then `go build -o bin/server ./cmd/server` |
| `task run` | `templ generate ./...` then `go run ./cmd/server` |
| `task fmt` | `templ fmt .` and `gofmt -w .` |
| `task clean` | Remove `bin/`, `tmp/`, all `*_templ.go` files |

### `.air.toml`

- Watches `.go` and `.templ` extensions
- Pre-build hook: `templ generate ./...`
- Build: `go build -o ./tmp/main ./cmd/server`
- Run: `./tmp/main`
- Excludes: `tmp/`, `bin/`, `node_modules/`

### Tool installation

```sh
go install github.com/a-h/templ/cmd/templ@latest
go install github.com/air-verse/air@latest
go install github.com/go-task/task/v3/cmd/task@latest
```

### Go dependencies

```sh
go get github.com/go-chi/chi/v5
go get github.com/starfederation/datastar-go
go get github.com/a-h/templ
```

## Smoke test acceptance criteria

After `task dev` is running:

1. `http://localhost:8080/` — visually matches the original `index.html` (fonts, fire strips, navbar/footer, working audio elements).
2. `http://localhost:8080/demo` loads with `data-json-signals` showing `{ "count": 0 }`.
3. Clicking **Bump** increments the displayed count and the JSON view in lockstep.
4. Clicking **Fetch** under "Server time" updates `#server-time` to the current server timestamp.
5. Clicking **Add note** appends a new `<li>` to `#notes` each time (without clearing existing entries).
6. Clicking **Dismiss** removes the entire `#banner` section from the DOM.
7. Clicking **Greet** triggers a browser `alert()` populated by the server.
8. Saving any `.templ` or `.go` file causes air to rebuild and the browser to re-fetch within a second or two.

If all 8 hold, every SDK method (`ReadSignals`, `NewSSE`, `MarshalAndPatchSignals`, `PatchElements` default + append + remove, `RemoveElement`, `ExecuteScript`) is exercised.

## What's deliberately deferred

- Extracting the inline fire-effect JS into a static file.
- Extracting the inline CSS into a static file.
- Hot-reload in the browser (Datastar's SSE retry handles reconnects on its own; that's enough).
- Production build (minification, embed FS via `embed.FS`, etc.).
- TLS, HTTP/2 tuning, graceful shutdown.

These are sensible next steps but not part of the scaffold.

## Open questions

None at spec-write time. All architectural decisions are settled. Implementation specifics (exact air config flags, exact chi static file mount syntax) will be decided in the implementation plan.
