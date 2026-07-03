/*!
 * katex-render.js — Pharos companion for KaTeX auto-render.
 *
 * Loaded inside lesson/reference iframes alongside katex.min.js and the
 * auto-render extension. Scans the document body for $...$, $$...$$, \(...\),
 * and \[...\] delimiters and renders them with KaTeX.
 *
 * Theming: KaTeX emits semantic HTML/CSS whose .katex root inherits color from
 * the container (currentColor for fraction bars, radicals, stretchy SVG). No
 * JS retint is needed on theme toggle — the existing data-theme CSS variable
 * swap recolours math automatically, unlike mermaid whose colours are baked
 * into per-diagram <style> at render time.
 */
(function () {
  document.addEventListener('DOMContentLoaded', function () {
    if (typeof renderMathInElement !== 'function') return;
    renderMathInElement(document.body, {
      delimiters: [
        { left: '$$', right: '$$', display: true },
        { left: '\\[', right: '\\]', display: true },
        { left: '$', right: '$', display: false },
        { left: '\\(', right: '\\)', display: false }
      ],
      throwOnError: false
    });
  });
})();
