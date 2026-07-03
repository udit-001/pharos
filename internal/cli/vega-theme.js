/*!
 * vega-theme.js — Pharos companion for Vega-Lite charts.
 *
 * Auto-renders charts from pure JSON specs. The agent writes a JSON spec
 * inside a <script type="application/json" id="my-chart"> tag and places a
 * <div class="chart" data-vega="my-chart"></div> where the chart should
 * appear. This companion discovers the pair, renders via vegaEmbed with
 * Nord theming, and re-renders every chart on theme toggle — no JS for
 * the agent to write.
 *
 * Theming: the Nord config is injected via vegaEmbed's options.config,
 * read fresh from data-theme at each render. Theme switching is a clean
 * re-render from the spec — no cached-SVG retint (unlike mermaid, whose
 * colours are baked into per-diagram <style> at render time).
 */
(function () {
  // WCAG 1.4.11-compliant Nord palettes — 3:1+ contrast against both
  // dark (#3b4252) and light (#ffffff) page backgrounds.
  var NORD_CATEGORICAL_DARK = [
    '#7ba3c8', '#d08088', '#a3be8c', '#d08770',
    '#b48ead', '#88c0d0', '#ebcb8b'
  ];
  var NORD_CATEGORICAL_LIGHT = [
    '#3d5a80', '#8b3a42', '#6b8c5a', '#a05a3c',
    '#7a5a82', '#5a8a96', '#a08840'
  ];

  function configFor(theme) {
    var dark = theme === 'dark';
    return {
      background: 'transparent',
      font: 'Inter',
      view: { stroke: null },
      title: { color: dark ? '#eceff4' : '#2e3440', fontSize: 13, fontWeight: 600 },
      style: {
        'guide-label': { fill: dark ? '#aebbcf' : '#4c566a', fontSize: 11 },
        'guide-title': { fill: dark ? '#d8dee9' : '#3b4252', fontSize: 11, fontWeight: 600 }
      },
      axis: {
        domainColor: dark ? '#81a1c1' : '#4c566a',
        tickColor: dark ? '#81a1c1' : '#4c566a',
        gridColor: dark ? '#88c0d0' : '#4c566a',
        gridOpacity: 0.3,
        domainOpacity: 0.3,
        tickOpacity: 0.3,
        labelFontSize: 11,
        titleFontSize: 11
      },
      legend: {
        labelColor: dark ? '#aebbcf' : '#4c566a',
        titleColor: dark ? '#d8dee9' : '#3b4252'
      },
      range: { category: dark ? NORD_CATEGORICAL_DARK : NORD_CATEGORICAL_LIGHT }
    };
  }

  function currentTheme() {
    return document.documentElement.dataset.theme || 'light';
  }

  function renderOne(el, theme) {
    var id = el.getAttribute('data-vega');
    if (!id) return;
    var script = document.getElementById(id);
    if (!script) return;
    var spec;
    try {
      spec = JSON.parse(script.textContent);
    } catch (e) {
      el.textContent = 'Invalid JSON in #' + id + ': ' + e.message;
      return;
    }
    if (typeof vegaEmbed !== 'function') return;
    vegaEmbed(el, spec, {
      config: configFor(theme || currentTheme()),
      actions: false,
      renderer: 'svg'
    });
  }

  function renderAll(theme) {
    document.querySelectorAll('.chart[data-vega]').forEach(function(el) { renderOne(el, theme); });
  }

  document.addEventListener('DOMContentLoaded', function() { renderAll(); });
  window.addEventListener('message', function (e) {
    if (e.data && e.data.type === 'theme') renderAll(e.data.theme);
  });
})();
