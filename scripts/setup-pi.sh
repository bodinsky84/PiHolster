#!/usr/bin/env bash
# scripts/setup-pi.sh — Installerar PiHolster på en körande Raspberry Pi OS Lite.
#
# Användning:
#   bash scripts/setup-pi.sh <PI_IP> [SSH_USER] [RELEASE_TAG]
#
# Exempel:
#   bash scripts/setup-pi.sh 192.168.1.100
#   bash scripts/setup-pi.sh 192.168.1.100 pi v0.1.0
#
# Krav på Pi:
#   - Raspberry Pi OS Lite (Bookworm 64-bit eller Bullseye 32-bit)
#   - SSH aktiverat (aktivera via rpi-imager eller: sudo touch /boot/ssh)
#   - sudo-åtkomst för SSH-usern (default: pi)
#
# Skriptet gör INTE:
#   - Flash SD-kortet (gör det med rpi-imager)
#   - Ändra DNS-inställningar på routern (gör det manuellt efteråt)

set -euo pipefail

# ---------------------------------------------------------------------------
# Argument och defaults
# ---------------------------------------------------------------------------

PI_IP="${1:-}"
SSH_USER="${2:-pi}"
RELEASE_TAG="${3:-}"

if [[ -z "${PI_IP}" ]]; then
    echo "Fel: ange Pi:ns IP-adress som första argument." >&2
    echo "Användning: bash scripts/setup-pi.sh <PI_IP> [SSH_USER] [RELEASE_TAG]" >&2
    exit 1
fi

# Autodetektera arkitektur — Raspberry Pi 3/4/5 kör arm64; Pi 2/Zero W kör arm-v7
# Skriptet frågar Pi:n direkt via SSH.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

log()  { echo "[setup-pi] $*"; }
die()  { echo "[setup-pi] FEL: $*" >&2; exit 1; }
step() { echo ""; echo "=== $* ==="; }

SSH_OPTS="-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10"
SSH="ssh ${SSH_OPTS} ${SSH_USER}@${PI_IP}"
SCP="scp ${SSH_OPTS}"

# ---------------------------------------------------------------------------
# Steg 0: Kontrollera SSH-anslutning
# ---------------------------------------------------------------------------

step "Kontrollerar SSH-anslutning till ${SSH_USER}@${PI_IP}"

if ! ${SSH} "echo 'SSH OK'" 2>/dev/null; then
    die "Kan inte ansluta via SSH till ${SSH_USER}@${PI_IP}.
Kontrollera:
  1. Pi:n är påslagen och på nätverket
  2. SSH är aktiverat (rpi-imager → Avancerade inställningar → SSH)
  3. Du kan nå Pi:n: ping ${PI_IP}"
fi
log "SSH-anslutning OK."

# ---------------------------------------------------------------------------
# Steg 1: Detektera arkitektur
# ---------------------------------------------------------------------------

step "Detekterar arkitektur"

ARCH=$(${SSH} "uname -m")
log "Rapporterad arkitektur: ${ARCH}"

case "${ARCH}" in
    aarch64)
        BINARY_SUFFIX="linux-arm64"
        ;;
    armv7l|armv6l)
        BINARY_SUFFIX="linux-arm-v7"
        ;;
    x86_64)
        # Tillåt installation på en vanlig Linux-dator för testning
        BINARY_SUFFIX="linux-amd64"
        log "VARNING: x86_64 är inte en Pi — installerar ändå för testning."
        ;;
    *)
        die "Okänd arkitektur: ${ARCH}. Stöder: aarch64, armv7l, x86_64."
        ;;
esac

log "Använder binärsuffix: ${BINARY_SUFFIX}"

# ---------------------------------------------------------------------------
# Steg 2: Hämta eller bygg binärer
# ---------------------------------------------------------------------------

step "Hämtar binärer"

