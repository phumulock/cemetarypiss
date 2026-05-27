---
name: rocket
description: Build Datastar Pro Rocket web components — the `rocket(tag, definition)` custom-element API with codec-normalized props, `$$` instance-local signals, and `html` / `svg` tagged templates. Use whenever the user asks to author a custom element with Rocket, mentions `rocket()`, `$$` signals, `observeProps`, codecs like `string.trim.lower`, `data-if` / `data-for` templates, Datastar Pro, or imports from `datastar-pro.js`. Trigger even if the user just describes "a custom element that uses Datastar signals" or "a reactive web component on top of Datastar" — Rocket's model (codec props + instance-scoped `$$` signals + Datastar attributes inside tagged templates) differs enough from Lit / Stencil / vanilla custom elements that defaulting to those patterns will produce wrong code. This skill is complementary to the `datastar` skill: use both together when a Rocket component talks to a Go SSE backend.
---

# Rocket

Rocket is **Datastar Pro's** custom-element API. You call `rocket(tag, definition)` once, and from then on every `<my-tag>` in the document becomes a reactive web component. The component's **public API** is a set of typed, codec-normalized `props`. Its **internal state** is a bag of Datastar signals exposed as `$$`. Its **template** is a tagged template literal returning DOM, with ordinary Datastar `data-*` attributes for reactivity.

There is **no** proprietary templating, no virtual DOM, no JSX. It's HTML, plus Datastar attributes, plus a thin lifecycle. The reactivity engine is the same one driving the rest of your Datastar app.

This skill assumes Datastar Pro. The bundle is `/bundles/datastar-pro.js` (not the OSS `datastar.js`). Mention this if the user's project doesn't already have it.

## When to use this skill

Use this skill when:

- You're calling `rocket(...)` or seeing it imported.
- The user wants to build a reusable custom element / web component on top of Datastar.
- You see `$$signalName` (note the **double** `$`) — that's a Rocket instance-scoped signal. Single `$signal` is a page-global Datastar signal; the two are different scopes.
- You see codec calls like `string.trim.lower.maxLength(48)`, `number.clamp(0, 100)`, `array(object({...}))` — that's the Rocket props DSL.
- The user mentions `observeProps`, `overrideProp`, `defineHostProp`, `onFirstRender`, or `publishRocketManifests`.

Use the sibling **`datastar`** skill in parallel when the Rocket component also fires actions at a Datastar backend (`@get`, `@post`, SSE patches). Rocket components freely mix global `$signals` and instance `$$signals`; both still drive Datastar reactivity.

Do **not** reach for Lit's `@property` decorator, Stencil's `@Component`, or React-style state hooks — Rocket is its own thing and they don't map cleanly.

## The mental model

A Rocket component has four moving parts, in order:

1. **Props** — the public API. Declared via a codec function: `props: ({ string, number, bool, array, object, oneOf }) => ({...})`. Each codec normalizes the incoming HTML attribute string (e.g. `string.trim.lower` lowercases, `number.clamp(0, 100)` bounds). Props are read in `setup` and `render` via `props.name`. They are **not** reactive in the signals sense — to react to prop changes, use `observeProps`.

2. **Setup** — runs once per instance, before first render. Receives `{$$, $, props, effect, cleanup, action, observeProps, render, host, ...}`. This is where you initialize instance signals (`$$.count = 0`), wire side effects, register actions, and hook prop observers. **Refs from `data-ref:name` are NOT yet available here** — use `onFirstRender` for DOM operations needing refs.

3. **Render** — a function returning DOM via the `html` (or `svg`) tagged template tag. Called on first mount, then on each prop change if `renderOnPropChange` is truthy (default behavior depends on whether you provide a `setup` — see `references/lifecycle.md`). Inside the template you use **ordinary Datastar attributes** (`data-on:click`, `data-bind`, `data-text`, `data-show`, `data-class`, etc.) and Rocket's two template directives: `<template data-if>` and `<template data-for>`.

4. **Cleanup** — `cleanup(fn)` registers a teardown that runs when the element disconnects. Use it for `clearInterval`, `AbortController.abort()`, websocket close, etc.

**Key invariant: `$$name` inside a template is automatically rewritten to be instance-scoped.** Two `<demo-counter>` elements on the same page each get their own `$$count`. To reference a *page-global* Datastar signal from inside a Rocket template, append `__root`: `data-text="$globalCounter__root"`. This is the single most surprising thing about Rocket — write it down.

