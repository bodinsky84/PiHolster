# Installera PiHolster på Raspberry Pi

Steg-för-steg-guide för att installera PiHolster på en Raspberry Pi med
Raspberry Pi OS Lite (utan att behöva bygga en SD-image från scratch).

---

## Krav

| Krav | Detaljer |
|------|----------|
| Hårdvara | Raspberry Pi 3, 4 eller 5 (arm64) **eller** Pi 2 / Zero 2 W (arm-v7) |
| OS på Pi:n | Raspberry Pi OS Lite — Bookworm (64-bit) rekommenderas |
| Nätverksåtkomst | Pi:n och din dator på samma nätverk, SSH aktiverat |
| Lokalt: bash | Git Bash, WSL eller macOS Terminal |
| Lokalt: curl | Används för att ladda ner binärer från GitHub |

---

## Steg 1 — Flasha Raspberry Pi OS Lite

1. Ladda ner **Raspberry Pi Imager**: https://www.raspberrypi.com/software/
2. Välj **Raspberry Pi OS Lite (64-bit)** (Bookworm)
3. Klicka på **kugghjulsikonen (Avancerade inställningar)** och konfigurera:
   - **Hostname**: `piholster` (gör att du kan nå den via `piholster.local`)
   - **SSH**: Aktivera, välj "Använd lösenordsautentisering"
   - **Användarnamn/lösenord**: Välj ett starkt temporärt lösenord (används bara under setup)
   - **WiFi**: Konfigurera om Pi:n inte är ansluten via kabel
4. Flasha till SD-kortet och starta Pi:n

> **Tips:** Använd alltid Ethernet under setup. WiFi-DNS-blockering fungerar men ger
> sämre prestanda och mer komplexitet.

---

## Steg 2 — Hitta Pi:ns IP-adress

Titta i routerns DHCP-lista, eller kör från din dator:

```bash
# macOS / Linux
ping -c 1 piholster.local

# Windows (PowerShell)
ping piholster.local
```

Notera IP-adressen (t.ex. `192.168.1.100`).

---

## Steg 3 — Kör setup-skriptet

Klona repot (eller se till att du är i repots rot):

```bash
git clone https://github.com/piholster/piholster.git
cd piholster
```

Kör installationsskriptet — det tar hand om allt:

```bash
# Med senaste GitHub-release (rekommenderas):
bash scripts/setup-pi.sh 192.168.1.100

# Med specifik release-tag:
bash scripts/setup-pi.sh 192.168.1.100 pi v0.1.0

# Med ett annat SSH-användarnamn:
bash scripts/setup-pi.sh 192.168.1.100 mittnamn
```

Skriptet gör automatiskt:
- Installerar systemberoenden (avahi, iptables, libcap2-bin, openssl)
- Skapar systemanvändare `piholster` (UID 999) och `piholster-arpd` (UID 998)
- Laddar ner och installerar binärer från GitHub Releases
- Sätter Linux capabilities (cap_net_bind_service, cap_net_raw)
- Installerar alla 4 systemd-tjänster
- Kör firstboot-setup (TLS-cert, admin-lösenord, blocklist, iptables)
- Startar tjänsterna

Installationen tar ungefär 2–5 minuter beroende på nätverkshastighet.

---

## Steg 4 — Logga in

När skriptet är klart visas:

```
================================================================
  PiHolster installerat!
================================================================

  Pi IP:         192.168.1.100
  Admin URL:     https://192.168.1.100/
  mDNS URL:      https://piholster.local/

  Admin-lösenord: ABCDEFGHIJK12LMNOPQRSTUV

  OBS: Lösenordet finns bara i RAM (/run/piholster/initial-password).
  Det försvinner vid omstart. Byt lösenord i UI:t direkt.
...
================================================================
```

1. Öppna `https://192.168.1.100/` i webbläsaren
2. Acceptera det självsignerade certifikatet (tryck "Avancerat" → "Fortsätt")
3. Logga in med lösenordet ovan
4. **Byt lösenord direkt** — det försvinner vid omstart av Pi:n

---

## Steg 5 — Peka routern på Pi:n som DNS-server

1. Logga in på din router (vanligtvis `http://192.168.1.1`)
2. Hitta **DHCP-inställningar** → **DNS-server**
3. Sätt primär DNS till Pi:ns IP: `192.168.1.100`
4. Spara och starta om routern (eller vänta tills DHCP förnyas)

Verifiera att DNS-blockering fungerar:

```bash
# Ska returnera NXDOMAIN eller 0.0.0.0 (blockerad)
nslookup doubleclick.net 192.168.1.100

# Ska returnera riktig IP (ej blockerad)
nslookup github.com 192.168.1.100
```

---

## Felsökning

### Tjänster startar inte

```bash
ssh pi@192.168.1.100
sudo systemctl status piholsterd
sudo journalctl -fu piholsterd
sudo journalctl -fu piholster-arpd
sudo journalctl -fu piholster-firstboot
```

### Inga enheter visas i nätverksöversikten

ARP-scannern kräver att Pi:n är på samma L2-segment som enheterna (dvs samma router-port eller VLAN). Kontrollera:

```bash
sudo systemctl status piholster-arpd
sudo journalctl -fu piholster-arpd
```

### Glömt lösenordet

Om Pi:n startades om försvann initial-lösenordet. Återskapa:

```bash
ssh pi@192.168.1.100
sudo /usr/local/bin/piholster-firstboot.sh  # om firstboot-done finns, behöver du ta bort sentineln
# ELLER direkt:
sudo su -s /bin/bash piholster -c \
  "head -c 32 /dev/urandom | base32 | tr -d '=' | head -c 24 > /run/piholster/initial-password"
```

Eller använd "Glömt lösenord"-flödet i UI:t om det implementerats.

### DNS fungerar men blocklistan blockerar rätt sajter

Kontrollera att blocklistan laddades:

```bash
ssh pi@192.168.1.100
wc -l /var/lib/piholster/blocklists/ads.txt
sudo journalctl -u piholsterd | grep -i blocklist
```

---

## Säkerhetskopiera

Se [docs/BACKUP.md](BACKUP.md) för instruktioner om att säkerhetskopiera databasen.

---

## Uppdatera PiHolster

```bash
# Ladda ner nya binärer och starta om
bash scripts/setup-pi.sh 192.168.1.100 pi v0.2.0
```

Skriptet är idempotent — det kan köras om utan att konfigurationen går förlorad.
Databasen och TLS-certifikatet bevaras i `/var/lib/piholster/`.
