# SECURITY-REVIEW-001: Säkerhetsgranskning av ADR-001

- **Status:** Granskat
- **Datum:** 2026-05-01
- **Granskare:** Säkerhets- och integritetsexpert (PiHolster)
- **Granskat dokument:** `docs/ADR-001-architecture.md`
- **Scope:** Hela arkitekturen, hotmodellen, byggkedjan, runtime-hardening, autentisering, krypto-baseline

---

## 1. Sammanfattning

ADR-001 är en ovanligt mogen säkerhetsbas för en v1. Beslut som per-device random initial password, Argon2id, fail-closed DoT/DoH, ingen telemetri, signerade images via minisign med 2-of-2-tröskel, och ingen Let's Encrypt-användning är alla rätt. Det finns dock ett antal konkreta luckor som måste adresseras innan publik release. De allvarligaste rör (a) `setcap CAP_NET_RAW` på en HTTP-serverande binär, (b) saknad CSRF/Origin-skydd för cookie-baserad admin, (c) en blottad attackyta vid första-boot innan firstboot-tjänsten kört, (d) avsaknad av subresource integrity och Content-Security-Policy för det embeddade frontend, samt (e) outbound-egress genom DNS-upstreams kan exfiltrera data om en kompromettering sker (DNS-tunneling).

---

## 2. Vad är bra (säkerhetsbeslut som är rätt)

### 2.1 Autentisering
- **Per-device slumpmässigt initiallösenord på 24 base32-tecken** (~120 bits entropi) — eliminerar Mirai-klassens kompromisser via default credentials. Detta är den enskilt viktigaste säkerhetsbeslutet i hela ADR:n.
- **Argon2id (m=64MB, t=3, p=2)** är rätt KDF och rätt parametrar för Pi 3-klass hårdvara. Memory-hard motstår GPU/ASIC.
- **Forced rotation** vid första login innan "remember me" kan utfärdas — bra, hindrar att initiallösenordet blir det permanenta.
- **Server-side sessions framför JWT** — korrekt val för single-node device. Eliminerar alg=none, kid-injection, och revoke-problemen i JWT.
- **Lockout 5 försök -> 60s exponential per source IP** — rimlig default. (Se dock 3.4 om source-IP-spoofing på LAN.)

### 2.2 Nätverksdesign
- **Fail-closed DoT/DoH** (SERVFAIL istället för cleartext fallback) — exemplariskt. Många DNS-resolvers har downgrade-buggar; att skriva ut detta som hård regel är rätt.
- **Bind endast till LAN-interface** — rätt. `0.0.0.0` skulle vara katastrofalt om någon råkade köra Pi:n bakom en publik IP.
- **Egress-firewall via iptables OUTPUT, inte bara applikationskod** — rätt, defence-in-depth.
- **Ingen Let's Encrypt** — rätt motivering. Public CT logs avslöjar `*.piholster.local` per device, och ACME kräver public DNS som vi inte vill ha.
- **Inget extern reverse proxy** — färre rörliga delar, mindre attackyta. Rätt val på 1 GB RAM.

### 2.3 Image build & supply chain
- **Pi-gen pinnad via submodule, kvartalsvis rebase** — bra. Surprise upstream changes är ett reellt hot.
- **Minisign-signed images + binärer, 2-of-2 YubiKey-tröskel** — exemplariskt. Detta är på nivå med Tor Project / Tails. Många MVP-projekt nöjer sig med en GPG-nyckel på en laptop.
- **Pure-Go SQLite-driver utan CGO** — minskar build-attackytan signifikant. CGO drar in glibc-versionsskillnader och en C-toolchain som CI-yta.
- **`go.sum` + Dependabot + vendored module review** — rimlig MVP-baseline. (Se 3.7 om vad som saknas.)
- **Ingen CD till prod** — rätt princip. Att kunna pusha kod till en hemroutter utan användarens bekräftelse är inkompatibelt med produktens hela värdeerbjudande.

