(function () {
	var cfg = window.__pharos;
	if (!cfg || !cfg.workspace || !cfg.docId) return;

	var NORD_COLORS = [
		'#EBCB8B', '#A3BE8C', '#B48EAD', '#8FBCBB', '#88C0D0',
		'#81A1C1', '#5E81AC', '#BF616A', '#D08770'
	];
	var DEFAULT_COLOR = '#EBCB8B';
	var PREFIX_LEN = 30;
	var SUFFIX_LEN = 30;

	var highlights = [];
	var marks = [];
	var toolbarTimer = null;
	var activeTooltip = null;
	var panel, panelBtn, panelOpen = false;
	var btnTooltipEl = null;

	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', init);
	} else {
		init();
	}

	function init() {
		injectStyles();
		fetchHighlights();
		bindSelectionEvents();
		buildPanel();
	}

	function apiBase() {
		return '/api/workspaces/name/' + encodeURIComponent(cfg.workspace) + '/highlights';
	}

	function fetchHighlights() {
		var url = apiBase() + '?docType=' + encodeURIComponent(cfg.docType) + '&docId=' + cfg.docId;
		fetch(url).then(function (r) { return r.json(); }).then(function (data) {
			highlights = data || [];
			anchorAll();
			renderPanel();
		}).catch(function () {});
	}

	// ── 1. Anchor matching + <mark> rendering ──

	function anchorAll() {
		marks = [];
		highlights.forEach(function (hl) {
			var range = findAnchor(hl);
			if (range) {
				wrapMark(hl, range);
			} else {
				hl._drifted = true;
			}
		});
	}

	function findAnchor(hl) {
		var anchor;
		try { anchor = JSON.parse(hl.anchorData || '{}'); } catch (_) { anchor = {}; }
		var text = anchor.text || '';
		if (!text) return null;

		var map = buildTextMap();
		var full = map.text;
		var prefix = anchor.prefix || '';
		var suffix = anchor.suffix || '';

		var positions = findAllOccurrences(full, text);
		if (positions.length === 0) return null;

		// Prefer a position where prefix + suffix match (exact re-anchor).
		var best = -1;
		for (var i = 0; i < positions.length; i++) {
			var pos = positions[i];
			var pre = full.substring(Math.max(0, pos - prefix.length), pos);
			var suf = full.substring(pos + text.length, pos + text.length + suffix.length);
			if (pre === prefix && suf === suffix) { best = pos; break; }
		}
		// If no exact match: anchor only when unambiguous (single occurrence).
		// Multiple occurrences without prefix/suffix confirmation = drifted
		// (can't know which instance was highlighted).
		if (best < 0 && positions.length === 1) best = positions[0];
		if (best < 0) return null;

		var start = mapOffsetToNode(best, map, false);
		var end = mapOffsetToNode(best + text.length, map, true);
		if (!start || !end) return null;

		var range = document.createRange();
		try {
			range.setStart(start.node, start.offset);
			range.setEnd(end.node, end.offset);
			if (range.toString() === text) return range;
		} catch (_) {}
		return null;
	}

	function buildTextMap() {
		var nodes = [];
		var text = '';
		var walker = document.createTreeWalker(
			document.body, NodeFilter.SHOW_TEXT, {
				acceptNode: function (n) {
					var p = n.parentElement;
					if (!p) return NodeFilter.FILTER_REJECT;
					var tag = p.tagName;
					if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'NOSCRIPT') return NodeFilter.FILTER_REJECT;
					// Exclude injected UI (panels, toolbar, tooltip, toast, buttons)
					// so panel quotes don't create false duplicate occurrences.
					if (p.closest('#pharos-highlights-panel, #pharos-toc-panel, #pharos-frame-buttons, .pharos-hl-toolbar, .pharos-hl-overlay, .pharos-hl-tooltip, .pharos-hl-toast')) return NodeFilter.FILTER_REJECT;
					return NodeFilter.FILTER_ACCEPT;
				}
			}
		);
		var node;
		while ((node = walker.nextNode())) {
			nodes.push({ node: node, start: text.length });
			text += node.textContent;
		}
		return { text: text, nodes: nodes };
	}

	function findAllOccurrences(haystack, needle) {
		var positions = [];
		var idx = 0;
		while ((idx = haystack.indexOf(needle, idx)) !== -1) {
			positions.push(idx);
			idx += needle.length;
		}
		return positions;
	}

	function mapOffsetToNode(offset, map, preferEnd) {
		var nodes = map.nodes;
		for (var i = nodes.length - 1; i >= 0; i--) {
			if (offset > nodes[i].start) {
				return { node: nodes[i].node, offset: offset - nodes[i].start };
			}
			if (offset === nodes[i].start) {
				// For end positions at a text-node boundary, snap to the end
				// of the previous node so the range doesn't spill into
				// inter-element whitespace (which makes surroundContents throw).
				if (preferEnd && i > 0) {
					return { node: nodes[i - 1].node, offset: nodes[i - 1].node.textContent.length };
				}
				return { node: nodes[i].node, offset: 0 };
			}
		}
		return null;
	}

	function wrapMark(hl, range) {
		var mark = document.createElement('mark');
		mark.className = 'pharos-highlight';
		mark.dataset.highlightId = hl.id;
		mark.dataset.hlColor = hl.color || DEFAULT_COLOR;
		mark.style.backgroundColor = hexToRgba(hl.color || DEFAULT_COLOR, 0.3);
		mark.style.borderBottom = '2px solid ' + (hl.color || DEFAULT_COLOR);
		mark.style.padding = '0 1px';
		mark.style.borderRadius = '2px';
		mark.style.cursor = 'pointer';

		function registerMark() {
			marks.push({ hl: hl, el: mark });
			mark.addEventListener('click', function (e) {
				e.stopPropagation();
				showMarkTooltip(mark, hl);
			});
		}

		try {
			range.surroundContents(mark);
			// Reject if surroundContents wrapped a block element (invalid
			// nesting like <mark><p>...</p></mark>). Undo and drift instead.
			if (mark.querySelector('p,h1,h2,h3,h4,h5,h6,div,li,td,tr,table,blockquote,pre')) {
				var parent = mark.parentNode;
				while (mark.firstChild) parent.insertBefore(mark.firstChild, mark);
				parent.removeChild(mark);
				hl._drifted = true;
				return;
			}
			registerMark();
		} catch (_) {
			hl._drifted = true;
		}
	}

	// ── 2. Selection toolbar ──

	function bindSelectionEvents() {
		document.addEventListener('mouseup', function () {
			clearTimeout(toolbarTimer);
			toolbarTimer = setTimeout(showSelectionToolbar, 300);
		});
		document.addEventListener('mousedown', function (e) {
			if (activeTooltip && !activeTooltip.contains(e.target) && !e.target.classList.contains('pharos-hl-toolbar-btn')) {
				closeTooltip();
			}
			var toolbar = document.querySelector('.pharos-hl-toolbar');
			if (!toolbar || !toolbar.contains(e.target)) {
				hideSelectionToolbar();
			}
		});
	}

	function getSelection() {
		var sel = window.getSelection();
		if (!sel || sel.isCollapsed || sel.rangeCount === 0) return null;
		var text = sel.toString().trim();
		if (text.length < 1) return null;
		var node = sel.anchorNode;
		if (!node || !node.parentElement) return null;
		// Don't show toolbar inside injected UI (panels, buttons, dialogs).
		if (node.parentElement.closest('#pharos-highlights-panel, #pharos-toc-panel, #pharos-frame-buttons, .pharos-hl-toolbar, .pharos-hl-overlay, .pharos-hl-tooltip, .pharos-hl-toast')) {
			return null;
		}
		// Don't show toolbar inside an existing mark (that's mark-click territory).
		if (node.parentElement.closest('.pharos-highlight')) {
			return { sel: sel, existing: node.parentElement.closest('.pharos-highlight') };
		}
		return { sel: sel, existing: null };
	}

	function showSelectionToolbar() {
		hideSelectionToolbar();
		var info = getSelection();
		if (!info) return;
		var sel = info.sel;
		var rect = sel.getRangeAt(0).getBoundingClientRect();
		if (rect.width === 0 && rect.height === 0) return;

		var toolbar = document.createElement('div');
		toolbar.className = 'pharos-hl-toolbar';
		toolbar.style.left = (rect.left + rect.width / 2 + window.scrollX) + 'px';
		toolbar.style.top = (rect.top + window.scrollY - 40) + 'px';

		var hlBtn = document.createElement('button');
		hlBtn.className = 'pharos-hl-toolbar-btn';
		if (info.existing) {
			hlBtn.textContent = 'Change color';
			hlBtn.onclick = function () {
				var existingId = parseInt(info.existing.dataset.highlightId, 10);
				var hl = highlights.filter(function (h) { return h.id === existingId; })[0];
				if (hl) openCreateDialog(hl, null, function (color, note) {
					patchHighlight(hl.id, color, note, function () {
						info.existing.style.backgroundColor = hexToRgba(color, 0.3);
						info.existing.style.borderBottomColor = color;
						hl.color = color;
						hl.noteText = note;
						renderPanel();
					});
					sel.removeAllRanges();
				});
			};
		} else {
			hlBtn.textContent = 'Highlight';
			hlBtn.onclick = function () {
				var range = sel.getRangeAt(0);
				captureSelection(sel, function (anchor, selectedText) {
					openCreateDialog(null, selectedText, function (color, note) {
						createHighlight(anchor, color, note, function (created) {
							highlights.push(created);
							wrapMark(created, range);
							renderPanel();
							sel.removeAllRanges();
						});
					});
				});
			};
		}
		toolbar.appendChild(hlBtn);

		var askBtn = document.createElement('button');
		askBtn.className = 'pharos-hl-toolbar-btn pharos-hl-toolbar-ask';
		askBtn.textContent = 'Ask agent';
		askBtn.disabled = true;
		askBtn.title = 'Ask agent coming soon';
		toolbar.appendChild(askBtn);

		var removeBtn = null;
		if (info.existing) {
			removeBtn = document.createElement('button');
			removeBtn.className = 'pharos-hl-toolbar-btn pharos-hl-toolbar-remove';
			removeBtn.textContent = 'Remove';
			removeBtn.onclick = function () {
				hideSelectionToolbar();
				var existingId = parseInt(info.existing.dataset.highlightId, 10);
				deleteHighlight(existingId, function () {
					var parent = info.existing.parentNode;
					while (info.existing.firstChild) parent.insertBefore(info.existing.firstChild, info.existing);
					parent.removeChild(info.existing);
					highlights = highlights.filter(function (h) { return h.id !== existingId; });
					marks = marks.filter(function (m) { return m.hl.id !== existingId; });
					renderPanel();
				});
				sel.removeAllRanges();
			};
			toolbar.appendChild(removeBtn);
		}

		document.body.appendChild(toolbar);
		// Adjust so toolbar doesn't go off-screen left.
		var tbRect = toolbar.getBoundingClientRect();
		if (tbRect.left < 4) toolbar.style.left = (4 + window.scrollX) + 'px';
	}

	function hideSelectionToolbar() {
		var existing = document.querySelector('.pharos-hl-toolbar');
		if (existing) existing.remove();
	}

	function captureSelection(sel, cb) {
		var range = sel.getRangeAt(0);
		var text = sel.toString();
		var fullMap = buildTextMap();
		var start = range.startContainer;
		var startOffset = range.startOffset;

		// Compute prefix: text before the selection start (up to PREFIX_LEN chars).
		var startMapIdx = -1;
		for (var i = 0; i < fullMap.nodes.length; i++) {
			if (fullMap.nodes[i].node === start) { startMapIdx = i; break; }
		}
		var absStart = startMapIdx >= 0 ? fullMap.nodes[startMapIdx].start + startOffset : 0;
		var prefix = fullMap.text.substring(Math.max(0, absStart - PREFIX_LEN), absStart);
		var suffix = fullMap.text.substring(absStart + text.length, absStart + text.length + SUFFIX_LEN);

		cb(JSON.stringify({ text: text, prefix: prefix, suffix: suffix }), text);
	}

	// ── 3. Create / edit dialog ──

	function openCreateDialog(existing, selectedText, onSave) {
		closeDialog();
		hideSelectionToolbar();
		var overlay = document.createElement('div');
		overlay.className = 'pharos-hl-overlay';

		var dialog = document.createElement('div');
		dialog.className = 'pharos-hl-dialog';

		var title = document.createElement('div');
		title.className = 'pharos-hl-dialog-title';
		title.textContent = existing ? 'Edit highlight' : 'New highlight';
		dialog.appendChild(title);

		if (selectedText) {
			var preview = document.createElement('div');
			preview.className = 'pharos-hl-preview';
			preview.textContent = '"' + (selectedText.length > 80 ? selectedText.substring(0, 80) + '…' : selectedText) + '"';
			dialog.appendChild(preview);
		}

		var colorLabel = document.createElement('label');
		colorLabel.className = 'pharos-hl-label';
		colorLabel.textContent = 'Color';
		dialog.appendChild(colorLabel);

		var picker = document.createElement('div');
		picker.className = 'pharos-hl-color-picker';
		var selectedColor = existing ? existing.color : DEFAULT_COLOR;
		NORD_COLORS.forEach(function (c) {
			var swatch = document.createElement('button');
			swatch.className = 'pharos-hl-swatch';
			swatch.style.backgroundColor = c;
			if (c === selectedColor) swatch.classList.add('pharos-hl-swatch-active');
			swatch.onclick = function (e) {
				e.preventDefault();
				selectedColor = c;
				var all = picker.querySelectorAll('.pharos-hl-swatch');
				for (var i = 0; i < all.length; i++) all[i].classList.remove('pharos-hl-swatch-active');
				swatch.classList.add('pharos-hl-swatch-active');
			};
			picker.appendChild(swatch);
		});
		dialog.appendChild(picker);

		var noteLabel = document.createElement('label');
		noteLabel.className = 'pharos-hl-label';
		noteLabel.textContent = 'Note (optional)';
		dialog.appendChild(noteLabel);

		var noteInput = document.createElement('textarea');
		noteInput.className = 'pharos-hl-note';
		noteInput.rows = 3;
		noteInput.value = existing ? (existing.noteText || '') : '';
		dialog.appendChild(noteInput);

		var actions = document.createElement('div');
		actions.className = 'pharos-hl-actions';

		var cancelBtn = document.createElement('button');
		cancelBtn.className = 'pharos-hl-btn pharos-hl-btn-cancel';
		cancelBtn.textContent = 'Cancel';
		cancelBtn.onclick = function () { closeDialog(); };
		actions.appendChild(cancelBtn);

		var saveBtn = document.createElement('button');
		saveBtn.className = 'pharos-hl-btn pharos-hl-btn-save';
		saveBtn.textContent = 'Save';
		saveBtn.onclick = function () {
			closeDialog();
			onSave(selectedColor, noteInput.value.trim());
		};
		actions.appendChild(saveBtn);

		dialog.appendChild(actions);
		overlay.appendChild(dialog);
		overlay.addEventListener('click', function (e) {
			if (e.target === overlay) closeDialog();
		});
		overlay._onKeyDown = function (e) {
			if (e.key === 'Escape') { e.preventDefault(); closeDialog(); }
		};
		document.addEventListener('keydown', overlay._onKeyDown);
		document.body.appendChild(overlay);
		requestAnimationFrame(function() {
			overlay.classList.add('pharos-hl-overlay--open');
		});
		noteInput.focus();
	}

	function closeDialog() {
		var overlay = document.querySelector('.pharos-hl-overlay');
		if (!overlay) return;
		if (overlay._closing) return;
		overlay._closing = true;
		overlay.classList.remove('pharos-hl-overlay--open');
		document.removeEventListener('keydown', overlay._onKeyDown);
		overlay.addEventListener('transitionend', function cb() {
			overlay.removeEventListener('transitionend', cb);
			overlay.remove();
		});
	}

	// ── 4. Per-mark tooltip ──

	function showMarkTooltip(mark, hl) {
		closeTooltip();
		var tooltip = document.createElement('div');
		tooltip.className = 'pharos-hl-tooltip';

		// Header: color dot + "Highlight" label
		var header = document.createElement('div');
		header.className = 'pharos-hl-tooltip-header';
		var dot = document.createElement('span');
		dot.className = 'pharos-hl-dot';
		dot.style.backgroundColor = hl.color || DEFAULT_COLOR;
		header.appendChild(dot);
		var label = document.createElement('span');
		label.className = 'pharos-hl-tooltip-label';
		label.textContent = 'Highlight';
		header.appendChild(label);
		tooltip.appendChild(header);

		// Body: the note, or a prompt to add one
		var body = document.createElement('div');
		body.className = 'pharos-hl-tooltip-body';
		if (hl.noteText) {
			body.textContent = hl.noteText;
		} else {
			var addNote = document.createElement('a');
			addNote.className = 'pharos-hl-tooltip-addnote';
			addNote.textContent = 'Add a note…';
			addNote.href = '#';
			addNote.onclick = function (e) {
				e.preventDefault();
				e.stopPropagation();
				closeTooltip();
				openCreateDialog(hl, null, function (color, note) {
					patchHighlight(hl.id, color, note, function () {
						mark.style.backgroundColor = hexToRgba(color, 0.3);
						mark.style.borderBottomColor = color;
						hl.color = color;
						hl.noteText = note;
						renderPanel();
					});
				});
			};
			body.appendChild(addNote);
		}
		tooltip.appendChild(body);

		// Actions: Edit + Delete
		var actions = document.createElement('div');
		actions.className = 'pharos-hl-tooltip-actions';

		var editBtn = document.createElement('button');
		editBtn.className = 'pharos-hl-tooltip-btn';
		editBtn.textContent = 'Edit';
		editBtn.onclick = function (e) {
			e.stopPropagation();
			closeTooltip();
			openCreateDialog(hl, null, function (color, note) {
				patchHighlight(hl.id, color, note, function () {
					mark.style.backgroundColor = hexToRgba(color, 0.3);
					mark.style.borderBottomColor = color;
					hl.color = color;
					hl.noteText = note;
					renderPanel();
				});
			});
		};
		actions.appendChild(editBtn);

		var deleteBtn = document.createElement('button');
		deleteBtn.className = 'pharos-hl-tooltip-btn pharos-hl-tooltip-btn-delete';
		deleteBtn.textContent = 'Delete';
		deleteBtn.onclick = function (e) {
			e.stopPropagation();
			closeTooltip();
			deleteHighlight(hl.id, function () {
				var parent = mark.parentNode;
				while (mark.firstChild) parent.insertBefore(mark.firstChild, mark);
				parent.removeChild(mark);
				highlights = highlights.filter(function (h) { return h.id !== hl.id; });
				marks = marks.filter(function (m) { return m.hl.id !== hl.id; });
				renderPanel();
			});
		};
		actions.appendChild(deleteBtn);
		tooltip.appendChild(actions);

		// Position: centered below the mark, with an arrow pointing up
		var rect = mark.getBoundingClientRect();
		tooltip.style.left = (rect.left + rect.width / 2 + window.scrollX) + 'px';
		tooltip.style.top = (rect.bottom + window.scrollY + 8) + 'px';

		document.body.appendChild(tooltip);
		activeTooltip = tooltip;

		// Adjust if off-screen
		var tRect = tooltip.getBoundingClientRect();
		if (tRect.left < 4) tooltip.style.left = (4 + window.scrollX) + 'px';
		if (tRect.right > window.innerWidth - 4) {
			tooltip.style.left = (window.innerWidth - tRect.width - 4 + window.scrollX) + 'px';
		}
	}

	function closeTooltip() {
		if (activeTooltip) { activeTooltip.remove(); activeTooltip = null; }
	}

	// ── 5. Highlights panel (slide-in) ──

	function getFrameButtonsContainer() {
		var c = document.getElementById('pharos-frame-buttons');
		if (!c) {
			c = document.createElement('div');
			c.id = 'pharos-frame-buttons';
			document.body.appendChild(c);
		}
		return c;
	}

	function buildPanel() {
		panelBtn = document.createElement('button');
		panelBtn.id = 'pharos-highlights-btn';
		panelBtn.setAttribute('aria-label', 'Highlights');
		panelBtn.setAttribute('data-tooltip', 'Highlights');
		panelBtn.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 11-6 6v3h9l3-3"/><path d="m22 12-4.6 4.6a2 2 0 0 1-2.8 0l-5.2-5.2a2 2 0 0 1 0-2.8L14 4"/></svg>';
		panelBtn.addEventListener('click', togglePanel);

		panelBtn.addEventListener('mouseenter', function () {
			if (btnTooltipEl) return;
			var r = this.getBoundingClientRect();
			btnTooltipEl = document.createElement('div');
			btnTooltipEl.textContent = 'Highlights';
			btnTooltipEl.style.position = 'fixed';
			btnTooltipEl.style.color = '#f8fafc';
			btnTooltipEl.style.whiteSpace = 'nowrap';
			btnTooltipEl.style.pointerEvents = 'none';
			btnTooltipEl.style.zIndex = '60';
			btnTooltipEl.style.background = '#1e293b';
			btnTooltipEl.style.borderRadius = '4px';
			btnTooltipEl.style.padding = '4px 10px';
			btnTooltipEl.style.fontSize = '.75rem';
			btnTooltipEl.style.fontWeight = '500';
			btnTooltipEl.style.lineHeight = '1.4';
			btnTooltipEl.style.right = (window.innerWidth - r.left + 10) + 'px';
			btnTooltipEl.style.top = (r.top + r.height / 2) + 'px';
			btnTooltipEl.style.transform = 'translateY(-50%)';
			var tt = document.querySelector('[tooltip-for="highlights"]');
			if (tt) tt.remove();
			btnTooltipEl.setAttribute('tooltip-for', 'highlights');
			document.body.appendChild(btnTooltipEl);
		});
		panelBtn.addEventListener('mouseleave', function () {
			if (btnTooltipEl) { btnTooltipEl.remove(); btnTooltipEl = null; }
		});

		panel = document.createElement('div');
		panel.id = 'pharos-highlights-panel';
		panel.setAttribute('aria-label', 'Highlights');
		panel.innerHTML = '<div class="pharos-hl-panel-header"><span class="pharos-hl-panel-title">Highlights</span><button class="pharos-hl-panel-close" aria-label="Close">&times;</button></div><div class="pharos-hl-panel-body"></div>';
		panel.querySelector('.pharos-hl-panel-close').addEventListener('click', closePanel);

		getFrameButtonsContainer().appendChild(panelBtn);
		document.body.appendChild(panel);

		document.addEventListener('click', function (e) {
			if (!panelOpen) return;
			if (panel.contains(e.target) || panelBtn.contains(e.target)) return;
			closePanel();
		});
	}

	function togglePanel() {
		if (panelOpen) closePanel(); else openPanel();
	}

	function openPanel() {
		panelOpen = true;
		panel.classList.add('pharos-hl-panel-open');
		panelBtn.classList.add('pharos-hl-btn-active');
		if (btnTooltipEl) { btnTooltipEl.remove(); btnTooltipEl = null; }
	}

	function closePanel() {
		panelOpen = false;
		panel.classList.remove('pharos-hl-panel-open');
		panelBtn.classList.remove('pharos-hl-btn-active');
	}

	function renderPanel() {
		var body = panel.querySelector('.pharos-hl-panel-body');
		if (!body) return;
		body.innerHTML = '';

		if (highlights.length === 0) {
			var empty = document.createElement('div');
			empty.className = 'pharos-hl-panel-empty';
			empty.textContent = 'No highlights yet. Select text to create one.';
			body.appendChild(empty);
			return;
		}

		var anchored = highlights.filter(function (h) { return !h._drifted; });
		var drifted = highlights.filter(function (h) { return h._drifted; });

		if (anchored.length > 0) {
			var section = document.createElement('div');
			section.className = 'pharos-hl-panel-section';
			anchored.forEach(function (hl) { section.appendChild(buildPanelRow(hl, false)); });
			body.appendChild(section);
		}

		if (drifted.length > 0) {
			var driftHeader = document.createElement('div');
			driftHeader.className = 'pharos-hl-panel-drift-header';
			driftHeader.textContent = 'Drifted (text changed)';
			body.appendChild(driftHeader);

			var driftSection = document.createElement('div');
			driftSection.className = 'pharos-hl-panel-section';
			drifted.forEach(function (hl) { driftSection.appendChild(buildPanelRow(hl, true)); });
			body.appendChild(driftSection);
		}
	}

	function buildPanelRow(hl, drifted) {
		var row = document.createElement('div');
		row.className = 'pharos-hl-panel-row';

		var dot = document.createElement('span');
		dot.className = 'pharos-hl-dot';
		dot.style.backgroundColor = hl.color || DEFAULT_COLOR;
		row.appendChild(dot);

		var content = document.createElement('div');
		content.className = 'pharos-hl-panel-row-content';

		var anchor;
		try { anchor = JSON.parse(hl.anchorData || '{}'); } catch (_) { anchor = {}; }
		var text = anchor.text || '(unknown)';

		if (drifted) {
			var badge = document.createElement('span');
			badge.className = 'pharos-hl-drift-badge';
			badge.textContent = 'drifted';
			content.appendChild(badge);
			content.appendChild(document.createTextNode(' "' + truncate(text, 50) + '"'));
			if (hl.noteText) {
				var note = document.createElement('div');
				note.className = 'pharos-hl-panel-row-note';
				note.textContent = hl.noteText;
				content.appendChild(note);
			}
		} else {
			var quote = document.createElement('div');
			quote.className = 'pharos-hl-panel-row-quote';
			quote.textContent = truncate(text, 60);
			quote.addEventListener('click', function () {
				var markEl = document.querySelector('.pharos-highlight[data-highlight-id="' + hl.id + '"]');
				if (markEl) {
					closePanel();
					markEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
					markEl.classList.add('pharos-hl-flash');
					setTimeout(function () { markEl.classList.remove('pharos-hl-flash'); }, 1200);
				}
			});
			content.appendChild(quote);

			if (hl.noteText) {
				var note2 = document.createElement('div');
				note2.className = 'pharos-hl-panel-row-note';
				note2.textContent = hl.noteText;
				content.appendChild(note2);
			}
		}
		row.appendChild(content);

		var delBtn = document.createElement('button');
		delBtn.className = 'pharos-hl-panel-row-delete';
		delBtn.setAttribute('aria-label', 'Delete highlight');
		delBtn.innerHTML = '&times;';
		delBtn.onclick = function (e) {
			e.stopPropagation();
			deleteHighlight(hl.id, function () {
				var markEl = document.querySelector('.pharos-highlight[data-highlight-id="' + hl.id + '"]');
				if (markEl) {
					var parent = markEl.parentNode;
					while (markEl.firstChild) parent.insertBefore(markEl.firstChild, markEl);
					parent.removeChild(markEl);
				}
				highlights = highlights.filter(function (h) { return h.id !== hl.id; });
				marks = marks.filter(function (m) { return m.hl.id !== hl.id; });
				renderPanel();
			});
		};
		row.appendChild(delBtn);

		return row;
	}

	// ── API calls ──

	function createHighlight(anchorData, color, note, cb) {
		fetch(apiBase(), {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				docType: cfg.docType, docId: cfg.docId,
				color: color, noteText: note, anchorData: anchorData
			})
		}).then(function (r) {
			if (!r.ok) { showToast('Failed to create highlight'); return; }
			return r.json();
		}).then(function (created) {
			if (created && created.id) cb(created);
		}).catch(function () { showToast('Failed to create highlight'); });
	}

	function patchHighlight(id, color, note, cb) {
		fetch(apiBase() + '/' + id, {
			method: 'PATCH',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ color: color, noteText: note })
		}).then(function (r) { if (r.ok) cb(); }).catch(function () { showToast('Failed to update'); });
	}

	function deleteHighlight(id, cb) {
		fetch(apiBase() + '/' + id, { method: 'DELETE' })
			.then(function (r) { if (r.ok) cb(); })
			.catch(function () { showToast('Failed to delete'); });
	}

	// ── Toast ──

	function showToast(msg) {
		var existing = document.querySelector('.pharos-hl-toast');
		if (existing) existing.remove();
		var toast = document.createElement('div');
		toast.className = 'pharos-hl-toast';
		toast.textContent = msg;
		document.body.appendChild(toast);
		setTimeout(function () { toast.classList.add('pharos-hl-toast-show'); }, 10);
		setTimeout(function () {
			toast.classList.remove('pharos-hl-toast-show');
			setTimeout(function () { toast.remove(); }, 300);
		}, 2500);
	}

	// ── Helpers ──

	function hexToRgba(hex, alpha) {
		var h = hex.replace('#', '');
		if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
		var r = parseInt(h.substring(0, 2), 16) || 235;
		var g = parseInt(h.substring(2, 4), 16) || 203;
		var b = parseInt(h.substring(4, 6), 16) || 139;
		return 'rgba(' + r + ',' + g + ',' + b + ',' + alpha + ')';
	}

	function truncate(s, max) {
		s = s.replace(/\s+/g, ' ').trim();
		return s.length > max ? s.substring(0, max) + '…' : s;
	}

	function injectStyles() {
		var style = document.createElement('style');
		style.textContent = '\
.pharos-highlight {\
	color: var(--slate-900, #2e3440);\
}\
[data-theme="dark"] .pharos-highlight {\
	color: #f8fafc;\
}\
.pharos-hl-toolbar {\
	position: absolute;\
	z-index: 10000;\
	display: flex;\
	gap: 1px;\
	background: #1e293b;\
	border-radius: 6px;\
	padding: 3px;\
	box-shadow: 0 4px 12px rgba(0,0,0,0.25);\
	transform: translateX(-50%);\
}\
.pharos-hl-toolbar-btn {\
	border: none;\
	background: #334155;\
	color: #f1f5f9;\
	padding: 5px 10px;\
	border-radius: 4px;\
	font-size: 0.75rem;\
	font-weight: 500;\
	cursor: pointer;\
	white-space: nowrap;\
	transition: background 0.1s;\
}\
.pharos-hl-toolbar-btn:hover:not(:disabled) { background: #475569; }\
.pharos-hl-toolbar-btn:disabled { opacity: 0.4; cursor: not-allowed; }\
.pharos-hl-toolbar-ask { opacity: 0.5; }\
.pharos-hl-toolbar-remove { color: #fca5a5; }\
.pharos-hl-toolbar-remove:hover:not(:disabled) { background: #7f1d1d; }\
\
.pharos-hl-overlay {\
	position: fixed;\
	inset: 0;\
	z-index: 10001;\
	background: rgba(0,0,0,0.3);\
	display: flex;\
	align-items: center;\
	justify-content: center;\
	opacity: 0;\
	transition: opacity 0.15s ease-out;\
}\
.pharos-hl-overlay--open {\
	opacity: 1;\
}\
.pharos-hl-dialog {\
	background: var(--color-white, #fff);\
	border-radius: 10px;\
	padding: 1.25rem;\
	width: 340px;\
	max-width: 90vw;\
	box-shadow: 0 10px 30px rgba(0,0,0,0.2);\
	transform-origin: center;\
	transform: scale(0.95);\
	opacity: 0;\
	transition: opacity 0.15s ease-out, transform 0.15s ease-out;\
}\
.pharos-hl-overlay--open .pharos-hl-dialog {\
	transform: scale(1);\
	opacity: 1;\
}\
@media (prefers-reduced-motion: reduce) {\
	.pharos-hl-overlay,\
	.pharos-hl-dialog {\
		transition: none !important;\
	}\
}\
[data-theme="dark"] .pharos-hl-dialog {\
	background: var(--color-slate-50, #353b4a);\
	color: var(--color-slate-800, #d8dee9);\
}\
.pharos-hl-dialog-title {\
	font-size: 0.9rem;\
	font-weight: 600;\
	margin-bottom: 0.75rem;\
}\
.pharos-hl-preview {\
	font-size: 0.75rem;\
	color: var(--color-slate-500, #6b7689);\
	font-style: italic;\
	margin-bottom: 0.75rem;\
	padding: 0.5rem;\
	background: var(--color-slate-50, #f8fafc);\
	border-radius: 6px;\
	border-left: 3px solid var(--color-amber-300, #EBCB8B);\
}\
[data-theme="dark"] .pharos-hl-preview {\
	background: var(--color-slate-100, #2e3440);\
	color: var(--color-slate-400, #81a1c1);\
}\
.pharos-hl-label {\
	display: block;\
	font-size: 0.6875rem;\
	font-weight: 500;\
	text-transform: uppercase;\
	letter-spacing: 0.04em;\
	color: var(--color-slate-500, #6b7689);\
	margin-bottom: 0.375rem;\
	margin-top: 0.75rem;\
}\
[data-theme="dark"] .pharos-hl-label { color: var(--color-slate-400, #81a1c1); }\
.pharos-hl-color-picker {\
	display: flex;\
	flex-wrap: wrap;\
	gap: 6px;\
	margin-bottom: 0.5rem;\
}\
.pharos-hl-swatch {\
	width: 28px;\
	height: 28px;\
	border-radius: 50%;\
	border: 2px solid transparent;\
	cursor: pointer;\
	padding: 0;\
	transition: transform 0.1s, border-color 0.1s;\
}\
.pharos-hl-swatch:hover { transform: scale(1.1); }\
.pharos-hl-swatch-active {\
	border-color: var(--color-slate-700, #2e3440);\
	transform: scale(1.1);\
}\
[data-theme="dark"] .pharos-hl-swatch-active { border-color: #d8dee9; }\
.pharos-hl-note {\
	width: 100%;\
	border: 1px solid var(--color-slate-200, #e2e8f0);\
	border-radius: 6px;\
	padding: 0.5rem;\
	font-size: 0.8125rem;\
	font-family: inherit;\
	resize: vertical;\
	box-sizing: border-box;\
}\
[data-theme="dark"] .pharos-hl-note {\
	background: var(--color-slate-100, #2e3440);\
	border-color: var(--color-slate-200, #434c5e);\
	color: var(--color-slate-800, #d8dee9);\
}\
.pharos-hl-actions {\
	display: flex;\
	gap: 0.5rem;\
	justify-content: flex-end;\
	margin-top: 1rem;\
}\
.pharos-hl-btn {\
	padding: 0.4rem 0.9rem;\
	border-radius: 6px;\
	font-size: 0.8125rem;\
	font-weight: 500;\
	cursor: pointer;\
	border: 1px solid transparent;\
}\
.pharos-hl-btn-cancel {\
	background: transparent;\
	color: var(--color-slate-500, #6b7689);\
	border-color: var(--color-slate-200, #e2e8f0);\
}\
.pharos-hl-btn-cancel:hover { background: var(--color-slate-50, #f8fafc); }\
[data-theme="dark"] .pharos-hl-btn-cancel { color: var(--color-slate-400, #81a1c1); border-color: var(--color-slate-200, #434c5e); }\
.pharos-hl-btn-save {\
	background: var(--color-amber-500, #EBCB8B);\
	color: #2e3440;\
}\
.pharos-hl-btn-save:hover { filter: brightness(0.95); }\
\
.pharos-hl-dot {\
	display: inline-block;\
	width: 10px;\
	height: 10px;\
	border-radius: 50%;\
	flex-shrink: 0;\
}\
\
.pharos-hl-tooltip {\
	position: absolute;\
	z-index: 10000;\
	background: var(--slate-800);\
	color: var(--slate-50);\
	border: none;\
	border-radius: 8px;\
	padding: 0;\
	box-shadow: 0 8px 24px rgba(0,0,0,0.2);\
	max-width: 260px;\
	min-width: 180px;\
	transform: translateX(-50%);\
	overflow: visible;\
}\
.pharos-hl-tooltip::before {\
	content: "";\
	position: absolute;\
	top: -7px;\
	left: 50%;\
	margin-left: -7px;\
	border: 7px solid transparent;\
	border-bottom-color: var(--slate-800);\
}\
.pharos-hl-tooltip-header {\
	display: flex;\
	align-items: center;\
	gap: 6px;\
	padding: 10px 14px 0.25rem;\
}\
.pharos-hl-tooltip-header .pharos-hl-dot {\
	width: 8px;\
	height: 8px;\
	margin: 0;\
}\
.pharos-hl-tooltip-label {\
	font-size: 0.625rem;\
	font-weight: 600;\
	text-transform: uppercase;\
	letter-spacing: 0.05em;\
	color: var(--slate-200);\
}\
.pharos-hl-tooltip-body {\
	padding: 0 14px 0.5rem;\
	font-size: 0.8125rem;\
	line-height: 1.5;\
	color: var(--slate-50);\
	word-wrap: break-word;\
}\
.pharos-hl-tooltip-addnote {\
	color: #8fbcbb;\
	text-decoration: none;\
	font-size: 0.75rem;\
	font-style: italic;\
	cursor: pointer;\
}\
.pharos-hl-tooltip-addnote:hover,\
.pharos-hl-tooltip-addnote:focus {\
	color: #8fbcbb;\
	text-decoration: underline;\
}\
[data-theme="dark"] .pharos-hl-tooltip-addnote,\
[data-theme="dark"] .pharos-hl-tooltip-addnote:hover,\
[data-theme="dark"] .pharos-hl-tooltip-addnote:focus { color: #5e81ac; }\
.pharos-hl-tooltip-actions {\
	display: flex;\
	gap: 0.375rem;\
	padding: 0.5rem 0.75rem;\
	border-top: 1px solid rgba(255,255,255,0.12);\
	justify-content: flex-end;\
}\
.pharos-hl-tooltip-btn {\
	border: none;\
	background: transparent;\
	color: var(--slate-200);\
	padding: 0.3rem 0.625rem;\
	border-radius: 5px;\
	font-size: 0.75rem;\
	font-weight: 500;\
	cursor: pointer;\
	transition: background 0.1s, color 0.1s;\
}\
.pharos-hl-tooltip-btn:hover {\
	background: rgba(255,255,255,0.1);\
	color: var(--slate-50);\
}\
.pharos-hl-tooltip-btn-delete { color: var(--slate-200); }\
.pharos-hl-tooltip-btn-delete:hover {\
	background: rgba(191, 97, 106, 0.25);\
	color: #bf616a;\
}\
[data-theme="dark"] .pharos-hl-tooltip-label { color: var(--slate-200, #d8dee9); }\
[data-theme="dark"] .pharos-hl-tooltip-body { color: var(--slate-100, #eceff4); }\
[data-theme="dark"] .pharos-hl-tooltip-addnote,\
[data-theme="dark"] .pharos-hl-tooltip-addnote:hover,\
[data-theme="dark"] .pharos-hl-tooltip-addnote:focus { color: var(--frost-1, #8fbcbb); }\
[data-theme="dark"] .pharos-hl-tooltip-btn { color: var(--slate-200, #d8dee9); }\
[data-theme="dark"] .pharos-hl-tooltip-btn:hover {\
	background: rgba(255,255,255,0.12);\
	color: var(--slate-50, #eceff4);\
}\
[data-theme="dark"] .pharos-hl-tooltip-btn-delete { color: var(--slate-200, #d8dee9); }\
[data-theme="dark"] .pharos-hl-tooltip-btn-delete:hover {\
	background: rgba(191, 97, 106, 0.25);\
	color: #bf616a;\
}\
\
#pharos-frame-buttons {\
	position: fixed;\
	top: 50%;\
	right: 0;\
	z-index: 998;\
	transform: translateY(-50%);\
	display: flex;\
	flex-direction: column;\
	gap: 2px;\
}\
@media (max-width: 767px) {\
	#pharos-frame-buttons {\
		top: auto;\
		bottom: 1rem;\
		right: 1rem;\
		transform: none;\
	}\
}\
\
#pharos-highlights-btn {\
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
	transition: background 0.1s, color 0.1s;\
}\
@media (max-width: 767px) {\
	#pharos-highlights-btn {\
		width: 36px;\
		height: 36px;\
		border: 1px solid var(--slate-200, #e5e9f0);\
		border-radius: 6px;\
	}\
}\
#pharos-highlights-btn:hover {\
	background: var(--slate-200, #e5e9f0);\
	color: var(--slate-700, #4c566a);\
}\
#pharos-highlights-btn.pharos-hl-btn-active {\
	background: var(--slate-100, #eceff4);\
	color: var(--blue-700, #5e81ac);\
	border-color: var(--slate-200, #e5e9f0);\
}\
[data-theme="dark"] #pharos-highlights-btn {\
	background: var(--slate-100, #353b4a);\
	border-color: var(--slate-200, #434c5e);\
	color: var(--slate-400, #81a1c1);\
}\
[data-theme="dark"] #pharos-highlights-btn:hover,\
[data-theme="dark"] #pharos-highlights-btn.pharos-hl-btn-active {\
	background: var(--slate-200, #434c5e);\
	color: var(--blue-700, #81a1c1);\
}\
\
#pharos-highlights-panel {\
	position: fixed;\
	top: 0;\
	right: 0;\
	width: 280px;\
	height: 100%;\
	z-index: 999;\
	background: var(--slate-100, #eceff4);\
	border-left: 1px solid var(--slate-200, #e5e9f0);\
	transform: translateX(100%);\
	transition: transform 0.2s ease;\
	display: flex;\
	flex-direction: column;\
}\
#pharos-highlights-panel.pharos-hl-panel-open { transform: translateX(0); }\
[data-theme="dark"] #pharos-highlights-panel {\
	background: var(--slate-100, #353b4a);\
	border-color: var(--slate-200, #434c5e);\
}\
.pharos-hl-panel-header {\
	display: flex;\
	align-items: center;\
	justify-content: space-between;\
	padding: 0.75rem 1rem 0.5rem;\
	flex-shrink: 0;\
}\
.pharos-hl-panel-title {\
	font-size: 0.6875rem;\
	font-weight: 500;\
	color: var(--slate-500, #6b7689);\
	letter-spacing: 0.04em;\
	text-transform: uppercase;\
}\
[data-theme="dark"] .pharos-hl-panel-title { color: var(--slate-400, #81a1c1); }\
.pharos-hl-panel-close {\
	width: 32px;\
	height: 32px;\
	padding: 0;\
	border: 1px solid transparent;\
	background: none;\
	color: var(--slate-400, #8891a0);\
	cursor: pointer;\
	font-size: 1.25rem;\
	border-radius: 6px;\
	display: flex;\
	align-items: center;\
	justify-content: center;\
}\
.pharos-hl-panel-close:hover {\
	background: var(--slate-200, #e5e9f0);\
	color: var(--slate-900, #2e3440);\
}\
[data-theme="dark"] .pharos-hl-panel-close:hover {\
	background: var(--slate-200, #434c5e);\
	color: var(--slate-800, #eceff4);\
}\
.pharos-hl-panel-body {\
	overflow-y: auto;\
	flex: 1;\
	padding: 0.375rem 0;\
}\
.pharos-hl-panel-empty {\
	padding: 1.5rem 1rem;\
	text-align: center;\
	font-size: 0.8125rem;\
	color: var(--slate-400, #8891a0);\
}\
.pharos-hl-panel-section { padding: 0 0.25rem; }\
.pharos-hl-panel-drift-header {\
	padding: 0.625rem 1rem 0.25rem;\
	font-size: 0.625rem;\
	font-weight: 600;\
	text-transform: uppercase;\
	letter-spacing: 0.06em;\
	color: var(--rose-500, #f43f5e);\
}\
.pharos-hl-panel-row {\
	display: flex;\
	align-items: flex-start;\
	gap: 0.5rem;\
	padding: 0.5rem 0.75rem;\
	border-radius: 6px;\
	margin: 0 0.25rem 2px;\
	transition: background 0.1s;\
}\
.pharos-hl-panel-row:hover { background: var(--slate-200, #e5e9f0); }\
[data-theme="dark"] .pharos-hl-panel-row:hover { background: rgba(255,255,255,0.05); }\
.pharos-hl-panel-row .pharos-hl-dot {\
	margin-top: 4px;\
	flex-shrink: 0;\
}\
.pharos-hl-panel-row-content {\
	flex: 1;\
	min-width: 0;\
}\
.pharos-hl-panel-row-quote {\
	font-size: 0.75rem;\
	color: var(--slate-700, #4c566a);\
	cursor: pointer;\
	line-height: 1.35;\
}\
.pharos-hl-panel-row-quote:hover { color: var(--blue-700, #5e81ac); }\
[data-theme="dark"] .pharos-hl-panel-row-quote { color: var(--slate-500, #94adcb); }\
.pharos-hl-panel-row-note {\
	font-size: 0.6875rem;\
	color: var(--slate-400, #8891a0);\
	margin-top: 2px;\
	font-style: italic;\
}\
.pharos-hl-drift-badge {\
	display: inline-block;\
	font-size: 0.5625rem;\
	font-weight: 600;\
	text-transform: uppercase;\
	letter-spacing: 0.04em;\
	background: var(--rose-100, #ffe4e6);\
	color: var(--rose-600, #e11d48);\
	padding: 1px 5px;\
	border-radius: 3px;\
	margin-right: 4px;\
}\
[data-theme="dark"] .pharos-hl-drift-badge {\
	background: rgba(244, 63, 94, 0.2);\
	color: #fda4af;\
}\
.pharos-hl-panel-row-delete {\
	flex-shrink: 0;\
	border: none;\
	background: none;\
	color: var(--slate-400, #8891a0);\
	cursor: pointer;\
	font-size: 1rem;\
	width: 22px;\
	height: 22px;\
	border-radius: 4px;\
	display: flex;\
	align-items: center;\
	justify-content: center;\
	opacity: 0;\
	transition: opacity 0.1s, background 0.1s, color 0.1s;\
}\
.pharos-hl-panel-row:hover .pharos-hl-panel-row-delete { opacity: 1; }\
.pharos-hl-panel-row-delete:hover { background: var(--rose-100, #ffe4e6); color: var(--rose-600, #e11d48); }\
[data-theme="dark"] .pharos-hl-panel-row-delete:hover { background: rgba(244,63,94,0.2); color: #fda4af; }\
\
.pharos-hl-flash {\
	animation: pharos-hl-flash 1.2s ease;\
}\
@keyframes pharos-hl-flash {\
	0%, 100% { outline: none; }\
	20%, 60% { outline: 3px solid var(--color-amber-300, #EBCB8B); outline-offset: 1px; }\
}\
\
.pharos-hl-toast {\
	position: fixed;\
	bottom: 1.5rem;\
	left: 50%;\
	transform: translate(-50%, 100px);\
	z-index: 10002;\
	background: #1e293b;\
	color: #f1f5f9;\
	padding: 0.625rem 1.25rem;\
	border-radius: 8px;\
	font-size: 0.8125rem;\
	box-shadow: 0 4px 12px rgba(0,0,0,0.3);\
	opacity: 0;\
	transition: transform 0.3s, opacity 0.3s;\
}\
.pharos-hl-toast-show {\
	transform: translate(-50%, 0);\
	opacity: 1;\
}\
\
@media (max-width: 767px) {\
	#pharos-highlights-panel { width: 100%; }\
	[data-theme="dark"] #pharos-highlights-btn { background: var(--slate-100, #353b4a); }\
}\
';
		document.head.appendChild(style);
	}
})();
