#!/bin/bash -e
# stage-piholster/00-install/01-run.sh
#
# Sub-stage 00-install, steg 2: Installera och aktivera systemd-enheter.
#
# Skapar tre service-filer och aktiverar dem:
#   piholster-firstboot.service  — oneshot, kör en gång vid första boot
#   piholster-arpd.service       — ARP-scanner, separat process (ADR-002 §3.1)
#   piholsterd.service           — huvuddaemon
#
# Ordering (ADR-002 §3.2.1):
#   firstboot -> arpd -> piholsterd
#   firstboot kör INNAN arpd och piholsterd startar.
#   piholsterd Requires=firstboot, dvs om firstboot misslyckas startar inte HTTP.

echo "[00-install/01-run.sh] Installerar systemd-enheter ..."

# ---------------------------------------------------------------------------
# 1. piholster-firstboot.service
#
#    Type=oneshot RemainAfterExit=yes: systemd anser tjänsten som "active"
#    efter att ExecStart-kommandot returnerat 0, vilket gör att
#    After=/Requires=-kedjan fungerar korrekt.
#
#    ConditionPathExists=!/var/lib/piholster/.firstboot-done:
#    kör bara om sentinelfilen SAKNAS — dvs vid första boot.
#    Vid efterföljande boots hoppas firstboot över direkt.
# ---------------------------------------------------------------------------

cat > /etc/systemd/system/piholster-firstboot.service <<'UNIT'
[Unit]
Description=PiHolster First Boot Setup
Documentation=https://github.com/piholster/piholster
# Nätverket behöver inte vara online för att generera cert och lösenord,
# men time-sync garanterar att TLS-certifikatets NotBefore-datum är korrekt.
After=systemd-networkd.service time-sync.target
Before=piholster-arpd.service piholsterd.service piholster-avahi-publish.service
# Kör bara om sentinel-filen saknas (= första boot)
ConditionPathExists=!/var/lib/piholster/.firstboot-done

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=/usr/local/bin/piholster-firstboot.sh
# Logga till journal så att man kan följa processen med journalctl -fu piholster-firstboot
StandardOutput=journal
StandardError=journal
# Scriptet körs som root men behöver inte eskalera till annan UID via SUID/capabilities
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
UNIT

# ---------------------------------------------------------------------------
# 2. piholster-arpd.service
#
#    Separat process för ARP-scanning (ADR-002 §3.1.1).
#    Kör som piholster-arpd (UID 998), har CAP_NET_RAW via setcap.
#
#    Type=notify: piholster-arpd skickar "READY=1" via sd_notify när
#    Unix socket är öppen — piholsterd.service väntar på det.
#
#    RuntimeDirectory=piholster: systemd skapar /run/piholster/ med
#    mode 0750 innan ExecStart, ägt av User/Group i denna unit.
#    Katalogen försvinner vid shutdown (tmpfs) och återskapas vid boot.
#
#    AmbientCapabilities vs setcap:
#    Vi använder setcap (i 00-run.sh) OCH AmbientCapabilities som
#    redundant skydd. AmbientCapabilities är det systemd-rekommenderade
#    sättet; setcap är en extra guard om unit-filen av misstag ändras.
# ---------------------------------------------------------------------------

cat > /etc/systemd/system/piholster-arpd.service <<'UNIT'
[Unit]
Description=PiHolster ARP Scanner
Documentation=https://github.com/piholster/piholster
After=network-online.target piholster-firstboot.service
Requires=piholster-firstboot.service
Before=piholsterd.service

[Service]
Type=notify
ExecStart=/usr/local/bin/piholster-arpd --socket=/run/piholster/arp.sock
User=piholster-arpd
Group=piholster-arpd
# Runtime-katalog för Unix socket: /run/piholster/ skapas av systemd
RuntimeDirectory=piholster
RuntimeDirectoryMode=0750
# Capabilities: CAP_NET_RAW för AF_PACKET (ARP-sniffning)
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW
NoNewPrivileges=true
# Filsystem-isolation
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
# AF_PACKET kräver access till nätverksenheter — PrivateDevices=false
PrivateDevices=false
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
# Tillåt bara nödvändiga adressfamiljer
RestrictAddressFamilies=AF_PACKET AF_UNIX AF_NETLINK
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
# Restart-policy
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
UNIT

# ---------------------------------------------------------------------------
# 3. piholsterd.service
#
#    Huvuddaemon. Kör som piholster (UID 999).
#    Binder till :53, :80, :443 via CAP_NET_BIND_SERVICE.
#
#    Requires=piholster-firstboot.service: om firstboot misslyckas
#    startar piholsterd INTE — det finns inget TLS-cert eller lösenord att använda.
#
#    Requires=piholster-arpd.service: ARP-socketen måste finnas
#    (Type=notify garanterar att arpd är redo när piholsterd startar).
#
#    ReadWritePaths: piholsterd behöver skriva till:
#      /var/lib/piholster  — SQLite-db, blocklists, TLS-cert
#      /run/piholster       — Unix socket (klient-sidan)
# ---------------------------------------------------------------------------

cat > /etc/systemd/system/piholsterd.service <<'UNIT'
[Unit]
Description=PiHolster Network Security Daemon
Documentation=https://github.com/piholster/piholster
After=network-online.target time-sync.target piholster-firstboot.service piholster-arpd.service
Requires=piholster-firstboot.service piholster-arpd.service

