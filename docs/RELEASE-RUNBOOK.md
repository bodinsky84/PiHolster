# Release-runbook för v0.1.0 GA

Steg-för-steg-instruktion för DevOps att tagga, bygga och publicera v0.1.0 GA.
Den här runbooken används **istället för** `release-image.yml` (som är `if: false`
för v0.1.0 — se workflow-header) eftersom CI-pipelinen för image-bygget inte är
färdigautomatiserad. Re-aktivering planerad till v0.1.1.

**Förutsätter:** GA-gate är passerad (CTO + IT-säk har postat "GA APPROVED" i
`release/v0.1.0-ga`-PR:en). Se `docs/GA-GATE-CHECKLIST.md`.

---

## 0. Förkrav

| Krav | Verifiering |
|------|-------------|
| GA-gate sign-off från CTO och IT-säk | Kommentarer i `release/v0.1.0-ga`-PR:en |
| Lokal build-host (Debian/Ubuntu/Pi OS) med rot-rättighet | `xz`, `sha256sum`, `rsync`, `git` installerat |
| Go ≥ 1.22 | `go version` |
| pnpm ≥ 9 | `pnpm --version` |
| `pi-gen`-submodulen initierad | `git submodule status image/pi-gen` ger ingen `-`-prefix |
| `minisign` installerat | `minisign -v` |
| Minisign-signeringsnyckel (`minisign.key`) tillgänglig och lösenord känt | DevOps personlig nyckel; **dela aldrig** |
| `gh` CLI inloggad mot bodinsky84/PiHolster | `gh auth status` |
| Master är fast-forwardable till GA-gate-commiten | `git log master --oneline -1` |

Om något saknas: avbryt här och åtgärda. Att starta runbooken halvvägs är
felkälla nr 1.

---

## 1. Pre-flight (lokalt)

Kör den automatiserade preflighten på master:

```bash
git checkout master
git pull --ff-only
bash scripts/pre-release-check.sh
```

Förväntat: `42/42 PASS`. Om något FAIL — fixa innan du fortsätter.

---

## 2. Tagga och pusha — triggar binär-release i CI

```bash
# Annoterad signerad tag om GPG-nyckel finns:
git tag -s v0.1.0 -m "v0.1.0 GA"

# Eller annoterad utan signering:
# git tag -a v0.1.0 -m "v0.1.0 GA"

git push origin v0.1.0
```

Detta triggar `.github/workflows/release-binary.yml` som:

1. Bygger frontend (`pnpm --filter web build`)
2. Cross-kompilerar `piholsterd` och `piholster-arpd` för
   `linux/arm/v7`, `linux/arm64`, `linux/amd64`
3. Skapar en **draft** GitHub Release `v0.1.0` och laddar upp 6 binärer

**Vänta** tills jobbet är grönt:

```bash
gh run watch --workflow="Release — Binary"
```

Verifiera att alla 6 binärer finns:

```bash
gh release view v0.1.0 --json assets --jq '.assets[].name'
```

Ska innehålla `piholsterd-linux-{arm-v7,arm64,amd64}` och
`piholster-arpd-linux-{arm-v7,arm64,amd64}`.

---

## 3. Bygg SD-card-imagen lokalt

Imagen byggs på en Debian/Pi OS-host (kan vara CTO:s Pi 4 eller en
x86-Debian-VM). pi-gen kräver root, chroot och loop-mount.

```bash
# 1. Hämta de cross-compilerade binärerna från release:
mkdir -p /tmp/piholster-bins
cd /tmp/piholster-bins
gh release download v0.1.0 -R bodinsky84/PiHolster \
  --pattern 'piholsterd-linux-arm64' \
  --pattern 'piholster-arpd-linux-arm64'
chmod +x piholsterd-linux-arm64 piholster-arpd-linux-arm64

# 2. Kör build.sh:
cd ~/piholster   # din lokala kopia
git checkout v0.1.0
git submodule update --init image/pi-gen

sudo PIHOLSTER_BINARY=/tmp/piholster-bins/piholsterd-linux-arm64 \
     PIHOLSTER_ARPD_BINARY=/tmp/piholster-bins/piholster-arpd-linux-arm64 \
     bash image/build.sh
```

Output hamnar i `dist/`:

- `piholster-0.1.0-YYYY-MM-DD.img.xz`
- `piholster-0.1.0-YYYY-MM-DD.img.xz.sha256`
- `MANIFEST`

**Verifiera MANIFEST** — alla fält ska vara ifyllda, ingen `unknown`:

```bash
cat dist/MANIFEST
```

Förväntade nycklar: `build_timestamp`, `image_name`, `piholster_version`,
`repo_commit`, `pigen_commit`, `piholsterd_sha256`, `piholster_arpd_sha256`,
`image_sha256`, `image_file`.

---

## 4. Signera imagen med minisign

```bash
cd dist
minisign -Sm piholster-0.1.0-*.img.xz -s ~/.minisign/minisign.key \
  -t "PiHolster v0.1.0 GA — built $(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

Output: `piholster-0.1.0-*.img.xz.minisig`.

**Verifiera signaturen** med public key (publicerad i `README.md`):

```bash
minisign -Vm piholster-0.1.0-*.img.xz \
  -P "$(cat ../docs/minisign.pub | tail -1)"
