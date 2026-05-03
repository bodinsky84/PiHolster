#!/bin/bash -e
# stage-piholster/00-install/00-run.sh
#
# Sub-stage 00-install, steg 1: Installera paket och skapa system-users.
#
# Körs av pi-gen inuti chroot på target-filsystemet.
# Env-variabler från pi-gen:
#   ROOTFS_DIR  — rotsystemets sökväg (= "/" inuti chroot)
#
# Notera: detta script körs MED chroot, dvs alla sökvägar är relativa
# till imagen, inte build-maskinen. Binärerna som ska kopieras in
# exponeras via ${ROOTFS_DIR} och kopias av pi-gen från files/-katalogen.

echo "[00-install/00-run.sh] Börjar installera PiHolster-beroenden ..."

# ---------------------------------------------------------------------------
# 1. Uppdatera apt-index och installera beroenden
#    --no-install-recommends håller imagen liten.
#    iptables-persistent sparar iptables-regler till /etc/iptables/rules.v4
#    och återläser dem vid boot via netfilter-persistent.service.
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] apt-get install ..."

apt-get update -y
apt-get install -y --no-install-recommends \
    avahi-daemon \
    iptables \
    iptables-persistent \
    netfilter-persistent \
    unattended-upgrades \
    apt-listchanges \
    ca-certificates \
    libcap2-bin

