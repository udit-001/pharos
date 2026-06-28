(function() {
  'use strict';

  var icons = {
    expand: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3H5a2 2 0 0 0-2 2v3"/><path d="M21 8V5a2 2 0 0 0-2-2h-3"/><path d="M3 16v3a2 2 0 0 0 2 2h3"/><path d="M16 21h3a2 2 0 0 0 2-2v-3"/></svg>',
    close: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',
    zoomIn: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/><path d="M11 8v6"/><path d="M8 11h6"/></svg>',
    zoomOut: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/><path d="M8 11h6"/></svg>',
    reset: '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>'
  };

  function Lightbox(svgEl) {
    this.svgEl = svgEl;
    this.clone = null;
    this.overlay = null;
    this.scale = 1;
    this.panX = 0;
    this.panY = 0;
    this.dragging = false;
    this.startX = 0;
    this.startY = 0;
    this.isOpen = false;
    this._handlers = [];
  }

  Lightbox.prototype._on = function(target, event, handler, options) {
    target.addEventListener(event, handler, options);
    this._handlers.push({ el: target, event: event, handler: handler, options: options });
  };

  Lightbox.prototype._clearListeners = function() {
    for (var i = 0; i < this._handlers.length; i++) {
      var h = this._handlers[i];
      h.el.removeEventListener(h.event, h.handler);
    }
    this._handlers = [];
  };

  Lightbox.prototype._applyTransform = function() {
    this.clone.style.transform = 'translate(' + this.panX + 'px, ' + this.panY + 'px) scale(' + this.scale + ')';
    this.clone.style.transformOrigin = '0 0';
  };

  Lightbox.prototype._onWheel = function(e) {
    e.preventDefault();
    var delta = e.deltaY > 0 ? -0.1 : 0.1;
    var newScale = Math.min(Math.max(this.scale + delta, 0.5), 5);
    var viewport = this.overlay.querySelector('.mermaid-lightbox-viewport');
    var rect = viewport.getBoundingClientRect();
    var cx = e.clientX - rect.left;
    var cy = e.clientY - rect.top;
    this.panX = cx - (cx - this.panX) * (newScale / this.scale);
    this.panY = cy - (cy - this.panY) * (newScale / this.scale);
    this.scale = newScale;
    this._applyTransform();
  };

  Lightbox.prototype._onMouseDown = function(e) {
    if (e.button !== 0) return;
    this.dragging = true;
    this.startX = e.clientX - this.panX;
    this.startY = e.clientY - this.panY;
    this.overlay.querySelector('.mermaid-lightbox-viewport').style.cursor = 'grabbing';
  };

  Lightbox.prototype._onMouseMove = function(e) {
    if (!this.dragging) return;
    this.panX = e.clientX - this.startX;
    this.panY = e.clientY - this.startY;
    this._applyTransform();
  };

  Lightbox.prototype._onMouseUp = function() {
    this.dragging = false;
    var vp = this.overlay && this.overlay.querySelector('.mermaid-lightbox-viewport');
    if (vp) vp.style.cursor = 'grab';
  };

  Lightbox.prototype._onBackdropClick = function(e) {
    if (e.target === this.overlay) this.close();
  };

  Lightbox.prototype._onKeyDown = function(e) {
    if (e.key === 'Escape') this.close();
  };

  Lightbox.prototype.open = function() {
    if (this.isOpen) return;

    this.scale = 1;
    this.panX = 0;
    this.panY = 0;
    this.dragging = false;

    this.clone = this.svgEl.cloneNode(true);
    this.clone.removeAttribute('width');
    this.clone.removeAttribute('height');
    this.clone.style.width = '100%';
    this.clone.style.height = 'auto';

    this.overlay = document.createElement('div');
    this.overlay.className = 'mermaid-lightbox';
    this.overlay.innerHTML =
      '<div class="mermaid-lightbox-panel">' +
        '<button class="mermaid-lightbox-close" aria-label="Close">' + icons.close + '</button>' +
        '<div class="mermaid-lightbox-viewport"></div>' +
        '<div class="mermaid-lightbox-controls">' +
          '<button class="mermaid-zoom-in" aria-label="Zoom in" title="Zoom in">' + icons.zoomIn + '</button>' +
          '<button class="mermaid-zoom-reset" aria-label="Reset zoom" title="Reset">' + icons.reset + '</button>' +
          '<button class="mermaid-zoom-out" aria-label="Zoom out" title="Zoom out">' + icons.zoomOut + '</button>' +
        '</div>' +
      '</div>';

    var viewport = this.overlay.querySelector('.mermaid-lightbox-viewport');
    viewport.appendChild(this.clone);
    document.body.appendChild(this.overlay);
    document.body.style.overflow = 'hidden';

    this.isOpen = true;

    var self = this;
    this._on(viewport, 'wheel', function(e) { self._onWheel(e); }, { passive: false });
    this._on(viewport, 'mousedown', function(e) { self._onMouseDown(e); });
    this._on(document, 'mousemove', function(e) { self._onMouseMove(e); });
    this._on(document, 'mouseup', function() { self._onMouseUp(); });
    this._on(this.overlay.querySelector('.mermaid-zoom-in'), 'click', function() { self.zoomIn(); });
    this._on(this.overlay.querySelector('.mermaid-zoom-out'), 'click', function() { self.zoomOut(); });
    this._on(this.overlay.querySelector('.mermaid-zoom-reset'), 'click', function() { self.reset(); });
    this._on(this.overlay.querySelector('.mermaid-lightbox-close'), 'click', function() { self.close(); });
    this._on(this.overlay, 'click', function(e) { self._onBackdropClick(e); });
    this._on(document, 'keydown', function(e) { self._onKeyDown(e); });
  };

  Lightbox.prototype.close = function() {
    if (!this.isOpen) return;
    this._clearListeners();
    if (this.overlay && this.overlay.parentNode) {
      this.overlay.parentNode.removeChild(this.overlay);
    }
    document.body.style.overflow = '';
    this.isOpen = false;
    this.overlay = null;
    this.clone = null;
  };

  Lightbox.prototype.zoomIn = function() {
    this.scale = Math.min(this.scale + 0.25, 5);
    this._applyTransform();
  };

  Lightbox.prototype.zoomOut = function() {
    this.scale = Math.max(this.scale - 0.25, 0.5);
    this._applyTransform();
  };

  Lightbox.prototype.reset = function() {
    this.scale = 1;
    this.panX = 0;
    this.panY = 0;
    this._applyTransform();
  };

  function addToolbar(el) {
    if (el.dataset.toolbar) return;
    if (!el.querySelector('svg')) return;
    el.dataset.toolbar = '1';

    var toolbar = document.createElement('div');
    toolbar.className = 'mermaid-toolbar';
    toolbar.innerHTML = '<button class="mermaid-expand-btn" aria-label="Expand diagram" title="Expand">' + icons.expand + '</button>';
    toolbar.querySelector('.mermaid-expand-btn').addEventListener('click', function(e) {
      e.stopPropagation();
      var svg = el.querySelector('svg');
      if (svg) new Lightbox(svg).open();
    });
    el.appendChild(toolbar);
  }

  window.mermaidLightbox = {
    addToolbar: addToolbar,
    Lightbox: Lightbox
  };
})();
