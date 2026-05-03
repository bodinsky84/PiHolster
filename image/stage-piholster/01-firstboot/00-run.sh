#!/bin/bash -e
# stage-piholster/01-firstboot/00-run.sh
#
# Sub-stage 01-firstboot: Installera firstboot-scriptet i imagen.
#
# pi-gen kör detta script inuti chroot på target-filsystemet.
# Scriptet kopieras från vår stage och läggs på rätt plats.

echo "[01-firstboot/00-run.sh] Installerar firstboot-script ..."

# ---------------------------------------------------------------------------
# 1. Installera piholster-firstboot.sh
#
#    Scriptet kopieras av pi-gen från 01-firstboot/ till chroot-miljön.
#    Vi installerar det på rätt plats med korrekt ägare och mode.
#
#    Scriptet körs som root av systemd (piholster-firstboot.service).
#    Det MÅSTE vara körbart och ägas av root.
# ---------------------------------------------------------------------------

# pi-gen lägger filer från stage-katalogen (inte files/) direkt i rootfs
# under samma relativa sökväg. piholster-firstboot.sh kopieras hit av pi-gen
# om det läggs i stages-katalogen, men enklast är att installera det explicit.

SRC="/piholster-firstboot.sh"  # pi-gen kopiar hit från vår stage-katalog

if [[ -f "${SRC}" ]]; then
    install -m 0755 -o root -g root "${SRC}" /usr/local/bin/piholster-firstboot.sh
    rm -f "${SRC}"
elif [[ -f "/usr/local/bin/piholster-firstboot.sh" ]]; then
    # Redan på plats (kan hända vid iterativ testning)
    chmod 0755 /usr/local/bin/piholster-firstboot.sh
    chown root:root /usr/local/bin/piholster-firstboot.sh
    echo "[01-firstboot/00-run.sh] piholster-firstboot.sh redan installerad."
else
    echo "[01-firstboot/00-run.sh] FATAL: piholster-firstboot.sh hittades inte." >&2
    exit 1
fi

# Verifiera att scriptet faktiskt är körbart
if [[ ! -x /usr/local/bin/piholster-firstboot.sh ]]; then
    echo "[01-firstboot/00-run.sh] FATAL: /usr/local/bin/piholster-firstboot.sh är inte körbar." >&2
    exit 1
fi

echo "[01-firstboot/00-run.sh] piholster-firstboot.sh installerad."

# ---------------------------------------------------------------------------
# 2. Verifiera att unit-filen som kallar scriptet finns
#    (installerades av 00-install/01-run.sh)
# ---------------------------------------------------------------------------

if [[ ! -f /etc/systemd/system/piholster-firstboot.service ]]; then
    echo "[01-firstboot/00-run.sh] VARNING: piholster-firstboot.service saknas." >&2
    echo "[01-firstboot/00-run.sh] Den borde ha installerats av 00-install/01-run.sh." >&2
fi

echo "[01-firstboot/00-run.sh] Sub-stage 01-firstboot klar."
