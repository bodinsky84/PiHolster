#!/usr/bin/env bash
# pre-release-check.sh — lokal preflight innan v0.1.0 GA-tagging.
#
# Kör allt CI gör + extra GA-specifika kontroller. Misslyckas tidigt om något
# är fel, så vi inte upptäcker problem först när release-image.yml triggas
# av en pushad tag.
#
# Det här scriptet är trafikljuset för CTO/PM innan US-27 (taggning).
# Detaljerad GA-checklista: docs/GA-GATE-CHECKLIST.md
#
# Användning:
#   bash scripts/pre-release-check.sh
#
# Exit codes:
#   0  alla automatiserbara kontroller PASS
#   1  minst en kontroll FAIL (se utskriften)
#   2  förutsättning saknas (verktyg, repo-state)

set -uo pipefail

# ---------------------------------------------------------------------------
# Konstanter och hjälpare
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${REPO_ROOT}"

# Räknare
checks_pass=0
checks_fail=0
checks_skip=0
failed_checks=()

# Färger om vi har en TTY
if [[ -t 1 ]]; then
    C_RED=$'\033[31m'
    C_GREEN=$'\033[32m'
    C_YELLOW=$'\033[33m'
    C_BLUE=$'\033[34m'
    C_RESET=$'\033[0m'
else
    C_RED=""
    C_GREEN=""
    C_YELLOW=""
    C_BLUE=""
    C_RESET=""
fi

section() {
    echo ""
    echo "${C_BLUE}=== $* ===${C_RESET}"
}

pass() {
    echo "  ${C_GREEN}PASS${C_RESET} $*"
    checks_pass=$((checks_pass + 1))
}

fail() {
    echo "  ${C_RED}FAIL${C_RESET} $*"
    checks_fail=$((checks_fail + 1))
    failed_checks+=("$*")
}

skip() {
    echo "  ${C_YELLOW}SKIP${C_RESET} $*"
    checks_skip=$((checks_skip + 1))
}

info() {
    echo "  $*"
}

# Kör ett kommando och rapportera PASS/FAIL utifrån exit-koden.
# Användning: run_check "beskrivning" -- kommando args ...
run_check() {
    local desc="$1"
    shift
    if [[ "${1:-}" == "--" ]]; then
        shift
    fi
    if "$@" > /tmp/pre-release-check.log 2>&1; then
        pass "${desc}"
        return 0
    else
        fail "${desc}"
        echo "    ↳ logg: /tmp/pre-release-check.log (sista 10 raderna):"
        tail -10 /tmp/pre-release-check.log | sed 's/^/      /'
        return 1
    fi
}

# ---------------------------------------------------------------------------
# 1. Förutsättningar — verktyg och repo-state
# ---------------------------------------------------------------------------

section "1. Förutsättningar"

# Git tree måste vara rent
if git diff --quiet && git diff --cached --quiet; then
    pass "git working tree är rent"
else
    fail "git working tree har okommittade ändringar — committa eller stash:a först"
    git status --short | sed 's/^/    /'
fi