### 2.4 Runtime hardening
- **Dedikerad `piholster`-användare med UID 999, inte root** — rätt.
- **`setcap` istället för setuid root** — rätt princip. (Men se 4.1 om CAP_NET_RAW-problemet.)
- **systemd `ProtectSystem=strict`, `NoNewPrivileges=true`, `PrivateTmp=true`** — bra start. (Se 5.2 för fler systemd-direktiv som saknas.)
- **`sysctl kernel.kptr_restrict=2`, `dmesg_restrict=1`, `tcp_syncookies=1`** — rätt baslinje.

### 2.5 Datahygien
- **DNS-query logs aldrig off-device utan explicit nedladdning** — exemplariskt.
- **Telemetri off by default, opt-in, payload visas före första sändning** — exemplariskt. Detta är det mest pro-användare beslutet i hela ADR:n.
- **GPL-3.0 även på frontend** — rätt. En MIT-frontend skulle vara ett kryphål för proprietära fork-leverantörer att bygga ovanpå auth-UI:t.

---

## 3. Risker och luckor i v1

### 3.1 CAP_NET_RAW på en HTTP-serverande binär (HÖG)
ADR säger:
```
setcap cap_net_bind_service,cap_net_raw=+ep /usr/local/bin/piholsterd
```
Att kombinera dessa två capabilities på samma binär är en betydande risk:
- `CAP_NET_RAW` ger raw socket-skapande, vilket vid en RCE i HTTP-handlern direkt låter angriparen skicka godtyckliga paket på LAN (ARP spoof, DHCP poison, NDP spoof, ICMP redirect).
- HTTP-handlern hanterar untrusted input från LAN-devices. ARP-skannern hanterar bara observerade frames.

**Föreslagen åtgärd:** dela på binär eller dela på process. Två rena lösningar:
1. **Process-separation:** `piholster-arpd` är en separat liten Go-binär med endast `CAP_NET_RAW`, kör som annan UID, kommunicerar med `piholsterd` via Unix domain socket på `/run/piholster/arp.sock` (root-skapad, mode 0660, group `piholster`). `piholsterd` har då bara `CAP_NET_BIND_SERVICE`.
2. **Privilege-drop:** `piholsterd` startar med båda caps, gör ARP-socket-bindning vid boot, släpper sedan `CAP_NET_RAW` via `prctl(PR_CAP_AMBIENT_LOWER)` innan HTTP-servern startar. Detta är svårare att få rätt och rekommenderas inte.

Process-separation är arkitektoniskt renare och bör väljas.

### 3.2 CSRF/Origin-skydd saknas i specifikationen (HÖG)
ADR specificerar `SameSite=Strict` cookies, men det räcker inte ensamt:
- En LAN-device som planterar en länk eller iframe kan i vissa edge-cases (gammal webview, kompromettade browser-extensions, samma-site-tolkning vid IP vs hostname) bypassa SameSite.
- Allt LAN-trafik ses som "samma site" av många användares webbläsare när de surfar via PiHolsters DNS.

**Föreslagen åtgärd:** lägg till explicit i ADR-001 §8.2:
- **Origin/Referer-validering** server-side på alla state-changing endpoints (POST/PUT/DELETE). Acceptera bara `Origin: https://piholster.local` eller den bundna LAN-IP:n.
- **Double-submit CSRF-token** för admin-actions: token i cookie + token i custom header `X-PiHolster-CSRF`, server kräver att de matchar. SameSite skyddar inte mot CSRF om en LAN-device hostar ett angrepps-UI.
- **DNS rebinding-skydd:** validera `Host`-header server-side mot en allowlist (`piholster.local`, `piholster.lan`, `<LAN-IP>`). Avvisa allt annat med 421 Misdirected Request. Utan detta kan en publik webbsida via DNS rebinding rikta sig mot 192.168.x.x och nå admin-UI:t från användarens browser-kontekst.

### 3.3 Firstboot-fönster: oskyddad period innan firstboot-tjänsten kört (KRITISK — se §4)
Se kritiskt fynd nr 2 nedan.

### 3.4 Source-IP-baserad lockout är otillräcklig på LAN (MEDIUM)
ADR §8.1: "60s exponential lockout per source IP". På en LAN är source-IP-spoofing trivialt (samma broadcast-domän, ARP, ingen rpfilter på de flesta hem-routrar).

