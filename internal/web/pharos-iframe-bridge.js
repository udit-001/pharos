(function() {
  if (window.parent === window) return;
  document.addEventListener('click', function(e) {
    try { window.parent.postMessage({ type: 'pharos-frame-click' }, '*'); } catch(_) {}
  });
})();
