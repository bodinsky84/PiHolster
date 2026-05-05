# GA-gate-checklista för v0.1.0

PM-mall för `release/v0.1.0-ga`-PR:en. Öppnas dag 14 av sprinten när alla US-19
till US-26 är klara. Ingen kod mergas i denna PR — den fungerar som
samlingsdokument för CTO- och IT-säk-sign-off. Merges först efter att US-27
(taggning) är klar.

Detaljerad spec finns i `docs/SPRINT-4.md` § "GA-gate".

---

## PR-skelett — kopiera till PR-bodyn

```markdown
# v0.1.0 GA-gate

Detta är samlings-PR:en för v0.1.0 GA-sign-off. Ingen kod mergas här.
US-27 (taggning) får inte påbörjas förrän både CTO och IT-säkerhet har
postat "GA APPROVED"-kommentar nedan.

Spec: docs/SPRINT-4.md § "GA-gate"

## Bilagor

### US-19 — 7-dagars soak (CTO)
- Soak-rapport (3–5 meningar): se kommentar nedan
- RAM-graf: `soak-rc1-ram.png` (bilaga)
- RAW CSV: `soak-rc1.csv` (bilaga)
- Cert-expiry verifierat: YYYY-MM-DD (10 år från firstboot)

### US-20 — Beta-testning (PM)
- Sammanställning: `docs/BETA-FEEDBACK.md` (PM-sammanställningssektionen ifylld)
- Antal testare: N/M lyckades
- Showstoppers åtgärdade: lista PR-länkar / inga showstoppers

### US-21 — WAL och backup/restore (Senior Go-dev)
- PRAGMA-output: `journal_mode=wal`, `synchronous=NORMAL`
- Strömavbrottstest: 3/3 PASS — `integrity_check` returnerade `ok` alla gånger
- Backup/restore-flöde verifierat manuellt: PASS

### US-22 — RELEASE_NOTES.md
- Granskat av CTO: ✓
- Innehåller M-05, L-03, "ingen WiFi", "ingen one-click update", "ingen cert-renewal"
- Soak/beta-fynd dokumenterade som "kända begränsningar"

### US-23 — SECURITY.md
- M-05-formulering granskad och godkänd av IT-säk: se kommentar nedan
- L-03-formulering: ✓
- Responsible Disclosure-policy: ✓

### US-24 — TestHTTPSRedirect
- Test mergat: commit ___________
- Bevis på att testet kan misslyckas: bifogad CI-körning eller skärmdump

### US-25 — M-04 MANIFEST-granskning
- IT-säk-utfall: se kommentar nedan
- `image/MANIFEST` innehåller: pi-gen commit, base-image SHA256, piholsterd_sha256,
  piholster_arpd_sha256, repo_commit, build-timestamp

### US-26 — Smoke 3 Pi-enheter
- Resultatmatris: `docs/SMOKE-MATRIX.md` (ifylld)
- Loggar: `smoke-results-A.log`, `-B.log`, `-C.log` (bilagda)
- IT-säk-sign-off: se kommentar nedan

## Kvarstående
- [ ] CTO sign-off (krav nedan)
- [ ] IT-säk sign-off (krav nedan)
- [ ] US-27 taggning (efter sign-off)
```

---

## CTO sign-off — krav

Alla rader ska vara `OK` innan CTO postar "GA APPROVED".

| Krav | Story | Verifiering | Status |
|------|-------|-------------|--------|
| 7-dagars soak utan omstart, ingen minnesläcka | US-19 | RAM-graf + soak-rapport | OK / NOK |
| Minst 2 av 3 (eller 3 av 5) beta-testare lyckades | US-20 | Beta-sammanfattning | OK / NOK |
| WAL-konsistens klarar 3 strömavbrott | US-21 | Testresultat i PR | OK / NOK |
| RELEASE_NOTES korrekt och fullständig | US-22 | CTO-läsning | OK / NOK |
| Cert-renewal-path verifierad ELLER dokumenterad som känd begränsning | US-19 §5 | Soak-rapport + RELEASE_NOTES | OK / NOK |

**CTO-kommentar — klistra exakt:**
```
GA APPROVED — CTO — YYYY-MM-DD
Alla krav i GA-gate är uppfyllda. OK att tagga v0.1.0.
```

---

## IT-säk sign-off — krav

| Krav | Story | Verifiering | Status |
|------|-------|-------------|--------|
| M-05 dokumenterat i SECURITY.md med korrekt riskbedömning | US-23 | PR-läsning | OK / NOK |
| M-04 verifierat — MANIFEST-generering godkänd | US-25 | Granskning av build.sh + rc1 MANIFEST | OK / NOK |
| Smoke-sviten (inkl. TestHTTPSRedirect) grön på 3 Pi 3-enheter | US-24 + US-26 | Tabell + loggar i PR | OK / NOK |

**IT-säk-kommentar — klistra exakt:**
```
GA APPROVED — IT-säk — YYYY-MM-DD
Alla krav i GA-gate är uppfyllda. OK att tagga v0.1.0.
```

---

## När båda kommentarer finns — gå till US-27

Steg-för-steg:

```bash
# 1. Verifiera att master är på den commit som ska taggas
git checkout master
git pull --ff-only
git log -1 --oneline

# 2. Skapa annoterad signerad tag (signering om GPG-nyckel finns)
git tag -s v0.1.0 -m "v0.1.0 GA"
# Eller utan signering om ingen GPG-nyckel:
# git tag -a v0.1.0 -m "v0.1.0 GA"

# 3. Pusha taggen — detta triggar release-image.yml automatiskt
git push origin v0.1.0
```

GitHub Actions bygger imagen från den taggade commiten, kör smoke-sviten i CI
och laddar upp `.img.xz`, `.img.xz.sha256` och `.img.xz.minisig` till en draft
GitHub Release.

PM:
1. Editar release-bodyn med v0.1.0-avsnittet ur `RELEASE_NOTES.md`.
2. Publicerar releasen (avmarkerar "draft").
3. Uppdaterar `README.md` med "Latest release: v0.1.0" och länk.
4. Stänger GA-gate-PR:en med kommentar `v0.1.0 taggad <SHA>`.

Sprint 4 klar.
