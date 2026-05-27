# Datastar actions reference

Actions are functions invoked from inside Datastar expressions using `@name(...)` syntax. They're the bridge between client interactions and the backend (HTTP actions) or the signal store (utility actions).

## Table of contents

- [HTTP actions](#http-actions)
- [Action options](#action-options)
- [Timing modifiers](#timing-modifiers-belong-on-the-event-not-the-action)
- [Utility actions](#utility-actions)
- [Pro-only actions](#pro-only-actions)

---

## HTTP actions

| Action | Use for |
|---|---|
| `@get(url, options?)` | Reads. Signals are serialized into the URL query string. |
| `@post(url, options?)` | Creates / arbitrary side effects. Signals as JSON body. |
| `@put(url, options?)` | Replace / idempotent updates. JSON body. |
| `@patch(url, options?)` | Partial updates. JSON body. |
| `@delete(url, options?)` | Deletes. JSON body. |

```html
<button data-on:click="@post('/save')">Save</button>
<input  data-bind:q data-on:input="@get('/search')" />
<button data-on:click="@delete(`/items/${$id}`)">Remove</button>
```

All five expect the server to respond with an SSE stream of `datastar-patch-elements` and/or `datastar-patch-signals` events. The connection closes when the server closes it.

### What gets sent

By default, **all signals** in the store are sent with every HTTP action call. Behavior depends on method:

- `@get` — signals are flattened into the query string as `datastar=<json>` (or expanded keys, depending on backend SDK).
- `@post`, `@put`, `@patch`, `@delete` — signals are JSON in the request body, `Content-Type: application/json`.

The Go SDK's `datastar.ReadSignals(r, &store)` knows the difference and unmarshals from either side.

---

## Action options

The second argument to any HTTP action is an options object. All keys are optional.

```js
@post('/checkout', {
  contentType: 'json',          // 'json' (default) or 'form'
  filterSignals: {              // limit which signals to send
    include: /^cart/,
    exclude: /^_/
  },
  headers: {                    // extra request headers
    'X-Csrf-Token': $csrf
  },
  openWhenHidden: false,        // keep SSE open when tab is hidden
  requestCancellation: 'auto',  // 'auto' | 'cleanup' | 'disabled' | AbortController
  retry: true,                  // auto-reconnect on disconnect
  retryInterval: 1000,          // ms before first retry
  retryScaler: 2,               // multiply interval after each fail
  retryMaxWait: 30000,          // ms cap on retry interval
  retryMaxCount: 5              // give up after this many attempts
})
```

### Key options explained

**`contentType: 'json'` vs `'form'`**

Default is `'json'`. Use `'form'` for file uploads (`<input type="file">`) — the browser will send `multipart/form-data` and the file content will be properly encoded. JSON encoding cannot carry binary file data, so `@post('/upload')` with a bound file input will fail unless you set `contentType: 'form'`.

**`filterSignals: {include, exclude}`**

Regex match against signal names. Both are optional. If `include` is set, only matching signals are sent. If `exclude` is set, those are dropped. Both can be combined.

```html
<!-- Only send cart-related signals -->
<button data-on:click="@post('/checkout', {filterSignals: {include: /^cart/}})">

<!-- Send everything except local UI state (underscore-prefixed convention) -->
<button data-on:click="@post('/save', {filterSignals: {exclude: /^_/}})">
```

Use this for sensitive state (auth tokens), large state (don't ship the full app store to every endpoint), or local UI state (`$_fetching`, `$_modalOpen`).

**`openWhenHidden: true`**

By default Datastar closes the SSE connection when the tab is hidden. For real-time push UIs where you *want* updates to keep arriving in the background, opt in.

**`requestCancellation`**

`'auto'` (default) — a new request from the same element cancels the old in-flight one. `'cleanup'` — let the old one finish. `'disabled'` — never auto-cancel. Pass your own `AbortController` for fine control.

---

## Timing modifiers belong on the event, not the action

Debouncing, throttling, and delaying happen at the event listener level, using **`__modifier`** syntax on the `data-on:*` attribute:

```html
<!-- Right -->
<input data-bind:q
       data-on:input__debounce.300ms="@get('/search')" />

<!-- Wrong — these are not action options -->
<input data-bind:q
       data-on:input="@get('/search', {debounce: 300})" />
```

See `attributes.md` → "Modifier reference" for the full list.

---

## Utility actions

These don't talk to the backend — they manipulate the local signal store.

### `@peek(() => expr)`

Read signals **without subscribing** to them. Useful when you want some signals to be reactive triggers and others to be just-read values.

```html
<!-- Re-renders only when $foo changes, NOT when $bar changes -->
<div data-text="$foo + ' ' + @peek(() => $bar)"></div>
```

### `@setAll(value, filter?)`

Bulk-set many signals at once.

```html
<!-- Set every signal whose name starts with "is" to true -->
<button data-on:click="@setAll(true, {include: /^is/})">Enable all</button>

<!-- No filter — set every signal -->
<button data-on:click="@setAll(false)">Reset</button>
```

### `@toggleAll(filter?)`

Flip every boolean signal matching the filter.

```html
<button data-on:click="@toggleAll({include: /^visible_/})">Toggle visibility</button>
```

---

## Pro-only actions

Require a Datastar Pro license. Not in the free bundle.

| Action | Purpose |
|---|---|
| `@clipboard(text, isBase64?)` | Copy to clipboard. |
| `@fit(v, oldMin, oldMax, newMin, newMax, clamp?, round?)` | Linear map a value from one range to another (useful for sliders, viz). |
| `@intl(type, value, options?, locale?)` | Wraps `Intl.*`. Types: `'datetime'`, `'number'`, `'pluralRules'`, `'relativeTime'`, `'list'`, `'displayNames'`. |

Free-core alternatives:

- Clipboard: a small inline expression using `navigator.clipboard.writeText($text)` in a `data-on:click`.
- `Intl` formatting: do the formatting server-side and send the pre-formatted string in a signal patch, **or** put the call directly in a `data-text` expression — `Intl` is available globally in browsers.
