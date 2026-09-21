// Pharos service worker — shell-only.
// The app is server-backed (SQLite + Go), so "offline" means "server not
// running," not "no network." The SW has three jobs and nothing else:
//
//   1. Navigation fallback: when a navigation can't reach the server,
//      serve the cached stopped page instead of the browser's
//      connection-error page.
//   2. Offline floor for the stopped page's one dependency (presence.js).
//   3. Versioned-cache serving for registered /js/ bundles (immutable
//      content-hash URLs — see internal/web/jsurl.go).
//
// Agent-created lesson assets (/api/lesson-html/*/assets/*, mermaid-theme.js,
// images, …) are deliberately NOT intercepted: they are mutable at a fixed
// URL, so they pass through and let http.ServeFile's Last-Modified /
// If-Modified-Since revalidation govern freshness. Same for CDNs and the API.
//
// CACHE is bumped only when THIS FILE's behavior changes (rare). Asset
// content changes never require a bump — versioned URLs make each build
// self-consistent, so a stale SW still fetches fresh bytes for new URLs.

var CACHE = 'pharos-v10';

// Precache is deliberately tiny: the stopped page (self-contained HTML +
// inline styles) and its only external dependency, presence.js — kept
// UNVERSIONED as an offline floor; versioned requests fall back to it via
// ignoreSearch only when the network is unreachable.
var PRECACHE = [
  '/stopped.html',
  '/js/presence.js'
];

self.addEventListener('install', function(e) {
  e.waitUntil(
    caches.open(CACHE).then(function(cache) {
      return cache.addAll(PRECACHE);
    }).then(function() {
      return self.skipWaiting();
    })
  );
});

self.addEventListener('activate', function(e) {
  e.waitUntil(
    caches.keys().then(function(keys) {
      return Promise.all(
        keys.filter(function(k) { return k !== CACHE; })
            .map(function(k) { return caches.delete(k); })
      );
    }).then(function() {
      return self.clients.claim();
    })
  );
});

self.addEventListener('fetch', function(e) {
  var req = e.request;
  var url = new URL(req.url);

  // Navigations: network-first, stopped-page fallback on failure.
  // Deliberately no identity guard on success: the original
  // `e.target instanceof WindowClient` gate can never match a fetch event
  // (it targets the worker scope) and never fired in production; client
  // lookups can't distinguish fresh lesson iframes from top-level launches
  // (empty clientId) and the resulting client is still pending (a
  // clients.get() on it hangs the navigation). Since sentinel-less 200s
  // include every lesson/reference iframe body, successful responses pass
  // through untouched.
  if (req.mode === 'navigate') {
    e.respondWith(
      fetch(req).then(function(resp) {
        return resp;
      }).catch(function() {
        return caches.match('/stopped.html').then(function(stopped) {
          return stopped || Response.error();
        });
      })
    );
    return;
  }

  // Registered /js/ bundles and /vendor/ library files (immutable
  // content-hash URLs, universe A). Tiers: exact (versioned) cache key →
  // network (fresh; uploads the new key) → unversioned precache floor
  // (offline). ignoreSearch is ONLY the last resort: used first, it would
  // collapse every version into one slot and defeat the versioning scheme.
  if (url.origin === self.location.origin &&
      (url.pathname.indexOf('/js/') === 0 || url.pathname.indexOf('/vendor/') === 0)) {
    e.respondWith(
      caches.match(req).then(function(cached) {
        if (cached) return cached;
        return fetch(req).then(function(resp) {
          if (resp.ok) {
            var clone = resp.clone();
            caches.open(CACHE).then(function(cache) { cache.put(req, clone); });
          }
          return resp;
        }).catch(function() {
          return caches.match(req, { ignoreSearch: true }).then(function(floor) {
            return floor || Response.error();
          });
        });
      })
    );
    return;
  }

  // Google Fonts (cross-origin, immutable CDN URLs): cache-first at runtime.
  if (url.hostname === 'fonts.googleapis.com' || url.hostname === 'fonts.gstatic.com') {
    e.respondWith(
      caches.match(req).then(function(cached) {
        if (cached) return cached;
        return fetch(req).then(function(resp) {
          if (resp.ok) {
            var clone = resp.clone();
            caches.open(CACHE).then(function(cache) { cache.put(req, clone); });
          }
          return resp;
        });
      })
    );
    return;
  }

  // Everything else (API, agent lesson assets, CDN, etc.): pass through.
});