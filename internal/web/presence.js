// Pharos server-presence watcher (LEARN: stopped-page recovery).
//
// Detects when the local Pharos server comes back up (down->up) or goes
// down (up->down), so a page can recover without the user hitting refresh.
//
// Why this exists: the server is localhost and can be stopped independently
// of the PWA window. When it's down there is no server process to push a
// "server is back" event (the SSE broker at /api/events dies with the
// server), so recovery has to be client-initiated: probe /healthz with
// exponential backoff until it answers, then report the transition.
//
// This is a deep module: a small interface (pharosWatchServer({onUp,onDown}))
// hides the whole retry/backoff/edge-triggering state machine. Two callers
// share it:
//   - stopped.html: onUp -> location.reload()  (the stopped page recovers)
//   - dashboard frame: onDown -> "Reconnecting" banner, onUp -> reload
//     (mid-session recovery after an upgrade/restart)
(function (global) {
  'use strict';

  var HEALTH = '/healthz';   // probe target (see server mux)
  var MIN = 1000;            // initial backoff, ms
  var MAX = 30000;           // backoff cap, ms
  var FACTOR = 2;            // multiplier between attempts
  var JITTER = 0.3;          // +/- 30% jitter to avoid lockstep
  var TIMEOUT = 2500;        // per-probe abort timeout, ms

  // state: 'up' | 'down' | 'unknown'. Transitions are edge-triggered:
  // onUp fires only on a down->up crossing, onDown only on up->down.
  function pharosWatchServer(opts) {
    opts = opts || {};
    var onUp = opts.onUp || function () {};
    var onDown = opts.onDown || function () {};
    var state = 'unknown';
    var backoff = MIN;
    var inFlight = false;
    var timer = null;
    var stopped = false;

    function transition(next) {
      if (next === state || stopped) return;
      var prev = state;
      state = next;
      // Initial settle (unknown -> first known state): establish the baseline
      // without firing. Otherwise a page that loads while the server is already
      // up would spuriously fire onUp and (e.g.) reload itself forever.
      if (prev === 'unknown') return;
      if (next === 'up' && prev !== 'up') onUp();
      else if (next === 'down' && prev !== 'down') onDown();
    }

    function probe() {
      if (stopped || inFlight) return;
      inFlight = true;
      var ctrl = new AbortController();
      var t = setTimeout(function () { ctrl.abort(); }, TIMEOUT);
      fetch(HEALTH, { cache: 'no-store', signal: ctrl.signal })
        .then(function (r) { clearTimeout(t); return r.ok ? 'up' : 'down'; })
        .catch(function () { clearTimeout(t); return 'down'; })
        .then(function (next) {
          inFlight = false;
          backoff = next === 'down' ? Math.min(backoff * FACTOR, MAX) : MIN;
          transition(next);
          schedule();
        });
    }

    function schedule() {
      if (stopped) return;
      var delay = Math.round(backoff * (1 + (Math.random() - 0.5) * 2 * JITTER));
      timer = setTimeout(probe, delay);
    }

    function onVisible() {
      // Browser throttles background-tab timers anyway, so probing already
      // slows while hidden. On return to the tab, probe immediately with a
      // fresh backoff so a just-started server is picked up without waiting
      // out the old backoff.
      if (document.visibilityState !== 'visible') return;
      if (inFlight) return; // a probe is out; it will schedule the next one
      if (timer) { clearTimeout(timer); timer = null; }
      backoff = MIN;
      probe();
    }

    document.addEventListener('visibilitychange', onVisible);

    probe(); // initial probe

    return {
      stop: function () {
        stopped = true;
        if (timer) clearTimeout(timer);
        document.removeEventListener('visibilitychange', onVisible);
      }
    };
  }

  global.pharosWatchServer = pharosWatchServer;
})(window);