**Föreslagen åtgärd:**
- Komplettera per-IP-lockout med **per-account global lockout** (samtidigt). Efter 20 globala misslyckade försök oavsett källa: lås kontot i 15 min och skicka Telegram-alert.
- Logga och alerta även framgångsrika logins från en ny IP.
- Överväg en tillfällig "device-LED blinkar rött" som fysisk feedback (Pi 3 har act-LED programmerbar via `/sys/class/leds/`).

### 3.5 CSP, SRI och frontend-injicerings-skydd saknas (HÖG)
ADR specificerar inte:
- **Content-Security-Policy header.** Med embedded SvelteKit-build kan vi sätta en mycket strikt CSP utan tredjepartsdomäner. Föreslagen baseline:
  ```
  Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' wss://piholster.local; frame-ancestors 'none'; base-uri 'none'; form-action 'self'
  ```
  (`'unsafe-inline'` på style krävs ofta av SvelteKit; alternativt använd nonce-injektion.)
- **Subresource Integrity (SRI):** eftersom allt är `go:embed`-ad statisk är SRI redan implicit (server kontrollerar bytes), men ADR bör explicit bekräfta att frontend INTE laddar någon resurs från CDN. Om en utvecklare av misstag lägger in `<script src="https://...">` ska CSP blocka det.
- **`X-Frame-Options: DENY`** eller `frame-ancestors 'none'` (gjort via CSP ovan).
- **`X-Content-Type-Options: nosniff`**.
- **`Referrer-Policy: no-referrer`**.
- **`Permissions-Policy`** som disablar geolocation/camera/microphone/USB/etc. (default-deny).

### 3.6 Svag specifikation kring TLS-cert: trust on first use vid hostbyte (MEDIUM)
ADR §3.6: "Self-signed root, generated on first boot, rotated annually."
- Ingen specifikation för **vad som händer vid rotation** — användaren får en cert-warning igen?
- Ingen specifikation för **cert-pinning i frontend** för WebSocket-anslutningen — om en LAN-MITM utfärdar eget self-signed cert (samma CN), klickar användaren bort warningen.
- Ingen specifikation för **certifikatets nyckelstorlek/algoritm**. Föreslå: ECDSA P-256 eller Ed25519, inte RSA 2048 (snabbare på Pi 3).

**Föreslagen åtgärd:**
- Specificera: certrotation gör inte automatisk re-trust-prompt; nytt cert signeras av samma local root (root-cert har 10 års giltighet, leaf-cert 13 månader).
- Frontend embed:ar root-cert SHA-256 fingerprint i en `<meta>` och WebSocket-klienten rejectar mismatch (defence i depth — browser har redan validerat, detta fångar PKI-bypass).
- Algoritm: Ed25519 root + ECDSA P-256 leaf.

### 3.7 Supply chain: `go.sum` + Dependabot är inte tillräckligt (MEDIUM)
ADR §8.6 nämner `go.sum`, Dependabot, vendored review. Saknas:
- **`govulncheck`** i `ci.yml` — Go's officiella vulnerability scanner som matchar mot symbol-användning, inte bara modulversion. Dependabot säger ofta "vuln in dep X" när din kod inte ens anropar den sårbara funktionen.
- **SLSA-provenance attestation** för release-artifacts. GitHub Actions har detta inbyggt sedan 2023; det kostar i princip ingenting och gör att användaren kan verifiera att en `.img.xz` faktiskt byggdes från en specifik commit i en specifik workflow.
- **SBOM (CycloneDX eller SPDX)** publicerad med varje release. Krav för transparens i en GPL-3.0-produkt och för att downstream packagers ska kunna spåra deras egen risk.
- **Pinning av GitHub Actions till SHA, inte tag.** Tags kan flyttas; flera supply-chain-incidenter (tj-actions/changed-files Mar 2025) hade mitigerats av SHA-pinning.
- **`gitleaks`/`trufflehog`** i CI för att fånga oavsiktligt incheckade tokens.
- **Dependency Review Action** för att blocka PR:er som introducerar nya deps med kända CVE:er.

