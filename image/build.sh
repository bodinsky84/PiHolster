#!/usr/bin/env bash
# PiHolster — SD-card image builder
#
# Kräver:
#   - Körs som root (pi-gen kräver det för chroot + loop-mount)
#   - pi-gen submodulen initialiserad: git submodule update --init image/pi-gen
#   - Debian/Raspberry Pi OS-baserat build-system (inte macOS, inte WSL utan bpfcc)
#
# Env-variabler (obligatoriska):
#   PIHOLSTER_BINARY      — absolut sökväg till piholsterd-binären (linux/arm64 eller linux/armv7)
#   PIHOLSTER_ARPD_BINARY — absolut sökväg till piholster-arpd-binären
#
# Valfria:
#   IMG_DATE   — datumsuffix i outputfilens namn, default: YYYY-MM-DD
#   WORK_DIR   — temporär katalog för pi-gen, default: /tmp/piholster-build-$$
#   OUT_DIR    — katalog där .img.xz och .sha256 hamnar, default: $(pwd)/dist
#   KEEP_WORK  — sätt till "1" för att behålla WORK_DIR efter bygget (debugging)

set -euo pipefail

# ---------------------------------------------------------------------------
# 0. Konstanter
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
PIGEN_DIR="${SCRIPT_DIR}/pi-gen"
STAGE_DIR="${SCRIPT_DIR}/stage-piholster"

IMG_DATE="${IMG_DATE:-$(date +%Y-%m-%d)}"
WORK_DIR="${WORK_DIR:-/tmp/piholster-build-$$}"
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/dist}"
KEEP_WORK="${KEEP_WORK:-0}"

# Läs version ur package.json om det finns, annars fallback till "dev"
VERSION="dev"
if [[ -f "${REPO_ROOT}/package.json" ]]; then
    VERSION="$(grep '"version"' "${REPO_ROOT}/package.json" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
fi

IMAGE_NAME="piholster-${VERSION}-${IMG_DATE}"

# ---------------------------------------------------------------------------
# 1. Validera miljön
# ---------------------------------------------------------------------------

log() { echo "[build.sh] $*"; }
die() { echo "[build.sh] FATAL: $*" >&2; exit 1; }

# Måste köras som root för chroot och loop-mount
if [[ "$(id -u)" -ne 0 ]]; then
    die "Måste köras som root. Kör: sudo bash image/build.sh"
fi

# Kontrollera att PIHOLSTER_BINARY är satt och pekar på en körbar fil
if [[ -z "${PIHOLSTER_BINARY:-}" ]]; then
    die "Env-var PIHOLSTER_BINARY är inte satt. Exempel: export PIHOLSTER_BINARY=/tmp/piholsterd-linux-arm64"
fi
if [[ ! -f "${PIHOLSTER_BINARY}" ]]; then
    die "PIHOLSTER_BINARY='${PIHOLSTER_BINARY}' — filen finns inte."
fi
if [[ ! -x "${PIHOLSTER_BINARY}" ]]; then
    die "PIHOLSTER_BINARY='${PIHOLSTER_BINARY}' — filen är inte körbar (chmod +x behövs)."
fi

# Kontrollera att PIHOLSTER_ARPD_BINARY är satt och pekar på en körbar fil
if [[ -z "${PIHOLSTER_ARPD_BINARY:-}" ]]; then
    die "Env-var PIHOLSTER_ARPD_BINARY är inte satt. Exempel: export PIHOLSTER_ARPD_BINARY=/tmp/piholster-arpd-linux-arm64"
fi
if [[ ! -f "${PIHOLSTER_ARPD_BINARY}" ]]; then
    die "PIHOLSTER_ARPD_BINARY='${PIHOLSTER_ARPD_BINARY}' — filen finns inte."
fi
if [[ ! -x "${PIHOLSTER_ARPD_BINARY}" ]]; then
    die "PIHOLSTER_ARPD_BINARY='${PIHOLSTER_ARPD_BINARY}' — filen är inte körbar (chmod +x behövs)."
fi

# Kontrollera att pi-gen submodulen är initialiserad
if [[ ! -f "${PIGEN_DIR}/build.sh" ]]; then
    die "pi-gen saknas på '${PIGEN_DIR}'. Kör: git submodule update --init image/pi-gen"
fi

# Kontrollera att vår stage finns
if [[ ! -d "${STAGE_DIR}" ]]; then
    die "stage-piholster saknas på '${STAGE_DIR}'."
fi

# Kontrollera att nödvändiga verktyg finns
for cmd in xz sha256sum rsync; do
    if ! command -v "${cmd}" &>/dev/null; then
        die "Saknar verktyg: ${cmd}. Installera med: apt-get install -y ${cmd}"
    fi
done

log "Validering OK."
log "  piholsterd:     ${PIHOLSTER_BINARY}"
log "  piholster-arpd: ${PIHOLSTER_ARPD_BINARY}"
log "  Version:        ${VERSION}"
log "  Image-namn:     ${IMAGE_NAME}"