TMPDIR_LOCAL="$(mktemp -d)"
trap 'rm -rf "${TMPDIR_LOCAL}"' EXIT

PIHOLSTERD_BIN="${TMPDIR_LOCAL}/piholsterd"
ARPD_BIN="${TMPDIR_LOCAL}/piholster-arpd"

if [[ -n "${RELEASE_TAG}" ]]; then
    # Ladda ner från GitHub Releases
    GH_BASE="https://github.com/piholster/piholster/releases/download/${RELEASE_TAG}"
    log "Laddar ner från GitHub Releases: ${RELEASE_TAG}"

    if command -v curl &>/dev/null; then
        curl -fsSL "${GH_BASE}/piholsterd-${BINARY_SUFFIX}" -o "${PIHOLSTERD_BIN}"
        curl -fsSL "${GH_BASE}/piholster-arpd-${BINARY_SUFFIX}" -o "${ARPD_BIN}"
    elif command -v wget &>/dev/null; then
        wget -q "${GH_BASE}/piholsterd-${BINARY_SUFFIX}" -O "${PIHOLSTERD_BIN}"
        wget -q "${GH_BASE}/piholster-arpd-${BINARY_SUFFIX}" -O "${ARPD_BIN}"
    else
        die "Varken curl eller wget finns tillgängligt. Installera curl och försök igen."
    fi
    log "Nedladdning klar."
else
    # Försök hitta senaste GitHub-release automatiskt
    log "Inget release-tag angivet — försöker hämta senaste release från GitHub."

    if command -v curl &>/dev/null; then
        LATEST_TAG=$(curl -fsSL \
            "https://api.github.com/repos/piholster/piholster/releases/latest" \
            | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
    else
        LATEST_TAG=""
    fi

    if [[ -n "${LATEST_TAG}" ]]; then
        log "Senaste release: ${LATEST_TAG}"
        GH_BASE="https://github.com/piholster/piholster/releases/download/${LATEST_TAG}"
        curl -fsSL "${GH_BASE}/piholsterd-${BINARY_SUFFIX}" -o "${PIHOLSTERD_BIN}" || true
        curl -fsSL "${GH_BASE}/piholster-arpd-${BINARY_SUFFIX}" -o "${ARPD_BIN}" || true
    fi

    # Fallback: bygg lokalt om binärerna inte hittades
    if [[ ! -s "${PIHOLSTERD_BIN}" ]]; then
        log "Ingen GitHub-release hittades — försöker bygga lokalt med Go."

        if ! command -v go &>/dev/null; then
            die "Go är inte installerat och ingen GitHub-release hittades.
Alternativ:
  1. Tagga och pusha en release: git tag v0.1.0 && git push origin v0.1.0
     (GitHub Actions bygger binärerna, sedan körs detta skript igen med release-tag:en)
  2. Installera Go lokalt och kör skriptet igen.
  3. Kör: bash scripts/setup-pi.sh ${PI_IP} ${SSH_USER} <release-tag>"
        fi

        # Bygg frontend om dist/ saknas innehåll
        DIST_DIR="${REPO_ROOT}/apps/piholsterd/internal/api/dist"
        if [[ ! -f "${DIST_DIR}/index.html" ]]; then
            log "Bygger frontend (pnpm build) ..."
            if ! command -v pnpm &>/dev/null; then
                die "pnpm är inte installerat. Kör: npm install -g pnpm"
            fi
            (cd "${REPO_ROOT}" && pnpm --filter web build)
        fi

        log "Kompilerar piholsterd för ${BINARY_SUFFIX} ..."
        (cd "${REPO_ROOT}" && \
            CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH%l}" GOARM=7 \
            go build -trimpath -ldflags="-s -w" \
            -o "${PIHOLSTERD_BIN}" \
            ./apps/piholsterd/cmd/piholsterd) || \
        (cd "${REPO_ROOT}" && \
            CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" \
            go build -trimpath -ldflags="-s -w" \
            -o "${PIHOLSTERD_BIN}" \
            ./apps/piholsterd/cmd/piholsterd)

        log "Kompilerar piholster-arpd för ${BINARY_SUFFIX} ..."
        (cd "${REPO_ROOT}" && \
            CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH%l}" GOARM=7 \
            go build -trimpath -ldflags="-s -w" \
            -o "${ARPD_BIN}" \
            ./apps/piholster-arpd/cmd/piholster-arpd) || \
        (cd "${REPO_ROOT}" && \
            CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" \
            go build -trimpath -ldflags="-s -w" \
            -o "${ARPD_BIN}" \
            ./apps/piholster-arpd/cmd/piholster-arpd)

        log "Binärer byggda lokalt."
    fi
