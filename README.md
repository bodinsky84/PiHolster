# PiHolster

PiHolster is an open source network-level ad and tracker blocker built for Raspberry Pi. It runs a DNS sinkhole on your local network, blocks malicious and tracking domains for every device in your home, monitors connected devices via ARP, and presents a family-friendly web interface — no technical knowledge required.

## How to install

**Recommended — setup script (fastest):**

1. Flash **Raspberry Pi OS Lite** to an SD card with [Raspberry Pi Imager](https://www.raspberrypi.com/software/) (enable SSH in advanced settings).
2. Boot the Pi, find its IP address (e.g. `192.168.1.100`).
3. Run the setup script from this repo:
   ```bash
   bash scripts/setup-pi.sh 192.168.1.100
   ```
4. Open `https://192.168.1.100/` in your browser, accept the self-signed cert, and log in.
5. Point your router's DHCP DNS to the Pi's IP address.

See [docs/INSTALL-PI.md](docs/INSTALL-PI.md) for the full step-by-step guide.

**Alternative — pre-built SD image:**

Download the latest `.img.xz` from the [Releases](https://github.com/bodinsky84/PiHolster/releases) page and burn it to a microSD card.

### Verifying releases

Every official `.img.xz` is signed with `minisign`. Before flashing, verify the
download against the public key shipped in this repo:

```bash
# 1. Verify the SHA-256 checksum:
sha256sum -c piholster-0.1.0-YYYY-MM-DD.img.xz.sha256

# 2. Verify the minisign signature against the public key in docs/minisign.pub:
minisign -Vm piholster-0.1.0-YYYY-MM-DD.img.xz \
  -P "$(tail -1 docs/minisign.pub)"
```

Expected: `Signature and comment signature verified`. If verification fails,
**do not flash the image** — re-download or open an issue.

## Requirements

- Raspberry Pi 3B+, 4, or Zero 2 W
- microSD card, 8 GB minimum, Class 10 or faster
- Ethernet connection to your router (Wi-Fi works but is not recommended for a DNS server)

## Lokal utveckling med Docker

Krav: Docker Desktop 4.x eller senare med Compose v2.

```bash
docker compose up        # bygg images och starta alla tjänster
docker compose down      # stoppa och ta bort containers
docker compose logs -f   # följ loggar i realtid
```

Tjänster som startar:

| Tjänst | Adress | Beskrivning |
|---|---|---|
| Web-UI | http://localhost:8080 | SvelteKit-frontend |
| DNS (UDP) | localhost:5353 | DNS sinkhole |
| DNS (TCP) | localhost:5353 | DNS sinkhole (TCP fallback) |

DNS-tjänsten lyssnar på port **5353** lokalt (inte 53) för att undvika konflikt med
operativsystemets inbyggda DNS-resolver. Det påverkar inte funktionen under utveckling.

### Lokal konfiguration

Kopiera exempelfilen och justera portar eller miljövariabler efter behov:

```bash
cp docker-compose.override.yml.example docker-compose.override.yml
# redigera docker-compose.override.yml
docker compose up
```

Filen `docker-compose.override.yml` är gitignorerad och påverkar bara din lokala miljö.

## Documentation

See the [`docs/`](docs/) directory for:
- [Installation guide — setup script](docs/INSTALL-PI.md)
- [Backup and restore](docs/BACKUP.md)
- Router configuration guide
- Advanced settings and blocklist management
- API reference for integrations
- Architectural decisions (ADRs)

See also:
- [Release notes](RELEASE_NOTES.md) — what's in each version, known limitations, upgrade guide
- [Security policy](SECURITY.md) — vulnerability reporting and known security issues

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/CHANGE-PROCESS.md](docs/CHANGE-PROCESS.md).

## License

SPDX-License-Identifier: GPL-3.0-or-later
