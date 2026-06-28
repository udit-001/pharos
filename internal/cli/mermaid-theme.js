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
    tertiaryColor: '#4c566a'
  };

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

  window.mermaidTheme = {
    swapColors: swapColors,
    themeVars: themeVars,
  };

  window.addEventListener('message', function(e) {
    if (e.data && e.data.type === 'theme') {
      document.documentElement.dataset.theme = e.data.theme;
      swapColors(e.data.theme);
    }
  });
})();
