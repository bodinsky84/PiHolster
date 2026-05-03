# Sprint 2 — SQLite-store, Auth, ARP-skanning, Telegram, API-koppling

**Sprintlängd:** 2 veckor
**Sprint-mål:** En körbar Pi som lagrar data i SQLite, kräver inloggning för admin,
scannar nätverket med `piholster-arpd` i isolerad process, skickar Telegram-varning vid ny enhet,
och visar verklig data i Web-UI via ett fungerande backend-API.

---

## User stories

### US-07 — SQLite-store med migrations och kärntabeller

**Som** backend
**vill jag** ha en versionsstyrd SQLite-databas med tabellerna `devices`, `dns_query_log`,
`settings` och `users`
**så att** alla tjänster kan läsa och skriva persistent data utan att data förloras vid omstart.

**Prioritet:** Hög (blockar US-08, US-09, US-10, US-11)
**Estimat:** 2 dagar
**Ansvarig:** Senior Go-utvecklare

**Acceptanskriterier:**
- Databasen skapas i `/var/lib/piholster/piholster.db` om den inte finns
- Migrations körs automatiskt vid start av `piholsterd` via ett schema-versioning-system
  (t.ex. en intern `schema_version`-tabell, eller `golang-migrate`)
- Tabell `users`: kolumner `id`, `username`, `argon2_hash`, `created_at`, `last_login_at`
- Tabell `devices`: kolumner `id`, `mac`, `ipv4`, `ipv6`, `hostname`, `vendor_oui`,
  `first_seen_at`, `last_seen_at`, `is_known` (boolean, default false)
- Tabell `dns_query_log`: kolumner `id`, `domain`, `client_ip`, `blocked` (boolean),
  `blocklist_match`, `resolved_at`; automatisk rotation som tar bort poster äldre än 30 dagar
- Tabell `settings`: kolumner `key` (TEXT PRIMARY KEY), `value`, `updated_at`
- Ett Go-paket `apps/piholsterd/internal/store` exponerar interface för CRUD-operationer;
  direkta SQL-anrop utanför paketet är inte tillåtna
- Migrationssteget är idempotent: att köra det två gånger ger inget fel och ändrar inte data
- Enhetstester täcker: migration från tom DB, migration från v1 till v2, CRUD för alla tabeller
- Databasen öppnas med `PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`

---

### US-08 — Admin-autentisering med Argon2id och sessionshantering

**Som** adminanvändare
**vill jag** logga in med ett lösenord och sedan ha en session som håller i 8 timmar
**så att** obehöriga på LAN inte kan ändra inställningar.

**Prioritet:** Hög (beror på US-07; blockar US-11 admin-endpoints)
**Estimat:** 2,5 dagar
**Ansvarig:** Senior Go-utvecklare + IT-säkerhet (review)

**Acceptanskriterier:**
- Lösenordshash lagras med Argon2id, parametrar: `memory=64MB, iterations=3, parallelism=4`
- Hash-jämförelse använder `subtle.ConstantTimeCompare` (krav från ADR-002 H-G)
- Sessions är server-side: ett opakt 256-bitars token genereras med `crypto/rand`,
  lagras i `sessions`-tabell i SQLite med kolumner `token_hash`, `user_id`,
  `created_at`, `expires_at`, `last_seen_at`
- Token skickas till klienten som HTTP-only, Secure, SameSite=Strict-cookie
- Session-token hashas med SHA-256 innan det lagras i DB (råtoken ska aldrig finnas i databasen)
- Session-livslängd: 8 timmar; förlängs med 1 timme vid varje aktivt anrop, max 24 timmar
- `POST /api/auth/login` returnerar 200 och sätter cookie vid korrekt lösenord;
  returnerar 401 med generisk felmeddelandetext vid fel (inga ledtrådar om vad som var fel)
- `POST /api/auth/logout` raderar session på server-sidan
- `GET /api/auth/me` returnerar 200 med `{username}` för inloggad, 401 annars
- Per-IP-lockout: 5 misslyckade inloggningar inom 10 minuter ger 429 i 15 minuter
- Global account-lockout: 20 misslyckade inloggningar mot samma konto ger Telegram-varning
  (Telegram-integrationen existerar inte i US-08; loggsteg räcker, Telegram kopplas i US-10)
- Firstboot-wizard skapar admin-användare med slumpmässigt 16-teckens initial-lösenord
  som visas en gång i terminalen och skrivs till `/run/piholster/initial-password` (mode 0600,
  raderas vid första inloggning)
