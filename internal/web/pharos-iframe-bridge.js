(function() {
  if (window.parent === window) return;
  document.addEventListener('click', function(e) {
    try { window.parent.postMessage({ type: 'pharos-frame-click' }, '*'); } catch(_) {}
  });
  document.addEventListener('keydown', function(e) {
    if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K' || e.key === '/')) {
      e.preventDefault();
      e.stopPropagation();
      try { window.parent.focus(); window.parent.postMessage({ type: 'pharos-open-palette' }, '*'); } catch(_) {}
    }
  });
})();
