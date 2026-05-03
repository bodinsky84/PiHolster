#!/bin/bash -e
# /usr/local/bin/piholster-firstboot.sh
#
# Körs av piholster-firstboot.service (Type=oneshot) vid FÖRSTA boot.
# Genererar per-enhetshemligheterna som inte kan baka in i imagen:
#   1. Slumpmässigt admin-lösenord (24 tecken, base32) — skrivs till tmpfs
#   2. Självsignerat TLS-certifikat (3650 dagar, P-256 ECDSA)
#   3. Device identity key (32 bytes, /dev/urandom)
#   4. Kopierar blocklist till data-katalogen
#   5. Öppnar iptables-portar (säkerhetsmodell: DROP tills firstboot är klar)
#   6. LED-feedback via ACT-LED
#   7. Skriver sentinel-fil .firstboot-done
#
# Säkerhetsmodell (ADR-002 §3.2.1):
#   - iptables startar i DEFAULT DROP (konfigurerat av 02-harden).
#   - Inga portar är öppna tills detta script har körts klart.
#   - piholsterd och piholster-arpd startar inte förrän denna tjänst är klar
#     (systemd Requires= + Before= i unit-filerna).
#   - Admin-lösenordet skrivs till /run/piholster/ (tmpfs) — aldrig till disk.

set -euo pipefail

DATADIR=/var/lib/piholster
RUNDIR=/run/piholster
LOGPREFIX="[piholster-firstboot]"

log()  { echo "${LOGPREFIX} $*"; }
die()  { echo "${LOGPREFIX} FATAL: $*" >&2; exit 1; }
warn() { echo "${LOGPREFIX} VARNING: $*" >&2; }

# ---------------------------------------------------------------------------
# LED-hjälpfunktion: styr Pi 3 ACT-LED (grön) via sysfs
#   fast-blink: setup pågår
#   on:         setup klar
# ---------------------------------------------------------------------------

LED_TRIGGER=/sys/class/leds/ACT/trigger
LED_DELAY_ON=/sys/class/leds/ACT/delay_on
LED_DELAY_OFF=/sys/class/leds/ACT/delay_off

led_blink_fast() {
    if [[ -e "${LED_TRIGGER}" ]]; then
        echo timer > "${LED_TRIGGER}" 2>/dev/null || true
        echo 100   > "${LED_DELAY_ON}"  2>/dev/null || true
        echo 100   > "${LED_DELAY_OFF}" 2>/dev/null || true
        log "LED: snabb blink (setup pågår)"
    fi
}

led_on_steady() {
    if [[ -e "${LED_TRIGGER}" ]]; then
        echo default-on > "${LED_TRIGGER}" 2>/dev/null || true
        log "LED: stadigt på (setup klar)"
    fi
}

# ---------------------------------------------------------------------------
# Huvudprogram
# ---------------------------------------------------------------------------

log "Startar firstboot-setup ..."
log "  Datakatalog: ${DATADIR}"

# Aktivera snabb LED-blink som signal till användaren att setup pågår
led_blink_fast

# Säkerställ att runtime-katalogen finns (systemd skapar den via RuntimeDirectory=
# men firstboot körs innan piholsterd — vi skapar den manuellt om den saknas)
if [[ ! -d "${RUNDIR}" ]]; then
    install -d -m 750 -o piholster -g piholster "${RUNDIR}"
fi

# ---------------------------------------------------------------------------
# 1. Idempotens-kontroll
#
#    Om sentinel-filen redan finns är firstboot redan körd. Avsluta direkt.
#    (systemd ConditionPathExists= skyddar också, men defence-in-depth.)
# ---------------------------------------------------------------------------

if [[ -f "${DATADIR}/.firstboot-done" ]]; then
    log "Firstboot redan utförd — avslutar."
    exit 0
fi

