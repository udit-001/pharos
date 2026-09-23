/**
 * pharos-workbench.js — relay for sql-workbench journal events (LEARN-232).
 *
 * Auto-injected by the server right after the vendored sql-workbench.js when
 * lesson HTML contains <sql-workbench>. The bench journals every run locally
 * and re-dispatches each event as a `workbench-event` CustomEvent on the
 * element; this glue relays them, batched, to the same-origin ingest endpoint
 * so the agent can read the practice journal via `pharos workbench log`
 * before writing the next lesson.
 *
 * Transport follows the locked journal + embed contract (LEARN-197), as
 * superseded by LEARN-206: flush on 10 events / 30s / visibilitychange;
 * ack-then-drop (events leave the queue only after the server acknowledges);
 * backoff retry 5s doubling to 60s — retry-safe because the server dedupes
 * on event id; the pending queue mirrors to IndexedDB so a server-down
 * window or a crash orphans nothing; batches chunked to 50 (server cap).
 *
 * Idempotent by design (legacy lessons may carry their own copy) and degrades
 * to no-op when window.__pharos carries no workspace (DB-unresolved lesson):
 * the bench keeps journaling locally exactly as it would standalone.
 */
(function () {
  'use strict';
  if (window.__pharosWorkbenchRelay) return;
  window.__pharosWorkbenchRelay = true;

  var cfg = window.__pharos;
  if (!cfg || !cfg.workspace) return;

  var apiURL = '/api/workspaces/name/' + encodeURIComponent(cfg.workspace) + '/workbench-events';
  var FLUSH_AT = 10;       // events queued → flush now (LEARN-197)
  var FLUSH_MS = 30000;    // otherwise flush after 30s
  var BACKOFF_START = 5000;
  var BACKOFF_MAX = 60000;

  var pending = [];   // wire-shaped events, oldest first
  var timer = null;   // 30s flush timer
  var retryMs = BACKOFF_START;
  var inflight = false;
  var idbPromise = openIdb();

  // ── Mapping: bench event → ingest wire shape ────────────────────────
  // The one place both contracts meet. Verbatim error strings pass through
  // untouched. `dataset ?? fixture` is the version-skew guard: v0.5.1 emits
  // `fixture`, SQLWB-12 renames it to `dataset` — both spellings accepted,
  // and the wire always speaks `dataset` (vocabulary ruling, LEARN-206).

  function benchDataset(ev) {
    return ev.dataset !== undefined && ev.dataset !== null ? ev.dataset : ev.fixture;
  }

  function toWire(ev) {
    var ds = benchDataset(ev);
    var wire;
    switch (ev.type) {
      case 'query':
        // Server validation: query needs sql + ok; ok:false needs error.
        if (typeof ev.sql !== 'string') return null;
        wire = { sql: ev.sql, ok: !!ev.ok, dataset: ds };
        if (ev.ok) {
          if (typeof ev.rows === 'number') wire.rows = ev.rows;
          if (typeof ev.ms === 'number') wire.ms = ev.ms;
        } else {
          if (typeof ev.error !== 'string') return null;
          wire.error = ev.error;
        }
        return { type: 'query', payload: wire };
      case 'csv-import':
        return { type: 'info', payload: { kind: 'csv-import', name: ev.name, rows: ev.rows } };
      case 'csv-import-removed':
        return { type: 'info', payload: { kind: 'csv-import-removed', name: ev.name } };
      case 'dataset-reset':
        if (!ds) return null;
        return { type: 'dataset', payload: { dataset: ds } };
      default:
        return null; // unknown bench type — skip; the journal stays authoritative
    }
  }

  // ── IndexedDB buffer (LEARN-197: ack-then-drop survives server-down) ─

  function openIdb() {
    return new Promise(function (resolve, reject) {
      var req = indexedDB.open('pharos-workbench-queue', 1);
      req.onupgradeneeded = function () {
        req.result.createObjectStore('kv');
      };
      req.onsuccess = function () { resolve(req.result); };
      req.onerror = function () { reject(req.error); };
    });
  }

  function idbSave() {
    return idbPromise.then(function (db) {
      return new Promise(function (resolve, reject) {
        var tx = db.transaction('kv', 'readwrite');
        tx.objectStore('kv').put(pending, 'pending');
        tx.oncomplete = resolve;
        tx.onerror = function () { reject(tx.error); };
      });
    }).catch(function () {
      // Buffering is best-effort: without IndexedDB the in-memory queue
      // still carries the session (the bench degrades the same way).
    });
  }

  // Crash/reload recovery: events enqueued by a previous page load that
  // never got acknowledged resume from the buffer.
  idbPromise.then(function (db) {
    return new Promise(function (resolve) {
      var tx = db.transaction('kv', 'readonly');
      var req = tx.objectStore('kv').get('pending');
      req.onsuccess = function () { resolve(req.result || []); };
      req.onerror = function () { resolve([]); };
    });
  }).then(function (restored) {
    if (restored.length) {
      pending = restored.concat(pending);
      scheduleFlush();
    }
  });

  // ── Listener: bind each <sql-workbench> element directly ────────────
  // The bench dispatches workbench-event WITHOUT bubbles (sql-workbench.ts
  // dispatches with only `detail`), so a document-level listener never
  // fires — bind per element instead. MutationObserver covers benches
  // mounted after this script runs (e.g. cards added dynamically).

  function bind(el) {
    if (!el || el.__pharosRelayBound) return;
    el.__pharosRelayBound = true;
    el.addEventListener('workbench-event', onEvent);
  }

  var mo = new MutationObserver(function (muts) {
    for (var i = 0; i < muts.length; i++) {
      for (var j = 0; j < muts[i].addedNodes.length; j++) {
        var node = muts[i].addedNodes[j];
        if (!node || node.nodeType !== 1) continue;
        if (node.tagName === 'SQL-WORKBENCH') bind(node);
        if (node.querySelectorAll) {
          var benches = node.querySelectorAll('sql-workbench');
          for (var k = 0; k < benches.length; k++) bind(benches[k]);
        }
      }
    }
  });

  function onEvent(e) {
    var ev = e.detail;
    if (!ev || !ev.id) return;
    var wire = toWire(ev);
    if (!wire) {
      console.warn('[pharos-workbench] skipping unrelayable event', ev);
      return;
    }
    var ns = (e.currentTarget && e.currentTarget.getAttribute('namespace')) || 'default';
    pending.push({
      id: ev.id,
      namespace: ns,
      type: wire.type,
      ts: new Date(ev.ts).toISOString(),
      payload: wire.payload,
    });
    idbSave();
    if (pending.length >= FLUSH_AT) flush();
    else scheduleFlush();
  }

  var benches = document.querySelectorAll('sql-workbench');
  for (var i = 0; i < benches.length; i++) bind(benches[i]);
  mo.observe(document.documentElement, { childList: true, subtree: true });

  function scheduleFlush() {
    if (timer) return;
    timer = setTimeout(function () {
      timer = null;
      flush();
    }, FLUSH_MS);
  }

  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'hidden') {
      if (timer) { clearTimeout(timer); timer = null; }
      flush();
    }
  });

  function flush() {
    if (inflight || !pending.length) return;
    inflight = true;
    var batch = pending.slice(0, 50);
    fetch(apiURL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ events: batch }),
      keepalive: true, // the visibilitychange flush may outlive the page
    }).then(function (res) {
      inflight = false;
      if (res.ok) {
        // Ack-then-drop: only acknowledged events leave the queue.
        pending = pending.slice(batch.length);
        retryMs = BACKOFF_START;
        idbSave();
        if (pending.length) flush(); // more than one chunk was queued
      } else {
        retryFlush();
      }
    }, retryFlush);
  }

  function retryFlush() {
    inflight = false;
    setTimeout(flush, retryMs);
    retryMs = Math.min(retryMs * 2, BACKOFF_MAX);
  }
})();
