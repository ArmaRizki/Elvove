// The admin panel's service worker. It exists for one reason: browsers only
// offer to install a site whose service worker answers fetches. It stores
// nothing on purpose — booking data is private and changes on the server, so a
// cached copy would be stale at best and left behind on a shared device at
// worst. Every request passes straight through to the network.
self.addEventListener("install", (event) => event.waitUntil(self.skipWaiting()));
self.addEventListener("activate", (event) => event.waitUntil(self.clients.claim()));
self.addEventListener("fetch", () => {});
