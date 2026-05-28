// Background music player (ported from fire.js's audio block) driving the
// footer controls. Plays each track to its end before advancing, with a
// short fade-out into the next; prev/next switch manually. No-ops if the
// expected elements aren't on the page.
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

  const MAX_VOL = 0.15;
  const FADE_MS = 3000;

  let playing    = false;
  let currentIdx = 0;
  let fadeOutTimer = null;
  // Generation counter: any pending async work (timers, metadata waits) keyed
  // to a stale generation is discarded. Bumped on every cycle start/stop.
  let gen = 0;
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

  // Cancel any in-progress fade and silence/stop a track immediately.
  function hardStop(el) {
    const prev = fadeState.get(el);
    if (prev) prev.cancelled = true;
    fadeState.delete(el);
    el.pause();
    el.currentTime = 0;
    el.volume = 0;
  }

  function clearCycleTimers() {
    if (fadeOutTimer) {
      clearTimeout(fadeOutTimer);
      fadeOutTimer = null;
    }
  }

  // Schedule the fade-out to begin FADE_MS before the track ends, using the
  // known duration. The 'ended' handler still drives the actual advance — the
  // fade just makes the transition smooth instead of an abrupt cut.
  function scheduleFadeOut(idx, myGen) {
    const track = tracks[idx];
    const duration = track.duration;
    if (!isFinite(duration) || duration <= 0) return;
    const remainingMs = Math.max(0, (duration - track.currentTime) * 1000 - FADE_MS);
    fadeOutTimer = setTimeout(() => {
      if (myGen !== gen || !playing) return;
      fadeOutTimer = null;
      fade(track, track.volume, 0, FADE_MS);
    }, remainingMs);
  }

  function runCycle(idx) {
    clearCycleTimers();
    gen++;
    const myGen = gen;
    currentIdx = idx;
    audioToggle.textContent = '♪ ' + trackNames[idx];
    const track = tracks[idx];
    track.currentTime = 0;
    track.volume = 0;
    track.play().catch(() => {});
    fade(track, 0, MAX_VOL, FADE_MS);

    if (isFinite(track.duration) && track.duration > 0) {
      scheduleFadeOut(idx, myGen);
    } else {
      const onMeta = () => {
        track.removeEventListener('loadedmetadata', onMeta);
        if (myGen !== gen) return;
        scheduleFadeOut(idx, myGen);
      };
      track.addEventListener('loadedmetadata', onMeta);
    }
  }

  // 'ended' fires when the track plays through naturally (the element no
  // longer has `loop` set). Drive the advance from here so the full tail of
  // every track is heard, even if duration metadata was slightly off.
  tracks.forEach((track, idx) => {
    track.addEventListener('ended', () => {
      if (!playing || idx !== currentIdx) return;
      clearCycleTimers();
      hardStop(track);
      runCycle((idx + 1) % tracks.length);
    });
  });

  function switchTrack(newIdx) {
    if (!playing) {
      playing = true;
      audioToggle.classList.add('on');
    }
    // Manual switch: stop the current track at once (no fade-out) and let the
    // new one fade in via runCycle. Fade-out is reserved for cycle completion.
    clearCycleTimers();
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
      clearCycleTimers();
      gen++;
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
