// Pharos service worker — caches the app shell + stopped page.
// The app is server-backed (SQLite + Go), so "offline" means "server not
// running," not "no network." The SW guards origin identity: if a foreign
// app occupies the port, the sentinel check falls back to the cached
// stopped page instead of rendering the foreign response.

var CACHE = 'pharos-v3';

var PRECACHE = [
  '/css/app.css',
  '/favicon.svg',
  '/favicon.png',
  '/favicon.ico',
  '/icon-192.png',
  '/icon-512.png',
  '/stopped.html'
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

var SENTINEL = '<meta name="pharos-app" content="1">';

self.addEventListener('fetch', function(e) {
  var req = e.request;

  // Navigations: network-first with identity guard.
  // Only applies to top-level document navigations — iframe loads
  // (also mode: navigate) are passed through, since their content
  // (lesson/reference HTML) doesn't carry the sentinel and shouldn't
  // be replaced with the stopped page.
  if (req.mode === 'navigate' && e.target instanceof WindowClient) {
    e.respondWith(
      fetch(req).then(function(resp) {
        var clone = resp.clone();
        return clone.text().then(function(body) {
          if (body.indexOf(SENTINEL) !== -1) {
            return resp;
          }
          return caches.match('/stopped.html').then(function(stopped) {
            return stopped || resp;
          });
        });
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
        req.destination === 'font' || url.pathname === '/css/app.css') {
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
