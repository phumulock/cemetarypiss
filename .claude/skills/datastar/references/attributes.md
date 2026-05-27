# Datastar attributes reference

Every `data-*` attribute Datastar recognizes, what it does, exact syntax, and a tiny example.

Free-core attributes are listed first. Pro-only attributes are in their own section at the bottom and **will silently not work** without a Datastar Pro license.

## Table of contents

- [Reactivity & display](#reactivity--display)
- [Event listeners](#event-listeners)
- [URL & browser](#url--browser)
- [Lifecycle & escape hatches](#lifecycle--escape-hatches)
- [Modifier syntax](#modifier-syntax)
- [Modifier reference](#modifier-reference)
- [Pro-only attributes](#pro-only-attributes)

---

## Reactivity & display

### `data-signals`

Initializes (or patches) signals into the reactive store. Place on any element; signals are global.

```html
<!-- Object form, multiple signals -->
<div data-signals="{count: 0, user: {name: 'Ada'}}"></div>

<!-- Single-signal shorthand -->
<div data-signals:count="0"></div>

<!-- Nested via dot notation -->
<div data-signals:user.name="'Ada'"></div>
```

Use the `__ifmissing` modifier to only set a signal if it doesn't already exist (don't clobber an existing value):

```html
<div data-signals:count__ifmissing="0"></div>
```

### `data-bind`

Two-way binds a form element (`<input>`, `<textarea>`, `<select>`, also `<input type="checkbox">`, `<input type="radio">`, `<input type="file">`) to a signal. Creates the signal if it doesn't exist.

```html
<input data-bind:search />            <!-- signal: $search -->
<input data-bind="firstName" />       <!-- equivalent alt syntax -->
<input data-bind:user.email />        <!-- nested: $user.email -->
```

Modifiers: `__case.camel|kebab|snake|pascal` (transform signal name), `__prop` (bind to a DOM property instead of attribute, e.g. for richer input components), `__event.input|change` (which event triggers the sync — `input` by default for text inputs).

### `data-text`

Sets `textContent` reactively from an expression. Safe — does not interpret HTML.

```html
<div data-text="$message"></div>
<div data-text="`Hello, ${$user.name}`"></div>
```

### `data-show`

Toggles `display: none` based on expression truthiness. Element stays in the DOM; it's a CSS-level hide.

```html
<div data-show="$count > 0">Some items</div>
<div data-show="!$loading">Done</div>
```

For *actual* DOM removal, emit a `mode: remove` patch from the server.

### `data-class`

Reactively adds/removes classes.

```html
<!-- Object map form: keys are class names, values are booleans -->
<div data-class="{active: $selected, disabled: !$enabled}"></div>

<!-- Single-class shorthand -->
<div data-class:font-bold="$emphasized"></div>
```

### `data-attr`

Reactively sets any HTML attribute.

```html
<button data-attr:disabled="$loading">Submit</button>
<a data-attr:href="`/users/${$id}`">Profile</a>

<!-- Object form: multiple attributes at once -->
<div data-attr="{'aria-label': $label, 'data-state': $state}"></div>
```

### `data-style`

Reactively sets inline CSS properties.

```html
<div data-style:color="$urgent ? 'red' : 'black'"></div>
<div data-style:display="$hiding ? 'none' : ''"></div>
```

### `data-computed`

Creates a read-only derived signal. Recomputed whenever its dependencies change.

```html
<input data-bind:firstName />
<input data-bind:lastName />
<div data-computed:fullName="`${$firstName} ${$lastName}`"
     data-text="$fullName"></div>
```

You cannot write to a `data-computed` signal (it's derived). Trying to set it is silently ignored.

### `data-effect`

Runs an expression when the element mounts and whenever any signal it references changes. Like a `useEffect` for the DOM element's lifetime.

```html
<div data-effect="document.title = `Count: ${$count}`"></div>
```

Use sparingly. Most things you'd reach for `data-effect` for are better expressed with `data-text`, `data-attr`, `data-class`, `data-show`, or a server-side patch.

### `data-ref`

Creates a signal whose value is the DOM element itself. Useful for measuring, focusing, calling methods on a third-party widget, etc.

```html
<div data-ref:viewport>...</div>
<button data-on:click="$viewport.scrollTo({top: 0})">Top</button>
```

### `data-indicator`

Sets a signal to `true` while a fetch initiated from this element is in flight; `false` otherwise. Convention is to name these with a leading underscore for "local UI state".

```html
<button data-indicator:_saving
        data-attr:disabled="$_saving"
        data-on:click="@post('/save')">
  <span data-show="!$_saving">Save</span>
  <span data-show="$_saving">Saving…</span>
</button>
```

### `data-json-signals`

Debug helper. Renders the entire signals store as live JSON. Drop one on the page while developing.

```html
<pre data-json-signals></pre>

<!-- Compact form -->
<pre data-json-signals__terse></pre>
```

---

## Event listeners

### `data-on:eventName`

Generic event listener. `eventName` is **any DOM event** — `click`, `input`, `change`, `submit`, `keydown`, `mouseenter`, `focus`, `blur`, `load`, `pointermove`, `transitionend`, …

```html
<button data-on:click="$count++">+</button>
<form data-on:submit__prevent="@post('/save')">…</form>
<input data-on:keydown__debounce.500ms="@get('/search')" />
<div data-on:load="@get('/init')"></div>
```

All `data-on` modifiers are listed in the [modifier reference](#modifier-reference).

### `data-on-intersect`

Fires when the element enters the viewport. Backed by IntersectionObserver.

```html
<div data-on-intersect__once="@get('/load-more')">Loading…</div>
```

Modifiers: `__once`, `__exit` (fire on leave instead), `__half` (50% visible), `__full` (100% visible), `__threshold.0.25` (custom ratio), `__delay.500ms`, `__debounce`, `__throttle`, `__viewtransition`.

### `data-on-interval`

Fires on a recurring interval.

```html
<div data-on-interval__duration.1s="@get('/heartbeat')"></div>
<div data-on-interval__duration.500ms="$tick++"></div>
```

Modifiers: `__duration.<ms>` (default 1s if omitted), `__viewtransition`.

### `data-on-signal-patch`

Fires whenever any signal in the store is patched. Pair with `data-on-signal-patch-filter` to scope.

```html
<div data-on-signal-patch="console.log('something changed')"></div>

<!-- Only fire when signals matching the regex change -->
<div data-on-signal-patch-filter="{include: /^counter$/}"
     data-on-signal-patch="@post('/log')"></div>
```

Modifiers: `__delay`, `__debounce`, `__throttle`.

---

## URL & browser

### `data-replace-url`

Replaces `window.location` via `history.replaceState` based on a template-literal expression. No reload.

```html
<div data-replace-url="`/page/${$page}`"></div>
```

Use for keeping the URL in sync with pagination, tabs, or filter state when you don't want a full navigation.

---

## Lifecycle & escape hatches

### `data-init`

Runs an expression once when the element initializes. Useful for one-shot setup.

```html
<div data-init="$count = 1"></div>
```

(For most cases, `data-signals` is what you want instead; `data-init` is for when you need imperative setup.)

### `data-ignore`

Marks the element and **all descendants** as Datastar-inert. Datastar will not parse, bind, or morph anything underneath.

```html
<div data-ignore>
  <!-- Third-party widget that mutates its own DOM -->
  <div id="thirdparty-mount"></div>
</div>
```

Modifier: `__self` to scope only to this element (descendants stay reactive).

### `data-ignore-morph`

The element is still reactive, but the morph algorithm won't touch its subtree. Use when a third-party widget owns the inner DOM but the wrapper still participates in Datastar.

```html
<div id="chart" data-ignore-morph>
  <!-- Chart library writes here; Datastar leaves it alone -->
</div>
```

### `data-preserve-attr`

Tells the morph not to overwrite a specific attribute. Crucial for things like `<details open>` where you want client-side toggle state to survive a re-morph.

```html
<details data-preserve-attr="open">
  <summary>More</summary>
  …
</details>

<!-- Preserve multiple -->
<details data-preserve-attr="open class">…</details>
```

---

## Modifier syntax

Datastar modifiers attach to an attribute name with **double underscore**:

```
data-<attr>:<key>__<modifier>__<another-modifier>="<expression>"
```

If a modifier takes an argument, the argument follows a single dot:

```
data-on:click__debounce.300ms__throttle.1s="…"
data-on-intersect__threshold.0.5="…"
data-on-interval__duration.500ms="…"
data-bind:userName__case.kebab
```

The `.` is **only** for the modifier argument. Stacking modifiers uses more `__`. Coming from Alpine, which uses dots for everything, this is the most common mistake.

---

## Modifier reference

### `data-on` modifiers

| Modifier | What it does |
|---|---|
| `__once` | Fire only the first time |
| `__passive` | Add `{passive: true}` to the listener |
| `__capture` | Listen in the capture phase |
| `__prevent` | Call `event.preventDefault()` |
| `__stop` | Call `event.stopPropagation()` |
| `__outside` | Fire only when the event originated *outside* this element |
| `__window` | Attach the listener to `window`, not the element |
| `__document` | Attach to `document` |
| `__debounce.<ms>` | Debounce by that many ms (e.g. `__debounce.300ms`) |
| `__throttle.<ms>` | Throttle by that many ms |
| `__delay.<ms>` | Delay each fire by that many ms |
| `__viewtransition` | Wrap the resulting DOM updates in a View Transition |
| `__case.<style>` | Case-transform the event name: `camel`, `kebab`, `snake`, `pascal` |

### `data-bind` modifiers

| Modifier | What it does |
|---|---|
| `__case.<style>` | Transform signal name case |
| `__prop` | Bind to a DOM property instead of attribute |
| `__event.<name>` | Sync on a different event (default is `input`) |

### `data-signals` modifiers

| Modifier | What it does |
|---|---|
| `__case.<style>` | Transform signal name case |
| `__ifmissing` | Only initialize if signal doesn't already exist |

### `data-on-intersect` modifiers

`__once`, `__exit`, `__half`, `__full`, `__threshold.<ratio>`, `__delay.<ms>`, `__debounce.<ms>`, `__throttle.<ms>`, `__viewtransition`.

### `data-on-interval` modifiers

`__duration.<ms>` (default 1s), `__viewtransition`.

### `data-on-signal-patch` modifiers

`__delay.<ms>`, `__debounce.<ms>`, `__throttle.<ms>`.

### `data-json-signals` modifiers

`__terse` — compact output.

### `data-ignore` modifiers

`__self` — only this element, not descendants.

---

## Pro-only attributes

The following require a **Datastar Pro** commercial license. They are not in the free bundle, so writing them in code that loads the free `datastar.js` will silently do nothing.

| Attribute | Purpose |
|---|---|
| `data-animate` | Animate attributes over time from signals |
| `data-custom-validity` | Custom form validation: `data-custom-validity="$a === $b ? '' : 'Mismatch'"` |
| `data-match-media` | Bind a signal to a media query result |
| `data-on-raf` | Run on every `requestAnimationFrame` |
| `data-on-resize` | Run when element dimensions change (ResizeObserver) |
| `data-persist` | Persist signals to localStorage / sessionStorage |
| `data-scroll-into-view` | Scroll element into view (many modifiers) |
| `data-view-transition` | Set `view-transition-name` reactively |
| `data-query-string` | Sync URL query string with signals (with history mode) |

If the user has confirmed they have Pro (or the project already uses these), then these are available — otherwise prefer free-core alternatives:

| Pro need | Free-core alternative |
|---|---|
| `data-persist` | Server-side session storage + signal patch on load |
| `data-match-media` | A `data-effect` with `window.matchMedia` listener |
| `data-on-raf` / `data-on-resize` | A `data-effect` registering the relevant listener |
| `data-animate` | CSS transitions triggered by `data-class` toggles |
| `data-query-string` | `data-replace-url` (URL sync, no auto-bind) |