### 3.8 Encrypted query logs men oklar key management (MEDIUM)
ADR §8.5: "per-row AES-GCM med en key derived from the device identity key." ADR-003 deferreras. Men v1 lovar redan att leverera detta. Risker:
- Om "device identity key" lagras på samma SD-kort som krypterad data är skyddet noll mot fysisk åtkomst (vilket dock 8.6 erkänner som accepterad gräns).
- Per-row AES-GCM utan associated data (AAD) kan vara sårbart för row-reordering attacks om en angripare har skrivåtkomst till DB:n. Använd row-id som AAD.
- Key rotation deferreras — men om vi någonsin ska rotera måste schemat redan ha en `key_version`-kolumn i v1, annars är vi skitnödiga senare.

**Föreslagen åtgärd:** lägg till `key_version SMALLINT NOT NULL DEFAULT 1` i `query_log`-schemat redan i v1. Kostnaden är 2 byte per row; kostnaden att lägga till senare är en migration på en miljardradstabell på en Pi 3.

### 3.9 Telegram bot-token är en single point of compromise (MEDIUM)
ADR §8.3 listar Telegram API som tillåten egress. Inte specificerat:
- Var lagras Telegram bot-token? Plaintext i SQLite `settings` är vanligast och är fel.
- Vad händer om Telegram bot-token läcker? En angripare kan använda den för att skicka falska larm till användaren ("klicka här för att verifiera ditt konto").

**Föreslagen åtgärd:**
- Bot-token krypteras med device identity key, samma KDF-väg som query_log.
- Outgoing Telegram messages signeras med en HMAC-prefix som användaren ser ("OK" eller "VERIFIERAD") och som angriparen inte kan reproducera utan device-state.
- Rate-limit utgående Telegram till säg 60 msg/h för att begränsa abuse om token läcker.

### 3.10 ARP-skannern: passiv vs aktiv inte specificerad (LÅG)
ADR säger "gopacket ARP scanner" men inte om vi gör aktiva ARP-probes (skickar ARP requests för 192.168.1.0/24) eller bara passivt sniffar.
- Aktiv probing: vi blir synliga på LAN, en kompromettead IoT-device kan fingerprinta oss på det specifika ARP-mönstret.
- Passiv: missar tysta devices.

Föreslagen åtgärd: passiv som default, aktiv probe gated bakom Admin-mode-toggle, randomiserad inter-probe-delay.

### 3.11 mDNS/Avahi exponerar device-info till hela LAN (LÅG)
ADR §3.5: publicera `_http._tcp` och `_https._tcp`. Detta inkluderar i många avahi-konfigurationer också `hostname`, OS-info, och TXT-records. En kompromettad IoT enhet kan enumerera hela LAN.

Föreslagen åtgärd: konfigurera Avahi `disable-publishing` förutom de specifika tjänsterna, och stänga av `publish-hinfo` och `publish-workstation` i `/etc/avahi/avahi-daemon.conf`.

### 3.12 Auto-update kan vara en TOCTOU-vektor (LÅG)
ADR §5.3: "Updates are signed binaries downloaded from `releases.piholster.net`, verified with the bundled minisign public key, atomically swapped, then `systemctl restart`."

Saknas:
- Anti-rollback-skydd: en angripare med MITM på update-server kan servera en äldre, korrekt signerad version med en känd CVE.
- Update-frequency: hur ofta polls servern? Korrelations-fingerprint (varje Pi som pollar samma sekund från samma IP-block är samma hushåll).

**Föreslagen åtgärd:**
- Embedded `min_version` i image, monotont ökande. Update aborteras om served version < installed.
- Update-poll med jitter (±2h), varje device får random offset i sin systemd-timer.

### 3.13 SSH disabled men inte locked (LÅG)
ADR §7.2 `02-harden`: "Disable SSH by default. Power users can enable via Admin UI."
- Hur enables SSH via UI? Vad är default port? Rate limit? Endast nyckel-auth?
- Saknas: när SSH enables ska det vara endast `PubkeyAuthentication yes`, `PasswordAuthentication no`, `PermitRootLogin no`, port stays 22 men `iptables` öppnas bara för LAN-CIDR.

---

## 4. Kritiska fynd (måste fixas innan release)

