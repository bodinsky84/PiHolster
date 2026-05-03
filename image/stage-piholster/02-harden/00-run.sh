#!/bin/bash -e
# stage-piholster/02-harden/00-run.sh
#
# Sub-stage 02-harden: Härdning av OS-konfigurationen.
#
# Körordning (ADR-002 §3.2.1):
#   Imagen levereras med en minimal initital iptables-regeluppsättning
#   som DEFAULT DROP:ar allt inkommande utom loopback och established.
#   Firstboot-scriptet öppnar specifika portar EFTER att hemligheterna
#   är genererade.
#
# Detta script körs av pi-gen inuti chroot under build — INTE vid boot.
# Allt som görs här är statisk konfiguration som baka in i imagen.

echo "[02-harden/00-run.sh] Börjar härdning ..."

# ---------------------------------------------------------------------------
# 1. Initiala iptables-regler: DEFAULT DROP
#
#    ADR-002 §3.2.1 Lager 1:
#    Imagen levereras med regler som DROP:ar all inkommande trafik
#    utom loopback och already-established sessioner.
#
#    ICMP echo-request är tillåtet men rate-limiterat (5/sek) för att
#    tillåta ping utan att öppna för ICMP flood.
#
#    Firstboot-scriptet lägger till de faktiska tjänste-portarna med
#    iptables -A INPUT och sparar sedan med iptables-save.
#
#    IPv6: vi DROP:ar all IPv6-input i MVP. PiHolster v1.0 är IPv4-first.
#    Loopback (::1) tillåts alltid för systemd-intern kommunikation.
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Konfigurerar iptables ..."

mkdir -p /etc/iptables

# IPv4-regler: DEFAULT DROP inbound
# Portar 53, 80, 443, 5353 öppnas av firstboot-scriptet vid första boot.
cat > /etc/iptables/rules.v4 <<'EOF'
*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [0:0]

# Tillåt loopback (systemd och local IPC)
-A INPUT -i lo -j ACCEPT

# Tillåt already-established och related (t.ex. svar på outbound DNS-queries)
-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Tillåt ICMP echo-request (ping) med rate-limiting mot flood
-A INPUT -p icmp --icmp-type echo-request -m limit --limit 5/sec -j ACCEPT

# Alla övriga paket DROP:as tills firstboot lägger till tjänste-portarna.
# Kommentar: piholster-firstboot.sh lägger till dessa ACCEPT-regler efter
# att TLS-cert och lösenord är genererade:
#   -A INPUT -p udp --dport 53   -j ACCEPT   # DNS
#   -A INPUT -p tcp --dport 53   -j ACCEPT   # DNS
#   -A INPUT -p tcp --dport 80   -j ACCEPT   # HTTP (redirect)
#   -A INPUT -p tcp --dport 443  -j ACCEPT   # HTTPS
#   -A INPUT -p udp --dport 5353 -j ACCEPT   # mDNS

COMMIT
EOF

# IPv6-regler: DROP allt inbound i MVP, tillåt loopback och established
cat > /etc/iptables/rules.v6 <<'EOF'
*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [0:0]

# Tillåt loopback
-A INPUT -i lo -j ACCEPT

# Tillåt already-established
-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# ICMPv6 behövs för IPv6-grundfunktion (ND, RS, RA) men vi kör IPv4-only
# Tillåt ändå lokal ICMPv6 (link-local) för att inte bryta systemd-networkd
-A INPUT -p icmpv6 -s fe80::/10 -j ACCEPT

COMMIT
EOF

echo "[02-harden/00-run.sh] iptables-regler skrivna."

# ---------------------------------------------------------------------------
# 2. sysctl-härdning
#
#    Paramtrarna läggs i en separat fil i /etc/sysctl.d/ för att inte
#    modifiera /etc/sysctl.conf direkt (undviker konflikter med paketet
#    procps som äger sysctl.conf).
#
#    tcp_syncookies:  skyddar mot TCP SYN flood (DDoS-attack)
#    kptr_restrict=2: döljer kernel-pekaradresser från alla, inklusive root
#                     (försvårar kernel-exploit-development)
#    dmesg_restrict:  hindrar icke-root från att läsa kernel-ring-buffer
#                     (kernel-adresser kan läcka via dmesg)
#    rp_filter=1:    strict reverse-path filtering (stoppar IP-spoofing)
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Konfigurerar sysctl ..."

