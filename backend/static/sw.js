// Service Worker for Browsedower Push Notifications

self.addEventListener('install', (event) => {
    console.log('Browsedower service worker installed');
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    console.log('Browsedower service worker activated');
    event.waitUntil(clients.claim());
});

self.addEventListener('push', (event) => {
    console.log('Push notification received:', event);
    
    let data = {
        title: 'Browsedower',
        body: 'You have a new notification',
        icon: '/admin/icon-192.png',
        url: '/admin/',
        tag: 'default'
    };
    
    if (event.data) {
        try {
            data = { ...data, ...event.data.json() };
        } catch (e) {
            console.error('Error parsing push data:', e);
        }
    }
    
    const options = {
        body: data.body,
        icon: data.icon || '/admin/icon-192.png',
        badge: '/admin/icon-192.png',
        tag: data.tag || 'default',
        requireInteraction: true,
        data: {
            url: data.url || '/admin/'
        }
    };
    
    event.waitUntil(
        self.registration.showNotification(data.title, options)
    );
});

self.addEventListener('notificationclick', (event) => {
    console.log('Notification clicked:', event);
    event.notification.close();
    
    const url = event.notification.data?.url || '/admin/';
    
    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true })
            .then((clientList) => {
                // Check if there's already a window open
                for (const client of clientList) {
                    if (client.url.includes('/admin') && 'focus' in client) {
                        client.navigate(url);
                        return client.focus();
                    }
                }
                // Open a new window
                if (clients.openWindow) {
                    return clients.openWindow(url);
                }
            })
    );
});

