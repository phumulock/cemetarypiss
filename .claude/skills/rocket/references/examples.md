# Worked examples

Full source for each pattern, runnable as-is once Rocket is imported.

## Counter with reset

Props-driven initial value, instance-local `$$count`, observeProps to re-sync when the prop changes externally.

```javascript
import { rocket } from '/bundles/datastar-pro.js'

rocket('demo-counter', {
  mode: 'light',
  props: ({ number, string }) => ({
    count: number.step(1).min(0).default(0),
    label: string.trim.default('Counter'),
  }),
  setup: ({ $$, observeProps, props }) => {
    $$.count = props.count
    observeProps(() => { $$.count = props.count }, 'count')
  },
  render: ({ html, props: { count, label } }) => html`
    <div class="stack gap-2">
      <button type="button" data-on:click="$$count += 1" data-text="'${label}: ' + $$count"></button>
      <template data-if="$$count !== ${count}">
        <button type="button" data-on:click="$$count = ${count}">Reset</button>
      </template>
    </div>
  `,
})
```

```html
<demo-counter count="5" label="Inventory"></demo-counter>
```

## Progress bar

Pure-prop component — no setup needed. `number.clamp` bounds the value, `data-style` drives width.

```javascript
rocket('demo-progress', {
  props: ({ number, bool, string }) => ({
    value: number.clamp(0, 100).default(0),
    striped: bool,
    label: string.trim.default('Progress'),
  }),
  render: ({ html, props: { value, striped, label } }) => html`
    <div>
      <strong>${label}</strong>
      <div class="bar" data-style="{ width: ${value} + '%' }"></div>
      <span data-show="${striped}">Striped</span>
    </div>
  `,
})
```

## Timer with autoplay toggle

`setInterval` driven, `cleanup` to clear on disconnect, `observeProps` to re-sync when the interval or autoplay prop changes.

```javascript
rocket('demo-timer', {
  props: ({ number, bool }) => ({
    intervalMs: number.min(50).default(1000),
    autoplay: bool,
  }),
  setup: ({ $$, cleanup, props, observeProps }) => {
    $$.seconds = 0
    let timerId = 0

    const syncTimer = () => {
      clearInterval(timerId)
      if (!props.autoplay) return
      timerId = window.setInterval(() => { $$.seconds += 1 }, props.intervalMs)
    }

    syncTimer()
    observeProps(syncTimer)
    cleanup(() => clearInterval(timerId))
  },
  render: ({ html }) => html`<p data-text="$$seconds"></p>`,
})
```

## Async data fetcher

Kick off `fetch` in `setup`, call `render({}, data, error)` when resolved. The render function branches on the tail args.

```javascript
rocket('demo-user-card', {
  props: ({ string, number }) => ({
    userId: number.min(1),
    fallbackName: string.default('Unknown user'),
  }),
  setup: ({ cleanup, render, props }) => {
    let cancelled = false
    ;(async () => {
      try {
        const r = await fetch('/users/' + props.userId + '.json')
        const user = await r.json()
        if (!cancelled) render({}, user, null)
      } catch (error) {
        if (!cancelled) render({}, null, error)
      }
    })()
    cleanup(() => { cancelled = true })
  },
  render: ({ html, props: { fallbackName } }, user = null, error = null) => {
    if (error) return html`<p>Failed to load user.</p>`
    if (!user) return html`<p>Loading user…</p>`
    return html`
      <article>
        <h3>${user.name ?? fallbackName}</h3>
        <p>${user.email ?? 'No email provided'}</p>
      </article>
    `
  },
})
```

## Input bridge — keep host `.value` in sync with native `<input>`

`data-ref:input` exposes the element, `onFirstRender` wires `overrideProp` to forward `host.value` reads/writes to the native input.

```javascript
rocket('demo-input-bridge', {
  props: ({ string }) => ({
    value: string.default(''),
  }),
  render: ({ html, props: { value } }) => html`
    <input data-ref:input value="${value}">
  `,
  onFirstRender: ({ overrideProp, refs }) => {
    overrideProp(
      'value',
      (getDefault) => refs.input?.value ?? getDefault(),
      (value, setDefault) => {
        const next = String(value ?? '')
        if (refs.input && refs.input.value !== next) refs.input.value = next
        setDefault(next)
      },
    )
  },
})
```

## Action handler with `@`-call

`action('copy', fn)` registers a named handler invocable from the template as `@copy()`. Pair with `cleanup` for any side timers.

