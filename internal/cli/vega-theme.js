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
  var NORD_CATEGORICAL = [
    '#5e81ac', '#bf616a', '#a3be8c', '#d08770',
    '#b48ead', '#88c0d0', '#ebcb8b'
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
        domainColor: dark ? '#434c5e' : '#d8dee9',
        tickColor: dark ? '#434c5e' : '#d8dee9',
        gridColor: dark ? '#3b4252' : '#eceff4',
        gridOpacity: 0.7,
        labelFontSize: 11,
        titleFontSize: 11
      },
      legend: {
        labelColor: dark ? '#aebbcf' : '#4c566a',
        titleColor: dark ? '#d8dee9' : '#3b4252'
      },
      range: { category: NORD_CATEGORICAL }
    };
  }

  function currentTheme() {
    return document.documentElement.dataset.theme || 'light';
  }

  function renderOne(el) {
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
      config: configFor(currentTheme()),
      actions: false,
      renderer: 'svg'
    });
  }

  function renderAll() {
    document.querySelectorAll('.chart[data-vega]').forEach(renderOne);
  }

  document.addEventListener('DOMContentLoaded', renderAll);
  window.addEventListener('message', function (e) {
    if (e.data && e.data.type === 'theme') renderAll();
  });
})();
