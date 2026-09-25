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
        // Actor attribution (LEARN-236/237): who ran — learner or agent.
        if (ev.actor === 'learner' || ev.actor === 'agent') wire.actor = ev.actor;
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
    if (!ev || !ev.id || !ev.ts) return;
    var wire = toWire(ev);
    if (!wire) {
      console.warn('[pharos-workbench] skipping unrelayable event', ev);
      return;
    }
    var ns = (e.currentTarget && e.currentTarget.getAttribute('namespace')) || 'default';
    var ts = new Date(ev.ts).toISOString(); // throws on invalid input — guarded above
    pending.push({
      id: ev.id,
      namespace: ns,
      type: wire.type,
      ts: ts,
      payload: wire.payload,
    });
    idbSave();
    if (pending.length >= FLUSH_AT) flush();
    else scheduleFlush();
  }

  var benches = document.querySelectorAll('sql-workbench');
  for (var i = 0; i < benches.length; i++) bind(benches[i]);
  mo.observe(document.documentElement, { childList: true, subtree: true });

  // ── Exec channel (LEARN-237): agent commands → bench → replies ─────
  // The server broadcasts workbench-command events on the SSE topic
  // "workbench:<workspace>"; this glue executes them against the named
  // bench element and posts replies. Rules (bench-kit/commands.ts):
  // commands serialize (the human's in-flight run is never preempted);
  // replies carry the outcome verbatim; failures are replies, not throws.
  // Token: cfg.workbenchToken (injected by the server) gates replies —
  // a forged reply would be a forged verdict.

  var token = cfg.workbenchToken || '';
  var replyURL = '/api/workspaces/name/' + encodeURIComponent(cfg.workspace) + '/workbench-replies';
  var byNamespace = {};

  function benchFor(ns) {
    if (byNamespace[ns]) return byNamespace[ns];
    var el = document.querySelector('sql-workbench[namespace="' + ns + '"]') ||
             (ns === 'default' ? document.querySelector('sql-workbench:not([namespace])') : null);
    if (el) byNamespace[ns] = el;
    return el;
  }

  var cmdChain = Promise.resolve();
  function enqueue(cmd) {
    var turn = cmdChain.then(function () { return executeCommand(cmd); });
    // A failed turn must not poison the queue for the next command.
    cmdChain = turn.catch(function () {});
    return turn;
  }

  function executeCommand(cmd) {
    return new Promise(function (resolve) {
      var done = function (reply) {
        // The reply carries the command id (bench-kit/commands.ts reply shape).
        fetch(replyURL, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Pharos-Workbench-Token': token,
          },
          body: JSON.stringify({ id: cmd.id, reply: Object.assign({ id: cmd.id }, reply) }),
        }).catch(function () {});
      };

      if (!cmd || !cmd.id || !cmd.op) { done({ ok: false, error: 'malformed command' }); return; }
      var el = benchFor(cmd.namespace || 'default');
      if (!el) { done({ ok: false, error: 'no bench element with namespace "' + (cmd.namespace || 'default') + '" on this page' }); return; }

      try {
        if (cmd.op === 'run') {
          if (typeof el.runGraded === 'function') {
            // runGraded journals with the given actor and returns the
            // outcome verbatim plus the step verdict when a problem is
            // loaded (older bundles lack it — fall back to run).
            el.runGraded(cmd.sql || '', { actor: cmd.actor || 'agent' })
              .then(function (r) {
                done({ ok: true, op: 'run', outcome: r.outcome, verdict: r.verdict });
              })
              .catch(function (err) { done({ ok: false, op: 'run', error: String(err && err.message || err) }); });
          } else {
            el.run(cmd.sql || '', { actor: cmd.actor || 'agent' })
              .then(function (outcome) {
                done({ ok: true, op: 'run', outcome: outcome });
              })
              .catch(function (err) { done({ ok: false, op: 'run', error: String(err && err.message || err) }); });
          }
        } else if (cmd.op === 'setProblem') {
          el.setProblem(cmd.problem || null);
          done({ ok: true, op: 'setProblem' });
        } else if (cmd.op === 'reset') {
          el.reset()
            .then(function () { done({ ok: true, op: 'reset' }); })
            .catch(function (err) { done({ ok: false, op: 'reset', error: String(err && err.message || err) }); });
        } else {
          done({ ok: false, op: 'run', error: 'unknown command op "' + cmd.op + '"' });
        }
      } catch (err) {
        done({ ok: false, op: 'run', error: String(err && err.message || err) });
      }
    });
  }

  function onCommandEvent(e) {
    try {
      var msg = JSON.parse(e.data);
      if (!msg || msg.type !== 'workbench-command' || !msg.data) return;
      enqueue(msg.data);
    } catch (err) { /* malformed SSE frame — ignore, stream continues */ }
  }

  var es = null;
  function connectCommands() {
    if (es || !token) return; // no token → no command channel (server governs)
    es = new EventSource('/api/events?topic=' + encodeURIComponent('workbench:' + cfg.workspace));
    es.onmessage = onCommandEvent;
    es.onerror = function () {
      // EventSource reconnects on its own; drop the dead instance so a
      // new one is created after the backoff reconnect.
      es = null;
      setTimeout(connectCommands, 5000);
    };
  }
  connectCommands();

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
