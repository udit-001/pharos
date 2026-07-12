function resolveTheme(mode) {
	return mode === 'system'
		? (window.matchMedia('(prefers-color-scheme:dark)').matches ? 'dark' : 'light')
		: mode;
}
function applyTheme(mode) {
	var html = document.documentElement;
	var actual = resolveTheme(mode);
	document.querySelectorAll('iframe').forEach(function(f) {
		try {
			if (f.contentDocument && f.contentDocument.documentElement) {
				f.contentDocument.documentElement.dataset.theme = actual;
			}
		} catch(e) {}
		try { f.contentWindow.postMessage({type: 'theme', theme: actual}, '*'); } catch(e) {}
	});
	html.dataset.theme = actual;
	localStorage.setItem('pharos_theme', mode);
	var m = document.getElementById('theme-color');
	if (m) m.content = actual === 'dark' ? '#2e3440' : '#eceff4';
	var sunIcon = document.querySelector('[data-theme-icon=sun]');
	var moonIcon = document.querySelector('[data-theme-icon=moon]');
	var contrastIcon = document.querySelector('[data-theme-icon=contrast]');
	if (sunIcon) sunIcon.classList.toggle('theme-hidden', mode !== 'light');
	if (moonIcon) moonIcon.classList.toggle('theme-hidden', mode !== 'dark');
	if (contrastIcon) contrastIcon.classList.toggle('theme-hidden', mode !== 'system');
}
function toggleTheme() {
	var current = localStorage.getItem('pharos_theme') || 'system';
	var modes = ['light', 'dark', 'system'];
	var idx = modes.indexOf(current);
	var next = modes[(idx + 1) % modes.length];
	applyTheme(next);
}
(function() {
	var t = localStorage.getItem('pharos_theme') || 'system';
	applyTheme(t);
	window.matchMedia('(prefers-color-scheme:dark)').addEventListener('change', function() {
		applyTheme(localStorage.getItem('pharos_theme') || 'system');
	});
})();
