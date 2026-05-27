# Datastar SSE events — wire format

In **v1.x**, Datastar uses exactly two SSE event types. Pre-v1 names (`datastar-merge-fragments`, `datastar-merge-signals`, `datastar-remove-fragments`, `datastar-remove-signals`, `datastar-execute-script`) are obsolete and **not interpreted by the v1 client** — using them is a no-op.

Backend SDKs (like `datastar-go`) handle this framing for you. You only need this file if you're hand-rolling SSE or debugging the wire.

## SSE refresher (basics that bite people)

- Each event is a sequence of `<field>: <value>` lines, **terminated by a blank line**. Missing the blank line means the browser buffers indefinitely.
- For multi-line values, repeat the field name once per line:
  ```
  data: line 1
  data: line 2
  ```
  The browser joins these with `\n` and dispatches once.
- Set response headers correctly: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, and either `Connection: keep-alive` (HTTP/1.1) or trust the HTTP/2 framing. Flush after every event.

---

## `datastar-patch-elements`

Patches the DOM.

```
event: datastar-patch-elements
data: selector #results
data: mode inner
data: useViewTransition false
data: elements <div class="row">
data: elements   <span>Found 3 matches</span>
data: elements </div>

```

### Fields

| Field | Required? | Values | Default |
|---|---|---|---|
| `selector` | Required for `inner`, `replace`, `append`, `prepend`, `before`, `after`, `remove` | Any CSS selector | (none) |
| `mode` | No | `outer`, `inner`, `replace`, `append`, `prepend`, `before`, `after`, `remove` | `outer` |
| `namespace` | No | `svg`, `mathml` | (HTML) |
| `useViewTransition` | No | `true`, `false` | `false` |
| `elements` | Required (except for `mode remove`) | HTML, can span multiple `data: elements` lines | (none) |

### Mode semantics

| Mode | Behavior |
|---|---|
| `outer` (default) | Find a live DOM element with the same `id` as the top-level element in `elements`; morph it. Selector is **not used**. |
| `inner` | Replace `innerHTML` of the element matching `selector`. |
| `replace` | Same as `outer` but doesn't morph — just hard-replace. |
| `append` | Append `elements` as children of `selector`. |
| `prepend` | Prepend `elements` as children of `selector`. |
| `before` | Insert `elements` immediately before `selector`. |
| `after` | Insert `elements` immediately after `selector`. |
| `remove` | Remove the element matching `selector`. `elements` not used. |

### Examples

**Default: morph by id**

```
event: datastar-patch-elements
data: elements <div id="counter">Count: 42</div>

```

`#counter` is found and its outer HTML is morphed to the new version. Datastar preserves any descendant signals and `data-*` reactivity where the IDs match.

**Replace inner content**

```
event: datastar-patch-elements
data: selector #list
data: mode inner
data: elements <li>a</li>
data: elements <li>b</li>
data: elements <li>c</li>

```

**Append a row to a table**

```
event: datastar-patch-elements
data: selector #rows
data: mode append
data: elements <tr><td>new</td></tr>

```

**Remove an element**

```
event: datastar-patch-elements
data: selector #notification
data: mode remove

```

**Execute a script** (replacement for the old `datastar-execute-script` event)

```
event: datastar-patch-elements
data: selector body
data: mode append
data: elements <script>console.log('hi')</script>

```

The browser executes the script as it's inserted. (Datastar-Go's `sse.ExecuteScript("…")` is just sugar for this.)

---

## `datastar-patch-signals`

Patches the signals store. The body is a JSON object that is **deep-merged** into the current store. Setting a key to `null` removes it.

```
event: datastar-patch-signals
data: signals {"user":{"name":"Ada"},"count":42}

```

### Fields

| Field | Required? | Values | Default |
|---|---|---|---|
| `signals` | Required | A single line of JSON | (none) |
| `onlyIfMissing` | No | `true`, `false` | `false` |

### `onlyIfMissing`

When `true`, keys that already exist in the client store are **not** overwritten. Use this for initial defaults that should not clobber user input.

```
event: datastar-patch-signals
data: onlyIfMissing true
data: signals {"theme":"dark","language":"en"}

```

### Removing signals

Set them to `null`:

```
event: datastar-patch-signals
data: signals {"_tempFlag":null,"draft":null}

```

The client store will have those keys deleted.

### Nested merging

The merge is deep. Given a store of `{user: {name: "Ada", email: "a@x"}}`, sending:

```
event: datastar-patch-signals
data: signals {"user":{"name":"Lovelace"}}

```

…leaves `user.email` intact and updates only `user.name`. To remove `user.email`, send `{"user":{"email":null}}`.

---

## Interleaving in one stream

A single SSE response can emit any mix of both event types, in any order, and end whenever it wants:

```
event: datastar-patch-signals
data: signals {"_fetching":true}

event: datastar-patch-elements
data: selector #results
data: mode inner
data: elements <p>Searching…</p>

event: datastar-patch-elements
data: selector #results
data: mode inner
data: elements <ul><li>a</li><li>b</li></ul>

event: datastar-patch-signals
data: signals {"_fetching":false,"resultsCount":2}

```

For request/response style, just close the stream after the last event. For server-push real-time UIs, keep it open and emit events as they happen — Datastar reconnects automatically (with backoff) if the connection drops.

---

## Debugging

- **Patch lands nowhere visible** — for `mode outer`, the top-level element in `elements` has an `id` that doesn't match anything on the page. Check ids. Or you meant `mode inner` + `selector`.
- **Browser hangs and shows no update** — missing blank line after the event. Each event ends with `\n\n`.
- **Multi-line HTML appears mangled** — every line of HTML needs its own `data: elements ` prefix (with the space after the colon). SDKs handle this; raw output does not.
- **Signal that should be removed is still there** — `null` removes; `""` or `undefined` does not.
- **Old event names silently ignored** — you're emitting `datastar-merge-fragments` or similar. Switch to `datastar-patch-elements` / `datastar-patch-signals`.

For Go specifically, the SDK has the framing right by default. If you're seeing wire-format issues with `datastar-go`, the bug is most likely in your selector or mode rather than the framing.
