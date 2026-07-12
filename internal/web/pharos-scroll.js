(function () {
	var cfg = window.__pharos;
	if (!cfg || cfg.docType !== 'lesson' || !cfg.workspace || !cfg.docId) return;

	var STORAGE_KEY = 'pharos_scroll_' + cfg.workspace + '_' + cfg.docType + '_' + cfg.docId;
	var SAVE_DELAY = 300;
	var RESTORE_MAX_FRAMES = 20;

	var scrollTimer = null;
	var lastY = 0;

	function save() {
		try { localStorage.setItem(STORAGE_KEY, String(lastY)); } catch (_) {}
	}

	// Debounced save on scroll — keeps localStorage current so a new tab
	// (which doesn't trigger beforeunload on this tab) can restore.
	window.addEventListener('scroll', function () {
		lastY = window.scrollY;
		if (scrollTimer) clearTimeout(scrollTimer);
		scrollTimer = setTimeout(save, SAVE_DELAY);
	}, { passive: true });

	// Final flush when tab loses focus (switching tabs, opening new tabs,
	// minimizing). beforeunload alone doesn't fire for the new-tab case.
	document.addEventListener('visibilitychange', function () {
		if (document.visibilityState === 'hidden') {
			if (scrollTimer) { clearTimeout(scrollTimer); scrollTimer = null; }
			lastY = window.scrollY;
			save();
		}
	});

	// Flush before reload/navigation (live-sync, manual refresh, close).
	window.addEventListener('beforeunload', function () {
		lastY = window.scrollY;
		save();
	});

	// Restore on load. Uses an rAF retry loop to handle async content
	// (mermaid, images) that expands the page after onload — scrollTo
	// silently clamps if scrollHeight < targetY.
	function restore() {
		var raw;
		try { raw = localStorage.getItem(STORAGE_KEY); } catch (_) { return; }
		if (raw == null) return;
		var target = parseInt(raw, 10);
		if (!target || target < 1) return;

		var attempts = 0;
		function tryScroll() {
			if (window.scrollY === target) return;
			window.scrollTo(0, target);
			if (window.scrollY >= target - 5 || window.scrollY === target) return;
			if (document.body.scrollHeight >= target && ++attempts > 3) return;
			if (attempts++ < RESTORE_MAX_FRAMES) requestAnimationFrame(tryScroll);
		}
		tryScroll();
	}

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', restore);
	} else {
		restore();
	}
})();