fi

# Verifiera att binärerna faktiskt finns och inte är tomma
[[ -s "${PIHOLSTERD_BIN}" ]] || die "piholsterd-binären saknas eller är tom."
[[ -s "${ARPD_BIN}" ]]       || die "piholster-arpd-binären saknas eller är tom."
log "Binärer OK: $(du -sh "${PIHOLSTERD_BIN}" | cut -f1) och $(du -sh "${ARPD_BIN}" | cut -f1)"

# ---------------------------------------------------------------------------
# Steg 3: Kopiera filer till Pi
# ---------------------------------------------------------------------------

step "Kopierar filer till Pi"

BLOCKLIST_SRC="${REPO_ROOT}/packages/blocklists/ads.txt"
FIRSTBOOT_SRC="${REPO_ROOT}/image/stage-piholster/01-firstboot/piholster-firstboot.sh"

[[ -f "${BLOCKLIST_SRC}" ]] || die "Blocklist saknas: ${BLOCKLIST_SRC}"
[[ -f "${FIRSTBOOT_SRC}" ]] || die "Firstboot-skript saknas: ${FIRSTBOOT_SRC}"

${SCP} "${PIHOLSTERD_BIN}"  "${SSH_USER}@${PI_IP}:/tmp/piholsterd"
${SCP} "${ARPD_BIN}"        "${SSH_USER}@${PI_IP}:/tmp/piholster-arpd"
${SCP} "${BLOCKLIST_SRC}"   "${SSH_USER}@${PI_IP}:/tmp/ads.txt"
${SCP} "${FIRSTBOOT_SRC}"   "${SSH_USER}@${PI_IP}:/tmp/piholster-firstboot.sh"

log "Filer kopierade till Pi."

# ---------------------------------------------------------------------------
# Steg 4: Installera på Pi (körs som root via sudo)
# ---------------------------------------------------------------------------

step "Installerar PiHolster på Pi"

${SSH} "sudo bash -s" <<'REMOTE'
set -euo pipefail

log()  { echo "[pi-install] $*"; }
die()  { echo "[pi-install] FEL: $*" >&2; exit 1; }

# 4a. Installera systemberoenden
log "Installerar systemberoenden ..."
apt-get update -q
apt-get install -y --no-install-recommends \
    avahi-daemon \
    iptables \
    iptables-persistent \
    netfilter-persistent \
    ca-certificates \
    libcap2-bin \
    openssl \
    unattended-upgrades \
    apt-listchanges
apt-get clean
log "Systemberoenden installerade."

# 4b. Skapa system-users (idempotent: id returnerar 0 om user finns)
log "Skapar system-users ..."
if ! id piholster-arpd &>/dev/null; then
    useradd --system --uid 998 --no-create-home --shell /usr/sbin/nologin \
        --comment "PiHolster ARP scanner daemon" piholster-arpd
    log "  piholster-arpd (UID 998) skapad."
else
    log "  piholster-arpd finns redan."
fi
if ! id piholster &>/dev/null; then
    useradd --system --uid 999 --no-create-home --shell /usr/sbin/nologin \
        --comment "PiHolster network security daemon" piholster
    log "  piholster (UID 999) skapad."
