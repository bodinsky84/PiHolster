# PiHolster

PiHolster is an open source network-level ad and tracker blocker built for Raspberry Pi. It runs a DNS sinkhole on your local network, blocks malicious and tracking domains for every device in your home, monitors connected devices via ARP, and presents a family-friendly web interface — no technical knowledge required.

## How to install

1. Download the latest `.img.xz` image from the [Releases](https://github.com/piholster/piholster/releases) page.
2. Burn the image to a microSD card (8 GB or larger) using [Raspberry Pi Imager](https://www.raspberrypi.com/software/) or `dd`:
   ```
   xz -dc piholster-vX.Y.Z.img.xz | sudo dd of=/dev/sdX bs=4M status=progress
   ```
3. Insert the SD card into your Raspberry Pi (3B+, 4, or Zero 2 W recommended).
4. Connect the Pi to your router via Ethernet and power it on.
5. Open `http://piholster.local` in any browser on your network.
6. Point your router's DNS to the Pi's IP address to protect all devices automatically.

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
- Router configuration guide
- Advanced settings and blocklist management
- API reference for integrations
- Architectural decisions (ADRs)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and [docs/CHANGE-PROCESS.md](docs/CHANGE-PROCESS.md).

## License

SPDX-License-Identifier: GPL-3.0-or-later
