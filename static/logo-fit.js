// Scales a streamed ASCII logo so it always fills the middle band between
// navbar and footer, recomputing whenever that band changes size.
// Grid size comes from #logo-box's data-cols/data-rows (falls back to
// measuring the current content for static logos).
(() => {
  const pre = document.getElementById('logo');
  const box = document.getElementById('logo-box');
  if (!pre || !box) return;

  let cols = parseInt(box.dataset.cols, 10);
  let rows = parseInt(box.dataset.rows, 10);
  if (!cols || !rows) {
    const lines = (pre.textContent || '').split('\n');
    rows = rows || lines.length || 1;
    cols = cols || lines.reduce((m, l) => Math.max(m, l.length), 0) || 1;
  }

  // Measure the monospace cell aspect once: width of one full row of `cols`
  // glyphs at a known font-size, with the same font/line settings as #logo.
  const probe = document.createElement('pre');
  probe.style.cssText =
    'position:absolute;visibility:hidden;margin:0;padding:0;white-space:pre;' +
    "line-height:1;font-family:'Courier New',ui-monospace,monospace;font-size:100px;";
  probe.textContent = '@'.repeat(cols);
  document.body.appendChild(probe);
  const rowWidthPerPx = probe.getBoundingClientRect().width / 100; // width of `cols` chars at font-size 1px
  document.body.removeChild(probe);

  function fit() {
    const w = box.clientWidth;
    const h = box.clientHeight;
    if (!w || !h || !rowWidthPerPx) return;
    const byWidth = w / rowWidthPerPx; // font-size where the row spans the box width
    const byHeight = h / rows;         // line-height:1 => rows lines == rows * font-size
    // Pure min-fit: the logo always fits inside the band on both axes, with
    // no horizontal overscan. Earlier versions allowed up to ~1.5× horizontal
    // overflow (relying on overflow:hidden to crop the sparse outer stippling
    // cols of cmlogo.txt) to grow the logo on portrait viewports, but on
    // small screens that visibly clipped the readable letterforms — so this
    // is the safe floor that guarantees the full glyph row stays on-screen.
    const size = Math.max(1.2, Math.floor(Math.min(byWidth, byHeight)));
    pre.style.fontSize = size + 'px';
  }

  // ResizeObserver fires on window resize, orientation change, and any layout
  // shift of the band (e.g. font load) — no separate resize listener needed.
  new ResizeObserver(fit).observe(box);
  fit();
})();