## Workflow

For any non-trivial Rocket component:

1. **Name the public API.** What attributes does `<my-element>` accept? Each becomes a prop. Pick the right codec: `number.clamp(...)` for bounded numeric, `string.trim.default('...')` for text, `bool` for flags, `array(object({...}))` for structured data, `oneOf('sm', 'md', 'lg')` for enums.
2. **Name the internal state.** What does the component remember between renders? That's a `$$` signal. Initialize it in `setup`. If it derives from a prop, set it in `setup` and refresh inside an `observeProps` callback.
3. **Decide rendering mode.** Default to `mode: 'light'` unless you genuinely need style isolation or `<slot>`. Light DOM = your page's CSS just works.
4. **Decide if you need refs.** If your render emits `data-ref:foo`, you can read `refs.foo` inside `onFirstRender` (NOT `setup`). Use this for measuring, focusing, bridging to native form-control behavior, etc.
5. **Decide if you need actions.** If a click handler is more than one expression, register it: `action('save', async () => {...})` and call `@save()` from the template.
6. **Always register cleanup.** Any `setInterval`, `setTimeout`, `addEventListener` on `window`, `AbortController`, subscription — pair it with `cleanup(() => ...)`. Custom elements can be torn down at any time.

## The shape — a complete minimal component

```javascript
import { rocket } from '/bundles/datastar-pro.js'

rocket('demo-counter', {
  mode: 'light',
  props: ({ number, string }) => ({
    start: number.min(0).default(0),
    label: string.trim.default('Count'),
  }),
  setup: ({ $$, props, observeProps }) => {
    $$.count = props.start
    observeProps(() => { $$.count = props.start }, 'start')
  },
  render: ({ html, props: { label } }) => html`
    <button type="button" data-on:click="$$count += 1">
      <span data-text="'${label}: ' + $$count"></span>
    </button>
  `,
})
```

Usage in HTML:

```html
<demo-counter start="5" label="Hits"></demo-counter>
```

Two instances on the same page have independent `$$count` — that's the auto-rewrite.

## Cheat sheet — codecs

| Codec | Common chain | Notes |
|---|---|---|
| `string` | `.trim.lower.maxLength(48)`, `.default('x')` | Also `.upper`, `.kebab`, `.camel`, `.snake`, `.pascal`, `.title`, `.prefix(s)`, `.suffix(s)` |
| `number` | `.clamp(0, 100)`, `.min(n)`, `.max(n)`, `.step(1)`, `.round`, `.default(0)` | Also `.ceil(d)`, `.floor(d)`, `.fit(inMin, inMax, outMin, outMax)` |
| `bool` | `.default(false)` | Truthy/falsy parsing of attribute strings |
| `date` | `.default(...)` | ISO-ish parsing |
| `json` | `.default({})` | Parse `JSON.parse` from attribute |
| `js` | `.default(...)` | Eval expression (use sparingly) |
| `bin` | `.default(...)` | Base64-decoded bytes |
| `array(codec)` | `array(string)` | Homogeneous list |
| `array(a, b, c)` | `array(number, string, bool)` | Typed tuple |
| `object({ k: codec })` | `object({ x: number, y: number })` | Fixed-shape object |
| `oneOf('a', 'b')` | `oneOf('sm', 'md', 'lg').default('md')` | Enum or union |

See `references/codecs.md` for every transformation chain and edge cases.

## Cheat sheet — setup context

Inside `setup({ ... })` you can destructure any of:

| Member | What it does |
|---|---|
| `props` | Normalized, decoded prop values (read-only-ish snapshot — see `observeProps` for change reactions) |
| `$$` | Instance-scoped Datastar signals. Write/read freely. Auto-rewritten inside templates. |
| `$` | Page-global Datastar signal store. Use sparingly from a component. |
| `effect(fn)` | Reactive effect; rerun whenever read signals change. Return a cleanup fn for per-run teardown. |
| `cleanup(fn)` | Register disconnect-time teardown. **Use this for every timer / listener / abort controller.** |
| `action(name, fn)` | Register instance-local action callable as `@name()` from the template. |
| `actions` | Global action registry — call other registered actions if needed. |
| `observeProps(fn, ...names)` | Run `fn(props, changes)` whenever any of the named props change. Omit names to observe all. |
| `overrideProp(name, getter, setter)` | Wrap the host's default property accessor — used to bridge to native input behavior. Pair with `onFirstRender`. |
| `defineHostProp(name, descriptor)` | Define a brand-new property/method on the host element. |
| `apply(root, merge?)` | Run Datastar's `apply` on a root manually — rarely needed. |
| `render(overrides, ...args)` | Re-run the render function from setup code, passing extra positional args. Useful for async data loads. |
| `host` | The custom-element instance itself. |