# ---------------------------------------------------------------------------
# 2. Exportera binärerna till stage-filsystemet
#    pi-gen kopierar stage/*/files/ till imagen automatiskt.
#    Vi lägger binärerna temporärt i 00-install/files/ och rensar efter bygget.
# ---------------------------------------------------------------------------

FILES_DIR="${STAGE_DIR}/00-install/files"

log "Kopierar binärer till ${FILES_DIR}/ ..."
cp "${PIHOLSTER_BINARY}"      "${FILES_DIR}/piholsterd"
cp "${PIHOLSTER_ARPD_BINARY}" "${FILES_DIR}/piholster-arpd"
chmod 755 "${FILES_DIR}/piholsterd" "${FILES_DIR}/piholster-arpd"

# Städa upp binärer vid exit (oavsett om bygget lyckas eller inte)
cleanup_binaries() {
    log "Städar bort temporära binärer från files/ ..."
    rm -f "${FILES_DIR}/piholsterd" "${FILES_DIR}/piholster-arpd"
}
trap cleanup_binaries EXIT

# ---------------------------------------------------------------------------
# 3. Skapa pi-gen konfiguration
# ---------------------------------------------------------------------------

log "Skapar pi-gen konfiguration ..."
mkdir -p "${WORK_DIR}"

# config-filen som pi-gen läser
cat > "${WORK_DIR}/config" <<EOF
# PiHolster pi-gen konfiguration
# Autogenererat av image/build.sh — ändra inte manuellt

IMG_NAME="${IMAGE_NAME}"
RELEASE=bookworm
DEPLOY_DIR="${OUT_DIR}"
WORK_DIR="${WORK_DIR}/pi-gen-work"
USE_QEMU=0

# Ingen desktop, ingen rekommenderade paket
ENABLE_SSH=0

# Locale och tidszon (neutrala defaults — firstboot kan justera)
LOCALE_DEFAULT=en_GB.UTF-8
TARGET_HOSTNAME=piholster
KEYBOARD_KEYMAP=gb
KEYBOARD_LAYOUT="English (UK)"
TIMEZONE_DEFAULT=UTC

# Ingen standard-user — piholster-user skapas av vår stage
FIRST_USER_NAME=""
FIRST_USER_PASS=""
DISABLE_FIRST_BOOT_USER_RENAME=1

# Komprimering (görs manuellt nedan med xz -9 för bättre kontroll)
COMPRESSION=""
EOF

# ---------------------------------------------------------------------------
# 4. Konfigurera vilka pi-gen-stages som körs
#    Vi hoppar över stage3 (skrivbordsmiljö) och stage4/5 (extra mjukvara).
#    SKIP-filer stoppar pi-gen från att köra en stage.
# ---------------------------------------------------------------------------

log "Konfigurerar pi-gen stages ..."
PIGEN_WORK="${WORK_DIR}/pi-gen-work"
mkdir -p "${PIGEN_WORK}"

# Hoppa över stage3, stage4, stage5 om de finns i pi-gen
for skip_stage in stage3 stage4 stage5; do
    if [[ -d "${PIGEN_DIR}/${skip_stage}" ]]; then
        touch "${PIGEN_DIR}/${skip_stage}/SKIP"
        touch "${PIGEN_DIR}/${skip_stage}/SKIP_IMAGES" 2>/dev/null || true
    fi
done

# Hoppa över default stage2-image (vi vill bara bygga vår egna stage)
touch "${PIGEN_DIR}/stage2/SKIP_IMAGES" 2>/dev/null || true

# Länka in vår custom stage i pi-gen-katalogen
PIGEN_STAGE_LINK="${PIGEN_DIR}/stage-piholster"
if [[ -L "${PIGEN_STAGE_LINK}" ]]; then
    rm "${PIGEN_STAGE_LINK}"
fi
ln -s "${STAGE_DIR}" "${PIGEN_STAGE_LINK}"

# ---------------------------------------------------------------------------
# 5. Exportera env-variabler som våra stage-scripts behöver
# ---------------------------------------------------------------------------

export PIHOLSTER_BINARY="${FILES_DIR}/piholsterd"
export PIHOLSTER_ARPD_BINARY="${FILES_DIR}/piholster-arpd"

# ---------------------------------------------------------------------------
# 6. Kör pi-gen
# ---------------------------------------------------------------------------

log "Startar pi-gen ..."
log "  Arbetsdir: ${PIGEN_WORK}"
log "  Output:    ${OUT_DIR}"

mkdir -p "${OUT_DIR}"

# pi-gen läser config-filen via --config eller som argument beroende på version.
# Nyare pi-gen stödjer --config; äldre source:ar config direkt.
pushd "${PIGEN_DIR}" > /dev/null

if bash build.sh --config "${WORK_DIR}/config" 2>&1 | tee "${WORK_DIR}/pi-gen.log"; then
    log "pi-gen klar."
else
    die "pi-gen misslyckades. Se logg: ${WORK_DIR}/pi-gen.log"
fi

popd > /dev/null

# ---------------------------------------------------------------------------
# 7. Hitta den råa .img-filen som pi-gen producerade
# ---------------------------------------------------------------------------