Rankat efter risk × sannolikhet × påverkan.

### KRITISK 1: CAP_NET_RAW på samma binär som hanterar HTTP-input
**Fil/sektion:** ADR-001 §5.3, §7.2

**Vektor:** Vid en RCE-bug i HTTP-handlern (även en read-side bug som låter angriparen kontrollera `syscall.Sendto`) kan angriparen skicka godtyckliga ARP/IPv6/ICMP-paket från Pi:n. Det förvandlar en w-vuln i en route-handler till full LAN-kompromettering inklusive ARP-spoof av default gateway.

**Påverkan om ej åtgärdat:** Privilege escalation från en HTTP RCE blir LAN-wide MITM, exfiltration av varje TCP-flöde i hushållet.

**Exakt fix:** dela på processer. Ny binär `piholster-arpd` (~200 LOC Go), egen UID `piholster-arp`, har `CAP_NET_RAW`, ingen nätverkslyssnare, kommunicerar bara via Unix socket `/run/piholster/arp.sock` med length-prefixed JSON. `piholsterd` får endast `CAP_NET_BIND_SERVICE`. Lägg till i `01-firstboot` att `/run/piholster` skapas med rätt mode.

### KRITISK 2: Firstboot-fönstret är oskyddat
**Fil/sektion:** ADR-001 §7.2 `01-firstboot`

**Vektor:** Mellan första bootens `network-online.target` och `piholster-firstboot.service` har slutfört kan följande inträffa:
- TLS-cert finns inte än → port 443 är nere ELLER serverar default cert.
- Admin-lösenordet är inte genererat än → vad svarar `/api/login`?
- Avahi annonserar redan `_https._tcp` → en LAN-angripare ser exakt när enheten sätts upp och kan race:a.

ADR specificerar inte att `piholsterd.service` har `After=piholster-firstboot.service` och att firstboot KÖRS innan piholsterd ens startar.

**Påverkan om ej åtgärdat:** En LAN-angripare som sniffar mDNS i vänteläge kan göra en "evil twin" som svarar på första HTTPS-requesten med ett annat self-signed cert, eller helt enkelt ARP-spoofa Pi:n innan första-användaren ens når dashboarden.

**Exakt fix:**
1. systemd-unit-deps: `piholsterd.service` har `Requires=piholster-firstboot.service` och `After=piholster-firstboot.service`. Firstboot är `Type=oneshot` med `RemainAfterExit=yes`.
2. Avahi-publishing startar inte förrän efter firstboot. Konfigurera ett `piholster-avahi-publish.service` som körs efter både och publicerar tjänsterna programmatiskt via `avahi-publish` snarare än statiska `.service`-filer i `/etc/avahi/services/`.
3. Innan firstboot är klar: alla portar (53, 80, 443) är blockerade av iptables. `02-harden` sätter en INPUT default DROP, firstboot lägger till accept-regler först efter password+cert genererat.
4. Lägg till en boot-LED-pattern: röd-blink under firstboot, grön-stadig efter — ger användaren visuell signal att de inte ska connecta innan dess.

### KRITISK 3: DNS rebinding-skydd saknas
**Fil/sektion:** ADR-001 §8.2, §8.3

**Vektor:** En användare surfar till en publik webbsida som innehåller JS som gör DNS rebinding till `192.168.1.50` (Pi:ns IP). Browsern tolkar same-origin baserat på hostname, inte IP — webbsidan kan nu göra `fetch()` mot Pi:ns admin-API från användarens browser, med användarens cookies. SameSite hjälper inte (samma site som angreppswebbsidan), CORS hjälper inte (no-cors `fetch()` läses inte men POSTAS).

**Påverkan om ej åtgärdat:** Varje PiHolster-installation är fjärr-CSRF-bar genom valfri webbsida användaren besöker. För en privacy-produkt är detta katastrofalt.

**Exakt fix:**
1. Server-side `Host`-header allowlist: `piholster.local`, `piholster.lan`, exakt LAN-IP. Avvisa allt annat med 421.
2. Origin-header validering på alla state-changing endpoints.
3. Double-submit CSRF-token (cookie + custom header).
4. CSP `frame-ancestors 'none'` för att blocka iframe-baserade rebinding.
5. Lägg till test i `ci.yml` som faktiskt testar DNS rebinding-skyddet (skicka request med `Host: evil.com`, assert 421).