See `references/setup-context.md` for signatures and examples.

## Cheat sheet — templates

Inside `render({ html, svg, props, host })`:

- `html\`...\`` — returns an `HTMLTemplateResult`. Interpolation is safe by default; events/attributes work as in normal HTML.
- `svg\`...\`` — same, for SVG namespace content.
- All standard Datastar attributes work: `data-bind`, `data-on:*`, `data-text`, `data-show`, `data-class:*`, `data-attr:*`, `data-style`, `data-computed:*`, `data-effect`, `data-ref:*`, `data-indicator:*`.
- `<template data-if="expr">…</template>` plus `<template data-else-if="expr">` and `<template data-else>` — conditional rendering.
- `<template data-for="$$items">…</template>` — loop, with `item` and `i` as default aliases. `data-for="letter in $$items"` for custom alias. `data-for="letter, row in $$items"` for alias + index.
- **Scope opt-out:** `data-bind:name__root`, `$globalSignal__root`, `$$rewritten__root` (etc.) skip Rocket's `$$` rewriting and reference page-scope.
- For `<slot>` content, use `mode: 'open'` or `mode: 'closed'`. In `mode: 'light'`, slots are not available — children render in the host directly.

See `references/templates.md` for the full directive list with examples.

## Common patterns (worked examples)

See `references/examples.md` for full source of each:

- **Counter with reset** — basic props + `$$` signal + observeProps.
- **Progress bar** — `number.clamp` + `data-style`.
- **Timer with autoplay toggle** — `setInterval` + `cleanup` + `observeProps` to re-sync on prop change.
- **Async data fetcher** — kick off `fetch` in `setup`, call `render({}, data, error)` when it resolves, branch in render on tail args.
- **Input bridge** — `data-ref:input` + `onFirstRender` + `overrideProp` to keep host `.value` and native input in sync.
- **Action handler with `@`-call** — `action('copy', async ...)` + `data-on:click="@copy()"`.
- **List rendering** — `array(object({...}))` prop + `data-for` template.
- **SVG content** — `svg` tagged template inside `html`.
- **Slots + manifest** — `mode: 'open'`, `<slot>` elements, `manifest` metadata, `publishRocketManifests`.

## Things that will catch you out

- **`$` vs `$$`.** Single dollar = global. Double dollar = instance. Mixing them up gives "works on one instance but not when there are two" bugs.
- **`__root` is the escape hatch.** When you genuinely need a page signal inside a Rocket template (e.g. global theme), append `__root` so Rocket doesn't rewrite it.
- **Refs aren't ready in `setup`.** Use `onFirstRender` if you need `refs.foo`.
- **Codecs run on attribute strings, not property assignments.** If JS code does `el.someProp = 'x'`, that bypasses the codec unless you wired `overrideProp`.
- **`renderOnPropChange` default depends on shape.** If you have `setup` *and* it writes to `$$` in response to `observeProps`, you usually want `renderOnPropChange: false` and let signal reactivity drive UI updates. If you have no setup and your render reads `props.*` directly, leave it on. See `references/lifecycle.md`.
- **Cleanup is not optional.** Custom elements can be moved across the document tree (which re-fires connect/disconnect) and removed without warning. Unsubscribe everything.
- **The Pro bundle is required.** `import { rocket } from '/bundles/datastar-pro.js'`. The free `datastar.js` bundle does not include Rocket.

## Reference files

- `references/codecs.md` — every codec, every chain, edge cases
- `references/setup-context.md` — full signatures for `effect`, `observeProps`, `overrideProp`, `defineHostProp`, `action`, `render(overrides, ...args)`, `apply`
- `references/templates.md` — `html` / `svg` semantics, every Datastar attribute usable inside Rocket, `data-if` / `data-for` directives, `__root` scope opt-out, slots
- `references/lifecycle.md` — `setup` vs `onFirstRender` vs `render`, `renderOnPropChange` semantics, disconnect ordering
- `references/examples.md` — worked components for each common pattern
