// Background music player (ported from fire.js's audio block) driving the
// footer controls. Cross-fades between tracks on a timer; prev/next switch
// manually. No-ops if the expected elements aren't on the page.
(() => {
  const tracks = [
    document.getElementById('bg-audio'),
    document.getElementById('bg-audio-2'),
  ];
  const trackNames = ['intro', 'terrorvoid'];
  const audioToggle = document.getElementById('audio-toggle');
  const prevBtn = document.getElementById('prev-btn');
  const nextBtn = document.getElementById('next-btn');
  if (!audioToggle || !prevBtn || !nextBtn || tracks.some((t) => !t)) return;

  const MAX_VOL  = 0.15;
  const CYCLE_MS = 30000;
  const FADE_MS  = 3000;

  let playing    = false;
  let currentIdx = 0;
  let cycleTimer = null;
  const fadeState = new WeakMap();

  function fade(el, from, to, ms, cb) {
    const prev = fadeState.get(el);
    if (prev) prev.cancelled = true;
    const token = { cancelled: false };
    fadeState.set(el, token);

    el.volume = Math.min(1, Math.max(0, from));
    const start = performance.now();
    function step(now) {
      if (token.cancelled) return;
      const t = Math.min(1, (now - start) / ms);
      el.volume = Math.min(1, Math.max(0, from + (to - from) * t));
      if (t >= 1) {
        fadeState.delete(el);
        if (cb) cb();
      } else {
        requestAnimationFrame(step);
      }
    }
    requestAnimationFrame(step);
  }

  function runCycle(idx) {
    currentIdx = idx;
    audioToggle.textContent = '♪ ' + trackNames[idx];
    const track = tracks[idx];
    track.volume = 0;
    track.play().catch(() => {});
    fade(track, 0, MAX_VOL, FADE_MS);
    cycleTimer = setTimeout(() => {
      fade(track, MAX_VOL, 0, FADE_MS, () => {
        track.pause();
        track.currentTime = 0;
        if (playing) runCycle((idx + 1) % tracks.length);
      });
    }, CYCLE_MS - FADE_MS);
  }

  // Cancel any in-progress fade and silence/stop a track immediately.
  function hardStop(el) {
    const prev = fadeState.get(el);
    if (prev) prev.cancelled = true;
    fadeState.delete(el);
    el.pause();
    el.currentTime = 0;
    el.volume = 0;
  }

  function switchTrack(newIdx) {
    clearTimeout(cycleTimer);
    if (!playing) {
      playing = true;
      audioToggle.classList.add('on');
    }
    // Manual switch: stop the current track at once (no fade-out) and let the
    // new one fade in via runCycle. Fade-out is reserved for cycle completion.
    hardStop(tracks[currentIdx]);
    runCycle(newIdx);
  }

  audioToggle.addEventListener('click', () => {
    playing = !playing;
    if (playing) {
      audioToggle.classList.add('on');
      runCycle(currentIdx);
    } else {
      audioToggle.textContent = '♪ off';
      audioToggle.classList.remove('on');
      clearTimeout(cycleTimer);
      tracks.forEach((t) => hardStop(t));
    }
  });

  prevBtn.addEventListener('click', () => {
    switchTrack((currentIdx - 1 + tracks.length) % tracks.length);
  });

  nextBtn.addEventListener('click', () => {
    switchTrack((currentIdx + 1) % tracks.length);
  });
})();
