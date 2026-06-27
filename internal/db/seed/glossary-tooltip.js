(function() {
  var m = window.location.pathname.match(/\/api\/lesson-html\/([^/]+)\//);
  if (!m) return;
  var wsName = m[1];
  var tip = document.createElement('div');
  tip.className = 'glossary-tooltip';
  document.body.appendChild(tip);
  var els = document.querySelectorAll('.glossary-term');
  if (!els.length) return;
  var defs = null;
  fetch('/api/workspaces/name/' + encodeURIComponent(wsName) + '/glossary-terms')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      defs = {};
      data.forEach(function(t) { defs[t.term] = t.definition; });
    })
    .catch(function() {});
  els.forEach(function(el) {
    el.addEventListener('mouseenter', function(e) {
      if (!defs) return;
      var term = el.getAttribute('data-term') || el.textContent.trim();
      var def = defs[term];
      if (!def) return;
      tip.textContent = def;
      var r = tip.getBoundingClientRect();
      var x = Math.min(e.clientX - r.width / 2, window.innerWidth - r.width - 10);
      var yAbove = e.clientY - r.height - 12;
      var yBelow = e.clientY + 12;
      var useAbove = yAbove >= 0;
      var y = useAbove ? yAbove : yBelow;
      tip.classList.toggle('tooltip-below', !useAbove);
      tip.style.left = Math.max(10, x) + 'px';
      tip.style.top = y + 'px';
      tip.classList.add('visible');
    });
    el.addEventListener('mouseleave', function() {
      tip.classList.remove('visible');
    });
  });
})();