```

Förväntat: `Signature and comment signature verified`.

---

## 5. Verifiera SHA256 mot MANIFEST

`build.sh` skriver `image_sha256` i MANIFEST. Verifiera att `.sha256`-filen och
MANIFEST är konsistenta:

```bash
diff <(awk '{print $1}' dist/piholster-0.1.0-*.img.xz.sha256) \
     <(grep '^image_sha256=' dist/MANIFEST | cut -d= -f2)
```

Förväntat: ingen output (= matchar).

---

## 6. Ladda upp image-filerna till release

```bash
cd dist
gh release upload v0.1.0 \
  piholster-0.1.0-*.img.xz \
  piholster-0.1.0-*.img.xz.sha256 \
  piholster-0.1.0-*.img.xz.minisig \
  MANIFEST \
  -R bodinsky84/PiHolster
```

Verifiera att alla 4 nya filer finns i releasen tillsammans med de 6
binärerna från steg 2:

```bash
gh release view v0.1.0 --json assets --jq '.assets[].name' | sort
```

Förväntat (10 filer):

```
MANIFEST
piholster-0.1.0-YYYY-MM-DD.img.xz
piholster-0.1.0-YYYY-MM-DD.img.xz.minisig
piholster-0.1.0-YYYY-MM-DD.img.xz.sha256
piholster-arpd-linux-amd64
piholster-arpd-linux-arm-v7
piholster-arpd-linux-arm64
piholsterd-linux-amd64
piholsterd-linux-arm-v7
piholsterd-linux-arm64
```

---

## 7. Editera release-bodyn med RELEASE_NOTES-avsnittet

```bash
# Extrahera v0.1.0-avsnittet ur RELEASE_NOTES.md (allt från första
# "## v0.1.0" till nästa "## " eller filslut):
awk '/^## v0\.1\.0/{flag=1; print; next} /^## /{flag=0} flag' \
  RELEASE_NOTES.md > /tmp/release-body.md

gh release edit v0.1.0 --notes-file /tmp/release-body.md \
  -R bodinsky84/PiHolster
```

Granska resultatet på https://github.com/bodinsky84/PiHolster/releases/tag/v0.1.0
— rendering av markdown och länkar.

---

## 8. Publicera releasen (avmarkera draft)

```bash
gh release edit v0.1.0 --draft=false -R bodinsky84/PiHolster
```

Releasen är nu publik.

---

## 9. Uppdatera README.md med "Latest release: v0.1.0"

```bash
git checkout master
# Editera README.md — lägg till eller uppdatera badge/sektion:
#   ## Latest release
#   v0.1.0 — https://github.com/bodinsky84/PiHolster/releases/tag/v0.1.0

git add README.md
git commit -m "docs: link to v0.1.0 release in README"
git push origin master
```

---

## 10. Stäng GA-gate-PR:en

På GitHub: gå till `release/v0.1.0-ga`-PR:en, postera kommentar:

```
v0.1.0 taggad — <commit-sha>
GitHub Release: https://github.com/bodinsky84/PiHolster/releases/tag/v0.1.0
Sprint 4 klar.
```

Stäng PR:en utan att merga (den är ett samlingsdokument, inte en kodändring).

---

## 11. Uppdatera memory + arkivera SPRINT-4-dokumentet

PM ansvarar för att:

- Uppdatera `MEMORY.md` (eller motsvarande projektmemory) med v0.1.0-status
- Skapa `docs/SPRINT-5.md`-skelett om en post-GA-sprint är planerad
- Lägga till GA-datum i `docs/ROADMAP.md`

---

## Rollback-procedur

Om steg 6–8 visar att imagen är trasig (smoke fail, MANIFEST fel, signatur fel):

```bash
# 1. Markera releasen som draft igen (eller radera helt):
gh release edit v0.1.0 --draft=true -R bodinsky84/PiHolster
# eller hård rollback:
gh release delete v0.1.0 --yes -R bodinsky84/PiHolster

# 2. Ta bort taggen lokalt och remote:
git tag -d v0.1.0
git push origin :refs/tags/v0.1.0

# 3. Fixa, bumpa till rc2, kör om från steg 1.
```

Notera: GA-gate-sign-off förblir giltig så länge ingen kodändring krävs.
Om en bugg i imagen kräver kodändring → ny GA-gate behövs (CTO + IT-säk
sign-off om).

---

## Tidsåtgång

Erfarenhetsmässigt (första gången):

| Steg | Tid |
|------|-----|
| 1. Pre-flight | 5 min |
| 2. Tag + binär-build i CI | 10 min |
| 3. Image-build på Pi/Debian | 30–60 min (pi-gen är långsamt) |
| 4. Minisign | 1 min |
| 5–6. Upload | 2 min |
| 7–8. Body + publish | 3 min |
| 9–10. README + PR-close | 5 min |

Total: ca 60–90 min. Reservera 2 timmar i kalendern.
