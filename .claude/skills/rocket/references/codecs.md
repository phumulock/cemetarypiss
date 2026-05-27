# Codecs — Rocket props DSL

A "codec" is a function that takes the raw attribute string from HTML and normalizes it into a typed JS value. Props are declared via a function that receives the codec registry and returns a shape:

```javascript
props: ({ string, number, bool, date, json, js, bin, array, object, oneOf }) => ({
  fieldName: codec.transformation.chain.default(value),
})
```

Each codec exposes a fluent chain. Transformations are applied left-to-right; the value at the end is what `props.fieldName` returns inside `setup` and `render`. If the attribute is absent, the chain's `.default(...)` value is used; if no default is set, the codec's intrinsic default applies (e.g. empty string, 0, false).

## string

| Method | Effect |
|---|---|
| `.trim` | Strip leading/trailing whitespace |
| `.upper` | Uppercase |
| `.lower` | Lowercase |
| `.kebab` | `foo-bar-baz` |
| `.camel` | `fooBarBaz` |
| `.snake` | `foo_bar_baz` |
| `.pascal` | `FooBarBaz` |
| `.title` | `Foo Bar Baz` |
| `.prefix(s)` | Prepend `s` if not already prefixed |
| `.suffix(s)` | Append `s` if not already suffixed |
| `.maxLength(n)` | Truncate to `n` chars |
| `.default(s)` | Fallback value if attribute absent |

Example:

```javascript
slug: string.trim.lower.kebab.maxLength(64).default('untitled')
```

## number

| Method | Effect |
|---|---|
| `.min(n)` | Lower bound |
| `.max(n)` | Upper bound |
| `.clamp(min, max)` | Both bounds in one call |
| `.step(step, base?)` | Snap to nearest `base + k*step` |
| `.round` | Round half-up to integer |
| `.ceil(decimals?)` | Round up; optional decimal precision |
| `.floor(decimals?)` | Round down; optional decimal precision |
| `.fit(inMin, inMax, outMin, outMax, clamped?, rounded?)` | Linear remap of one range to another |
| `.default(n)` | Fallback |

Example:

```javascript
volume: number.clamp(0, 100).round.default(50)
ratio:  number.fit(0, 1, 0, 360).default(0)  // normalized → degrees
```

## bool

```javascript
visible: bool.default(true)
```

Attribute presence semantics: an attribute like `<x-foo visible>` is truthy. Explicit string values like `"false"`, `"0"`, `""` are falsy.

## date

```javascript
when: date.default(new Date())
```

Parses ISO-style date strings into `Date` instances.

## json

Parses the attribute as JSON.

```javascript
config: json.default({ theme: 'light' })
```

```html
<x-card config='{"theme":"dark","compact":true}'></x-card>
```

## js

Evaluates the attribute as a JS expression. **Avoid if you can** — use `json` for data and `string` + chain transforms for everything else.

```javascript
factory: js.default(() => null)
```

## bin

Base64-decoded bytes (returns `Uint8Array` or similar). Used for binary payloads.

## array

Two forms:

**Homogeneous list** — same codec applied to each element:

```javascript
tags: array(string.trim.lower)
```

```html
<x-post tags='["JS","Web","datastar"]'></x-post>
```

**Typed tuple** — positional codecs:

```javascript
point: array(number, number)  // [x, y]
```

## object

Fixed-shape object with per-key codecs:

```javascript
position: object({
  x: number.default(0),
  y: number.default(0),
})
```

Combines well with `array`:

```javascript
items: array(object({
  href: string.trim.default('#'),
  label: string.trim.default('Untitled'),
}))
```

## oneOf

Enum (literal values) or union (codec alternatives):

```javascript
size:   oneOf('sm', 'md', 'lg').default('md')
value:  oneOf(number, string).default(0)
```

`.default(v)` provides an explicit fallback; without one, the first alternative's default is used.

## Codec edge cases

- **Codecs run only on attribute strings.** When JS code does `el.someProp = 42`, that bypasses the codec entirely. If you need property-set normalization, use `overrideProp` in `onFirstRender` (see `setup-context.md`).
- **Missing attribute → default chain.** The chain's `.default(...)` (if any) wins. If no default is set, the codec's intrinsic default is used.
- **Codecs do not throw on invalid input.** Bad JSON in a `json` prop yields the default. Out-of-range numbers are clamped if `.clamp` is in the chain, or coerced if not.
- **Order matters in the chain.** `string.trim.lower.prefix('user:')` trims first, then lowercases, then prefixes. Reordering changes the result.
