# Setup context

`setup(context)` runs **once per instance**, before first render. The context object is also passed (minus a few members) to `onFirstRender` and as part of the render context.

## Members

### `props`

Read-only-ish snapshot of the codec-normalized props at setup time. Reading `props.foo` gives the current value, but `props` is not a reactive signals object — to react to changes, use `observeProps`.

### `$$`

Instance-scoped Datastar signal store. Writing `$$.count = 0` creates a reactive signal; reading `$$.count` inside an effect or template subscribes to it. Each component instance has its own `$$`.

Inside templates, `$$count` is auto-rewritten so each instance sees its own value. To reference a *page-global* signal from inside a Rocket template, append `__root`: `$globalThing__root`.

### `$`

Page-global Datastar signal store. Same one used by `data-signals` at the page level. Use sparingly from inside a component — coupling component behavior to global signals leaks abstraction.

### `effect(fn)`

Runs `fn` immediately, then re-runs it whenever any signal it read changes. Like Datastar's `data-effect`, but in JS.

```javascript
setup: ({ $$, effect }) => {
  $$.firstName = ''
  $$.lastName = ''
  effect(() => {
    document.title = `${$$.firstName} ${$$.lastName}`.trim() || 'Untitled'
  })
}
```

If `fn` returns a function, that function runs before the next re-run (per-run cleanup).

### `cleanup(fn)`

Registers a teardown that runs when the element disconnects from the DOM. **Pair every long-lived resource with a cleanup.**

```javascript
setup: ({ cleanup }) => {
  const ctrl = new AbortController()
  window.addEventListener('resize', onResize, { signal: ctrl.signal })
  cleanup(() => ctrl.abort())
}
```

### `action(name, fn)`

Registers an instance-local action. Inside the template, call it with `@name()`.

```javascript
setup: ({ $$, action, props }) => {
  action('copy', async () => {
    await navigator.clipboard.writeText(props.text)
    $$.copied = true
  })
}
```

```html
<button data-on:click="@copy()">Copy</button>
```

Actions can be async, can read/write `$$` and `$`, and can call other actions.

### `actions`

The global action registry. Use to invoke side-wide actions from inside the component:

```javascript
$$.formatted = actions.intl('number', props.amount, { maximumFractionDigits: 0 }, 'en-US')
```

### `observeProps(fn, ...propNames)`

Runs `fn(props, changes)` whenever any of the named props change. Omit `propNames` to observe **all** props.

```javascript
setup: ({ $$, props, observeProps }) => {
  $$.count = props.start
  observeProps(() => { $$.count = props.start }, 'start')
}
```

The `changes` argument is an object mapping changed prop names to their previous values — useful for diff logic.

### `overrideProp(name, getter, setter)`

Wraps the host element's default property accessor. Both `getter` and `setter` are optional; the unwrapped behavior is passed in as `getDefault` / `setDefault`.

Typical use: bridge to a native input element so `host.value` stays in sync with `<input>.value`.

```javascript
onFirstRender: ({ refs, overrideProp }) => {
  overrideProp(
    'value',
    (getDefault) => refs.input?.value ?? getDefault(),
    (value, setDefault) => {
      const next = String(value ?? '')
      if (refs.input && refs.input.value !== next) refs.input.value = next
      setDefault(next)
    },
  )
}
```

Almost always paired with `onFirstRender` because it needs `refs`.

### `defineHostProp(name, descriptor)`

Defines a brand-new property or method on the host element. Useful for exposing instance methods to outside callers:

```javascript
setup: ({ defineHostProp, $$ }) => {
  defineHostProp('reset', {
    value: () => { $$.count = 0 },
  })
}
```

Now `document.querySelector('demo-counter').reset()` works.

### `apply(root, merge?)`

Runs Datastar's `apply` pass on an arbitrary root. Rarely needed — only when you've inserted DOM outside Rocket's render path and want Datastar attributes on it to come alive.

### `render(overrides, ...args)`

Re-runs the render function from setup code, passing extra positional args that land after the render context. The most common use is async data loading:

```javascript
setup: ({ render, cleanup, props }) => {
  let cancelled = false
  ;(async () => {
    try {
      const r = await fetch('/users/' + props.userId + '.json')
      const user = await r.json()
      if (!cancelled) render({}, user, null)
    } catch (err) {
      if (!cancelled) render({}, null, err)
    }
  })()
  cleanup(() => { cancelled = true })
},
render: ({ html, props: { fallbackName } }, user = null, error = null) => {
  if (error) return html`<p>Failed to load user.</p>`
  if (!user) return html`<p>Loading…</p>`
  return html`<article><h3>${user.name ?? fallbackName}</h3></article>`
}
```

The first arg to `render(...)` is an overrides object (typically `{}`); subsequent args are forwarded as positional render args.

### `host`

The custom-element instance itself. Use for low-level DOM operations, dispatching events, reading attributes directly (rarely needed — prefer `props`).

```javascript
host.dispatchEvent(new CustomEvent('close', { bubbles: true, composed: true }))
```

## `onFirstRender` context

`onFirstRender` receives the same context as `setup`, **plus** a `refs` object containing every element that was tagged with `data-ref:name` in the render output. Use this lifecycle hook for any DOM operation that needs an actual element reference.

```javascript
onFirstRender: ({ refs, $$ }) => {
  refs.input.focus()
  $$.width = refs.box.getBoundingClientRect().width
}
```

`refs` is **not** available in `setup` — at setup time, render hasn't run yet, so the elements don't exist.