RAW_IMG="$(find "${OUT_DIR}" -maxdepth 2 -name "*.img" | sort | tail -1)"

if [[ -z "${RAW_IMG}" ]]; then
    # pi-gen kan lägga output i DEPLOY_DIR under ett datumstämplat subdir
    RAW_IMG="$(find "${PIGEN_DIR}/deploy" -maxdepth 2 -name "*.img" 2>/dev/null | sort | tail -1 || true)"
fi

if [[ -z "${RAW_IMG}" ]]; then
    die "Kunde inte hitta .img-filen efter pi-gen-körningen. Kontrollera ${WORK_DIR}/pi-gen.log"
fi

log "Hittade råbild: ${RAW_IMG}"

# Flytta till OUT_DIR med rätt namn om den inte redan är där
FINAL_IMG="${OUT_DIR}/${IMAGE_NAME}.img"
if [[ "${RAW_IMG}" != "${FINAL_IMG}" ]]; then
    mv "${RAW_IMG}" "${FINAL_IMG}"
fi

# ---------------------------------------------------------------------------
# 8. Komprimera till .img.xz
#    xz -9 ger bäst komprimering, -T0 använder alla CPU-kärnor.
# ---------------------------------------------------------------------------

COMPRESSED="${FINAL_IMG}.xz"

log "Komprimerar till ${COMPRESSED} (xz -9, kan ta flera minuter) ..."
xz -9 --threads=0 --keep --force "${FINAL_IMG}"

# Ta bort råbilden för att spara utrymme
rm -f "${FINAL_IMG}"

log "Komprimering klar: ${COMPRESSED}"

# ---------------------------------------------------------------------------
# 9. Generera SHA256-checksumma
# ---------------------------------------------------------------------------

CHECKSUM_FILE="${COMPRESSED}.sha256"

log "Genererar SHA256-checksumma ..."
# sha256sum producerar "HASH  FILNAMN" — vi vill bara ha filnamnet, inte hela sökvägen
pushd "$(dirname "${COMPRESSED}")" > /dev/null
sha256sum "$(basename "${COMPRESSED}")" > "$(basename "${CHECKSUM_FILE}")"
popd > /dev/null

log "Checksumma: $(cat "${CHECKSUM_FILE}")"

# ---------------------------------------------------------------------------
# 10. Skriv ut resultat
# ---------------------------------------------------------------------------

# ---------------------------------------------------------------------------
# 10. Generera MANIFEST (US-13 krav 5)
#     Spårar exakt vad som bakades in i denna image-build.
# ---------------------------------------------------------------------------

MANIFEST_FILE="${OUT_DIR}/MANIFEST"
PIGEN_COMMIT="$(cd "${PIGEN_DIR}" && git rev-parse HEAD 2>/dev/null || echo "unknown")"
BASE_IMAGE_SHA="$(cat "${CHECKSUM_FILE}" | awk '{print $1}')"

cat > "${MANIFEST_FILE}" <<EOF
# PiHolster Image MANIFEST
# Autogenererat av image/build.sh — ändra inte manuellt

build_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
image_name=${IMAGE_NAME}
piholster_version=${VERSION}
pigen_commit=${PIGEN_COMMIT}
image_sha256=${BASE_IMAGE_SHA}
image_file=$(basename "${COMPRESSED}")
EOF

log "MANIFEST skriven: ${MANIFEST_FILE}"

log ""
log "Build klar."
log "  Image:      ${COMPRESSED}"
log "  SHA256:     ${CHECKSUM_FILE}"
log "  MANIFEST:   ${MANIFEST_FILE}"
log "  Storlek:    $(du -h "${COMPRESSED}" | cut -f1)"
log ""
log "Flasha till SD-kort med:"
log "  rpi-imager --cli ${COMPRESSED} /dev/sdX"
log "  eller: xzcat ${COMPRESSED} | dd of=/dev/sdX bs=4M status=progress"

# ---------------------------------------------------------------------------
# 11. Städning av temporär arbetskatalog
# ---------------------------------------------------------------------------

# Ta bort symlinken vi skapade i pi-gen
rm -f "${PIGEN_STAGE_LINK}"

# Återställ SKIP_IMAGES för stage2 om det var vi som lade dit den
# (om den redan fanns är det ett pi-gen-default; lämna kvar)
rm -f "${PIGEN_DIR}/stage3/SKIP" "${PIGEN_DIR}/stage3/SKIP_IMAGES" 2>/dev/null || true
rm -f "${PIGEN_DIR}/stage4/SKIP" "${PIGEN_DIR}/stage4/SKIP_IMAGES" 2>/dev/null || true
rm -f "${PIGEN_DIR}/stage5/SKIP" "${PIGEN_DIR}/stage5/SKIP_IMAGES" 2>/dev/null || true

if [[ "${KEEP_WORK}" != "1" ]]; then
    log "Tar bort temporär arbetsdir: ${WORK_DIR}"
    rm -rf "${WORK_DIR}"
else
    log "KEEP_WORK=1 — arbetsdir bevarad: ${WORK_DIR}"
fi

exit 0