### KRITISK 4: CSP och säkerhetsheaders inte specificerade
**Fil/sektion:** ADR-001 §3.2, §8 (saknas)

**Vektor:** Utan CSP är en XSS i SvelteKit-frontenden (även en framtida regression) full session-hijack. Med strikt CSP får angriparen ingenstans att skicka stulna data.

**Påverkan om ej åtgärdat:** En enda XSS-bug i admin-UI:t = admin-takeover på varje deployerad Pi. Med 10K installationer i framtiden är detta en koordinerad attack på 10K hushåll.

**Exakt fix:** lägg till i ADR-001 §8 ny subsektion "8.7 HTTP Security Headers" med:
```
Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' wss://piholster.local wss://piholster.lan; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; report-uri /api/csp-report
Strict-Transport-Security: max-age=63072000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Permissions-Policy: geolocation=(), camera=(), microphone=(), usb=(), payment=()
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Resource-Policy: same-origin
```
HSTS är OK på LAN eftersom frontend bara serveras över HTTPS från Go-binären; preloading skall inte aktiveras.

### KRITISK 5: Update-mekanism saknar anti-rollback
**Fil/sektion:** ADR-001 §5.3, §8.4

**Vektor:** Angripare kompromiterar `releases.piholster.net` (eller MITM:ar från en network position) och serverar en äldre, korrekt signerad version (säg v1.0.3) som har en känd RCE. Minisign-signaturen är giltig — den signerades verkligen av CTO:erna en gång — men versionen är vulnerable. Pi:n applicerar gladligen "uppdateringen".

**Påverkan om ej åtgärdat:** Hela signing-infrastrukturens värde är reducerat. En tidigare-signed-but-vulnerable release är en perpetuell zero-day-leverans.

**Exakt fix:**
1. Embedded `MIN_VERSION` (semver) i image/binär. Update aborteras om server-served `version < MIN_VERSION` ELLER `< current_version`.
2. Signaturen täcker även en `released_at` timestamp; klient avvisar om `released_at < current_installed_released_at`.
3. Lägg till `revoked_versions.json` (signerad av CTO) som listar uttryckligen återkallade versioner; embed:as i nya releases och uppdateras vid varje release-poll.

---

## 5. Rekommendationer (utöver kritiska fynd)

| # | Rekommendation | Prioritet | Fil/Sektion |
|---|---|---|---|
| R1 | Process-separation av ARP-scanner (se KRITISK 1) | Kritisk | §5.3, §7.2 |
| R2 | Firstboot ordering + iptables-stängd-tills-klar (se KRITISK 2) | Kritisk | §7.2 |
| R3 | DNS rebinding-skydd: Host allowlist + Origin + CSRF token (se KRITISK 3) | Kritisk | §8.2 |
| R4 | Security headers + CSP (se KRITISK 4) | Kritisk | ny §8.7 |
| R5 | Anti-rollback för uppdateringar (se KRITISK 5) | Kritisk | §8.4 |
| R6 | Per-account global lockout som komplement till per-IP-lockout, alert vid ny IP-login | Hög | §8.1 |
| R7 | `govulncheck` + SLSA-provenance + SBOM + SHA-pinned Actions + gitleaks i CI | Hög | §6.1 |
| R8 | systemd hardening: `ProtectKernelTunables=true`, `ProtectKernelModules=true`, `ProtectControlGroups=true`, `RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX AF_NETLINK`, `RestrictNamespaces=true`, `LockPersonality=true`, `MemoryDenyWriteExecute=true`, `SystemCallFilter=@system-service`, `SystemCallArchitectures=native`, `CapabilityBoundingSet=CAP_NET_BIND_SERVICE` (efter ARP-split), `ReadWritePaths=/var/lib/piholster /run/piholster` | Hög | §5.3 |
| R9 | Cert-cipher: Ed25519 root + ECDSA P-256 leaf, 13-månaders leaf, 10-års root, automatisk pre-expiry-rotation utan re-trust-prompt | Hög | §3.6 |
| R10 | `key_version`-kolumn i `query_log` redan i v1 | Hög | §8.5 |
| R11 | Telegram-token krypterad med device key, rate-limited utgång, signed messages med användar-synlig prefix | Medium | §8.3 |
| R12 | Avahi `publish-hinfo=no`, `publish-workstation=no`, `disable-publishing=yes` förutom explicit publishade tjänster | Medium | §3.5 |
| R13 | Update poll med ±2h jitter per device | Medium | §5.3 |
| R14 | SSH-konfig hardened när användaren enables: pubkey-only, ingen root, LAN-CIDR-only iptables-regel | Medium | §7.2 `02-harden` |
| R15 | DoH/DoT certificate pinning för upstream-resolvers (Quad9/Cloudflare publika nycklar embedded), aborterar resolution om CA-chain validerar men SPKI inte matchar | Medium | §3.3 |
| R16 | Sätt explicit i ADR att frontend-builden auditeras för externa URL:er (CI-step: grep `_app/` for `https://` förutom whitelisted) | Medium | §3.2 |
| R17 | ARP-skannern: passiv default, aktiv gated bakom Admin + jitter | Låg | §3.3 (egentligen ny sektion) |
| R18 | Action LED som boot-status-indikator (röd-blink firstboot, grön klart) | Låg | §7.2 |
| R19 | Backup-export av settings via Admin UI ska kräva password re-entry + producera krypterad fil med passphrase | Låg | nytt |
| R20 | CSP `report-uri` som loggar lokalt — om en CSP-rapport utlöses är det en larm-värd händelse | Låg | ny §8.7 |