- IT-säkerhet granskar PR innan merge

---

### US-09 — `piholster-arpd`: separat binär med CAP_NET_RAW-isolering

**Som** systemet
**vill jag** att ARP-skanning körs i en separat process med minimala privilegier
**så att** en eventuell RCE i HTTP-hanteraren inte ger angriparen tillgång till nätverkets råtrafik.

**Prioritet:** Hög (MVP-blocker per ADR-002 fynd 1; beror på US-07 för devices-tabell)
**Estimat:** 3 dagar
**Ansvarig:** Senior Go-utvecklare + IT-säkerhet (review)

**Acceptanskriterier:**
- Binären `piholster-arpd` byggs som ett separat Go-modul under `apps/piholster-arpd/`
  med repo-layout enligt ADR-002 §3.1.2
- `piholster-arpd` körs som systemd-tjänst med `User=piholster-arp` (UID 998),
  `AmbientCapabilities=CAP_NET_RAW`, `CapabilityBoundingSet=CAP_NET_RAW`,
  `NoNewPrivileges=true` och övriga begränsningar per ADR-002 §3.1.2
- `piholsterd` har INTE `cap_net_raw`; CI-steget `getcap piholsterd | grep -v cap_net_raw`
  failar bygget om detta bruts
- IPC sker via Unix socket `/run/piholster/arp.sock`, protokoll: length-prefixed protobuf
  per ADR-002 §3.1.2, max frame 64 KiB
- `piholster-arpd` validerar SO_PEERCRED och accepterar enbart anslutningar från UID 999
  (`piholster`)
- `piholsterd` validerar SO_PEERCRED och accepterar enbart svar från UID 998
  (`piholster-arp`)
- `piholster-arpd` öppnar AF_PACKET-socket tidigt, droppar sedan `CAP_NET_RAW` från
  effective set; capability-drop loggas
- Nya och förändrade enheter skrivs till `devices`-tabellen via `piholsterd`
  (arpd skriver aldrig direkt till DB)
- Heartbeat-meddelanden skickas var 30:e sekund; `piholsterd` loggar varning om heartbeat
  uteblir mer än 90 sekunder
- systemd-unit för `piholster-arpd` startar före `piholsterd.service` och har
  `Before=piholsterd.service`
- Enhetstester täcker: protobuf encode/decode, capability-drop-logik (mock), IPC-handskakning
- IT-säkerhet granskar PR innan merge

---

### US-10 — Telegram-notiser vid ny okänd enhet

**Som** hemmaägare
**vill jag** få ett Telegram-meddelande när en enhet ansluter till mitt nätverk som jag inte
känner igen
**så att** jag kan reagera om någon obehörig kopplar upp sig mot mitt WiFi.

**Prioritet:** Hög (beror på US-07 `devices`-tabell och US-09 ARP-flöde)
**Estimat:** 1,5 dagar
**Ansvarig:** Senior Go-utvecklare

**Acceptanskriterier:**
- En Telegram-bot-token och ett chat-ID konfigureras via `settings`-tabellen i SQLite
  (nycklarna `telegram.bot_token`, `telegram.chat_id`)
- Varning skickas när `piholsterd` tar emot ett `DeviceObserved`-meddelande från arpd
  för ett MAC-adress som inte finns i `devices`-tabellen (dvs. `is_known = false` och
  `first_seen_at` sätts nu)
- Meddelandeformat (exempel):
  ```
  [PiHolster] Ny enhet sedd pa natverket
  MAC: aa:bb:cc:dd:ee:ff
  IP: 192.168.1.42
  Tillverkare: Apple, Inc.
  Forst sedd: 2026-05-01 14:32:01
  ```
- Rate-limiting: max 1 Telegram-meddelande per MAC per 24 timmar (forhindrar spam vid
  DHCP-fornyelse)
- Om Telegram-anropet misslyckas (nätverksfel, ogiltigt token) loggas felet men arpd-flödet
  avbryts inte
- Bot-token lagras aldrig i klartext i loggar; maskeras som `tg:***` i loggmeddelanden
- `POST /api/settings/telegram/test` skickar ett testmeddelande och returnerar 200 eller
  felmeddelande
- Telegram-konfiguration kan lämnas tom; varningar skickas då inte men övrig funktion
  påverkas inte
- Enhetstester täcker: ny enhet triggar notis, känd enhet triggar inte, rate-limit respekteras

---

### US-11 — Backend-API: endpoints för enheter, DNS-log och inställningar