# Rensa apt-cache för att minska imagen
apt-get clean
rm -rf /var/lib/apt/lists/*

echo "[00-install/00-run.sh] Paket installerade."

# ---------------------------------------------------------------------------
# 2. Skapa system-users
#
#    piholster-arpd (UID 998): kör ARP-scannern. Ingen shell, ingen home.
#    piholster      (UID 999): kör huvuddaemon. Ingen shell, ingen home.
#
#    UID:er är hårdkodade (ADR-001 §7.2) för att vara förutsägbara i imagen
#    och matchas mot systemd-unit-direktivet User= och SO_PEERCRED-check
#    i piholsterd-klienten.
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] Skapar system-users ..."

# piholster-arpd (UID 998) — ARP-scanner med CAP_NET_RAW
useradd \
    --system \
    --uid 998 \
    --no-create-home \
    --shell /usr/sbin/nologin \
    --comment "PiHolster ARP scanner daemon" \
    piholster-arpd

# piholster (UID 999) — huvuddaemon med CAP_NET_BIND_SERVICE
useradd \
    --system \
    --uid 999 \
    --no-create-home \
    --shell /usr/sbin/nologin \
    --comment "PiHolster network security daemon" \
    piholster

echo "[00-install/00-run.sh] Users skapade: piholster-arpd (UID 998), piholster (UID 999)."

# ---------------------------------------------------------------------------
# 3. Skapa kataloger
#
#    /var/lib/piholster — persistent data (SQLite-db, TLS-cert, blocklists)
#    /run/piholster     — runtime-socket och PID-filer (tmpfs vid boot)
#
#    /run/ är tmpfs och återskapas vid varje boot av systemd via
#    RuntimeDirectory= i service-filerna, men vi skapar den här också
#    för att image-strukturen ska vara korrekt.
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] Skapar kataloger ..."

install -d -m 750 -o piholster     -g piholster     /var/lib/piholster
install -d -m 750 -o piholster-arpd -g piholster-arpd /run/piholster

# Katalog för delade blocklists (kopieras av firstboot-scriptet)
install -d -m 755 /usr/share/piholster

echo "[00-install/00-run.sh] Kataloger skapade."

# ---------------------------------------------------------------------------
# 4. Kopiera binärer
#
#    pi-gen har redan kopierat contents av files/ till ${ROOTFS_DIR}/
#    med samma relativa sökväg. Vi installerar dem på rätt plats.
#
#    OBS: PIHOLSTER_BINARY och PIHOLSTER_ARPD_BINARY pekar på
#    filerna i build.sh:s files/-export, som pi-gen lade i /tmp/
#    eller liknande. Vi kopierar dem via install(1) för korrekt mode.
#
#    Eftersom pi-gen kopierar files/ till ROOTFS_DIR direkt innan
#    scriptet körs, ligger binärerna redan på rätt ställe om man
#    lägger dem i files/usr/local/bin/. Vi väljer att kopiera
#    explicit för full kontroll och loggning.
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] Installerar binärer ..."

# Binärerna lades i files/ av build.sh och kopierades av pi-gen till /
# under samma relativa sökväg. Pi-gen lägger files/-innehåll direkt i
# rootfs utan ytterligare underkataloger — dvs files/piholsterd hamnar
# i /piholsterd. Vi hämtar dem därifrån.

if [[ -f /piholsterd ]]; then
    install -m 755 /piholsterd /usr/local/bin/piholsterd
    rm -f /piholsterd
elif [[ -f /tmp/piholsterd ]]; then
    install -m 755 /tmp/piholsterd /usr/local/bin/piholsterd
fi

if [[ -f /piholster-arpd ]]; then
    install -m 755 /piholster-arpd /usr/local/bin/piholster-arpd
    rm -f /piholster-arpd
elif [[ -f /tmp/piholster-arpd ]]; then
    install -m 755 /tmp/piholster-arpd /usr/local/bin/piholster-arpd
fi

# Verifiera att binärerna faktiskt finns
if [[ ! -f /usr/local/bin/piholsterd ]]; then
    echo "[00-install/00-run.sh] FATAL: /usr/local/bin/piholsterd saknas efter installation." >&2
    exit 1
fi
if [[ ! -f /usr/local/bin/piholster-arpd ]]; then
    echo "[00-install/00-run.sh] FATAL: /usr/local/bin/piholster-arpd saknas efter installation." >&2
    exit 1
fi

echo "[00-install/00-run.sh] Binärer installerade."

# ---------------------------------------------------------------------------
# 5. Sätt Linux capabilities
#
#    Vi använder setcap istället för SUID. Capabilities är snävt avgränsade
#    och ger principen om lägsta möjliga privilegium:
#
#    piholsterd:
#      cap_net_bind_service=+ep  — binda till privilegierade portar (:53, :80, :443)
#      INTE cap_net_raw           — ADR-002 §3.1.1: ARP är isolerat i piholster-arpd
#
#    piholster-arpd:
#      cap_net_raw=+ep           — öppna AF_PACKET-socket för ARP-sniffning
#      cap_net_bind_service       — behövs inte; arpd binder inte till låga portar
#
#    CI-check i ci.yml assertar att piholsterd INTE har cap_net_raw.
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] Sätter capabilities ..."

setcap cap_net_bind_service=+ep /usr/local/bin/piholsterd
setcap cap_net_raw=+ep          /usr/local/bin/piholster-arpd

# Verifiera capabilities
echo "[00-install/00-run.sh] Verifierar capabilities ..."
getcap /usr/local/bin/piholsterd
getcap /usr/local/bin/piholster-arpd

# Säkerhetscheck: piholsterd får INTE ha cap_net_raw (ADR-002 §3.1.1)
if getcap /usr/local/bin/piholsterd | grep -q cap_net_raw; then
    echo "[00-install/00-run.sh] FATAL: piholsterd har cap_net_raw — detta är inte tillåtet." >&2
    exit 1
fi

echo "[00-install/00-run.sh] Capabilities satta korrekt."

# ---------------------------------------------------------------------------
# 6. Kopiera blocklist-filen till /usr/share/piholster/
#
#    ads.txt ligger i files/ och kopieras av pi-gen till rootfs.
#    Firstboot-scriptet kopierar sedan den till /var/lib/piholster/blocklists/
#    (som ägs av piholster-usern och skapas vid firstboot).
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] Installerar blocklist ..."

if [[ -f /ads.txt ]]; then
    install -m 644 /ads.txt /usr/share/piholster/ads.txt
    rm -f /ads.txt
elif [[ -f /usr/share/piholster/ads.txt ]]; then
    echo "[00-install/00-run.sh] ads.txt redan på plats."
else
    echo "[00-install/00-run.sh] VARNING: ads.txt hittades inte — blocklist kommer saknas." >&2
fi

# ---------------------------------------------------------------------------
# 7. Ta bort default pi-user
#
#    Pi OS Lite skapar en 'pi'-user. Inga shell-konton ska finnas på
#    prod-imagen (ADR-001 §7.2, ADR-002 §3.2.1).
#    userdel -r tar bort hem-katalogen om den finns.
# ---------------------------------------------------------------------------

echo "[00-install/00-run.sh] Tar bort default pi-user ..."

userdel -r pi 2>/dev/null && echo "[00-install/00-run.sh] pi-user borttagen." \
    || echo "[00-install/00-run.sh] pi-user existerade inte (OK om ny imagen)."

echo "[00-install/00-run.sh] Sub-stage 00-install steg 1 klar."
