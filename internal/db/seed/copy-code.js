(function() {
  'use strict';

  var clipboardSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/></svg>';
  var checkSvg = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>';

  var pres = document.querySelectorAll('pre:not([data-no-copy])');
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
          fallbackCopy(code || pre, btn);
        });
      } else {
        fallbackCopy(code || pre, btn);
      }
    });

    pre.appendChild(btn);
  });

  function fallbackCopy(el, btn) {
    var range = document.createRange();
    range.selectNodeContents(el);
    var selection = window.getSelection();
    selection.removeAllRanges();
    selection.addRange(range);
    btn.innerHTML = checkSvg;
    btn.classList.add('copied');
    setTimeout(function() {
      btn.innerHTML = clipboardSvg;
      btn.classList.remove('copied');
      selection.removeAllRanges();
    }, 2000);
  }
})();
