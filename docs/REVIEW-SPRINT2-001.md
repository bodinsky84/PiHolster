# REVIEW-SPRINT2-001 - PiHolster Sprint 2

**Datum:** 2026-05-01
**Granskare:** Ghost Reviewer (claude-sonnet-4-6)
**Branch:** sprint-2 -> develop

---

## GODKAND MED KOMMENTARER

Inga blockers. Tva sakerhetsanmarkningar (S-01, S-02) maste atgardas fore merge.
Sex forbattringsforslag ar icke-blockerande.

---

## 1. Kritiska fel (blockers)

Inga blockers identifierades.

---

## 2. Sakerhetsanmarkningar

### S-01 - GET /api/devices och GET /api/status ar publika utan autentisering

**Fil:** internal/api/router.go:31-32
**Allvarlighet:** HOG - maste atgardas fore merge

    mux.HandleFunc("GET /api/status", statusHandler.Status)
    mux.HandleFunc("GET /api/devices", devicesHandler.List)

Dessa endpoints returnerar MAC-adresser, IP-adresser och hostnamn for alla natverksenheter
utan autentisering. En angripare som nar granssnittet far komplett natverkskarta utan
att logga in.

admin/+page.svelte:37-39 forstoker anvanda GET /api/devices som sessionskontroll, men
eftersom endpointen alltid returnerar 200 fungerar detta aldrig som sessionskontroll.

**Fix:** Skydda /api/devices med RequireAdmin. Om startsidans status-widget maste vara
publik, skapa en separat /api/health som returnerar bara status/statistik utan MAC/IP-data.

---

### S-02 - Cookie Path /admin tacker inte /api/-routes

**Fil:** internal/api/auth_handler.go:97 och 121
**Allvarlighet:** HOG - maste atgardas fore merge

Bada SetCookie-anropen (Login rad 97, Logout rad 121) satter Path: "/admin".
De skyddade API-endpoints ligger under /api/auth/change-password,
/api/devices/{mac}/trust och /api/devices/{mac}/rename.
Webblasaren skickar inte cookien till dessa paths eftersom de inte borjar med /admin.
RequireAdmin-middleware exekveras alltsa mot en cookie som aldrig medfotjer requesten,
vilket i praktiken gor alla skyddade endpoints tillgangliga utan giltig session.

**Fix:** Satt Path: "/" sa att cookien medfotjer alla requests till samma origin.
Alternativt: flytta admin-API:et under /admin/api/.

---

### S-03 - dummyHash-kommentar forklarar inte user == nil-garantin

**Fil:** internal/api/auth_handler.go:208
**Allvarlighet:** LAG

Timing-normaliseringen vid okant anvandarnamn ar korrekt. Kommentaren forklarar dock inte
att user == nil-grenen (rad 70) garanterar 401 oberoende av compare-resultatet. En
laggrannsare kan missuppfatta att en hash-kollision mot de kanda AAAA-salterna ger access.

**Fix:** Lagg till en kommentar vid rad 70 som explicit forklarar att user == nil
garanterar 401 oavsett compare-resultatet.

---

## 3. Forbattringar per modul

### Auth (US-08)

**F-01 - pruneLoop goroutine kan inte stoppas**
Fil: internal/auth/ratelimit.go:25

    go r.pruneLoop()

NewRateLimiter tar ingen context. pruneLoop startar en goroutine som lever for evigt.
Avviker fran det ovriga monster (scanner.Run, client.Connect tar context) och
omojliggor korrekt test utan goroutine-lacka.

Fix: Lagg till context-parameter till NewRateLimiter(ctx) och avsluta pruneLoop
pa <-ctx.Done().

---

### ARP (US-09)