[Service]
Type=notify
ExecStart=/usr/local/bin/piholsterd
User=piholster
Group=piholster
# Capabilities: binda till privilegierade portar (:53, :80, :443)
# INTE CAP_NET_RAW — den är isolerad i piholster-arpd (ADR-002 §3.1.1)
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
# Miljövariabler (kan overridas via /etc/systemd/system/piholsterd.service.d/override.conf)
Environment=DNS_PORT=53
Environment=HTTP_PORT=80
Environment=HTTPS_PORT=443
Environment=DB_PATH=/var/lib/piholster/piholster.db
Environment=BLOCKLIST_PATH=/var/lib/piholster/blocklists/ads.txt
Environment=TLS_CERT=/var/lib/piholster/tls.crt
Environment=TLS_KEY=/var/lib/piholster/tls.key
Environment=ARP_SOCK=/run/piholster/arp.sock
# Filsystem-isolation
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
# piholsterd behöver skriva data och läsa ARP-sockeln
ReadWritePaths=/var/lib/piholster /run/piholster
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
# Restart-policy
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
UNIT

# ---------------------------------------------------------------------------
# 4. piholster-avahi-publish.service
#
#    Publicerar piholster._https._tcp via Avahi-daemon EFTER att
#    piholsterd är redo (ADR-002 §3.2.1 Lager 3).
#    Statiska /etc/avahi/services/-filer används INTE; Avahi-publish
#    körs programmatiskt så att .local-adressen inte är synlig under
#    firstboot-fönstret.
# ---------------------------------------------------------------------------

cat > /etc/systemd/system/piholster-avahi-publish.service <<'UNIT'
[Unit]
Description=PiHolster mDNS Service Publishing
Documentation=https://github.com/piholster/piholster
# Publicera EFTER att piholsterd är redo — inte under firstboot-fönstret
After=piholsterd.service avahi-daemon.service
Requires=piholsterd.service avahi-daemon.service

[Service]
Type=simple
ExecStart=/usr/bin/avahi-publish -s piholster _https._tcp 443 "PiHolster Admin UI"
Restart=on-failure
RestartSec=5s
# Minimal isolering (avahi-publish är ett enkelt CLI-verktyg)
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
UNIT

# ---------------------------------------------------------------------------
# 5. Konfigurera Avahi: minimera information-leakage
#
#    ADR-002 §3.2.1: publicera inte HINFO, workstation eller AAAA-on-IPv4.
#    Dessa läcker enhetstyp och OS-version till nätverket.
# ---------------------------------------------------------------------------

echo "[00-install/01-run.sh] Konfigurerar Avahi ..."

# Säkerhetskopia av default-konfiguration
AVAHI_CONF=/etc/avahi/avahi-daemon.conf
if [[ -f "${AVAHI_CONF}" ]]; then
    cp "${AVAHI_CONF}" "${AVAHI_CONF}.orig"
fi

# Patch-funktion: sätter en nyckel i en specifik INI-sektion.
# Om nyckeln finns i sektionen ersätts den in-place.
# Om nyckeln saknas läggs den till direkt efter sektionsrubriken.
# Använder en temporär fil för att undvika att skriva halvfärdiga configs.
avahi_set_key() {
    local section="$1"  # t.ex. "publish"
    local key="$2"
    local value="$3"
    local tmpfile
    tmpfile="$(mktemp)"

    awk -v section="${section}" -v key="${key}" -v value="${value}" '
        BEGIN { in_section=0; key_written=0 }
        /^\[/ {
            # Om vi lämnar sektionen och nyckeln inte skrivits — skriv den nu
            if (in_section && !key_written) {
                print key "=" value
                key_written=1
            }
            in_section = ($0 == "[" section "]")
        }
        in_section && $0 ~ "^" key "=" {
            # Ersätt befintlig rad
            print key "=" value
            key_written=1
            next
        }
        { print }
        END {
            # Sektionen var sist i filen och nyckeln skrevs aldrig
            if (in_section && !key_written) {
                print key "=" value
            }
            # Sektionen saknas helt — lägg till den
            if (!in_section && !key_written) {
                print ""
                print "[" section "]"
                print key "=" value
            }
        }
    ' "${AVAHI_CONF}" > "${tmpfile}" && mv "${tmpfile}" "${AVAHI_CONF}"
}

# Applicera säkerhetsinställningar (ADR-002 §3.2.1)
avahi_set_key "publish" "publish-hinfo"       "no"
avahi_set_key "publish" "publish-workstation" "no"
avahi_set_key "publish" "publish-aaaa-on-ipv4" "no"
avahi_set_key "publish" "disable-publishing"  "no"

echo "[00-install/01-run.sh] Avahi konfigurerad."

# ---------------------------------------------------------------------------
# 6. Aktivera alla services
#
#    systemctl enable skapar symlinkar i /etc/systemd/system/multi-user.target.wants/
#    Tjänsterna startas automatiskt vid boot.
# ---------------------------------------------------------------------------

echo "[00-install/01-run.sh] Aktiverar systemd-tjänster ..."

systemctl enable piholster-firstboot.service
systemctl enable piholster-arpd.service
systemctl enable piholsterd.service
systemctl enable piholster-avahi-publish.service

# Ladda om systemd-daemon för att läsa de nya unit-filerna
systemctl daemon-reload 2>/dev/null || true

echo "[00-install/01-run.sh] Tjänster aktiverade:"
echo "  piholster-firstboot.service  (oneshot, kör vid första boot)"
echo "  piholster-arpd.service       (ARP-scanner, UID 998)"
echo "  piholsterd.service           (huvuddaemon, UID 999)"
echo "  piholster-avahi-publish.service (mDNS, publicerar efter piholsterd)"

echo "[00-install/01-run.sh] Sub-stage 00-install steg 2 klar."
