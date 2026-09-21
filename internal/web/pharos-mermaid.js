/**
 * pharos-mermaid.js — mermaid init + render glue.
 *
 * Auto-injected by the server when lesson HTML contains .mermaid containers
 * (after assets/mermaid.min.js and assets/mermaid-theme.js, which must be
 * vendored via the vendor cache). Lessons never write the initialize/run
 * boilerplate by hand.
 *
 * Why theme 'base': themeVariables control all diagram colors, enabling
 * dark/light switching. mermaid-theme.js re-renders each diagram on theme
 * toggle (retint via mermaid.render, copying only <style> — layout preserved).
 *
 * Idempotent: legacy lessons may carry their own inline init next to this
 * glue. The window guard prevents double-run of this glue; mermaid itself
 * skips data-processed elements on re-run, so coexisting inits are harmless.
 */
(function () {
  'use strict';

  function init() {
    if (window.__pharosMermaidInit) return;
    if (!window.mermaid) return;
    window.__pharosMermaidInit = true;

    window.mermaid.initialize({
      startOnLoad: false,
      theme: 'base',
      themeVariables: window.mermaidTheme ? window.mermaidTheme.themeVars() : undefined
    });

    window.mermaid.run({
      querySelector: '.mermaid',
      postRenderCallback: function (id) {
        var svg = document.getElementById(id);
        var el = svg && svg.closest ? svg.closest('.mermaid') : null;
        if (!el && svg) {
          var p = svg.parentNode;
          while (p && p !== document.body) {
            if (p.classList && p.classList.contains('mermaid')) { el = p; break; }
            p = p.parentNode;
          }
        }
        if (el && window.mermaidLightbox) window.mermaidLightbox.addToolbar(el);
      }
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
