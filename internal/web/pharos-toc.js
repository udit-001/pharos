(function () {
	var headings, tocItems, usedIds, tocNav, tocBtn, tocPanel, isOpen, isMobile, resizeTimer, tocTooltipEl;

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', init);
	} else {
		init();
	}

	function init() {
		headings = document.querySelectorAll('.container h1, .container h2, .container h3');
		if (headings.length < 2) return;
		usedIds = {};
		tocItems = [];
		headings.forEach(function (h, idx) {
			var level = parseInt(h.tagName.charAt(1));
			var id = h.id || slugify(h.textContent, usedIds);
			if (!h.id) h.id = id;
			tocItems.push({ level: level, id: id, text: trimText(h.textContent) });
		});
		if (tocItems.length < 2) return;
		buildTOC();
	}

	function trimText(s) {
		return s.replace(/\s+/g, ' ').trim();
	}

	function slugify(text, used) {
		var slug = text.toLowerCase()
			.replace(/[^\w\s-]/g, '')
			.replace(/\s+/g, '-')
			.replace(/-+/g, '-')
			.replace(/^-|-$/g, '');
		if (!slug) slug = 'section';
		var unique = slug;
		var n = 1;
		while (used[unique]) {
			n++;
			unique = slug + '-' + n;
		}
		used[unique] = true;
		return unique;
	}

	function buildTOC() {
		var style = document.createElement('style');
		style.textContent = getTOCStyles();
		document.head.appendChild(style);

		tocBtn = document.createElement('button');
		tocBtn.id = 'pharos-toc-btn';
		tocBtn.setAttribute('aria-label', 'Table of contents');
		tocBtn.setAttribute('data-tooltip', 'Table of contents');
		tocBtn.innerHTML = '<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>';
		tocBtn.addEventListener('click', toggleTOC);

		tocBtn.addEventListener('mouseenter', function () {
			if (tocTooltipEl) return;
			var r = this.getBoundingClientRect();
			tocTooltipEl = document.createElement('div');
			tocTooltipEl.textContent = 'Table of contents';
			tocTooltipEl.style.position = 'fixed';
			tocTooltipEl.style.color = '#f8fafc';
			tocTooltipEl.style.whiteSpace = 'nowrap';
			tocTooltipEl.style.pointerEvents = 'none';
			tocTooltipEl.style.zIndex = '60';
			tocTooltipEl.style.background = '#1e293b';
			tocTooltipEl.style.borderRadius = '4px';
			tocTooltipEl.style.padding = '4px 10px';
			tocTooltipEl.style.fontSize = '.75rem';
			tocTooltipEl.style.fontWeight = '500';
			tocTooltipEl.style.lineHeight = '1.4';
			tocTooltipEl.style.right = (window.innerWidth - r.left + 10) + 'px';
			tocTooltipEl.style.top = (r.top + r.height / 2) + 'px';
			tocTooltipEl.style.transform = 'translateY(-50%)';
			var tt = document.querySelector('.sidebar-tooltip[tooltip-for="toc"]');
			if (tt) tt.remove();
			tocTooltipEl.setAttribute('tooltip-for', 'toc');
			document.body.appendChild(tocTooltipEl);
		});
		tocBtn.addEventListener('mouseleave', function () {
			if (tocTooltipEl) { tocTooltipEl.remove(); tocTooltipEl = null; }
		});

		tocPanel = document.createElement('nav');
		tocPanel.id = 'pharos-toc-panel';
		tocPanel.setAttribute('aria-label', 'Table of contents');
		tocPanel.innerHTML = '<div class="pharos-toc-header"><span class="pharos-toc-title">On this page</span><button class="pharos-toc-close" aria-label="Close table of contents">&times;</button></div>';
		tocPanel.querySelector('.pharos-toc-close').addEventListener('click', closeTOC);

		var list = document.createElement('ul');
		list.className = 'pharos-toc-list';
		var minLevel = Math.min.apply(null, tocItems.map(function (i) { return i.level; }));
		tocItems.forEach(function (item) {
			var li = document.createElement('li');
			li.className = 'pharos-toc-item pharos-toc-level-' + (item.level - minLevel);
			var a = document.createElement('a');
			a.href = '#' + item.id;
			a.textContent = item.text;
			a.addEventListener('click', function (e) {
				e.preventDefault();
				var target = document.getElementById(item.id);
				if (target) {
					target.scrollIntoView({ behavior: 'smooth', block: 'start' });
				}
				if (isMobile) closeTOC();
			});
			li.appendChild(a);
			list.appendChild(li);
		});
		tocPanel.appendChild(list);

		document.body.appendChild(tocBtn);
		document.body.appendChild(tocPanel);

		document.addEventListener('click', function (e) {
			if (!isOpen) return;
			if (tocPanel.contains(e.target) || tocBtn.contains(e.target)) return;
			closeTOC();
		});

		var activeLink = null;
		var tocLinks = list.querySelectorAll('a');
		var tocObserver = new IntersectionObserver(function (entries) {
			entries.forEach(function (entry) {
				if (entry.isIntersecting) {
					var id = entry.target.id;
					tocLinks.forEach(function (link) {
						if (link.getAttribute('href') === '#' + id) {
							if (activeLink) activeLink.classList.remove('pharos-toc-active');
							link.classList.add('pharos-toc-active');
							activeLink = link;
							link.closest('.pharos-toc-list').querySelector('.pharos-toc-active') && link.closest('.pharos-toc-list').querySelector('.pharos-toc-active').scrollIntoView({ block: 'nearest', behavior: 'smooth' });
						}
					});
				}
			});
		}, { rootMargin: '-64px 0px -50% 0px', threshold: 0 });

		headings.forEach(function (h) { tocObserver.observe(h); });

		checkMobile();
		window.addEventListener('resize', function () {
			clearTimeout(resizeTimer);
			resizeTimer = setTimeout(checkMobile, 150);
		});
	}

	function toggleTOC() {
		if (isOpen) closeTOC(); else openTOC();
	}

	function openTOC() {
		isOpen = true;
		tocPanel.classList.add('pharos-toc-open');
		tocBtn.classList.add('pharos-toc-btn-open');
		if (tocTooltipEl) { tocTooltipEl.remove(); tocTooltipEl = null; }
	}

	function closeTOC() {
		isOpen = false;
		tocPanel.classList.remove('pharos-toc-open');
		tocBtn.classList.remove('pharos-toc-btn-open');
	}

	function checkMobile() {
		isMobile = window.innerWidth < 768;
		if (isMobile && isOpen) closeTOC();
		tocBtn.style.display = '';
	}

	function getTOCStyles() {
		return '\
#pharos-toc-btn {\
	position: fixed;\
	top: 50%;\
	right: 0;\
	z-index: 998;\
	width: 34px;\
	height: 44px;\
	padding: 0;\
	border: 1px solid var(--slate-200, #e5e9f0);\
	border-right: none;\
	border-radius: 4px 0 0 4px;\
	background: var(--slate-100, #eceff4);\
	color: var(--slate-500, #6b7689);\
	cursor: pointer;\
	display: flex;\
	align-items: center;\
	justify-content: center;\
	transform: translateY(-50%);\
	transition: background 0.1s, color 0.1s;\
}\
#pharos-toc-btn:hover {\
	background: var(--slate-200, #e5e9f0);\
	color: var(--slate-700, #4c566a);\
}\
#pharos-toc-btn.pharos-toc-btn-open {\
	background: var(--slate-100, #eceff4);\
	color: var(--blue-700, #5e81ac);\
	border-color: var(--slate-200, #e5e9f0);\
}\
[data-theme="dark"] #pharos-toc-btn {\
	background: var(--slate-100, #353b4a);\
	border-color: var(--slate-200, #434c5e);\
	color: var(--slate-400, #81a1c1);\
}\
[data-theme="dark"] #pharos-toc-btn:hover,\
[data-theme="dark"] #pharos-toc-btn.pharos-toc-btn-open {\
	background: var(--slate-200, #434c5e);\
	color: var(--blue-700, #81a1c1);\
}\
#pharos-toc-panel {\
	position: fixed;\
	top: 0;\
	right: 0;\
	width: 260px;\
	height: 100%;\
	z-index: 999;\
	background: var(--slate-100, #eceff4);\
	border-left: 1px solid var(--slate-200, #e5e9f0);\
	transform: translateX(100%);\
	transition: transform 0.2s ease;\
	display: flex;\
	flex-direction: column;\
}\
#pharos-toc-panel.pharos-toc-open {\
	transform: translateX(0);\
}\
[data-theme="dark"] #pharos-toc-panel {\
	background: var(--slate-100, #353b4a);\
	border-color: var(--slate-200, #434c5e);\
}\
.pharos-toc-header {\
	display: flex;\
	align-items: center;\
	justify-content: space-between;\
	padding: 0.75rem 1rem 0.5rem;\
	flex-shrink: 0;\
}\
.pharos-toc-title {\
	font-size: 0.6875rem;\
	font-weight: 500;\
	color: var(--slate-500, #6b7689);\
	letter-spacing: 0.04em;\
	text-transform: uppercase;\
}\
[data-theme="dark"] .pharos-toc-title {\
	color: var(--slate-400, #81a1c1);\
}\
.pharos-toc-close {\
	width: 32px;\
	height: 32px;\
	padding: 0;\
	border: 1px solid transparent;\
	background: none;\
	color: var(--slate-400, #8891a0);\
	cursor: pointer;\
	font-size: 1.25rem;\
	display: flex;\
	align-items: center;\
	justify-content: center;\
	border-radius: 6px;\
	line-height: 1;\
	flex-shrink: 0;\
	transition: background 0.1s, color 0.1s, border-color 0.1s;\
}\
.pharos-toc-close:hover {\
	background: var(--slate-200, #e5e9f0);\
	color: var(--slate-900, #2e3440);\
	border-color: var(--slate-200, #e5e9f0);\
}\
[data-theme="dark"] .pharos-toc-close:hover {\
	background: var(--slate-200, #434c5e);\
	color: var(--slate-900, #eceff4);\
	border-color: var(--slate-200, #434c5e);\
}\
.pharos-toc-list {\
	list-style: none;\
	margin: 0;\
	padding: 0.375rem 0;\
	overflow-y: auto;\
	flex: 1;\
}\
.pharos-toc-item {\
	padding: 0;\
	margin: 0;\
}\
.pharos-toc-item a {\
	display: block;\
	padding: 0.3rem 1rem;\
	font-size: 0.8125rem;\
	line-height: 1.4;\
	color: var(--slate-700, #4c566a);\
	text-decoration: none;\
	border-left: 2px solid transparent;\
	transition: background 0.1s, color 0.1s, border-color 0.1s;\
}\
.pharos-toc-item a:hover {\
	background: rgba(226, 232, 240, 0.5);\
	color: #334155;\
}\
[data-theme="dark"] .pharos-toc-item a {\
	color: var(--slate-500, #94adcb);\
}\
[data-theme="dark"] .pharos-toc-item a:hover {\
	background: rgba(255, 255, 255, 0.05);\
	color: var(--slate-800, #d8dee9);\
}\
.pharos-toc-level-1 a {\
	padding-left: 0.75rem;\
}\
.pharos-toc-level-2 a {\
	padding-left: 1.75rem;\
	font-size: 0.75rem;\
}\
.pharos-toc-level-3 a {\
	padding-left: 2.75rem;\
	font-size: 0.6875rem;\
}\
.pharos-toc-item a.pharos-toc-active {\
	color: var(--blue-700, #5e81ac);\
	border-left-color: var(--blue-700, #5e81ac);\
	background: rgba(94, 126, 172, 0.08);\
	font-weight: 500;\
}\
[data-theme="dark"] .pharos-toc-item a.pharos-toc-active {\
	color: var(--blue-700, #81a1c1);\
	border-left-color: var(--blue-700, #81a1c1);\
	background: rgba(129, 161, 193, 0.12);\
}\
@media (max-width: 767px) {\
	#pharos-toc-panel {\
		width: 100%;\
		top: auto;\
		bottom: 0;\
		height: 55%;\
		border-left: none;\
		border-top: 1px solid var(--slate-200, #e5e9f0);\
		transform: translateY(100%);\
		border-radius: 8px 8px 0 0;\
	}\
	#pharos-toc-panel.pharos-toc-open {\
		transform: translateY(0);\
	}\
	#pharos-toc-btn {\
		top: auto;\
		bottom: 1rem;\
		right: 1rem;\
		width: 36px;\
		height: 36px;\
		border: 1px solid var(--slate-200, #e5e9f0);\
		border-radius: 6px;\
		transform: none;\
		background: var(--slate-100, #eceff4);\
	}\
	[data-theme="dark"] #pharos-toc-btn {\
		background: var(--slate-100, #353b4a);\
	}\
}\
';
	}
})();
