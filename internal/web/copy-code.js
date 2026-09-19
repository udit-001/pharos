/**
 * Copy-button logic for lesson code blocks marked with `data-copy`.
 *
 * Auto-injected by the server when lesson HTML contains `data-copy`
 * (mirrors the glossary-tooltip detection). Lessons never include this
 * file by hand — mark the block and the logic arrives:
 *
 *   <pre data-copy><code>SELECT 1;</code></pre>
 */
(function() {
  'use strict';

  var clipboardSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>';
  var checkSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>';

  function fallbackCopy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch (e) {}
    ta.remove();
  }

  function init() {
    var pres = document.querySelectorAll('pre[data-copy]');
    pres.forEach(function(pre) {
      if (pre.querySelector('.copy-code-btn')) return;

      var pos = window.getComputedStyle(pre).position;
      if (pos === 'static') {
        pre.style.position = 'relative';
      }

      var btn = document.createElement('button');
      btn.className = 'copy-code-btn';
      btn.setAttribute('aria-label', 'Copy code');
      btn.innerHTML = clipboardSvg;

      btn.addEventListener('click', function(e) {
        e.stopPropagation();
        var code = pre.querySelector('code');
        var text = code ? code.textContent : pre.textContent;

        if (navigator.clipboard) {
          navigator.clipboard.writeText(text).then(function() {
            btn.innerHTML = checkSvg;
            btn.classList.add('copied');
            setTimeout(function() {
              btn.innerHTML = clipboardSvg;
              btn.classList.remove('copied');
            }, 2000);
          }).catch(function() {
            fallbackCopy(text);
            btn.innerHTML = checkSvg;
            btn.classList.add('copied');
            setTimeout(function() {
              btn.innerHTML = clipboardSvg;
              btn.classList.remove('copied');
            }, 2000);
          });
        } else {
          fallbackCopy(text);
        }
      });

      pre.appendChild(btn);
    });
  }

  // Injection lands in <head>, before the body is parsed — wait for the DOM.
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
