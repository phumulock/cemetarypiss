# Lifecycle — setup, render, onFirstRender, cleanup

A Rocket component goes through these phases per instance:

```
construct  →  setup()  →  render()  →  [DOM mounted]  →  onFirstRender()
                ↑                                              ↓
                |  ← observeProps fires on each prop change → |
                |                                              ↓
                |              (signals drive UI naturally)
                ↓
            disconnect  →  cleanup() callbacks run
```

## `setup(context)`

Runs **once per instance**, before the first render.

- Receives the full setup context (`props`, `$$`, `$`, `effect`, `cleanup`, `action`, `observeProps`, `overrideProp`, `defineHostProp`, `render`, `host`, `apply`, `actions`).
- Use it to: initialize `$$` signals, set up `effect`s, register `action`s, kick off async data loads, register prop observers.
- **Do not** try to read DOM refs here — render hasn't happened yet. Move ref-touching code into `onFirstRender`.
- `setup` is optional. A component with no internal state can skip it entirely.

## `render(context, ...extras)`

Returns the DOM for this instance.

- Receives `{ html, svg, props, host }` plus any positional `extras` passed by a `render(overrides, ...args)` call from setup.
- Called on first mount.
- Called again on subsequent prop changes **only if `renderOnPropChange` is truthy** (see below).
- Should be a pure function of `(props, extras)`. State that changes over time should live in `$$` signals and drive the DOM via Datastar attributes — not by re-rendering.
- Returning `html\`...\`` or `svg\`...\`` produces a fragment. You can also return an array of fragments.

## `onFirstRender(context)`

Runs **once**, after the first render has been applied to the DOM.

- Receives the setup context **plus** a `refs` object containing every `data-ref:name` element from the render output.
- Use it for: focusing inputs, measuring layout, wiring `overrideProp` against native input elements, attaching native event listeners that need refs.
- Not called again on later re-renders — it's truly first-render only.

## `renderOnPropChange`

Controls whether the render function is re-invoked when props change. Three forms:

| Value | Behavior |
|---|---|
| `true` | Always re-render on any prop change. |
| `false` | Never re-render on prop change. Signals are the only thing that updates the UI. |
| `(context) => boolean` | Conditional — receives `{ host, props, changes }`, returns whether to re-render. |

**When to set `false`:**

If your `setup` reads props once and pushes them into `$$` signals (typically via `observeProps`), and your render uses those `$$` signals via `data-text` / `data-show` / etc., you don't need re-renders — signal reactivity already updates the DOM. Setting `renderOnPropChange: false` avoids redundant work.

**When to leave it on (the default):**

If your render reads `props.foo` directly in interpolations (`<h3>${props.foo}</h3>`) and you want it to update when the prop changes, you need re-renders.

**Hybrid via callback:**

```javascript
renderOnPropChange: ({ changes }) => 'theme' in changes
```

Only re-renders when the `theme` prop changes; everything else flows through signals.

## `cleanup(fn)`

Registers a teardown callback. Cleanups run when the element disconnects from the DOM. Custom elements can move (re-fire connect/disconnect) and can be removed at any time — never assume a component lives forever.

Always pair these with cleanup:

- `setInterval` / `setTimeout` → `clearInterval` / `clearTimeout`
- `addEventListener` on `window`, `document`, or anything outside the host → `removeEventListener` (or use an `AbortController`)
- `IntersectionObserver`, `ResizeObserver`, `MutationObserver` → `.disconnect()`
- WebSocket / EventSource → `.close()`
- Active `fetch` → `AbortController.abort()` (or just set a cancelled flag)

## Reconnect semantics

If a custom element is moved within the DOM (e.g. via `appendChild` to a new parent), it disconnects then reconnects. Standard custom-element lifecycle — Rocket runs cleanup then re-runs setup. Don't store state in `setup`-local closures expecting it to survive a move; put persistent state on the host or use `$` if it needs cross-instance persistence.