# ---------------------------------------------------------------------------
# 2. Generera slumpmässigt admin-lösenord
#
#    24 tecken, base32-kodad (alfabet: A-Z, 2-7).
#    base32 ger ett alfabet utan tvetydiga tecken (0/O, 1/l/I).
#    Entropin: 24 tecken × log2(32) = 120 bitar.
#
#    SÄKERHET (B-01): lösenordet skrivs till /run/piholster/initial-password
#    som är tmpfs — finns bara i RAM och försvinner vid omstart. Skrivs
#    ALDRIG till /var/lib/piholster/ (blocklagring/SD-kort).
#    piholsterd läser filen vid UserCount()==0, hashar med Argon2id och
#    raderar filen efter första lyckade inloggning.
# ---------------------------------------------------------------------------

log "Genererar admin-lösenord ..."

# head -c 32 ger 32 slumpmässiga bytes = 256 bitar råentropy
ADMIN_PASS="$(head -c 32 /dev/urandom | base32 | tr -d '=' | head -c 24)"

if [[ ${#ADMIN_PASS} -ne 24 ]]; then
    die "Lösenordsgenerering misslyckades — fel längd: ${#ADMIN_PASS} (förväntade 24)"
fi

# Skriv till tmpfs — mode 0640, ägd av root:piholster
# piholsterd (UID 999, group piholster) kan läsa filen
install -m 0640 -o root -g piholster /dev/null "${RUNDIR}/initial-password"
printf '%s\n' "${ADMIN_PASS}" > "${RUNDIR}/initial-password"

# Rensa variabeln från bash-processminnet
ADMIN_PASS=""

log "Admin-lösenord genererat: ${RUNDIR}/initial-password (tmpfs, försvinner vid omstart)"

# ---------------------------------------------------------------------------
# 3. Generera självsignerat TLS-certifikat
#
#    P-256 ECDSA (B-03): snabbare generering och handshake på Pi 3 ARM
#    jämfört med RSA 4096. Genereringstid <1s vs 30-120s för RSA 4096.
#    3650 dagars löptid (B-02): 10 år — icke-tekniska användare behöver
#    inte förnya manuellt under produktens förväntade livstid.
#
#    SAN täcker alla namn användaren kan nå enheten på.
#    -nodes: ingen passphrase — daemon startar annars inte utan manuell prompt.
# ---------------------------------------------------------------------------

log "Genererar TLS-certifikat (P-256 ECDSA, 3650 dagar) ..."

openssl req \
    -x509 \
    -newkey ec \
    -pkeyopt ec_paramgen_curve:P-256 \
    -keyout "${DATADIR}/tls.key" \
    -out    "${DATADIR}/tls.crt" \
    -days 3650 \
    -nodes \
    -subj "/C=SE/O=PiHolster/CN=piholster.local" \
    -addext "subjectAltName=DNS:piholster.local,DNS:piholster.lan,IP:127.0.0.1" \
    2>&1

# Sätt korrekta rättigheter
chmod 0640 "${DATADIR}/tls.crt"
chmod 0600 "${DATADIR}/tls.key"
chown piholster:piholster "${DATADIR}/tls.crt" "${DATADIR}/tls.key"

# Verifiera att certifikatet är parsbart
if ! openssl x509 -noout -in "${DATADIR}/tls.crt" 2>/dev/null; then
    die "TLS-certifikat genererades men verifiering misslyckades."
fi

log "TLS-certifikat genererat (giltigt 10 år)."

# ---------------------------------------------------------------------------
# 4. Generera device identity key (B-04)
#
#    32 bytes kryptografiskt slumpmässiga data (256 bitar).
#    Används som rot för framtida per-rad AES-GCM-kryptering av query_log
#    och som suffix i HTTP User-Agent: sha256(device-id.key)[:8].
#    mode 0400: bara piholster-usern kan läsa.
# ---------------------------------------------------------------------------

log "Genererar device identity key ..."

DEVICE_ID_FILE="${DATADIR}/device-id.key"

if [[ ! -f "${DEVICE_ID_FILE}" ]]; then
    # Skapa filen med rätt mode innan write (undviker race mot chmod)
    install -m 0400 -o piholster -g piholster /dev/null "${DEVICE_ID_FILE}"
    head -c 32 /dev/urandom > "${DEVICE_ID_FILE}"
    chmod 0400 "${DEVICE_ID_FILE}"
    chown piholster:piholster "${DEVICE_ID_FILE}"

    # Verifiera att 32 bytes skrevs
    KEY_SIZE=$(wc -c < "${DEVICE_ID_FILE}")
    if [[ "${KEY_SIZE}" -ne 32 ]]; then
        die "Device identity key har fel storlek: ${KEY_SIZE} bytes (förväntade 32)"
    fi

    log "Device identity key genererad: ${DEVICE_ID_FILE} (32 bytes)"
else
    log "Device identity key finns redan — hoppar över."
fi

# ---------------------------------------------------------------------------
# 5. Kopiera blocklists till data-katalogen
#
#    /usr/share/piholster/ads.txt lades dit av 00-install.
#    Vi kopierar till /var/lib/piholster/blocklists/ som piholsterd läser.
# ---------------------------------------------------------------------------

log "Kopierar blocklists ..."

install -d -m 750 -o piholster -g piholster "${DATADIR}/blocklists"

if [[ -f /usr/share/piholster/ads.txt ]]; then
    cp /usr/share/piholster/ads.txt "${DATADIR}/blocklists/ads.txt"
    chown piholster:piholster "${DATADIR}/blocklists/ads.txt"
    chmod 640 "${DATADIR}/blocklists/ads.txt"
    log "Blocklist kopierad: $(wc -l < "${DATADIR}/blocklists/ads.txt") rader."
else
    warn "ads.txt saknas i /usr/share/piholster/ — blocklist är tom vid start."
fi

# ---------------------------------------------------------------------------
# 6. Öppna iptables-portar
#
#    ADR-002 §3.2.1 Lager 1: imagen levereras med DEFAULT DROP på INPUT.
#    Firstboot öppnar portarna EFTER att TLS-cert och lösenord är klara.
# ---------------------------------------------------------------------------

log "Öppnar iptables-portar ..."

if command -v iptables &>/dev/null; then
    iptables -A INPUT -p udp --dport 53   -j ACCEPT
    iptables -A INPUT -p tcp --dport 53   -j ACCEPT
    iptables -A INPUT -p tcp --dport 80   -j ACCEPT
    iptables -A INPUT -p tcp --dport 443  -j ACCEPT
    iptables -A INPUT -p udp --dport 5353 -j ACCEPT

    if command -v iptables-save &>/dev/null; then
        iptables-save > /etc/iptables/rules.v4
        log "iptables-regler sparade till /etc/iptables/rules.v4"
    fi
else
    warn "iptables inte tillgängligt — portar öppnas inte av firstboot."
fi

# ---------------------------------------------------------------------------
# 7. Skriv sentinel-fil
#
#    Måste skapas SIST. set -e garanterar att om något steg ovan
#    misslyckas körs inte touch-kommandot — firstboot körs om vid nästa boot.
# ---------------------------------------------------------------------------

log "Skriver sentinel-fil ..."
touch "${DATADIR}/.firstboot-done"
chmod 444 "${DATADIR}/.firstboot-done"
chown root:root "${DATADIR}/.firstboot-done"

# ---------------------------------------------------------------------------
# 8. Sätt LED till stadigt på — setup klar
# ---------------------------------------------------------------------------

led_on_steady

log "PiHolster firstboot klar."
log "  Admin-lösenord: ${RUNDIR}/initial-password (tmpfs)"
log "  TLS-cert:       ${DATADIR}/tls.crt"
log "  Device ID:      ${DATADIR}/device-id.key"
log "  Blocklist:      ${DATADIR}/blocklists/ads.txt"
log ""
log "Enheten är nu redo. Öppna https://piholster.local i din webbläsare."

exit 0
