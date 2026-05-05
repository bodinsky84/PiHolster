# IT-säkerhetsgranskning — MANIFEST-generering (US-25 / M-04)

**Granskad komponent:** `image/build.sh` — MANIFEST-genereringen (avsnitt 10)  
**Granskad version:** v0.1.0-rc1  
**Datum:** 2026-05-03  
**Granskare:** IT-säkerhet (PiHolster)  
**Status:** GODKÄNT med åtgärdat fynd

---

## Syfte

Granska att `image/MANIFEST` genereras automatiskt, innehåller tillräcklig
information för image-integritetskontroll och dokumenterar supply chain så att
en granskare kan verifiera att ingen oauktoriserad komponent ingår i imagen.

---

## Granskningspunkter

### 1. Är pi-gen commit-hash med och korrekt?

**Före fix:** Ja. `pigen_commit` sätts via `git rev-parse HEAD` i pi-gen-underkatalogen.  
**Bedömning:** OK.

### 2. Är base-image SHA256 med och verifierbar?

**Före fix:** Ja. `image_sha256` innehåller SHA256 av den komprimerade `.img.xz`-filen,
hämtad från den genererade `.sha256`-filen.  
**Bedömning:** OK. Notera att det är SHA256 av `.img.xz`, inte av den råa `.img`-filen —
det är korrekt eftersom `.img.xz` är den distribuerade artefakten.

### 3. Är piholsterd- och piholster-arpd-versioner med?

**Före fix:** **NEJ** — detta var det enda kritiska fyndet.

MANIFEST innehöll `piholster_version` (läst ur `package.json`) men **ingen
SHA256-checksumma för de enskilda binärerna** `piholsterd` och `piholster-arpd`.
Det innebar att en angripare som kunde ersätta binärerna i `stage-piholster/00-install/files/`
(t.ex. via en komprometterad CI-miljö) inte skulle synas i MANIFEST — `image_sha256`
täcker hela imagen men kräver att man monterar och extraherar den för att verifiera
individuella filer.

**Åtgärd (implementerad):** `build.sh` uppdaterat. Fälten `piholsterd_sha256` och
`piholster_arpd_sha256` beräknas nu med `sha256sum` på binärerna i `FILES_DIR`
*innan* de bakas in i imagen, och skrivs till MANIFEST.

Även `repo_commit` lades till för att fästa MANIFEST vid en specifik
repo-commit.

**Bedömning efter fix:** OK.

### 4. Är build-timestamp med?

**Bedömning:** Ja. `build_timestamp` sätts till UTC ISO 8601-format. OK.

### 5. Genereras MANIFEST automatiskt eller finns risk för manuella misstag?

**Bedömning:** MANIFEST genereras helt automatiskt av `build.sh` och skrivs
till `$OUT_DIR/MANIFEST`. Det finns ingen manuell inmatning i processen.

`build.sh` kör med `set -euo pipefail` — alla kommandon som misslyckas
avbryter bygget. Om `sha256sum` eller `git rev-parse` misslyckas avbryts
bygget och inget MANIFEST genereras.

Enda riskpunkt: om `package.json` saknas läser versionen "dev" — detta är
avsiktligt och dokumenterat i scriptet. OK.

---

## Fynd

| ID   | Allvarlighet | Beskrivning | Status |
|------|-------------|-------------|--------|
| M-04-1 | Medium | Individuella binär-SHA256 (piholsterd, piholster-arpd) saknades i MANIFEST | **Åtgärdat** i build.sh |
| M-04-2 | Info | repo-commit saknades — gör det svårare att koppla MANIFEST till källkod | **Åtgärdat** i build.sh |

---

## MANIFEST-format efter fix

```
# PiHolster Image MANIFEST
# Autogenererat av image/build.sh — ändra inte manuellt

build_timestamp=2026-05-03T12:00:00Z
image_name=piholster-0.1.0-2026-05-03
piholster_version=0.1.0
repo_commit=<git-sha>
pigen_commit=<git-sha>
piholsterd_sha256=<sha256>
piholster_arpd_sha256=<sha256>
image_sha256=<sha256>
image_file=piholster-0.1.0-2026-05-03.img.xz
```

---

## Utfall

M-04-fyndet är åtgärdat. MANIFEST-genereringen är nu godkänd för v0.1.0.

**Sign-off ges i GA-gate-PR med texten:**
```
M-04 verifierat — MANIFEST-generering godkänd. [IT-säk initialer] — [datum]
```