**Som** Web-UI
**vill jag** hämta verklig data via ett REST-API
**så att** Mormor-vy och Avancerat-vy visar faktisk information från nätverket.

**Prioritet:** Hög (beror på US-07 store och US-08 auth; blockar US-12)
**Estimat:** 2 dagar
**Ansvarig:** Senior Go-utvecklare

**Acceptanskriterier:**
- Alla endpoints returnerar JSON med `Content-Type: application/json`
- Middleware-ordning per ADR-002 §3.3.2: HostAllowlist, SecurityHeaders, RequireOrigin,
  CSRF, Session.Verify, RequestID, Logger, Recover
- Publika endpoints (kräver ingen inloggning):
  - `GET /api/status` — returnerar `{dns_blocked_today, devices_online, uptime_seconds,
    dns_ok: bool}`
- Autentiserade endpoints (kräver giltig session):
  - `GET /api/devices` — paginerad lista (`?page=1&per_page=50`), returnerar lista med
    `id, mac, ipv4, hostname, vendor_oui, first_seen_at, last_seen_at, is_known`
  - `PATCH /api/devices/{id}` — uppdaterar `is_known` och valfritt `hostname`
  - `GET /api/dns/log` — paginerad DNS-logg (`?page=1&per_page=100&blocked=true`)
  - `GET /api/settings` — returnerar alla inställningar utom hemliga värden (tokens
    returneras som `"***"`)
  - `PUT /api/settings/{key}` — uppdaterar ett inställningsvärde; validerar tillåtna nycklar
    mot en whitelist
  - `POST /api/settings/telegram/test` — skickar testmeddelande (se US-10)
- HTTP 401 returneras för autentiserade endpoints om session saknas eller har utgatt
- HTTP 421 returneras av HostAllowlist-middleware om Host-header inte matchar
- HTTP 403 returneras om Origin-header saknas eller CSRF-token är fel vid POST/PUT/PATCH
- Alla svar innehåller headers definierade i ADR-002 §3.4 (CSP, HSTS, X-Frame-Options m.fl.)
- Integrationstester (i `tests/`-mappen) täcker: status-endpoint utan auth, enheter kräver
  auth, felaktig session ger 401, CSRF-skydd blockerar POST utan token

---

### US-12 — Web-UI kopplad till verklig backend-data

**Som** användare
**vill jag** att Mormor-vy och Avancerat-vy visar data som faktiskt stämmer med vad som
händer i mitt nätverk
**så att** jag kan lita på informationen och agera på den.

**Prioritet:** Medium (beror på US-11)
**Estimat:** 2 dagar
**Ansvarig:** Senior Go-utvecklare (fullstack, SvelteKit)

**Acceptanskriterier:**
- Mormor-vyn hämtar `GET /api/status` och visar verkliga värden för:
  - Trafikljus: grönt om `dns_ok = true`, rött annars
  - Antal blockerade annonser idag (`dns_blocked_today`)
  - Antal enheter online (`devices_online`)
- Mormor-vyn uppdateras automatiskt var 30:e sekund utan helsidesladda
- Avancerat-vyn visar enhetslistan från `GET /api/devices` med IP, MAC, tillverkare
  och senast sedd
- Avancerat-vyn visar de senaste 50 DNS-frågorna från `GET /api/dns/log`; blockerade
  visas med röd markering
- Admin-vyn kräver att användaren är inloggad; omdirigeras till `/login` annars
- Login-sidan skickar `POST /api/auth/login`, hanterar 401 med felmeddelande,
  omdirigerar vid 200
- Logout-knapp i Admin-vyn anropar `POST /api/auth/logout`
- CSRF-token läses från cookie och skickas som `X-PiHolster-CSRF`-header på alla
  state-changing fetch-anrop (POST, PUT, PATCH)
- Platshållartexten "Kommer i Sprint 2" är borttagen i sin helhet
- UI fungerar i mobilwebbläsare (375 px bredd)
- Inga externa resurser laddas (font-filer, CSS-ramverk m.m. är bundlade);
  CI:s external-URL-check passerar (ADR-002 §3.4.2)

---

## Sprint-backlog sammanfattning

