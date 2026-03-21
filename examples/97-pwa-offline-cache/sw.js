const OFFLINE_URL = '/97-pwa-offline-cache/offline.html';
const REPLAY_SYNC_TAG = 'gwc-offline-demo-replay';

async function notifyClients(message) {
  const clients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
  await Promise.all(clients.map(client => client.postMessage(message)));
}

self.addEventListener('install', event => {
  self.skipWaiting();
  event.waitUntil(Promise.resolve());
});

self.addEventListener('activate', event => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('sync', event => {
  if (event.tag !== REPLAY_SYNC_TAG) {
    return;
  }
  event.waitUntil(notifyClients({ type: 'OFFLINE_REPLAY_SYNC', tag: event.tag }));
});

self.addEventListener('fetch', event => {
  if (event.request.method !== 'GET') {
    return;
  }
  event.respondWith((async () => {
    const cached = await caches.match(event.request);
    if (cached) {
      return cached;
    }
    try {
      return await fetch(event.request);
    } catch (error) {
      if (event.request.mode === 'navigate') {
        const offline = await caches.match(OFFLINE_URL);
        if (offline) {
          return offline;
        }
      }
      throw error;
    }
  })());
});