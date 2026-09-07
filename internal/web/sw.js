// Pharos service worker — caches the app shell + stopped page.
// The app is server-backed (SQLite + Go), so "offline" means "server not
// running," not "no network." The SW's job: when a navigation can't
// reach the server, serve the cached stopped page instead of the
// browser's connection-error page. Successful responses pass through
// untouched (no identity guard — see the navigation handler below).

var CACHE = 'pharos-v8';

var PRECACHE = [
  '/css/app.css',
  '/favicon.svg',
  '/favicon.png',
  '/favicon.ico',
  '/icon-192.png',
  '/icon-512.png',
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

  // Navigations: network-first, stopped-page fallback on failure.
  // There is deliberately no sentinel-based identity guard on success:
  // the original `e.target instanceof WindowClient` gate can never match
  // a fetch event (it targets the worker scope), so the guard never
  // fired in production; and client-tree lookups cannot distinguish a
  // fresh lesson iframe from a top-level launch — both arrive with an
  // empty clientId, and the *resulting* client is still pending (a
  // clients.get() on it hangs the navigation). Since sentinel-less
  // 200s include every lesson/reference iframe body, swapping them to
  // stopped.html breaks lessons. So: successful responses pass through
  // untouched, connection failures serve the cached stopped page.
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

  // Static shell (same-origin CSS/icons): cache-first.
  var url = new URL(req.url);
  if (url.origin === self.location.origin) {
    if (req.destination === 'style' || req.destination === 'image' ||
        req.destination === 'font' || url.pathname === '/css/app.css' ||
        url.pathname === '/js/presence.js') {
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
  }

  // Google Fonts (cross-origin): cache-first at runtime.
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

  // Everything else (API, POST, etc.): pass through.
});