cat > /etc/sysctl.d/99-piholster-harden.conf <<'EOF'
# PiHolster security hardening (ADR-002)
# Aktiveras automatiskt av sysctl.service vid boot.

# TCP SYN cookie-skydd mot SYN flood
net.ipv4.tcp_syncookies = 1

# Dölj kernel-pekaradresser (skyddar mot kernel-exploits)
kernel.kptr_restrict = 2

# Begränsa tillgång till kernel-ring-buffer
kernel.dmesg_restrict = 1

# Strict reverse-path filtering (stoppar IP-spoofing)
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Ignorera ICMP redirects (kan användas för routing-attack)
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.default.accept_redirects = 0
net.ipv6.conf.all.accept_redirects = 0

# Skicka inte ICMP redirects (vi är inte en router)
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.default.send_redirects = 0

# Logga spoofade, source-routade och redirect-paket
net.ipv4.conf.all.log_martians = 1
EOF

echo "[02-harden/00-run.sh] sysctl konfigurerat."

# ---------------------------------------------------------------------------
# 3. Inaktivera SSH som standard
#
#    SSH är av säkerhetsskäl INTE aktiverat på prod-imagen.
#    Avancerade användare kan aktivera det via Admin UI (kräver admin-lösenord).
#    ADR-001 §7.2: "Ingen default pi-user" — utan user-konto är SSH meningslöst
#    ändå, men vi inaktiverar explicit för defence-in-depth.
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Inaktiverar SSH ..."

systemctl disable ssh       2>/dev/null || true
systemctl disable sshd      2>/dev/null || true
systemctl disable ssh.socket 2>/dev/null || true

# Ta bort ssh-serverpaket om det installerats (Pi OS Lite inkluderar det ibland)
if dpkg -l openssh-server &>/dev/null 2>&1; then
    apt-get remove -y openssh-server
    apt-get autoremove -y
    apt-get clean
fi

echo "[02-harden/00-run.sh] SSH inaktiverat."

# ---------------------------------------------------------------------------
# 4. Konfigurera unattended-upgrades: säkerhets-only
#
#    ADR-001 §7.2: automatiska säkerhetsuppdateringar är på som standard.
#    Vi begränsar till security-pocket för att undvika att en ny pakeversion
#    bryter PiHolster-funktionaliteten vid nästa upgrade.
#
#    AutoFixInterruptedDpkg: reparera automatiskt om en apt-session
#    avbröts mitt i (strömavbrott etc).
#    MinimalSteps: uppgradera ett paket i taget — minskar risken för
#    systemet att vara i inkonsistent tillstånd vid strömavbrott.
#    Remove-Unused-Dependencies: kör autoremove efter uppgradering.
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Konfigurerar unattended-upgrades ..."

# Huvudkonfiguration: vilka origins som tillåts uppgradera automatiskt
cat > /etc/apt/apt.conf.d/50unattended-upgrades <<'EOF'
// PiHolster: automatiska säkerhetsuppdateringar
// Endast Debian/Raspbian security-pocket — INTE stabila uppdateringar.
Unattended-Upgrade::Allowed-Origins {
    "${distro_id}:${distro_codename}-security";
    "Raspbian:${distro_codename}";
};

// Reparera automatiskt avbruten dpkg-session
Unattended-Upgrade::AutoFixInterruptedDpkg "true";

// Uppgradera ett paket i taget (trygg vid strömavbrott)
Unattended-Upgrade::MinimalSteps "true";

// Ta bort oanvända paket efter uppgradering
Unattended-Upgrade::Remove-Unused-Dependencies "true";