# Branch måste vara master eller release/*
branch="$(git rev-parse --abbrev-ref HEAD)"
case "${branch}" in
    master|release/*)
        pass "branch '${branch}' är ok för release"
        ;;
    *)
        fail "branch '${branch}' — release ska göras från master eller release/*"
        ;;
esac

# Verktyg som måste finnas
for tool in go bash sha256sum; do
    if command -v "${tool}" > /dev/null 2>&1; then
        pass "${tool} hittades ($(command -v "${tool}"))"
    else
        fail "${tool} saknas i PATH"
    fi
done

# Optional verktyg
for tool in pnpm node; do
    if command -v "${tool}" > /dev/null 2>&1; then
        info "(optional) ${tool} hittades — frontend-checks körs"
    else
        info "(optional) ${tool} saknas — frontend-checks hoppas över"
    fi
done

# ---------------------------------------------------------------------------
# 2. Backend — go vet + go test
# ---------------------------------------------------------------------------

section "2. Backend (Go)"

run_check "go vet ./apps/piholsterd/..." -- \
    go vet ./apps/piholsterd/...

run_check "go vet ./apps/piholster-arpd/..." -- \
    go vet ./apps/piholster-arpd/...

# Tester körs utan -race på Windows (race detector kräver CGO som inte alltid
# är konfigurerat på developer-maskiner). CI kör med -race.
run_check "go test ./apps/piholsterd/..." -- \
    go test -count=1 ./apps/piholsterd/...

run_check "go test ./apps/piholster-arpd/..." -- \
    go test -count=1 ./apps/piholster-arpd/...

# ---------------------------------------------------------------------------
# 3. Smoke-tester (compile-only utan PI_IP)
# ---------------------------------------------------------------------------

section "3. Smoke-tester"

if [[ -d "tests/smoke" ]]; then
    # PI_IP unset → TestMain bailar tidigt; vi vill bara verifiera att paketet
    # kompilerar utan fel.
    run_check "go vet ./tests/smoke/..." -- \
        go vet ./tests/smoke/...

    run_check "tests/smoke kompilerar (compile-only utan PI_IP)" -- \
        env -u PI_IP go test -run='^$' ./tests/smoke/...
else
    skip "tests/smoke saknas"
fi

# ---------------------------------------------------------------------------
# 4. Cross-compile (linux/arm/v7) — så vi vet att Pi-byggen funkar
# ---------------------------------------------------------------------------

section "4. Cross-compile linux/arm/v7"

# Vi bygger till /dev/null på Unix; på Windows skriver vi till en temp-fil
# som vi raderar efteråt.
tmp_bin="$(mktemp)"
trap 'rm -f "${tmp_bin}"' EXIT

if GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "${tmp_bin}" \
    ./apps/piholsterd/cmd/piholsterd > /tmp/pre-release-check.log 2>&1; then
    pass "piholsterd cross-compilerar för linux/arm/v7"
else
    fail "piholsterd cross-compile för linux/arm/v7"
    tail -10 /tmp/pre-release-check.log | sed 's/^/      /'
fi

if GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w" -o "${tmp_bin}" \
    ./apps/piholster-arpd/cmd/piholster-arpd > /tmp/pre-release-check.log 2>&1; then
    pass "piholster-arpd cross-compilerar för linux/arm/v7"
else
    fail "piholster-arpd cross-compile för linux/arm/v7"
    tail -10 /tmp/pre-release-check.log | sed 's/^/      /'
fi

# ---------------------------------------------------------------------------
# 5. Bash-syntax på shell-scripts
# ---------------------------------------------------------------------------

section "5. Bash-syntax"

# Hitta alla *.sh och kör 'bash -n' på dem.
shellcheck_failed=0
while IFS= read -r script; do
    if bash -n "${script}" 2>/tmp/pre-release-check.log; then
        pass "bash -n ${script}"
    else
        fail "bash -n ${script}"
        cat /tmp/pre-release-check.log | sed 's/^/      /'
        shellcheck_failed=1
    fi
done < <(find scripts image -name '*.sh' -type f 2>/dev/null | sort)

# ---------------------------------------------------------------------------
# 6. MANIFEST format regression test (US-25 / M-04)
# ---------------------------------------------------------------------------

section "6. MANIFEST format regression"

if [[ -f "image/manifest_format_test.sh" ]]; then
    run_check "MANIFEST-fält i build.sh matchar MANIFEST.example" -- \
        bash image/manifest_format_test.sh
else
    fail "image/manifest_format_test.sh saknas — krävs av US-25"
fi

# ---------------------------------------------------------------------------
# 7. Frontend (om pnpm finns)
# ---------------------------------------------------------------------------

section "7. Frontend"

if command -v pnpm > /dev/null 2>&1; then
    if [[ -f "pnpm-lock.yaml" ]]; then
        run_check "pnpm install --frozen-lockfile" -- \
            pnpm install --frozen-lockfile

        if [[ $? -eq 0 ]]; then
            run_check "pnpm --filter web lint" -- \
                pnpm --filter web lint

            run_check "pnpm --filter web build" -- \
                pnpm --filter web build
        fi
    else
        skip "pnpm-lock.yaml saknas"
    fi
else
    skip "pnpm saknas — frontend-checks hoppas över (CI kör dem)"
fi

# ---------------------------------------------------------------------------
# 8. GA-dokumentation finns
# ---------------------------------------------------------------------------

section "8. GA-dokumentation"

required_docs=(
    "RELEASE_NOTES.md"
    "SECURITY.md"
    "README.md"
    "docs/SOAK-RUNBOOK.md"
    "docs/BETA-FEEDBACK.md"
    "docs/SMOKE-MATRIX.md"
    "docs/GA-GATE-CHECKLIST.md"
    "docs/BACKUP.md"
    "docs/INSTALL-PI.md"
    "image/MANIFEST.example"
)

for doc in "${required_docs[@]}"; do
    if [[ -f "${doc}" ]]; then
        pass "${doc} finns"
    else
        fail "${doc} saknas"
    fi
done

# RELEASE_NOTES måste innehålla v0.1.0
if [[ -f "RELEASE_NOTES.md" ]]; then
    if grep -q "v0.1.0" RELEASE_NOTES.md; then
        pass "RELEASE_NOTES.md innehåller 'v0.1.0'"
    else
        fail "RELEASE_NOTES.md saknar 'v0.1.0'-avsnitt"
    fi
fi

# package.json version måste vara 0.1.0
pkg_version="$(grep '"version"' package.json | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"
if [[ "${pkg_version}" == "0.1.0" ]]; then
    pass "package.json version=0.1.0"
else
    fail "package.json version='${pkg_version}', förväntat '0.1.0'"
fi

# ---------------------------------------------------------------------------
# 9. GA-blockers — manuell checklista (kan inte automatiseras)
# ---------------------------------------------------------------------------

section "9. GA-blockers (manuell sign-off krävs)"

cat <<'EOF'
  Följande kan INTE automatiseras och måste verifieras manuellt enligt
  docs/GA-GATE-CHECKLIST.md innan v0.1.0-taggen pushas:

    [ ] US-19  7-dagars soak utan omstart           (CTO sign-off)
    [ ] US-20  Beta-testning, minst 2/3 lyckas      (PM sign-off)
    [ ] US-21  Strömavbrottstest 3/3 PASS           (CTO sign-off)
    [ ] US-23  M-05/L-03 granskat av IT-säk         (IT-säk sign-off)
    [ ] US-25  M-04 MANIFEST-granskning             (IT-säk sign-off)
    [ ] US-26  Smoke på 3 olika Pi 3-enheter        (IT-säk sign-off)

  Bägge "GA APPROVED"-kommentarer (CTO + IT-säk) måste finnas i
  release/v0.1.0-ga-PR:en innan US-27 (taggning) påbörjas.
EOF

# ---------------------------------------------------------------------------
# 10. Sammanfattning
# ---------------------------------------------------------------------------

section "Sammanfattning"

total=$((checks_pass + checks_fail + checks_skip))
echo "  Totalt automatiserade kontroller: ${total}"
echo "  ${C_GREEN}PASS${C_RESET}: ${checks_pass}"
echo "  ${C_YELLOW}SKIP${C_RESET}: ${checks_skip}"
echo "  ${C_RED}FAIL${C_RESET}: ${checks_fail}"

if [[ ${checks_fail} -gt 0 ]]; then
    echo ""
    echo "${C_RED}Misslyckade kontroller:${C_RESET}"
    for f in "${failed_checks[@]}"; do
        echo "  - ${f}"
    done
    echo ""
    echo "${C_RED}Pre-release-check FAIL.${C_RESET} Åtgärda ovan innan US-27 (taggning)."
    exit 1
fi

echo ""
echo "${C_GREEN}Alla automatiserbara pre-release-checks PASS.${C_RESET}"
echo "Nästa steg: bekräfta de manuella GA-blockerna i sektion 9, sedan"
echo "kör 'bash scripts/release.sh v0.1.0' när bägge sign-offs finns."
exit 0
