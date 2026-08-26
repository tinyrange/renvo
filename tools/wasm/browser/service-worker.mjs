const CACHE = "renvo-web-ide-v34";
const CORE = [
  "./", "./index.html", "./styles.css", "./app.mjs", "./worker.mjs",
  "./editor-navigation.mjs", "./language-path.mjs", "./asset-fetch.mjs", "./serial-plotter.mjs",
  "./esp-webserial.mjs", "./esp-webusb.mjs", "./esp-webusb-jtag.mjs", "./project-archive.mjs",
  "./device-profile.mjs", "./test-project.mjs", "./workspace-store.mjs",
  "./c-language.mjs", "./rtg-language.mjs",
];

self.addEventListener("install", (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(CORE)).then(() => self.skipWaiting()));
});

self.addEventListener("activate", (event) => {
  event.waitUntil(caches.keys().then((keys) => Promise.all(keys.filter((key) => key !== CACHE && key.startsWith("renvo-web-ide-")).map((key) => caches.delete(key)))).then(() => self.clients.claim()));
});

self.addEventListener("fetch", (event) => {
  if (event.request.method !== "GET") return;
  const url = new URL(event.request.url);
  if (url.origin !== location.origin && !url.hostname.endsWith("jsdelivr.net")) return;
  const remember = (response) => {
    if (response.ok || response.type === "opaque") caches.open(CACHE).then((cache) => cache.put(event.request, response.clone()));
    return response;
  };
  if (url.origin === location.origin) {
    if (url.searchParams.has("v")) {
      event.respondWith(caches.match(event.request).then((cached) => cached || fetch(event.request).then(remember)));
      return;
    }
    event.respondWith(fetch(event.request).then(remember).catch(() => caches.match(event.request).then((cached) => cached || Response.error())));
    return;
  }
  event.respondWith(caches.match(event.request).then((cached) => cached || fetch(event.request).then(remember)));
});