---

## 6. Hotmodell-komplettering

ADR-001 §8.6 listar 5 hot. Den är på rätt nivå men har luckor. Följande hot saknas eller är otillräckligt behandlade:

### 6.1 Hot som saknas

**H-A: DNS rebinding från publik webbsida (KRITISK)**
- Ej nämnd. Se KRITISK 3. Detta är det mest realistiska remote-attacken mot LAN-bound admin-UI:n.
- Mitigering: Host allowlist + Origin + CSRF.

**H-B: Kompromettead device på LAN (kategori, inte instans) (HÖG)**
- ADR nämner "compromised IoT" som hostile environment men har inte en explicit attack-tree. En kompromettead smart-TV kan: ARP-spoofa Pi:n, sniffa DoH/DoT TLS handshakes (ej innehåll men SNI), mDNS-spoofa `piholster.local` mot andra LAN-devices, brute-force admin via /api/login.
- Mitigering finns delvis (lockout, TLS) men SNI-läckage av upstream-domäner är inte adresserad. ECH (Encrypted Client Hello) bör nämnas som framtida åtgärd i ADR-002/003.

**H-C: Browser-extension i admin-användarens browser (MEDIUM)**
- En malicious extension kan läsa cookies oavsett HttpOnly om den har host-permissioner. Mitigering: kort idle timeout (redan 30 min), Telegram-alert vid nya logins, Web Authentication (passkey/WebAuthn) som andra-faktor i framtid.

**H-D: Side-channel via DNS-svarstider (MEDIUM)**
- En LAN-angripare som mäter DNS-svarstider kan skilja "blocklist hit" från "uppstream resolved" från "cache hit". Detta läcker browsing-historik i aggregat.
- Mitigering: konstant svarstid via artificiell delay till maxgolv (säg 30 ms), eller responsen padded med EDNS0-padding (`EDE`-koden 24, "Synthesized response").

**H-E: Supply-chain på frontend dev-deps (MEDIUM)**
- ADR §8.6 nämner Go modules men inte npm. SvelteKit har många transitive npm-deps. En postinstall-script i en kompromettead dep kan exfiltrera signing-keys från en utvecklares laptop.
- Mitigering: `pnpm`'s `--ignore-scripts` i CI; lockfile review; `pnpm audit` blocking i ci.yml; ingen npm-publish-key på utvecklarmaskiner.

**H-F: Pi:n flyttad mellan nätverk (LÅG)**
- Användaren tar Pi:n till en vän. mDNS broadcast på det nya LAN exponerar nu vännens nätverkstopologi. Old session-tokens kvarstår.
- Mitigering: detektera default-route-byte; vid byte tvinga ny login + alert.