```javascript
rocket('demo-copy-button', {
  props: ({ string, number }) => ({
    text: string.default('Copy me'),
    resetMs: number.min(100).default(1200),
  }),
  setup: ({ $$, action, cleanup, props }) => {
    $$.copied = false
    $$.label = () => ($$.copied ? 'Copied' : 'Copy')
    let timerId = 0

    action('copy', async () => {
      await navigator.clipboard.writeText(props.text)
      $$.copied = true
      clearTimeout(timerId)
      timerId = window.setTimeout(() => { $$.copied = false }, props.resetMs)
    })

    cleanup(() => clearTimeout(timerId))
  },
  render: ({ html, props: { text } }) => html`
    <button data-on:click="@copy()">
      <span data-text="$$label"></span>
      <small>${text}</small>
    </button>
  `,
})
```

## Conditional rendering

Three branches with `data-if` / `data-else-if` / `data-else`.

```javascript
rocket('demo-status', {
  mode: 'light',
  setup: ({ $$ }) => { $$.step = 0 },
  render: ({ html }) => html`
    <div class="stack gap-2">
      <button type="button" data-on:click="$$step = ($$step + 1) % 3">Next</button>
      <template data-if="$$step === 0"><p>Idle</p></template>
      <template data-else-if="$$step === 1"><p>Loading</p></template>
      <template data-else><p>Ready</p></template>
    </div>
  `,
})
```

## List rendering — array-of-objects prop

Nested codec: `array(object({...}))`. Iterate with `data-for`, or interpolate `.map(...)` for static lists.

```javascript
rocket('demo-nav-list', {
  props: ({ array, object, string }) => ({
    items: array(object({
      href: string.trim.default('#'),
      label: string.trim.default('Untitled'),
    })),
    title: string.trim.default('Navigation'),
  }),
  render: ({ html, props: { items, title } }) => html`
    <nav aria-label="${title}">
      <h3>${title}</h3>
      <ul>
        ${items.map((item) => html`
          <li><a href="${item.href}">${item.label}</a></li>
        `)}
      </ul>
    </nav>
  `,
})
```

Reactive list (uses `$$` signal so additions update without re-render):

```javascript
rocket('demo-letter-list', {
  mode: 'light',
  setup: ({ $$ }) => { $$.letters = ['A', 'B', 'C'] },
  render: ({ html }) => html`
    <ul>
      <template data-for="letter, row in $$letters">
        <li><strong data-text="row + 1"></strong> <span data-text="letter"></span></li>
      </template>
    </ul>
    <button data-on:click="$$letters.push(String.fromCharCode(65 + $$letters.length))">Add</button>
  `,
})
```

## SVG content

```javascript
rocket('demo-meter-ring', {
  props: ({ number, string }) => ({
    value: number.clamp(0, 100).default(0),
    stroke: string.default('#0f172a'),
  }),
  render: ({ html, svg, props: { value, stroke } }) => {
    const C = 2 * Math.PI * 28
    return html`
      <figure class="stack gap-2">
        ${svg`
          <svg viewBox="0 0 64 64" width="64" height="64" aria-hidden="true">
            <circle cx="32" cy="32" r="28" fill="none" stroke="#e5e7eb" stroke-width="8"></circle>
            <circle cx="32" cy="32" r="28" fill="none"
                    stroke="${stroke}" stroke-width="8"
                    stroke-dasharray="${C}"
                    stroke-dashoffset="${C - (value / 100) * C}"
                    transform="rotate(-90 32 32)"></circle>
          </svg>
        `}
        <figcaption>${value}%</figcaption>
      </figure>
    `
  },
})
```

## Slots + manifest

Open shadow DOM for slot support. The `manifest` field documents the component for tooling.

```javascript
import { publishRocketManifests, rocket } from '/bundles/datastar-pro.js'

rocket('demo-dialog', {
  mode: 'open',
  props: ({ string, bool }) => ({
    title: string.default('Dialog'),
    open: bool,
  }),
  manifest: {
    slots: [
      { name: 'default', description: 'Dialog body content.' },
      { name: 'footer',  description: 'Action row content.' },
    ],
    events: [
      { name: 'close', kind: 'custom-event', bubbles: true, composed: true,
        description: 'Fired when the dialog requests dismissal.' },
    ],
  },
  render: ({ html, props: { title, open } }) => html`
    <section data-show="${open}">
      <header>${title}</header>
      <slot></slot>
      <footer><slot name="footer"></slot></footer>
    </section>
  `,
})

// Optionally publish manifests to a backend endpoint for docs tooling:
await publishRocketManifests({ endpoint: '/api/rocket/manifests' })
```

## Exposing instance methods on the host

`defineHostProp` adds properties/methods to the host element itself, so outside JS code can call them.

```javascript
rocket('demo-imperative', {
  setup: ({ $$, defineHostProp }) => {
    $$.count = 0
    defineHostProp('increment', { value: () => { $$.count += 1 } })
    defineHostProp('current',   { get: () => $$.count })
  },
  render: ({ html }) => html`<span data-text="$$count"></span>`,
})
```

```javascript
document.querySelector('demo-imperative').increment()
console.log(document.querySelector('demo-imperative').current)
```
