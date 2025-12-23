# Watchtower Web Filter

A web filter approval system consisting of a Go backend API and a Chrome browser extension. The extension blocks URLs that don't match approved patterns and allows users to request access, which administrators can approve or deny through a web interface.

## Features

- **URL Pattern Filtering**: Block or allow URLs based on glob-style patterns
- **Access Request Workflow**: Users can request access to blocked sites
- **Admin Dashboard**: Manage approval requests, patterns, devices, and users
- **Multiple Device Support**: Each browser extension instance registers as a separate device
- **Flexible Expiration**: Approve URLs for specific durations (15 min, 30 min, 1 hour, 8 hours, 24 hours, 1 week, custom, or permanent)
- **Real-time Sync**: Extensions receive pattern updates instantly via WebSocket
- **Push Notifications**: Browser notifications for new requests and device status changes
- **Device Monitoring**: Track device status (active, inactive, uninstalled) with heartbeat detection
- **Pattern Toggle**: Enable/disable patterns without deleting them
- **Mobile-Responsive UI**: Admin dashboard works on desktop and mobile devices
- **Container Support**: Build and deploy as an OCI container

## Architecture

```
┌─────────────────────┐     ┌─────────────────────┐
│  Chrome Extension   │────▶│    Go Backend API   │
│  (Web Filter)       │◀────│    + Admin UI       │
└─────────────────────┘     └──────────┬──────────┘
         ▲                             │
         │ WebSocket                   ▼
         └───────────────── ┌─────────────────────┐
                            │   SQLite Database   │
                            └─────────────────────┘
```

## Quick Start

### Backend

1. Navigate to the backend directory:
   ```bash
   cd backend
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build and run:
   ```bash
   go build -o watchtower .
   ./watchtower
   ```

   The server starts on `http://localhost:8080` by default.

4. Access the admin panel at `http://localhost:8080/admin/`

5. On first launch, you'll be prompted to create an administrator account.

### Container Deployment

Build and run using Podman or Docker:

```bash
cd backend

# Build the container
podman build -t watchtower -f Containerfile .

# Run the container
podman run -d -p 8080:8080 -v watchtower-data:/data watchtower
```

### Chrome Extension

1. Open Chrome and navigate to `chrome://extensions/`

2. Enable "Developer mode" (toggle in top right)

3. Click "Load unpacked" and select the `extension` directory

4. Click the extension icon and configure:
   - API URL: `http://localhost:8080`
   - Device Token: (get from admin panel after creating a device)

## Usage

### Admin Panel

1. **Create a Device**: Go to Devices → Add Device. Copy the generated token.

2. **Configure Extension**: Enter the API URL and token in the extension popup.

3. **Add Patterns**: Manually add allow/deny patterns, or approve access requests.

4. **Review Requests**: When users try to access blocked URLs, requests appear in the queue.

5. **Monitor Devices**: View device status (active/inactive) and last seen timestamps.

### Pattern Syntax

Patterns use glob-style matching:

- `*` matches any characters except `/`
- `**` matches any characters including `/`
- Patterns are matched against `hostname + path + query`

Examples:
- `example.com/*` - Allow all pages on example.com
- `*.google.com/*` - Allow all Google subdomains
- `reddit.com/r/programming/*` - Allow only r/programming

### Filtering Logic

1. URLs matching **deny** patterns are always blocked (listed first in admin panel)
2. If **allow** patterns exist, only matching URLs are permitted
3. If no patterns exist, all URLs are allowed
4. Disabled patterns are ignored during filtering

## API Endpoints

### Extension Endpoints (Token Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/patterns` | Get patterns for device |
| POST | `/api/requests` | Submit access request |
| POST | `/api/heartbeat` | Send device heartbeat |
| GET | `/api/ws` | WebSocket connection for real-time updates |

### Auth Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/setup/status` | Check if first-time setup is needed |
| POST | `/api/setup/create-user` | Create first admin user (setup only) |
| POST | `/api/auth/login` | Admin login |
| POST | `/api/auth/logout` | Admin logout |
| POST | `/api/auth/change-password` | Change current user's password |

### Admin Endpoints (Session Auth)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/admin/requests` | List access requests |
| POST | `/api/admin/requests/:id/approve` | Approve request |
| POST | `/api/admin/requests/:id/deny` | Deny request |
| GET | `/api/admin/patterns` | List all patterns |
| POST | `/api/admin/patterns` | Create pattern |
| PUT | `/api/admin/patterns/:id` | Update pattern |
| DELETE | `/api/admin/patterns/:id` | Delete pattern |
| POST | `/api/admin/patterns/:id/toggle` | Enable/disable pattern |
| GET | `/api/admin/devices` | List devices |
| POST | `/api/admin/devices` | Create device |
| DELETE | `/api/admin/devices/:id` | Delete device |
| POST | `/api/admin/devices/:id/regenerate-token` | Regenerate device token |
| GET | `/api/admin/users` | List users |
| POST | `/api/admin/users` | Create user |
| GET | `/api/admin/push/vapid-key` | Get VAPID public key |
| POST | `/api/admin/push/subscribe` | Subscribe to push notifications |
| POST | `/api/admin/push/unsubscribe` | Unsubscribe from push notifications |
| GET | `/api/admin/notifications/prefs` | Get notification preferences |
| PUT | `/api/admin/notifications/prefs` | Update notification preferences |

## Configuration

### Backend

Environment variables:
- `PORT`: Server port (default: `8080`)
- `DB_PATH`: SQLite database path (default: `./watchtower.db`)

### Extension

Configure via the popup:
- **API URL**: Backend server URL
- **Device Token**: Token from device registration

## Database Schema

The database schema is managed via migrations. Key tables:

```sql
-- Admin users
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    notify_new_requests INTEGER DEFAULT 1,
    notify_device_status INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Registered browser extensions
CREATE TABLE devices (
    id INTEGER PRIMARY KEY,
    token TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    last_seen DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- URL patterns (allow/deny)
CREATE TABLE patterns (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    type TEXT CHECK(type IN ('allow', 'deny')) NOT NULL,
    enabled INTEGER DEFAULT 1,
    expires_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Access requests from users
CREATE TABLE requests (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    suggested_pattern TEXT,
    status TEXT CHECK(status IN ('pending', 'approved', 'denied')) DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME
);

-- Push notification subscriptions
CREATE TABLE push_subscriptions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT UNIQUE NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Security Notes

- First-time setup requires creating an admin account (no default credentials)
- Change your password via the sidebar menu if needed
- Device tokens should be kept secret
- The extension stores the token in local storage
- Session cookies are HTTP-only for admin authentication
- Consider using HTTPS in production
- Push notification VAPID keys are auto-generated on first use

## Development

### Building the Backend

```bash
cd backend
go build -o watchtower .
```

### Running Migrations

Migrations run automatically on startup. The schema is managed in `backend/database/migrations/`.

### Loading the Extension

1. Make changes to extension files
2. Go to `chrome://extensions/`
3. Click the refresh icon on the extension card

## License

MIT

**Made with ❤️ + 🤖 + ☕️**
