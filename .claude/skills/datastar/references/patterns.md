# Datastar patterns

Battle-tested templates for common UI patterns. Pick the closest match and adapt the names; do not re-derive from primitives if a pattern fits.

## Table of contents

- [Active search (debounced)](#active-search-debounced)
- [Click-to-edit](#click-to-edit)
- [Form submit with validation errors](#form-submit-with-validation-errors)
- [Infinite scroll](#infinite-scroll)
- [Real-time push from server](#real-time-push-from-server)
- [Optimistic UI with rollback](#optimistic-ui-with-rollback)
- [Loading indicators](#loading-indicators)
- [Lazy load on view / on demand](#lazy-load-on-view--on-demand)
- [Modal dialog](#modal-dialog)
- [Tabs (URL-synced)](#tabs-url-synced)
- [Polling](#polling)
- [Confirm before action](#confirm-before-action)
- [Streaming AI / progressive output](#streaming-ai--progressive-output)

---

## Active search (debounced)

```html
<input type="text"
       data-bind:query
       data-on:input__debounce.300ms="@get('/search')"
       placeholder="Search…" />

<ul id="results"></ul>
```

Server reads `$query`, returns `<ul id="results">…</ul>`. Default `outer` mode morphs by id.

**Why debounce.** Without it, every keystroke fires a request. 300ms is the sweet spot — fast enough to feel live, slow enough to not pummel the server.

**Why `data-bind` separate from `data-on:input`.** `data-bind` keeps `$query` in sync; `data-on:input` triggers the request. Together they let the signal be reused (e.g., to show "results for $query").

## Click-to-edit

```html
<div id="profile">
    <p>Name: John Doe</p>
    <button data-indicator:_fetching
            data-attr:disabled="$_fetching"
            data-on:click="@get('/profile/edit')">Edit</button>
</div>
```

Server returns the edit form as `<div id="profile">…inputs and Save/Cancel buttons…</div>`. Default `outer` morph swaps the read-only view for the editable view.

Cancel button can either request the read-only view back (`@get('/profile')`) or use purely client-side state.

## Form submit with validation errors

```html
<form data-signals="{errors: {}}"
      data-on:submit__prevent="@post('/users')">
    <label>
        Name <input data-bind:name required />
        <span data-show="$errors.name" data-text="$errors.name" class="error"></span>
    </label>

    <label>
        Email <input data-bind:email type="email" />
        <span data-show="$errors.email" data-text="$errors.email" class="error"></span>
    </label>

    <button data-indicator:_submitting
            data-attr:disabled="$_submitting"
            type="submit">
        <span data-show="!$_submitting">Create</span>
        <span data-show="$_submitting">Submitting…</span>
    </button>
</form>
```

Server validates, then either:
- On error: `MarshalAndPatchSignals({"errors": {"email": "Already taken"}})`. The `data-show` reactively reveals the message, `data-text` populates it.
- On success: `MarshalAndPatchSignals({"errors": null})` and `PatchElements` something like a success message or `Redirect("/users/123")`.

## Infinite scroll

```html
<div id="feed">
    <!-- initial items rendered server-side -->
</div>
<div id="sentinel" data-on-intersect__once="@get('/feed?page=2')">Loading…</div>
```

Server:
1. Appends items into `#feed` with `mode: append`.
2. Replaces `#sentinel` with a new sentinel pointing at `page=N+1`. (Or removes it when there are no more results.)

The `__once` modifier ensures each sentinel only fires once — important since IntersectionObserver fires every time visibility changes.

## Real-time push from server

```html
<!-- Long-lived SSE; openWhenHidden keeps it alive in background tabs -->
<div data-on:load="@get('/stream', {openWhenHidden: true})"></div>

<p>Online: <span data-text="$online">…</span></p>
<ul id="activity"></ul>
```

Server handler keeps the response open and emits events as data arrives:

```go
for event := range eventBus.Subscribe() {
    switch e := event.(type) {
    case PresenceUpdate:
        sse.MarshalAndPatchSignals(map[string]any{"online": e.Count})
    case Activity:
        sse.PatchElements(fmt.Sprintf(`<li>%s</li>`, html.EscapeString(e.Text)),
            datastar.WithSelector("#activity"),
            datastar.WithMode(datastar.ElementsModePrepend))
    }
}
```

Datastar auto-reconnects with backoff if the connection drops.

## Optimistic UI with rollback

```html
<button data-on:click="$count++; @post('/inc')">+</button>
<p data-text="$count"></p>
```

Client increments `$count` synchronously. Then the request fires. Server can:
- Confirm: send the authoritative `$count` back via signal patch (typically equals the optimistic value).
- Reject: send back `$count - 1` (or the prior value) plus an error signal to display.

For collection operations (adding an item), the pattern is similar:

```html
<button data-on:click="
    $items = [...$items, {id: 'temp', text: $draft, pending: true}];
    $draft = '';
    @post('/items')
">Add</button>
```

Server returns the canonical list (with real ids and `pending: false`).

## Loading indicators

Three layers, pick what you need:

**Per-button** (`data-indicator`):

```html
<button data-indicator:_saving
        data-attr:disabled="$_saving"
        data-on:click="@post('/save')">
    <span data-show="!$_saving">Save</span>
    <span data-show="$_saving">…</span>
</button>
```

**Per-region** (server-driven):

```go
sse.PatchElements(`<div id="results"><p>Loading…</p></div>`)
results := slowQuery()
sse.PatchElements(`<div id="results"><ul>…</ul></div>`)
```

**Global** (any in-flight request anywhere):

```html
<!-- A wrapper sets _fetching on any request -->
<body data-indicator:_fetching>
    <div data-show="$_fetching" class="global-spinner"></div>
    …
</body>
```

(Note: `data-indicator` is scoped to the element it's on; for a truly global one, server-driven signal patches work better.)

## Lazy load on view / on demand

**On-view (deferred render):**

```html
<div id="comments" data-on-intersect__once="@get('/comments')">
    <p>Loading comments…</p>
</div>
```

**On-demand:**

```html
<details>
    <summary>Comments</summary>
    <div id="comments" data-on:toggle__once="@get('/comments')">
        <p>Loading…</p>
    </div>
</details>
```

Server returns `<div id="comments">…actual content…</div>` and the morph swaps in the real content.

## Modal dialog

```html
<button data-on:click="$modal = true">Open</button>

<div data-show="$modal" class="modal-backdrop"
     data-on:click="$modal = false">
    <div class="modal" data-on:click__stop>
        <h2>Title</h2>
        <p>Content</p>
        <button data-on:click="$modal = false">Close</button>
    </div>
</div>
```

**Why `__stop` on the inner div.** Clicks inside the modal would bubble to the backdrop and close it. Stop propagation at the modal boundary.

For modals whose content comes from the server:

```html
<button data-on:click="@get('/edit/123'); $modal = true">Edit</button>
<div data-show="$modal" id="modal-content"></div>
```

Server returns `<div id="modal-content">…form…</div>`.

## Tabs (URL-synced)

```html
<div data-signals:tab="'overview'"
     data-replace-url="`/dashboard?tab=${$tab}`">

    <nav>
        <button data-on:click="$tab = 'overview'"
                data-class:active="$tab === 'overview'">Overview</button>
        <button data-on:click="$tab = 'settings'"
                data-class:active="$tab === 'settings'">Settings</button>
    </nav>

    <section data-show="$tab === 'overview'">…</section>
    <section data-show="$tab === 'settings'">…</section>
</div>
```

`data-replace-url` keeps the URL in sync so refresh / share-link / back-button works (you'd hydrate from `?tab=` on initial render).

For lazy-loaded tab contents, swap `data-show` for an `@get` on click that targets a `#tab-content` region.

## Polling

```html
<div data-on-interval__duration.5s="@get('/status')"></div>

<p>Status: <span data-text="$status"></span></p>
```

Server returns `MarshalAndPatchSignals({"status": "healthy"})`. For higher-frequency polling, prefer the [real-time push pattern](#real-time-push-from-server) — one long-lived SSE is cheaper than N short polls.

## Confirm before action

```html
<button data-on:click="confirm('Delete this item?') && @delete(`/items/${$id}`)">
    Delete
</button>
```

The native `confirm()` blocks until the user answers, and `&&` short-circuits the action. Simple and works.

For a custom UI confirm dialog, use the [modal pattern](#modal-dialog) — set `$confirmFor = 'delete'`, render a confirm modal, and have the OK button fire the actual action.

## Streaming AI / progressive output

```html
<form data-on:submit__prevent="@post('/chat')">
    <input data-bind:prompt />
    <button data-indicator:_streaming type="submit">Ask</button>
</form>

<div id="output" class="prose"></div>
```

Server:

```go
sse := datastar.NewSSE(w, r)

sse.PatchElements(`<div id="output"></div>`) // clear previous

for chunk := range llmStream(s.Prompt) {
    sse.PatchElements(
        fmt.Sprintf(`<span>%s</span>`, html.EscapeString(chunk)),
        datastar.WithSelector("#output"),
        datastar.WithMode(datastar.ElementsModeAppend))
}
```

Each chunk is appended live. The `_streaming` indicator stays true until the handler returns.

For markdown rendering during the stream, accumulate the raw text in a signal and re-render the whole block on each tick — or render server-side per-chunk if you have an incremental markdown parser.
