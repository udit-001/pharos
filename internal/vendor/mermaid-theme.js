(function() {
  'use strict';

  var NORD_LIGHT = {
    fontFamily: 'Inter, sans-serif',
    fontSize: '16px',
    background: '#ffffff',
    primaryColor: '#eceff4',
    primaryBorderColor: '#81a1c1',
    primaryTextColor: '#2e3440',
    lineColor: '#6b7689',
    secondaryColor: '#e5e9f0',
    tertiaryColor: '#f8fafc'
  };

  var NORD_DARK = {
    fontFamily: 'Inter, sans-serif',
    fontSize: '16px',
    background: '#2e3440',
    primaryColor: '#3b4252',
    primaryBorderColor: '#81a1c1',
    primaryTextColor: '#e5e9f0',
    lineColor: '#8891a0',
    secondaryColor: '#434c5e',
    tertiaryColor: '#4c566a',
    // Per-section mindmap colours (cScale) + matching label text colours
    // (cScaleLabel). Without these, mermaid derives every section from
    // primaryColor (#3b4252, L≈15%); hue-shifting such a dark base collapses
    // every section to near-black. Explicit Nord frost/aurora hues keep each
    // branch distinct, with label colours chosen for contrast against each bg.
    cScale0: '#3b4252', cScaleLabel0: '#e5e9f0',
    cScale1: '#5e81ac', cScaleLabel1: '#eceff4',
    cScale2: '#a3be8c', cScaleLabel2: '#2e3440',
    cScale3: '#ebcb8b', cScaleLabel3: '#2e3440',
    cScale4: '#d08770', cScaleLabel4: '#2e3440',
    cScale5: '#bf616a', cScaleLabel5: '#eceff4',
    cScale6: '#b48ead', cScaleLabel6: '#2e3440',
    cScale7: '#88c0d0', cScaleLabel7: '#2e3440',
    cScale8: '#8fbcbb', cScaleLabel8: '#2e3440',
    cScale9: '#81a1c1', cScaleLabel9: '#2e3440',
    cScale10: '#4c566a', cScaleLabel10: '#e5e9f0',
    cScale11: '#434c5e', cScaleLabel11: '#e5e9f0',
    cScale12: '#5e81ac', cScaleLabel12: '#eceff4',
    // Root node (section-root) is filled by git0 + labelled by gitBranchLabel0,
    // not cScale. In dark mode mermaid darkens git0 by 25%, turning the
    // primaryColor #3b4252 into near-black #060608. Pin both so the root
    // reads as the anchor of the diagram instead of a black hole.
    git0: '#4c566a',
    gitBranchLabel0: '#eceff4'
  };

  // Map used only to retint an already-open lightbox clone on toggle. Inline
  // diagrams are retinted by copying a fresh <style> into the live SVG (see
  // retint below), which handles every colour notation — including the hsl()
  // values mermaid emits for mindmap nodes — that a plain hex string swap
  // cannot reach.
  var LIGHT_TO_DARK = {
    '#eceff4': '#3b4252',
    '#2e3440': '#e5e9f0',
    '#6b7689': '#8891a0',
    '#e5e9f0': '#434c5e',
    '#f8fafc': '#4c566a',
    '#ffffff': '#2e3440',
    '#81a1c1': '#81a1c1',
    '#4c566a': '#f8fafc',
    '#434c5e': '#e5e9f0',
    '#8891a0': '#6b7689',
    '#3b4252': '#eceff4',
  };

  var DARK_TO_LIGHT = {};
  for (var k in LIGHT_TO_DARK) {
    DARK_TO_LIGHT[LIGHT_TO_DARK[k]] = k;
  }

  function swapColors(theme) {
    var map = theme === 'dark' ? LIGHT_TO_DARK : DARK_TO_LIGHT;
    var styles = document.querySelectorAll('.mermaid svg style, .mermaid-lightbox-viewport svg style');
    for (var i = 0; i < styles.length; i++) {
      styles[i].textContent = styles[i].textContent.replace(/#[0-9a-f]{6}/gi, function(m) {
        return map[m.toLowerCase()] || m;
      });
    }
  }

  function themeVars() {
    return document.documentElement.dataset.theme === 'dark' ? NORD_DARK : NORD_LIGHT;
  }

  // Store each diagram's source text before mermaid replaces it with an SVG.
  // Registered from the <head> script, so it runs ahead of the boilerplate's
  // own DOMContentLoaded listener that calls mermaid.run — guaranteeing we
  // capture the raw source first.
  function captureSources() {
    var nodes = document.querySelectorAll('.mermaid');
    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      if (el.getAttribute('data-mermaid-src') != null) continue;
      if (el.querySelector('svg')) continue; // already rendered — source lost
      el.setAttribute('data-mermaid-src', el.textContent);
    }
  }

  // Retint every diagram to the new palette WITHOUT re-running the layout.
  // mermaid.run is non-idempotent: a second pass lays the diagram out
  // differently from the first (different viewBox), so replacing the SVG on
  // toggle makes the diagram jump size. Instead we render each diagram to a
  // throwaway, then copy only its <style> (scoped to the SVG's id) and
  // gradient stops into the live SVG. Geometry, viewBox, and the lightbox
  // toolbar are all left untouched — only colour changes.
  function retint(theme) {
    document.documentElement.dataset.theme = theme;
    // Synchronous fallback: update lightbox clones immediately (hex colors).
    // The async mermaid.render will follow and handle hsl() values later.
    swapColors(theme);
    if (!window.mermaid || !window.mermaid.render) {
      return;
    }
    window.mermaid.initialize({ startOnLoad: false, theme: 'base', themeVariables: themeVars() });
    var nodes = document.querySelectorAll('.mermaid');
    var tasks = [];
    for (var i = 0; i < nodes.length; i++) {
      var el = nodes[i];
      var src = el.getAttribute('data-mermaid-src');
      var svg = el.querySelector('svg');
      if (src == null || !svg) continue;
      tasks.push(retintOne(svg, src, i));
    }
    // Update lightbox clones after each diagram retints, not just at the end.
    // This avoids waiting for ALL diagrams before the lightbox catches up.
    var lightboxPokes = tasks.map(function(t) {
        return (t || Promise.resolve()).then(retintLightboxClones, function() {});
    });
    Promise.all(lightboxPokes).then(function() {}, function() { retintLightboxClones(); swapColors(theme); });
  }

  // An open lightbox shows a clone of the live SVG. The clone keeps the old
  // <style>, so copy the just-retinted style (+ gradient stops) from the live
  // SVG into the clone. Clones share the original SVG's id (duplicate), so we
  // match by id and copy textContent directly — no re-render needed.
  function retintLightboxClones() {
    var clones = document.querySelectorAll('.mermaid-lightbox-viewport svg');
    for (var i = 0; i < clones.length; i++) {
      var clone = clones[i];
      var id = clone.getAttribute('id');
      if (!id) continue;
      var live = document.querySelector('.mermaid svg[id="' + id + '"]');
      if (!live) continue;
      var liveStyle = live.querySelector('style');
      var cloneStyle = clone.querySelector('style');
      if (liveStyle && cloneStyle) cloneStyle.textContent = liveStyle.textContent;
      var liveGrad = live.querySelector('linearGradient');
      var cloneGrad = clone.querySelector('linearGradient');
      if (liveGrad && cloneGrad) cloneGrad.innerHTML = liveGrad.innerHTML;
    }
  }

  function retintOne(visibleSvg, src, idx) {
    var visId = visibleSvg.getAttribute('id') || ('mermaid-' + idx);
    var renderId = 'mermaid-retint-' + idx + '-' + Date.now();
    return window.mermaid.render(renderId, src).then(function(out) {
      var doc = new DOMParser().parseFromString(out.svg, 'image/svg+xml');
      var tSvg = doc.documentElement;
      var tmpId = tSvg.getAttribute('id') || renderId;
      var tmpStyle = tSvg.querySelector('style');
      if (!tmpStyle) return;
      // Rewrite the throwaway SVG's id (and its gradient id refs) to the
      // live SVG's id so the copied rules still match the live elements.
      var styleText = tmpStyle.textContent.split(tmpId).join(visId);
      var liveStyle = visibleSvg.querySelector('style');
      if (liveStyle) liveStyle.textContent = styleText;
      // Swap gradient stop colours so neo-look strokes retint too.
      var tmpGrad = tSvg.querySelector('linearGradient');
      var liveGrad = visibleSvg.querySelector('linearGradient');
      if (tmpGrad && liveGrad) liveGrad.innerHTML = tmpGrad.innerHTML;
    }).catch(function() { /* leave the live SVG as-is on failure */ });
  }

  // Defer mermaid.run until web fonts have loaded. Mermaid measures label
  // text during layout, so rendering before Inter is ready sizes the diagram
  // to the fallback font; a later re-render (e.g. theme toggle) then uses
  // Inter metrics and the diagram jumps size. Waiting for document.fonts.ready
  // makes the initial render match subsequent re-renders. Once fonts are
  // settled, document.fonts.ready resolves immediately, so re-renders aren't
  // delayed.
  if (window.mermaid && window.mermaid.run && document.fonts && document.fonts.ready) {
    var origRun = window.mermaid.run.bind(window.mermaid);
    window.mermaid.run = function(opts) {
      return document.fonts.ready.then(function() { return origRun(opts); });
    };
  }

  window.mermaidTheme = {
    swapColors: swapColors,
    themeVars: themeVars,
    retint: retint
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', captureSources);
  } else {
    captureSources();
  }

  window.addEventListener('message', function(e) {
    if (e.data && e.data.type === 'theme') {
      retint(e.data.theme);
    }
  });
})();