else
    log "  piholster finns redan."
fi

# 4c. Skapa kataloger
log "Skapar kataloger ..."
install -d -m 750 -o piholster      -g piholster      /var/lib/piholster
install -d -m 755                                      /usr/share/piholster
# /run/piholster skapas av systemd via RuntimeDirectory= vid boot;
# vi skapar den nu så att firstboot-scriptet kan skriva dit direkt.
install -d -m 750 -o piholster-arpd -g piholster-arpd /run/piholster

# 4d. Installera binärer
log "Installerar binärer ..."
install -m 755 /tmp/piholsterd       /usr/local/bin/piholsterd
install -m 755 /tmp/piholster-arpd   /usr/local/bin/piholster-arpd
install -m 755 /tmp/piholster-firstboot.sh /usr/local/bin/piholster-firstboot.sh
rm -f /tmp/piholsterd /tmp/piholster-arpd /tmp/piholster-firstboot.sh
log "Binärer installerade i /usr/local/bin/."

# 4e. Sätt Linux capabilities
log "Sätter capabilities ..."
setcap cap_net_bind_service=+ep /usr/local/bin/piholsterd
setcap cap_net_raw=+ep          /usr/local/bin/piholster-arpd
# Verifiera
getcap /usr/local/bin/piholsterd
getcap /usr/local/bin/piholster-arpd
if getcap /usr/local/bin/piholsterd | grep -q cap_net_raw; then
    die "Säkerhetsbrott: piholsterd har cap_net_raw — bör inte förekomma."
fi
log "Capabilities OK."

# 4f. Installera blocklist
log "Installerar blocklist ..."
install -m 644 /tmp/ads.txt /usr/share/piholster/ads.txt
rm -f /tmp/ads.txt
log "Blocklist: $(wc -l < /usr/share/piholster/ads.txt) rader."

# 4g. Installera systemd-tjänster
log "Installerar systemd-tjänster ..."

cat > /etc/systemd/system/piholster-firstboot.service <<'UNIT'
[Unit]
Description=PiHolster First Boot Setup
Documentation=https://github.com/piholster/piholster
After=systemd-networkd.service time-sync.target
Before=piholster-arpd.service piholsterd.service piholster-avahi-publish.service
ConditionPathExists=!/var/lib/piholster/.firstboot-done

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/piholster-firstboot.sh
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/systemd/system/piholster-arpd.service <<'UNIT'
[Unit]
Description=PiHolster ARP Scanner
Documentation=https://github.com/piholster/piholster
After=network-online.target piholster-firstboot.service
Requires=piholster-firstboot.service
Before=piholsterd.service

[Service]
Type=simple
ExecStart=/usr/local/bin/piholster-arpd --socket=/run/piholster/arp.sock
User=piholster-arpd
Group=piholster-arpd
RuntimeDirectory=piholster
RuntimeDirectoryMode=0750
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=false
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_PACKET AF_UNIX AF_NETLINK
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/systemd/system/piholsterd.service <<'UNIT'
[Unit]
Description=PiHolster Network Security Daemon
Documentation=https://github.com/piholster/piholster
After=network-online.target time-sync.target piholster-firstboot.service piholster-arpd.service
Requires=piholster-firstboot.service piholster-arpd.service

[Service]
Type=simple
ExecStart=/usr/local/bin/piholsterd
User=piholster
Group=piholster
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
Environment=DNS_PORT=53
Environment=HTTP_PORT=80
Environment=HTTPS_PORT=443
Environment=DB_PATH=/var/lib/piholster/piholster.db
Environment=BLOCKLIST_PATH=/var/lib/piholster/blocklists/ads.txt
Environment=TLS_CERT=/var/lib/piholster/tls.crt
Environment=TLS_KEY=/var/lib/piholster/tls.key
Environment=ARP_SOCK=/run/piholster/arp.sock
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
ReadWritePaths=/var/lib/piholster /run/piholster
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
UNIT