**H-G: Tidsattack mot Argon2id-jämförelsen (LÅG, defensivt mitigerat redan)**
- `subtle.ConstantTimeCompare` ska användas explicit. Värt en mening i ADR.

**H-H: Kallstartsfönster mellan boot och första NTP-sync (LÅG)**
- Pi 3 har ingen RTC. Klockan vid boot kan vara 1970 tills NTP synkar. Detta påverkar:
  - TLS-cert-validering (cert kan se "ej giltigt än" ut om klocka är 1970)
  - JWT/session expires_at-jämförelser
  - DoT/DoH som behöver giltig tid
- Mitigering: vänta med att starta `piholsterd` tills `time-sync.target` (`After=time-sync.target`).

**H-I: CSRF via WebSocket (MEDIUM)**
- WebSockets är inte skyddade av CORS. ADR specificerar inte Origin-validering på WS-handshake. En LAN-angripare med CSRF-möjlighet kan öppna en WS från en kompromettead browser.
- Mitigering: validera `Origin`-header vid WS-handshake; reject allt utom egna kända origins. Kräv CSRF-token i första WS-meddelandet.

**H-J: Subdomain-takeover av blocklist-källor (LÅG)**
- ADR §3.3 / §4 packages/blocklists. Om en blocklist-källa går offline och dess domän blir tillgänglig, kan angripare publicera en "blocklist" som listar `your-bank.com` etc.
- Mitigering: blocklists pinnas till specifika commits/SHA256; en uppdatering kräver review.

### 6.2 Befintliga hot som behöver utvecklas

**Stolen SD card (ADR §8.6):** ADR säger "settings + device list NOT encrypted". Det är ett rimligt v1-tradeoff men hotet har en glömd dimension: **WiFi-SSID och router-modeller i ARP-data avslöjar hushållets exakta hardware-fingerprint** för en angripare som hittar en kasserad SD-card. Lägg till en mening att "user-facing setup wizard varnar tydligt vid factory-reset/recycling".

**Malicious upstream DNS (ADR §8.6 "out of scope"):** Bör inte vara fullt out of scope. Lägg till mitigering: vi bundlar minst 2 default-upstreams (Quad9 + Cloudflare) och rotar mellan dem; en användare kan välja att kräva DNSSEC-validering i Admin (CoreDNS `dnssec`-plugin); `forward` plugin med `policy random` för att inte ge en enskild upstream hela query-strömmen.

---

## 7. Slutsats

ADR-001 är en stark grund. Inga av de kritiska fynden är arkitektoniska felval — alla är utelämnade detaljer som ska in i §8 eller en kommande ADR-002. Mitt rekommenderade flöde innan release:

1. Adressera de 5 KRITISKA fynden i en revidering av ADR-001 (eller en explicit ADR-001b "Security baseline addendum").
2. Ge ADR-002 (remote access / Tailscale) ett uttalat krav att inte exponera admin-UI över Tailscale utan en separat opt-in-flagga och en separat session-context.
3. Lyfta fram ADR-003 (encryption-at-rest) före v1-release om query log-kryptering ska levereras i v1; alternativt avgränsa v1 till "loggar är off-by-default" tills ADR-003 levereras.

Med dessa ändringar är PiHolster v1 säker att släppa till en publik open-source-publik. Utan dem skulle den första bug bounty-rapporten vara DNS rebinding inom en vecka.

---

## 8. Referenser

- Granskat dokument: `C:\Users\bodin\piholster\docs\ADR-001-architecture.md`
- Relaterade dokument: `C:\Users\bodin\piholster\docs\ROADMAP.md`, `C:\Users\bodin\piholster\docs\SPRINT-1.md`, `C:\Users\bodin\piholster\docs\CHANGE-PROCESS.md`
- Externa standarder: OWASP ASVS 4.0 §3 (Session), §4 (Access Control), §13 (API), §14 (Config); CIS Raspberry Pi OS Benchmark; SLSA v1.0.

---

*Granskning klar 2026-05-01.*
