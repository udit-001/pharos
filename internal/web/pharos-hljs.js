/**
 * pharos-hljs.js — syntax highlighting glue for highlight.js.
 *
 * Auto-injected by the server when lesson HTML marks code blocks with
 * language-* classes (opt-in highlighting — plain pre>code stays untouched).
 * Requires assets/highlight.min.js, served ahead of this bundle.
 *
 * Idempotent: the window guard makes double injection (legacy lessons that
 * call hljs.highlightAll() themselves plus this glue) a no-op on our side.
 */
(function () {
  'use strict';

  function init() {
    if (window.__pharosHljsDone) return;
    if (!window.hljs) return;
    window.__pharosHljsDone = true;
    window.hljs.highlightAll();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