cat > /etc/systemd/system/piholster-avahi-publish.service <<'UNIT'
[Unit]
Description=PiHolster mDNS Service Publishing
Documentation=https://github.com/piholster/piholster
After=piholsterd.service avahi-daemon.service
Requires=piholsterd.service avahi-daemon.service

[Service]
Type=simple
ExecStart=/usr/bin/avahi-publish -s piholster _https._tcp 443 "PiHolster Admin UI"
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable piholster-firstboot.service
systemctl enable piholster-arpd.service
systemctl enable piholsterd.service
systemctl enable piholster-avahi-publish.service
log "Systemd-tjänster aktiverade."

# 4h. Konfigurera Avahi (minimera information-leakage)
log "Konfigurerar Avahi ..."
AVAHI_CONF=/etc/avahi/avahi-daemon.conf
if [[ -f "${AVAHI_CONF}" ]]; then
    # Inaktivera HINFO, workstation och AAAA-on-IPv4 (läcker OS-info)
    sed -i 's/^#\?publish-hinfo=.*/publish-hinfo=no/'           "${AVAHI_CONF}"
    sed -i 's/^#\?publish-workstation=.*/publish-workstation=no/' "${AVAHI_CONF}"
    sed -i 's/^#\?publish-aaaa-on-ipv4=.*/publish-aaaa-on-ipv4=no/' "${AVAHI_CONF}"
fi
log "Avahi konfigurerad."

log "Installation klar."
REMOTE

# ---------------------------------------------------------------------------
# Steg 5: Kör firstboot-skriptet direkt (behöver inte vänta på reboot)
# ---------------------------------------------------------------------------

step "Kör firstboot-setup"

${SSH} "sudo /usr/local/bin/piholster-firstboot.sh"

# ---------------------------------------------------------------------------
# Steg 6: Starta tjänsterna
# ---------------------------------------------------------------------------

step "Startar PiHolster-tjänster"

${SSH} "sudo systemctl start piholster-arpd.service && sudo systemctl start piholsterd.service"

log "Väntar 3 sekunder på att tjänsterna ska starta ..."
sleep 3

# Visa status
${SSH} "sudo systemctl is-active piholster-arpd.service piholsterd.service || true"

# ---------------------------------------------------------------------------
# Steg 7: Hämta initial-lösenord
# ---------------------------------------------------------------------------

step "Hämtar admin-lösenord"

INITIAL_PASSWORD=$(${SSH} "sudo cat /run/piholster/initial-password 2>/dev/null || echo '(lösenord läst av piholsterd — logga in med journalctl)'")

# ---------------------------------------------------------------------------
# Klart — visa sammanfattning
# ---------------------------------------------------------------------------

echo ""
echo "================================================================"
echo "  PiHolster installerat!"
echo "================================================================"
echo ""
echo "  Pi IP:         ${PI_IP}"
echo "  Admin URL:     https://${PI_IP}/"
echo "  mDNS URL:      https://piholster.local/"
echo ""
echo "  Admin-lösenord: ${INITIAL_PASSWORD}"
echo ""
echo "  OBS: Lösenordet finns bara i RAM (/run/piholster/initial-password)."
echo "  Det försvinner vid omstart. Byt lösenord i UI:t direkt."
echo ""
echo "  Nästa steg:"
echo "  1. Öppna https://${PI_IP}/ i webbläsaren (acceptera självsignerat cert)"
echo "  2. Logga in med lösenordet ovan"
echo "  3. Peka routerns DHCP-DNS på ${PI_IP}"
echo "  4. Verifiera DNS-blockering: nslookup doubleclick.net ${PI_IP}"
echo ""
echo "  Loggar:"
echo "    sudo journalctl -fu piholsterd"
echo "    sudo journalctl -fu piholster-arpd"
echo "================================================================"