// Starta om automatiskt vid kernel-uppgraderingar (kräver säker boot)
// Satt till false i MVP — användaren notifieras via Admin UI istället.
Unattended-Upgrade::Automatic-Reboot "false";

// Logga alla åtgärder
Unattended-Upgrade::SyslogEnable "true";
EOF

# Schema: kör dagligen vid 02:00-06:00 (utspritt för att inte hammra upstream)
cat > /etc/apt/apt.conf.d/20auto-upgrades <<'EOF'
// PiHolster: schemalägg automatiska uppdateringar
APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Download-Upgradeable-Packages "1";
APT::Periodic::AutocleanInterval "7";
APT::Periodic::Unattended-Upgrade "1";
EOF

echo "[02-harden/00-run.sh] unattended-upgrades konfigurerat."

# ---------------------------------------------------------------------------
# 5. Säkra /proc-filsystemet
#
#    hidepid=2 hindrar icke-root-processer från att se andra processers
#    /proc/PID/-kataloger. Minskar informationsläckage om piholster-processerna.
#
#    Läggs till i /etc/fstab — monteras vid boot.
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Konfigurerar /proc-härdning ..."

# hidepid=invisible (kernel 5.8+) ersätter hidepid=2.
# Raspberry Pi OS Bookworm kör kernel 6.x — hidepid=2 fungerar men ger
# deprecation-varning. hidepid=invisible är den rekommenderade syntaxen.
# gid=proc: processer i proc-gruppen kan se alla PID trots hidepid
# (behövs av ps, top och systemd-components).

# Skapa proc-grupp om den saknas
if ! getent group proc &>/dev/null; then
    groupadd --system proc
fi

# Kontrollera att /proc-raden inte redan finns
if ! grep -q 'hidepid' /etc/fstab 2>/dev/null; then
    echo "proc /proc proc defaults,hidepid=invisible,gid=proc 0 0" >> /etc/fstab
    echo "[02-harden/00-run.sh] /proc hidepid=invisible aktiverat i fstab."
else
    echo "[02-harden/00-run.sh] /proc hidepid redan konfigurerat."
fi

# ---------------------------------------------------------------------------
# 6. Begränsa core dumps
#
#    Core dumps kan innehålla känslig data (lösenords-hash, TLS-nycklar
#    i minnet). Vi inaktiverar dem globalt.
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Inaktiverar core dumps ..."

cat > /etc/security/limits.d/99-piholster-no-coredump.conf <<'EOF'
# PiHolster: inaktivera core dumps (kan läcka känslig data)
* soft core 0
* hard core 0
root soft core 0
root hard core 0
EOF

# Också via sysctl
cat >> /etc/sysctl.d/99-piholster-harden.conf <<'EOF'

# Inaktivera core dumps via sysctl
fs.suid_dumpable = 0
kernel.core_pattern = /dev/null
EOF

echo "[02-harden/00-run.sh] Core dumps inaktiverade."

# ---------------------------------------------------------------------------
# 7. Verifiering av slutresultat
# ---------------------------------------------------------------------------

echo "[02-harden/00-run.sh] Verifierar härdning ..."

# Kontrollera att iptables-filer finns
[[ -f /etc/iptables/rules.v4 ]] || { echo "FATAL: rules.v4 saknas" >&2; exit 1; }
[[ -f /etc/iptables/rules.v6 ]] || { echo "FATAL: rules.v6 saknas" >&2; exit 1; }

# Kontrollera att DEFAULT DROP är satt i rules.v4
if ! grep -q ':INPUT DROP' /etc/iptables/rules.v4; then
    echo "[02-harden/00-run.sh] FATAL: iptables/rules.v4 saknar ':INPUT DROP'" >&2
    exit 1
fi

# Kontrollera att sysctl-filen finns
[[ -f /etc/sysctl.d/99-piholster-harden.conf ]] || { echo "FATAL: sysctl-fil saknas" >&2; exit 1; }

echo "[02-harden/00-run.sh] Härdning verifierad OK."
echo "[02-harden/00-run.sh] Sub-stage 02-harden klar."
