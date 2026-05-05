# Release Notes

## v0.1.0 (2026-05-03)

First public release of PiHolster — a self-hosted network security appliance for
Raspberry Pi 3.  Designed for non-technical households: one SD card, one power
cable, done.

### What's included

**DNS ad-blocking**
- Recursive DNS resolver with configurable upstream (DNS-over-HTTPS via Cloudflare
  1.1.1.1 by default)
- Block-list loaded from `packages/blocklists/ads.txt` at startup; non-fatal when
  absent (empty blocklist until populated)

**Device discovery and Telegram alerts**
- ARP-based device scanner (`piholster-arpd`) detects new hosts on the LAN
- Telegram notification when an untrusted device joins the network
- Devices can be named and trusted via the web UI

**Web UI**
- SvelteKit static build served directly from the daemon (`go:embed`)
- HTTPS only; strict `Content-Security-Policy: script-src 'self'; style-src 'self'`
  (no `'unsafe-inline'`, no nonces — BB-07)
- HTTP → HTTPS redirect on port 80 (US-24)

**API**
- `GET /api/health` — unauthenticated liveness probe (returns `dns_running`, `uptime_s`)
- `GET /api/status` — authenticated status summary
- `GET /api/devices`, `POST /api/devices/{mac}/trust`, `POST /api/devices/{mac}/rename`
- `POST /api/auth/login`, `POST /api/auth/logout`, `POST /api/auth/change-password`
- Rate limiter on login endpoint (brute-force protection)

**Security hardening**
- All HTTP responses include: `Strict-Transport-Security`, `X-Frame-Options: DENY`,
  `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`,
  `Permissions-Policy`, `Cross-Origin-Embedder-Policy`, `Cross-Origin-Opener-Policy`
- DNS-rebinding protection via `AllowedHosts` middleware (421 for unknown `Host` headers)
- Session tokens stored in SQLite with WAL mode (`PRAGMA journal_mode=WAL`) for
  crash consistency
- Firstboot admin password generated with `crypto/rand` (16 bytes, base64url)

**Deployment**
- Reproducible Raspberry Pi OS image via `image/build.sh`
- MANIFEST file with SHA-256 checksums for all installed components
- systemd services: `piholsterd.service`, `piholster-arpd.service`, `piholster-firstboot.service`
- Firewall: ufw deny-all default, allow 53/udp, 80/tcp, 443/tcp from LAN subnet only

### Known limitations

| ID   | Severity | Description                                                              | Target   |
|------|----------|--------------------------------------------------------------------------|----------|
| M-05 | Medium   | tmpfs admin password file created with mode 0640 instead of 0400         | v0.1.1   |
| L-03 | Low      | `LockPersonality` missing from `piholster-firstboot.service`             | v0.1.1   |
| —    | Info     | No backup function in the web UI — use `scp` (see [BACKUP.md](docs/BACKUP.md)) | v0.1.1 |
| —    | Info     | No TLS certificate auto-renewal — cert is valid for 10 years from firstboot | v1.0   |
| —    | Info     | Wi-Fi not supported — Ethernet connection to router is required           | v1.0     |
| —    | Info     | No one-click update — upgrading requires reflashing the SD card           | v1.0     |

See [SECURITY.md](SECURITY.md) for the security policy and full details on M-05 and L-03.

### Upgrade guide

There is currently no automatic upgrade path.  To upgrade to a future version:

1. Take a backup of `/var/lib/piholster/piholster.db` (see [docs/BACKUP.md](docs/BACKUP.md)).
2. Burn the new image to the SD card.
3. Restore the database backup.

All trusted device names and settings are preserved through a backup/restore cycle.
The admin password hash is also preserved — you do not need to reconfigure from scratch.

### Roadmap

v1.0 is planned approximately 6 months after v0.1.0 GA and is expected to include
one-click in-place updates, scheduled blocklist refresh, and a query history view
in the web UI.  No dates are committed.

### Checksums

SHA-256 checksums for the release image and individual binaries are published in
the `MANIFEST` file bundled with the image archive.  Verify before flashing:

```bash
sha256sum -c piholster-v0.1.0.img.xz.sha256
```
