const CACHE = "renvo-web-ide-v61";
const CORE = [
  "./", "./index.html", "./styles.css", "./app.mjs", "./worker.mjs", "./build-readiness.mjs",
  "./target-capabilities.mjs",
  "./editor-navigation.mjs", "./language-path.mjs", "./asset-fetch.mjs", "./serial-plotter.mjs",
  "./esp-webserial.mjs", "./esp-webusb.mjs", "./esp-webusb-jtag.mjs", "./pico-cmsis-dap.mjs", "./pico-webusb-monitor.mjs", "./project-archive.mjs",
  "./device-profile.mjs", "./test-project.mjs", "./workspace-store.mjs",
  "./c-language.mjs", "./rtg-language.mjs",
  "./firmware/renvo-rp2-monitor.uf2",
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
