# v0.1.0 GA-gate

Detta är samlings-PR:en för v0.1.0 GA-sign-off. Ingen kod mergas här.
US-27 (taggning) får inte påbörjas förrän både CTO och IT-säkerhet har
postat "GA APPROVED"-kommentar i PR:en.

Spec: `docs/SPRINT-4.md` § "GA-gate"
Mall: `docs/GA-GATE-CHECKLIST.md`

PM ansvarar för att hålla detta dokument uppdaterat under sprinten.
Fyll i fält efterhand som varje story landar — det blir den slutliga
PR-bodyn när release/v0.1.0-ga öppnas mot master.

---

## Bilagor

### US-19 — 7-dagars soak (CTO)

- Soak-start (firstboot-datum): _________
- Soak-slut (dag 7): _________
- Pi-modell / serienummer: _________
- SD-kort: _________
- Soak-rapport (3–5 meningar): _se kommentar i PR:en_
- RAM-graf: `soak-rc1-ram.png` (bilaga, laddas upp i PR:en)
- RAW CSV: `soak-rc1.csv` (bilaga)
- Cert-expiry verifierat: _________ (10 år från firstboot)
- Cert-renewal-path: [ ] verifierad / [ ] dokumenterad som känd begränsning i RELEASE_NOTES

### US-20 — Beta-testning (PM)

- Sammanställning: `docs/BETA-FEEDBACK.md` (PM-sektionen ifylld före GA-gate)
- Antal testare rekryterade: _ / _
- Antal som lyckades självständigt: _ / _
- Showstoppers åtgärdade: _lista PR-länkar, eller "inga showstoppers"_

### US-21 — WAL och backup/restore (Senior Go-dev)

- PRAGMA-output: `journal_mode=wal`, `synchronous=NORMAL` — bekräftad
- Strömavbrottstest: _ / 3 PASS — `integrity_check` returnerade `ok` alla gånger
- Backup-flöde testat (scp av `piholster.db` enligt `docs/BACKUP.md`): [ ] PASS
- Restore-flöde testat (stop, ersätt db, start, verifiera config): [ ] PASS

### US-22 — RELEASE_NOTES.md

- [ ] Granskat av CTO
- [ ] Innehåller M-05, L-03, "ingen WiFi", "ingen one-click update", cert-renewal-status
- [ ] Soak/beta-fynd dokumenterade som "kända begränsningar"
- [ ] Versionsdatum uppdaterat till GA-dagen

### US-23 — SECURITY.md

- [ ] M-05-formulering granskad och godkänd av IT-säk (kommentar i PR)
- [ ] L-03-formulering: ✓
- [ ] Responsible Disclosure-policy: ✓

### US-24 — TestHTTPSRedirect

- Test mergat i: `3960736` (HTTP -> HTTPS) och utbyggd i `tests/smoke/smoke_test.go`
- Bevis på att testet kan misslyckas: _bilägg CI-logg från test-run där redirect tillfälligt inaktiverats_

### US-25 — M-04 MANIFEST-granskning

- IT-säk-utfall: _se `docs/it-security-review-us25.md` + kommentar i PR:en_
- `image/MANIFEST.example` täcker: pi-gen commit, base-image SHA256,
  piholsterd_sha256, piholster_arpd_sha256, repo_commit, build-timestamp
- Format-regression-test: `image/manifest_format_test.sh` — körs i CI (`f53a446`)

### US-26 — Smoke 3 Pi-enheter

- Resultatmatris: `docs/SMOKE-MATRIX.md` (ifylld)
- Loggar: `smoke-results-A.log`, `smoke-results-B.log`, `smoke-results-C.log` (bilagor)
- IT-säk-sign-off: _se kommentar i PR:en_

---

## CTO sign-off — krav

| Krav | Story | Status |
|------|-------|--------|
| 7-dagars soak utan omstart, ingen minnesläcka | US-19 | OK / NOK |
| Minst 2 av 3 (eller 3 av 5) beta-testare lyckades | US-20 | OK / NOK |
| WAL-konsistens klarar 3 strömavbrott | US-21 | OK / NOK |
| RELEASE_NOTES korrekt och fullständig | US-22 | OK / NOK |
| Cert-renewal-path verifierad ELLER dokumenterad som känd begränsning | US-19 §5 | OK / NOK |

CTO klistrar exakt detta som kommentar när alla rader är OK:

```
GA APPROVED — CTO — YYYY-MM-DD
Alla krav i GA-gate är uppfyllda. OK att tagga v0.1.0.
```

---

## IT-säk sign-off — krav

| Krav | Story | Status |
|------|-------|--------|
| M-05 dokumenterat i SECURITY.md med korrekt riskbedömning | US-23 | OK / NOK |
| M-04 verifierat — MANIFEST-generering godkänd | US-25 | OK / NOK |
| Smoke-sviten (inkl. TestHTTPSRedirect) grön på 3 Pi 3-enheter | US-24 + US-26 | OK / NOK |

IT-säk klistrar exakt detta som kommentar när alla rader är OK:

```
GA APPROVED — IT-säk — YYYY-MM-DD
Alla krav i GA-gate är uppfyllda. OK att tagga v0.1.0.
```

---

## Kvarstående

- [ ] CTO sign-off
- [ ] IT-säk sign-off
- [ ] US-27 taggning (efter sign-off)
- [ ] GitHub Release publicerad med `.img.xz`, `.img.xz.sha256`, `.img.xz.minisig`
- [ ] README.md uppdaterad med "Latest release: v0.1.0"

---

## Pre-release-check (automatisk preflight)

Senaste körning av `bash scripts/pre-release-check.sh` ska bifogas eller
klistras in nedan av PM precis innan US-27.

```
Datum: _________
Resultat: __ / __ PASS
```
