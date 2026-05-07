# PiHolster — Roadmap

## Vision

PiHolster är ett nätverkssäkerhetsverktyg för hemmet som körs på Raspberry Pi 3 eller senare.
Målet är att ge alla hushåll — oavsett teknisk nivå — synlighet och kontroll över sitt hemnätverk:
vad som blockeras, vilka enheter som finns och när något okänt ansluter.

Designprincipen är "mormor ska klara det": installation under 10 minuter, UI utan jargong,
vettiga standardinställningar direkt ur lådan.

---

## MVP (v0.1) — Grundläggande skydd fungerar

Mål: En fungerande DNS-blockerare med nätverksöversikt och Telegram-notiser.
Allt ska gå att installera från en anpassad Raspberry Pi OS-image.

### Inkluderat i MVP

- DNS-server (dnsmasq eller Unbound) med blocklist för annonser, trackers och känd malware
- ARP-nätverksskanning — listar alla enheter på LAN med IP, MAC och (om möjligt) hostname
- Varning via Telegram-bot när en okänd enhet ansluter till nätverket
- Web-UI med tre lägen:
  - Mormor-läge: stora knappar, trafikljusstatus, inga tekniska termer
  - Avancerat läge: statistik, blocklist-hantering, enhetslista med detaljer
  - Admin-läge: lösenordsskyddat, nätverkskonfiguration, Telegram-inställningar
- DNS-over-HTTPS (DoH) som standard, DNS-over-TLS (DoT) som alternativ
- Anpassad Raspberry Pi OS-image med PiHolster förinstallerat och autostart

### Ej inkluderat i MVP

- VPN-funktionalitet
- Flera blocklist-profiler per enhet
- Mobilapp (web-UI fungerar i mobil-browser)
- Cloud-synk eller remote-management

---

## v0.1.1 — Patch-release efter GA

Mål: Adressera de kända begränsningar som dokumenterades i v0.1.0 GA-noterna,
samt återaktivera image-build-pipelinen i CI.

- M-05 fix: `/etc/piholster/initial-password` skrivs med 0400 (inte 0640)
- L-03 fix: `LockPersonality=true` läggs till i `firstboot.service`
- Cert-renewal-flöde: web-UI eller `piholsterctl regen-cert`-CLI för att
  rotera self-signed-certet innan 10-årsgränsen (eller vid komprometterad nyckel)
- Backup/restore via web-UI (idag endast manuellt via `docs/BACKUP.md`)
- Image-CI-pipeline: re-aktivera `release-image.yml` med:
  - Self-hosted ARM64-runner registrerad mot repot
  - `MINISIGN_SECRET_KEY`-secret provisionerat i GitHub Secrets
  - Hämta binärerna från `release-binary.yml`-jobbet via `actions/download-artifact`
  - Smoke-svit som kan köras headless i runner (mock-Pi eller emulerad nätverksstack)
- One-click update i web-UI (kräver signerade update-paket — använder samma minisign-nyckel)
- **Nördläge-dashboard** (`/nerd`-route): live SSE-flöde, regex-filter, latency-percentiler
  (p50/p95/p99/max), top-N blockerade domäner och klienter, Go-runtime-kort
  (uptime, goroutines, heap, GC). `/advanced` får 1h-sparkline och median-latency.
  Implementation klar på `feat/dashboard-nerd-mode`.

---

## v1.0 — Polerat och stabil release

Mål: Redo för bredare publik. Fokus på stabilitet, dokumentation och UX.

- Uppdateringshantering via web-UI (one-click update)
- Schemalagd blocklist-uppdatering med konfigurerbar frekvens
- Per-enhet DNS-regler (undantag och extra blockering per MAC/IP)
- Historikvy: senaste 24h/7d med sökbar logg
- Exportera enhetslista till CSV
- Komplett installationsdokumentation och troubleshooting-guide
- Automatiserade integrationstester i CI
- Säkerhetsaudit genomförd (IT-säkerhet signerar av)

---

## v2.0 — Utökat skydd och ekosystem

Mål: PiHolster som plattform, inte bara ett verktyg.

- Intrusion Detection System (IDS) med regelbaserad trafik-analys (Suricata)
- Bandbreddsöversikt per enhet
- Stöd för flera blocklist-profiler (t.ex. "Barn", "Arbete", "Standard")
- Schemalagda regler: t.ex. blockera sociala medier 22:00–07:00
- Plugin-API så att tredjepartsutvecklare kan bygga tillägg
- Valfri molnbackup av konfiguration (opt-in, end-to-end krypterad)
- Stöd för Raspberry Pi 5 och ARM64-arkitektur

---

## Versionsstrategi

| Version | Status      | Beräknad release  |
|---------|-------------|-------------------|
| v0.1.0  | Pågår (GA-gate) | Sprint 4      |
| v0.1.1  | Planerad    | ca 4–6 veckor efter v0.1.0 |
| v1.0    | Planerad    | ca 6 månader      |
| v2.0    | Vision      | ca 12–18 månader  |

Releasekandidater taggas som `vX.Y.0-rc1` och genomgår säkerhetsgranskning
innan de mergas till `main` och publiceras.
