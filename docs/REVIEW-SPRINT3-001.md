# Code Review — Sprint 3

**Reviewer:** IT-säkerhet + Code Review
**Datum:** 2026-05-03
**Branch:** develop (Sprint 3-leverans)
**Filer granskade:** image/build.sh, stage-piholster/00-install/*, stage-piholster/01-firstboot/*, stage-piholster/02-harden/*, apps/piholsterd/internal/api/static.go, apps/web/svelte.config.js, Makefile

**Utfall: GODKÄND MED BLOCKERS** — 5 blockers och 4 major-findings korrigerades av Senior Dev i samma session. Se "Åtgärdat" under varje punkt.

---

## BLOCKERS

### B-01 — Lösenord skrivs till disk, inte tmpfs

**Risk:** Hög. Admin-lösenordet skrivs till `/var/lib/piholster/initial-admin-password.txt` som är på den vanliga blockenheten. Vid kompromettering av filsystemet (SD-kortet kopieras) är lösenordet läsbart. US-14 kräver explicit tmpfs (`/run/piholster/`).

**Original:** `install ... "${DATADIR}/initial-admin-password.txt"`

**Åtgärdat:** Ändrad till `/run/piholster/initial-password` (tmpfs, skapas av systemd RuntimeDirectory). Lösenordet finns bara i RAM och försvinner vid omstart.

---

### B-02 — TLS-certifikat: 365 dagar (ska vara 3650)

**Risk:** Hög funktionell. Certifikatet löper ut efter ett år. HTTPS slutar fungera och användaren kan inte nå Admin UI utan att SSH in och regenerera. US-14 kräver 3650 dagar (10 år) för att undvika att icke-tekniska användare tvingas förnya.

**Åtgärdat:** `-days 365` → `-days 3650`.

---

### B-03 — RSA 4096 istället för P-256 ECDSA

**Risk:** Medium funktionell. RSA 4096-generering tar 30–120 sekunder på Pi 3 ARM Cortex-A53. Firstboot-fönstret förlängs markant. US-14 explicit: "P-256 föredras: mindre cert, snabbare handshake på Pi 3."

**Åtgärdat:** Ändrat till `-newkey ec -pkeyopt ec_paramgen_curve:P-256`. Genereringstid på Pi 3: <1 sekund.

---

### B-04 — Device identity key genereras inte

**Risk:** Hög. US-14 krav 4 specificerar att firstboot-skriptet ska generera `/var/lib/piholster/device-id.key` (32 bytes, `/dev/urandom`, mode 0400). Nyckeln används som rot för framtida per-rad AES-GCM-kryptering av query_log och som suffix i User-Agent. Saknas helt i implementationen.

**Åtgärdat:** Lagt till steg i firstboot.sh som genererar och verifierar 32-byte identity key.

---

### B-05 — Hårdkodad `--interface=eth0` i piholster-arpd.service

**Risk:** Hög funktionell. Raspberry Pi OS Bookworm (Debian 12) aktiverar predictable network names som standard. Ethernet-porten heter typiskt `end0` eller `enp1s0`, inte `eth0`. ARP-scannern hittar inget interface och startar inte — hela nätverksöversikten uteblir. piholster-arpd:s scanner har redan auto-detect av default route interface implementerat.

**Åtgärdat:** Tagit bort `--interface=eth0` från ExecStart. arpd auto-detekterar interface via default route.

---

## MAJOR

### M-01 — Duplicerad [publish]-sektion i Avahi-konfiguration

**Risk:** Medium. Scriptet gör `cat >>` till avahi-daemon.conf. Om [publish]-sektionen redan finns hamnar säkerhetsinställningarna i en andra sektion som Avahi ignorerar (läser bara första). publish-hinfo=no och publish-workstation=no appliceras då inte.

**Åtgärdat:** Ersätt med `sed`-baserad patch som uppdaterar befintliga nycklar in-place istället för att appenda.

---

### M-02 — `NoNewPrivileges=false` i piholster-firstboot.service

**Risk:** Låg-medium. Firstboot körs som root och behöver inte eskalera. `NoNewPrivileges=false` är dock anti-defence-in-depth — om SUID-binärer anropas av scriptet kan de eskalera till annan användare. Korrekt är `true`.

**Åtgärdat:** `NoNewPrivileges=false` → `NoNewPrivileges=true`.

---

### M-03 — `hidepid=2` deprecated på kernel 5.8+

**Risk:** Låg. Raspberry Pi OS Bookworm kör kernel 6.x. `hidepid=2` fungerar fortfarande men ger kernel-varning. Korrekt syntax på kernel 6.x är `hidepid=invisible` med `gid=proc`.

**Åtgärdat:** fstab-raden ändrad till `proc /proc proc defaults,hidepid=invisible,gid=proc 0 0`. En `proc`-grupp skapas och systemd-processer läggs i den.

---

### M-04 — MANIFEST-generering saknas i build.sh

**Risk:** Låg för säkerhet, hög för release-tracking. US-13 krav 5 specificerar att `image/MANIFEST` ska uppdateras med pi-gen commit, base-image SHA256, piholsterd version och build-timestamp.

**Åtgärdat:** Lagt till MANIFEST-generering i slutet av build.sh.

---

## MINOR (noterat, ej fixat i denna sprint)

- **Minisign-verifiering** av binärer saknas i build.sh. build.sh accepterar binärer via env-variabler från den betrodda build-maskinen, vilket är acceptabelt för MVP. Minisign-verifiering är scope för release-image CI-workflow (release-image.yml) snarare än build.sh. Noteras som teknisk skuld till v1.0.
- **Lösenordet hashas inte av firstboot-skriptet.** piholsterd läser lösenordet från `/run/piholster/initial-password` vid UserCount()==0 och kör Argon2id-hashning internt. Detta kräver koordination — piholsterd måste implementera firstboot-lösenordsinläsning. Noteras som implementation-scope för US-14 Go-del.

---

## go:embed / static.go

Inga säkerhetsfynd. `fs.Sub`, SPA-fallback och cache-headers är korrekt implementerade. CSP nonce-injection via `%%CSP_NONCE%%`-platshållare är **inte implementerat** i static.go (BB-07 i SPRINT-3.md). static.go serverar index.html rakt av utan nonce-injektion. Detta är ett öppet acceptance-kriterium för US-16 och kräver CTO-beslut per BB-07.

---

## Sammanfattning

| ID   | Typ     | Status     |
|------|---------|------------|
| B-01 | BLOCKER | Fixad      |
| B-02 | BLOCKER | Fixad      |
| B-03 | BLOCKER | Fixad      |
| B-04 | BLOCKER | Fixad      |
| B-05 | BLOCKER | Fixad      |
| M-01 | MAJOR   | Fixad      |
| M-02 | MAJOR   | Fixad      |
| M-03 | MAJOR   | Fixad      |
| M-04 | MAJOR   | Fixad      |
| S-01 | MINOR   | Teknisk skuld v1.0 |
| S-02 | MINOR   | Scope piholsterd firstboot-impl |
| BB-07 | ÖPPEN  | CTO-beslut krävs (nonce-injection) |
