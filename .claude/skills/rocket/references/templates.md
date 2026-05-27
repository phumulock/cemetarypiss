# Templates — `html`, `svg`, directives, scope

## The `html` tag

`render({ html, ... })` gets a tagged template tag named `html`. Use it to return DOM:

```javascript
render: ({ html, props: { name } }) => html`
  <section>
    <h3>Hello, ${name}</h3>
    <button data-on:click="$$count += 1">Hit me</button>
  </section>
`
```

Interpolation rules:

- **Text content** (`${value}` between tags) is HTML-escaped automatically.
- **Attribute values** (`attr="${value}"`) are coerced to strings and properly quoted.
- **Datastar attribute expressions** are passed through as-is — they're expressions Datastar evaluates at runtime, not interpolations.

A render function can return a single `html` result or an array of them (e.g. when mapping). For arrays:

```javascript
render: ({ html, props: { items } }) => html`
  <ul>
    ${items.map((it) => html`<li>${it.label}</li>`)}
  </ul>
`
```

## The `svg` tag

`render({ svg, ... })` gets a parallel `svg` tag for SVG-namespace content. Use it for `<svg>`, `<path>`, `<circle>`, etc. — these elements need the SVG namespace to render correctly.

```javascript
render: ({ html, svg, props: { value } }) => html`
  <figure>
    ${svg`
      <svg viewBox="0 0 64 64" width="64" height="64">
        <circle cx="32" cy="32" r="28" fill="none" stroke="currentColor" stroke-width="8"></circle>
      </svg>
    `}
    <figcaption>${value}%</figcaption>
  </figure>
`
```

## Datastar attributes inside a Rocket template

**Every** Datastar `data-*` attribute works inside a Rocket render output. The common ones:

| Attribute | What it does |
|---|---|
| `data-bind:fieldName` | Two-way bind input value to signal `$fieldName` (or `$$fieldName` inside Rocket) |
| `data-text="expr"` | Replace text content with `expr` |
| `data-show="expr"` | Toggle `display: none` based on `expr` |
| `data-class:foo="expr"` | Toggle class `foo` when `expr` is truthy |
| `data-attr:disabled="expr"` | Set the `disabled` attribute reactively |
| `data-style="{ width: $$x + 'px' }"` | Reactive inline styles |
| `data-on:click="…"` | Click handler — can be JS expression or `@actionName()` call |
| `data-on:input__debounce.300ms="…"` | Event handler with modifier |
| `data-effect="…"` | Side effect that re-runs when its read signals change |
| `data-computed:label="$$a + $$b"` | Derived signal |
| `data-ref:name` | Tag this element so `refs.name` works in `onFirstRender` |
| `data-indicator:_loading` | Set `$_loading` truthy while a backend action is in flight |
| `data-signals="{x: 0}"` | Initialize local signals declaratively (also works) |

## Rocket-specific directives

### `<template data-if>` / `data-else-if` / `data-else`

Conditional rendering. Must be `<template>` elements.

```html
<template data-if="$$step === 0">
  <p>Idle</p>
</template>
<template data-else-if="$$step === 1">
  <p>Loading</p>
</template>
<template data-else>
  <p>Ready</p>
</template>
```

### `<template data-for>`

Loop rendering. Default item alias is `item`, default index is `i`.

```html
<template data-for="$$letters">
  <li data-text="item"></li>
</template>
```

Custom alias:

```html
<template data-for="letter in $$letters">
  <li data-text="letter"></li>
</template>
```

Alias + index:

```html
<template data-for="letter, row in $$letters">
  <li>
    <strong data-text="row + 1"></strong>
    <span data-text="letter"></span>
  </li>
</template>
```

The iterated value can be any expression: `data-for="x in $$items.filter(i => i.active)"`.

## Scope rewriting and the `__root` opt-out

Inside a Rocket template, **`$$signal` is rewritten to be instance-scoped**. The mechanism is part of how Rocket parses templates — at render time, each `$$name` becomes a unique reference to *this* component's signal store.

To reference a **page-global** signal (anything in `$`, including signals defined by `data-signals` at the page level), append `__root` to the name:

```html
<!-- Read a global theme signal from inside a Rocket component -->
<div data-class:dark="$theme__root === 'dark'">…</div>

<!-- Bind a global signal -->
<input data-bind:userQuery__root />
```

The `__root` modifier is the single most common cause of "this signal isn't updating where I expect" bugs. When in doubt: are you trying to read instance state or page state? Pick `$$` or `$…__root` accordingly.

## Slots

Slots only work when `mode: 'open'` or `mode: 'closed'` — i.e. when the component uses Shadow DOM. In `mode: 'light'` (the default for most patterns), children are placed directly into the host.

Open-mode component with slots:

```javascript
rocket('demo-card', {
  mode: 'open',
  render: ({ html, props: { title } }) => html`
    <article>
      <header>${title}</header>
      <slot></slot>
      <footer><slot name="footer"></slot></footer>
    </article>
  `,
})
```

Usage:

```html
<demo-card title="Hello">
  <p>This goes in the default slot.</p>
  <button slot="footer">Action</button>
</demo-card>
```

Document slots in the component's `manifest` so consumers know what's available:

```javascript
manifest: {
  slots: [
    { name: 'default', description: 'Body content.' },
    { name: 'footer', description: 'Action row.' },
  ],
}
```

## Choosing `mode`

| `mode` | When to use |
|---|---|
| `'light'` | Default-ish choice. No style isolation, no slot semantics — children render in the host element directly. Page CSS works without configuration. Best for components that should blend into the page. |
| `'open'` | Style isolation + slots. Encapsulated; outside CSS does not bleed in (unless you use `::part` or CSS custom properties). External code can still inspect `el.shadowRoot`. |
| `'closed'` | Same as `'open'` but `el.shadowRoot` returns `null` to outside code. Use only if you need that boundary — most apps don't. |

If you reach for shadow DOM by reflex, ask whether you actually need slots or isolation. For 90% of components, `mode: 'light'` is the right call and makes styling much easier.