**F-02 - probeSweep sweepas alltid /24 oavsett faktisk natmask**
Fil: internal/arp/scanner.go:105

    for i := 1; i < 255; i++ {

Koden hamtar korrekt interfacets CIDR (rad 98) men ignorerar natmasken och sweepas
alltid exakt 254 adresser. Kommentaren pa rad 96 erkanner antagandet ("every host
in the /24") men det ar odokumenterat som medveten begransning.

Fix: Lagg till en TODO-kommentar som erkanner /24-antagandet, eller iterera
faktiskt over cidr.

**F-03 - reverseLookup blockerar ARP-read-loopen**
Fil: internal/arp/scanner.go:142

    dev := parseDevice(pkt)   // anropar reverseLookup med 500ms DNS-timeout inuti
    s.upsert(dev)

reverseLookup anropas synkront i listenReplies-loopen. Under DNS-timeoutens 500ms
kan inga ARP-paket las in, vilket ger paketforlust pa nat med manga enheter.

Fix: Lat upsert ske direkt med tomt Hostname, och gor DNS-lookupen i en goroutine
som uppdaterar enheten nar svar kommer.

---

### Telegram (US-10)

**F-04 - Send returnerar alltid nil trots error-signatur**
Fil: internal/alerts/telegram.go:35

    func (t *TelegramClient) Send(ctx context.Context, message string) error {

Alla fel loggas internt. Returvardet ar alltid nil. Kommentaren //nolint:errcheck i
notifier.go:73 ar ett symptom pa att signaturen ar vilseledande. Fire-and-forget ar
rimligt for notifieringar men error-returvardet ar da missledande.

Fix: Antingen returnera faktiska errors och lat anroparen bestamma, eller andra
signaturen till att inte returnera error alls.

---

### API + Web-UI (US-11)

**F-05 - admin/+page.svelte anvander sessionStorage som session-indikator**
Fil: apps/web/src/routes/admin/+page.svelte:37-39

    if (sessionStorage.getItem("ph_admin") === "1") {
        await loadDashboard();
    }

sessionStorage anvands som auth-indikator. Nar S-01 atgardas och /api/devices skyddas
kommer loadDashboard att fa 401 vid utgangen cookie och hanteras korrekt (rad 115-117).
Ingen direktsakerhetsrisk, men logiken ar skor och bor ses over nar S-01 atgardas.

---

### CSP-headers (US-12)

**F-06 - HSTS includeSubDomains pa ett lokalt nat**
Fil: internal/api/middleware/security.go:39

    h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

includeSubDomains kan paverka andra .local-subdomaner pa klienten under 2 ar. Risken
ar minimal pa ett hemmanat men bor dokumenteras med en kommentar.

---

## 4. Positivt

**Argon2id-implementation:**
Parametrarna (m=65536, t=3, p=2), subtle.ConstantTimeCompare och PHC-format-parsning ar
korrekt implementerade. Timing-normaliseringen vid okant anvandarnamn med dummyHash ar
en ovanligt genomtankt detalj som skyddar mot username enumeration via timing-analys.

**SQL-parameterisering:**
Samtliga queries i store/users.go, store/sessions.go och store/devices.go anvander
parameteriserade queries utan undantag. Ingen strankkonkatenering forekomme.

**Cookie-attribut:**
HttpOnly, Secure och SameSite=Strict korrekt satta (auth_handler.go:99-101). Ingen JWT.
Sessioner lagras server-side med expires_at-validering i SQL (sessions.go:37).

**CSP-nonce:**
Genereras med crypto/rand per request (security.go:58-63), injiceras i context.
Testsviten verifierar att nonce ar unik per request och aterfinns i CSP-headern.

**Host-allowlist:**
421 Misdirected Request for okand host. canonicalHost hanterar IPv6 och port-stripping
korrekt. EXTRA_ALLOWED_HOSTS via miljovariabel ger flexibilitet.

**Rate limiting:**
Sliding window 5 forsok/60 s per IP. Record anropas bara vid faktiskt misslyckad
inloggning (auth_handler.go:71) -- inte vid interna serverfel, vilket undviker att
legitima anvandare lasas ute av serverproblem.

**IPC-arkitektur:**
Unix-socket 0660-rattigheter, snapshot-on-connect, exponentiell backoff 500ms-30s.
Inga hardkodade tokens eller credentials.

**Testsvit:**
Notifier-testerna tacker ny enhet, betrodd enhet, gammal enhet och tomt hostnamn med
fake HTTP-server som fanger faktiska requests. Middleware-testerna ar parallella och
tacker header-narvaro, nonce-unicitet, host-allowlist och Chain-ordning.
password_test.go verifierar korrekthet och salt-unicitet per anrop.

**Loggning:**
log/slog konsekvent. Inga losenord, tokens eller session-varden loggas nagonsin.

---

## 5. Sammanfattning av atgardspunkter

| ID   | Prioritet | Fil                            | Beskrivning                                       |
|------|-----------|--------------------------------|---------------------------------------------------|
| S-01 | HOG       | api/router.go:31-32            | /api/devices exponerar MAC/IP utan autentisering  |
| S-02 | HOG       | api/auth_handler.go:97,121     | Cookie Path /admin tacker inte /api/-routes       |
| S-03 | LAG       | api/auth_handler.go:208        | dummyHash saknar user==nil-garanti i kommentar    |
| F-01 | MEDIUM    | auth/ratelimit.go:25           | pruneLoop goroutine kan inte stoppas              |
| F-02 | LAG       | arp/scanner.go:105             | probeSweep sweepas alltid /24                     |
| F-03 | MEDIUM    | arp/scanner.go:142             | reverseLookup blockerar ARP-read-loopen           |
| F-04 | LAG       | alerts/telegram.go:35          | Send returnerar alltid nil trots error-signatur   |
| F-05 | LAG       | web/admin/+page.svelte:37      | sessionStorage som session-indikator ar skor      |
| F-06 | LAG       | api/middleware/security.go:39  | HSTS includeSubDomains pa lokalt nat              |

**S-01 och S-02 maste atgardas innan merge till develop.**