| ID    | Story                                  | Prioritet | Estimat   | Ansvarig              |
|-------|----------------------------------------|-----------|-----------|-----------------------|
| US-07 | SQLite-store med migrations            | Hög       | 2 dagar   | Senior Go-dev         |
| US-08 | Admin-auth: Argon2id + sessioner       | Hög       | 2,5 dagar | Senior Go-dev + Säk   |
| US-09 | `piholster-arpd` separat binär         | Hög       | 3 dagar   | Senior Go-dev + Säk   |
| US-10 | Telegram-notiser vid ny enhet          | Hög       | 1,5 dagar | Senior Go-dev         |
| US-11 | Backend-API: enheter, DNS-log, inst.   | Hög       | 2 dagar   | Senior Go-dev         |
| US-12 | Web-UI kopplad till backend-data       | Medium    | 2 dagar   | Senior Go-dev         |

**Total estimering:** 13 dagar (~2 veckors sprint med marginal for review, IT-sak-granskning
och integrationstest pa fysisk Pi)

---

## Beroendediagram

```
US-07 (SQLite store)
  |
  +---> US-08 (Auth, users-tabell)
  |       |
  |       +---> US-11 (API, kräver session-verify)
  |               |
  |               +---> US-12 (Web-UI, kräver API)
  |
  +---> US-09 (ARP, devices-tabell)
          |
          +---> US-10 (Telegram, kräver ARP-event)
          |
          +---> US-11 (API, devices-endpoint kräver data)
```

**Kritisk väg:** US-07 -> US-09 -> US-10 -> US-11 -> US-12

US-07 måste vara klar senast dag 3 i sprinten for att inte blockera allt annat.
US-08 och US-09 kan utvecklas parallellt efter att US-07 är klar.
US-10 och US-11 kan påbörjas när US-09 respektive US-07+US-08 är klara.
US-12 kan bara slutföras när US-11 är klar.

---

## Definition of Done (per story)

1. Koden är mergad till `develop` via godkänd PR (minst 1 review; IT-säkerhet krävs
   for US-08 och US-09 per CHANGE-PROCESS.md)
2. CI-pipeline är grön: lint, enhetstester, Docker-image bygger
3. For US-09: CI-steget `getcap piholsterd | grep -v cap_net_raw` passerar
4. Acceptanskriterierna är manuellt verifierade av en annan teammedlem på fysisk Pi 3
   eller Pi-emulator
5. Inga öppna [BLOCK]-kommentarer i PR-review
6. SPRINT-2.md uppdateras om scope förändrats (PM-ansvar)
7. Inga hemligheter (bot-tokens, lösenord) är commitade i klartext

---

## Blockers att bevaka

**BB-01 — gopacket/libpcap på ARM**
`piholster-arpd` använder gopacket som kräver libpcap-headers vid cross-kompilering.
DevOps måste verifiera att `CGO_ENABLED=1` cross-compile till `GOARCH=arm` fungerar i CI.
Om det inte gör det: fallback till ren `AF_PACKET`-socket i Go (ingen cgo).
Eskalering: CTO beslutar senast dag 1 i sprinten.

**BB-02 — protobuf-toolchain i CI**
`arpproto.proto` kräver `protoc` och `protoc-gen-go` installerat i CI-imagen.
DevOps lägger till dessa i `ci.yml` som ett setup-steg. Risk: versionskonflikter med
framtida protobuf-filer. Beslut: pinnas till specifik `protoc`-version i CI.

**BB-03 — SvelteKit nonce-integration**
ADR-002 §3.4.1 kräver att SvelteKit producerar nonce-baserade `<style>`-taggar,
inte `style="..."` inline. ADR-002 §6 noterar att `adapter-static` ger inline-styles
utan nonce (teknisk skuld, ~1 dag). For Sprint 2 accepteras `'unsafe-inline'` pa
`style-src` som en temporär lösning. CTO beslutar om det ska fixas i Sprint 2 eller
skjutas till Sprint 3. PM markerar som blocker om CTO kräver fix nu.

**BB-04 — Firstboot-sekvens och initial lösenord**
US-08 kräver att firstboot-skriptet genererar ett initial-lösenord och lagrar det
i `/run/piholster/initial-password`. Om firstboot-implementationen (frán ADR-002 §3.2.1)
inte är klar vid Sprint 2-start blockar det US-08. PM spårar: är firstboot-tjänsten
klar fran Sprint 1 eller är det skuld att hantera i Sprint 2?
Eskalering: Senior Go-dev rapporterar status dag 1.

**BB-05 — IT-säkerhets tillgänglighet**
US-08 och US-09 kräver IT-säkerhets-review enligt CHANGE-PROCESS.md §3 steg 4.
PM bokar in IT-säkerhet for review i mitten av sprinten (dag 6–8) sa att PRs
inte fastnar i kö i slutet av vecka 2.
```
