# ADR-002: Security Hardening (Response to SECURITY-REVIEW-001)

- **Status:** Accepted
- **Date:** 2026-05-01
- **Author:** CTO (PiHolster)
- **Decision scope:** Response to all 5 critical findings + 10 nya hot från SECURITY-REVIEW-001. Konkret implementeringsanvisning per fynd. MVP/v1.0-prioritering. Hotmodell-uppdatering.
- **Supersedes:** ADR-001 §5.3 (CAP_NET_RAW), §7.2 (firstboot ordering), §8.2 (sessions/CSRF), §8.4 (update integrity). Lägger till ny §8.7 (HTTP security headers) i ADR-001-kanonen.
- **Relaterat:** `docs/SECURITY-REVIEW-001.md`, `docs/ADR-001-architecture.md`.

---

## 1. Context

Säkerhetsteamet har granskat ADR-001 och identifierat 5 kritiska luckor som måste stängas innan publik release. Granskningen är solid — inga av fynden är arkitektoniska felval, alla är utelämnade detaljer eller saknade implementeringsbeslut. Denna ADR accepterar samtliga fem kritiska fynd, beslutar exakt implementering, och låser MVP/v1.0-scope.

**Bottom line:**
- **Alla 5 kritiska fynd accepteras.** Ingen modifieras nedåt; två modifieras uppåt (mer aggressivt skydd än granskningen föreslog).
- **Fynd 1, 2, 3, 4 är MVP (v0.1).** De är blockers för första publika image. Utan dem är produkten inte säker att släppa.
- **Fynd 5 (anti-rollback) är v1.0.** Update-mekanismen i sig levereras inte i v0.1 (MVP är "flash SD, kör lokalt"); anti-rollback måste finnas första gången update-mekanismen tas i bruk.
- **`piholster-arpd` blir en separat binär, inte subprocess via `cmd.Process`.** Motivering i §3.1.

---

## 2. Decision Summary

| # | Fynd | Beslut | Scope | Övergripande mekanism |
|---|---|---|---|---|
| 1 | CAP_NET_RAW på samma binär som HTTP | **Accepterat — mer aggressivt** | MVP (v0.1) | Separat binär `piholster-arpd`, egen UID, Unix socket, length-prefixed protobuf |
| 2 | Firstboot-fönster oskyddat | **Accepterat** | MVP (v0.1) | systemd ordering + iptables default-DROP + LED-mönster + Avahi-publish gated |
| 3 | DNS rebinding-skydd saknas | **Accepterat** | MVP (v0.1) | Host-allowlist middleware + Origin-validering + double-submit CSRF + WS Origin-check |
| 4 | CSP/säkerhetsheaders | **Accepterat** | MVP (v0.1) | Strikt CSP via centralt `headers`-middleware, `report-uri` lokalt loggad |
| 5 | Anti-rollback saknas | **Accepterat** | v1.0 (när auto-update aktiveras) | Embedded `MIN_VERSION` + `released_at` i signaturen + `revoked_versions.json` |

---

## 3. Detaljerade beslut

### 3.1 Fynd 1: Process-separation av ARP-scanner

**Beslut:** Accepterat. `piholster-arpd` blir en **separat statisk Go-binär**, inte en subprocess startad via `os/exec` från `piholsterd`.

#### 3.1.1 Motivering: separat binär framför `cmd.Process`-subprocess

Granskningen lämnade frågan öppen. Här är beslutet och varför:

| Kriterium | Separat binär | Subprocess via `cmd.Process` |
|---|---|---|
| Capability-isolation | `setcap CAP_NET_RAW=+ep` endast på `piholster-arpd`. `piholsterd` har aldrig `CAP_NET_RAW`, ens som föräldraprocess. | Föräldern måste ha `CAP_NET_RAW` i ambient set för att barnet ska ärva, eller barnet måste ha setcap. Om barnet har setcap är det de facto en separat binär. |
| UID-isolation | `User=piholster-arp` (UID 998) i systemd-unit. `piholsterd` (UID 999) kan inte signala/`ptrace`. | Subprocess ärver UID från föräldern om inte `Credential` sätts. Komplicerad. |
| Restart-isolation | systemd `Restart=on-failure` per unit. ARP-crash dödar inte HTTP. | Föräldraprocessen måste implementera supervision logic. Om `piholsterd` crashar dör barnet med, även om barnet var stabilt. |
| Update atomicitet | Binärerna kan signeras separat. `piholster-arpd` ändras sällan; vi behöver inte re-signa hela paketet vid en HTTP-handler-ändring. | Allt i en binär = monolitisk update. |
| Memory footprint | ~5 MB RSS för `piholster-arpd` separat. Inom budget på 1 GB. | Identisk RSS, men allt i en process; vid OOM dör allt. |
| Debugging | `journalctl -u piholster-arpd` ger ren logg. | Blandade loggar i föräldraprocess. |
| Komplexitet i `piholsterd` | Nej — pratar bara med Unix socket. | Måste hantera lifecycle, stderr, restart, zombie-reaping. |

Separat binär vinner på alla relevanta axlar utom "1 binär att deploya" — vilket är värdelöst när vi redan har en signed-image-pipeline.

#### 3.1.2 Implementering

**Repo-layout (utöka ADR-001 §4):**
```
apps/
├── piholsterd/                  (befintlig)
└── piholster-arpd/              (NY)
    ├── cmd/piholster-arpd/main.go
    ├── internal/
    │   ├── scanner/             (gopacket passive sniffer + active probe)
    │   ├── proto/               (delad med piholsterd via internal module)
    │   └── ipc/                 (Unix socket server)
    └── go.mod
packages/
└── arpproto/                    (NY — delat protobuf-schema)
    ├── arpproto.proto
    └── go.mod
```

**Wire-protokoll: length-prefixed protobuf, inte JSON.**
Granskningen föreslog "length-prefixed JSON". Vi går med protobuf istället eftersom (a) paketstorlek är mindre, (b) field-validering är typad, (c) ARP-frames är binär data som klär sig dåligt i JSON. Schema:

```protobuf
// packages/arpproto/arpproto.proto
syntax = "proto3";
package arpproto;

message Envelope {
  oneof msg {
    DeviceObserved observed = 1;
    ProbeRequest probe_req = 2;
    ProbeAck probe_ack = 3;
    Heartbeat hb = 4;
  }
}

message DeviceObserved {
  bytes mac = 1;            // 6 bytes
  bytes ipv4 = 2;           // 4 bytes, optional
  bytes ipv6 = 3;           // 16 bytes, optional
  int64 first_seen_unix = 4;
  int64 last_seen_unix = 5;
  string vendor_oui = 6;    // resolved on piholsterd-side, not in arpd
}

message ProbeRequest {
  bytes target_ipv4 = 1;
  uint32 jitter_ms = 2;
}

message ProbeAck {
  bool accepted = 1;
  string reason = 2;
}

message Heartbeat {
  int64 unix_nano = 1;
  uint32 frames_seen = 2;
  uint32 errors = 3;
}
```

Frame-format på socket: `uint32 BE length || protobuf bytes`. Maxlängd 64 KiB; större = drop + close.

**Unix socket:**
- Path: `/run/piholster/arp.sock`
- Owner: `root:piholster`
- Mode: `0660`
- Skapas av en systemd `RuntimeDirectory=piholster` på `piholster-arpd.service`.
- `piholsterd` ansluter som client; `piholster-arpd` är server.

**Senior Go-utvecklarens checklista för `piholster-arpd`:**

```go
// apps/piholster-arpd/cmd/piholster-arpd/main.go
func main() {
    // 1. Validera capabilities innan vi rör nätverket. Om CAP_NET_RAW saknas: exit 1.
    if err := requireCap(unix.CAP_NET_RAW); err != nil { log.Fatal(err) }

    // 2. Open AF_PACKET-socket EARLY, sedan drop CAP_NET_RAW from effective set.
    handle, err := openARPHandle("eth0") // gopacket pcap
    if err != nil { log.Fatal(err) }
    if err := dropCap(unix.CAP_NET_RAW); err != nil { log.Fatal(err) }

    // 3. Lyssna på Unix socket. SO_PEERCRED för att verifiera att klient är UID 999 (piholsterd).
    ln, err := net.Listen("unix", "/run/piholster/arp.sock")
    if err != nil { log.Fatal(err) }
    defer ln.Close()

    // 4. systemd notify "READY=1" så piholsterd.service kan vänta på oss via After=.

    // 5. Två goroutines: scanner.Run(ctx, sink) och ipc.Serve(ctx, ln, sink).
    //    Scanner skriver till en buffered channel; IPC-serveren broadcastar.

    // 6. Active probe: endast efter ProbeRequest från klient. Validera target inom LAN-CIDR.

    // 7. Graceful shutdown via SIGTERM.
}
```

**`piholsterd`-sidan (klient):**

```go
// apps/piholsterd/internal/arp/client.go
type Client struct {
    conn net.Conn
    mu   sync.RWMutex
    devs map[string]*Device // by MAC
}

func Dial(ctx context.Context) (*Client, error) {
    c, err := (&net.Dialer{}).DialContext(ctx, "unix", "/run/piholster/arp.sock")
    if err != nil { return nil, err }
    // SO_PEERCRED-check: server måste vara UID 998 (piholster-arp)
    if err := verifyPeerUID(c, 998); err != nil { c.Close(); return nil, err }
    return &Client{conn: c, devs: map[string]*Device{}}, nil
}
```

**systemd-units:**

```ini
# /etc/systemd/system/piholster-arpd.service
[Unit]
Description=PiHolster ARP scanner (privileged)
After=network-online.target piholster-firstboot.service
Requires=piholster-firstboot.service
Before=piholsterd.service

[Service]
Type=notify
ExecStart=/usr/local/bin/piholster-arpd --interface=eth0 --socket=/run/piholster/arp.sock
User=piholster-arp
Group=piholster
RuntimeDirectory=piholster
RuntimeDirectoryMode=0750
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=false           # AF_PACKET kräver /dev/packet — keep
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_PACKET AF_UNIX AF_NETLINK
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/piholsterd.service (uppdaterad)
[Unit]
Description=PiHolster main daemon
After=network-online.target time-sync.target piholster-firstboot.service piholster-arpd.service
Requires=piholster-firstboot.service piholster-arpd.service

[Service]
Type=notify
ExecStart=/usr/local/bin/piholsterd
User=piholster
Group=piholster
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_INET AF_INET6 AF_UNIX
RestrictNamespaces=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallFilter=@system-service
SystemCallArchitectures=native
ReadWritePaths=/var/lib/piholster /run/piholster
Restart=on-failure
RestartSec=2s

[Install]
WantedBy=multi-user.target
```

**Borttagning från `image/stage-piholster/00-install`:**
Den gamla raden
```
setcap cap_net_bind_service,cap_net_raw=+ep /usr/local/bin/piholsterd
```
ersätts med
```
setcap cap_net_bind_service=+ep /usr/local/bin/piholsterd
setcap cap_net_raw=+ep         /usr/local/bin/piholster-arpd
```

**CI:** ny check i `ci.yml` som assertar att `getcap piholsterd` INTE innehåller `cap_net_raw`. Om någon av misstag lägger tillbaka det fallerar bygget.

---

### 3.2 Fynd 2: Firstboot-fönstret oskyddat

**Beslut:** Accepterat. Tre lager skyddar firstboot-fönstret.

#### 3.2.1 Implementering

**Lager 1: iptables default-DROP innan firstboot.**

`02-harden` modifieras så att `iptables-persistent` levererar en initial regelfil som DROP:ar **allt** inkommande utom loopback. Firstboot-tjänsten lägger till accept-reglerna **efter** att password+cert är genererat.

```bash
# image/stage-piholster/02-harden/files/iptables-initial.rules
*filter
:INPUT DROP [0:0]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [0:0]
-A INPUT -i lo -j ACCEPT
-A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
-A INPUT -p icmp --icmp-type echo-request -m limit --limit 5/sec -j ACCEPT
COMMIT
```

Firstboot körs `Type=oneshot RemainAfterExit=yes`. Sista steget i firstboot:

```bash
# /var/lib/piholster/firstboot.sh, sista steget
iptables -A INPUT -p udp --dport 53   -j ACCEPT
iptables -A INPUT -p tcp --dport 53   -j ACCEPT
iptables -A INPUT -p tcp --dport 80   -j ACCEPT
iptables -A INPUT -p tcp --dport 443  -j ACCEPT
iptables -A INPUT -p udp --dport 5353 -j ACCEPT  # mDNS
iptables-save > /etc/iptables/rules.v4
```

**Lager 2: systemd ordering.**

```
piholster-firstboot.service  (oneshot, RemainAfterExit)
   |
   +-- After: network-online.target, time-sync.target
   +-- Before: piholster-arpd.service, piholsterd.service, piholster-avahi.service
```

`piholsterd.service` har `Requires=piholster-firstboot.service` så att om firstboot misslyckas startar inte HTTP-servern alls.

**Lager 3: Avahi-publish gated.**

ADR-001 §3.5 publicerade `_http._tcp` och `_https._tcp` via statiska `/etc/avahi/services/*.service`-filer. Vi tar bort dessa och publicerar programmatiskt **efter** firstboot:

```ini
# /etc/systemd/system/piholster-avahi-publish.service
[Unit]
After=piholsterd.service
Requires=piholsterd.service

[Service]
Type=simple
ExecStart=/usr/bin/avahi-publish -s piholster _https._tcp 443
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

`avahi-daemon.conf` får också:
```
[publish]
publish-hinfo=no
publish-workstation=no
disable-publishing=no
publish-aaaa-on-ipv4=no
```

**Lager 4: LED-feedback.**

Pi 3 ACT-LED är programmerbar via `/sys/class/leds/led0/`. Firstboot:
- Start: `echo timer > /sys/class/leds/led0/trigger; echo 100 > /sys/class/leds/led0/delay_on; echo 100 > /sys/class/leds/led0/delay_off` (snabb-blink röd via PWR-LED-pseudonym)
- Slut: `echo default-on > /sys/class/leds/led0/trigger`

Detta dokumenteras i packaging-insertet: "Vänta tills den gröna lampan lyser stadigt innan du connectar."

**Lager 5: Klockvänta.**

`piholsterd.service` får `After=time-sync.target` (täcker hot H-H, kallstartsfönstret). På Pi 3 utan RTC: chrony eller systemd-timesyncd ska ha synkat innan vi gör TLS-validering.

#### 3.2.2 Test

Lägg till i `ci.yml` ett Pi-in-loop-test:
```bash
# Flash fresh image, boot, omedelbart innan firstboot är klar:
nmap -p 53,80,443 $PI_IP   # ska returnera "filtered" på alla
sleep 60                    # vänta firstboot
nmap -p 53,80,443 $PI_IP   # ska returnera "open" på alla
```

---

### 3.3 Fynd 3: DNS rebinding-skydd

**Beslut:** Accepterat. Fyra mekanismer i kombination.

#### 3.3.1 Implementering

**Mekanism 1: Host-header allowlist (server-side middleware).**

```go
// apps/piholsterd/internal/api/middleware/host.go
package middleware

import (
    "net"
    "net/http"
    "strings"
)

type HostAllowlist struct {
    Allowed map[string]struct{} // lower-cased, port stripped
    LANCIDR *net.IPNet
}

func NewHostAllowlist(lanCIDR *net.IPNet) *HostAllowlist {
    return &HostAllowlist{
        Allowed: map[string]struct{}{
            "piholster.local": {},
            "piholster.lan":   {},
        },
        LANCIDR: lanCIDR,
    }
}

func (h *HostAllowlist) Wrap(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        host := r.Host
        if i := strings.LastIndex(host, ":"); i != -1 {
            host = host[:i]
        }
        host = strings.ToLower(host)

        if _, ok := h.Allowed[host]; ok {
            next.ServeHTTP(w, r)
            return
        }
        if ip := net.ParseIP(host); ip != nil && h.LANCIDR.Contains(ip) {
            next.ServeHTTP(w, r)
            return
        }
        http.Error(w, "Misdirected Request", http.StatusMisdirectedRequest) // 421
    })
}
```

LAN-CIDR detekteras vid boot från default-route-interfacet. Hot-reload när `default-route` ändras (täcker hot H-F, "Pi flyttad mellan nätverk").

**Mekanism 2: Origin-header validering (state-changing endpoints).**

```go
// apps/piholsterd/internal/api/middleware/origin.go
func RequireOrigin(allowedOrigins []string) func(http.Handler) http.Handler {
    set := map[string]struct{}{}
    for _, o := range allowedOrigins { set[o] = struct{}{} }
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
                next.ServeHTTP(w, r)
                return
            }
            origin := r.Header.Get("Origin")
            if origin == "" {
                http.Error(w, "Origin required", http.StatusForbidden)
                return
            }
            if _, ok := set[origin]; !ok {
                http.Error(w, "Origin not allowed", http.StatusForbidden)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

`allowedOrigins` byggs vid boot: `https://piholster.local`, `https://piholster.lan`, `https://<LAN-IP>`. Uppdateras på interface-byte.

**Mekanism 3: Double-submit CSRF-token.**

```go
// apps/piholsterd/internal/api/middleware/csrf.go
const csrfHeader = "X-PiHolster-CSRF"
const csrfCookie = "piholster_csrf"

func CSRF(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
            // Mint token if not present
            if _, err := r.Cookie(csrfCookie); err != nil {
                tok := randomBase64(32)
                http.SetCookie(w, &http.Cookie{
                    Name: csrfCookie, Value: tok, Path: "/",
                    HttpOnly: false, // läsbar av JS för att kunna skickas i header
                    Secure: true, SameSite: http.SameSiteStrictMode,
                })
            }
            next.ServeHTTP(w, r)
            return
        }
        c, err := r.Cookie(csrfCookie)
        if err != nil {
            http.Error(w, "CSRF cookie missing", http.StatusForbidden); return
        }
        h := r.Header.Get(csrfHeader)
        if h == "" || subtle.ConstantTimeCompare([]byte(h), []byte(c.Value)) != 1 {
            http.Error(w, "CSRF token mismatch", http.StatusForbidden); return
        }
        next.ServeHTTP(w, r)
    })
}
```

Frontend SvelteKit api-client läser cookie och sätter header på varje fetch.

**Mekanism 4: WebSocket Origin-check (täcker hot H-I).**

```go
// apps/piholsterd/internal/api/ws.go
var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return originAllowlist[r.Header.Get("Origin")]
    },
}
```

Plus krav på CSRF-token i första WS-meddelandet (auth-handshake).

#### 3.3.2 Middleware-ordning

I `apps/piholsterd/internal/api/router.go`:
```
chi.Router.Use(
    HostAllowlist.Wrap,        // 1. avsluta DNS rebinding tidigt
    SecurityHeaders,           // 2. se 3.4
    RequireOrigin,             // 3. på state-changing
    CSRF,                      // 4. double-submit
    Session.Verify,            // 5. session-auth
    RequestID, Logger, Recover,
)
```

#### 3.3.3 Test

Ny `ci.yml`-step:
```bash
# DNS rebinding-skydd
curl -k -H "Host: evil.com" https://localhost/api/login -o /dev/null -w "%{http_code}\n" | grep -q 421
curl -k -H "Origin: https://evil.com" -X POST https://localhost/api/login | grep -q 403
```

---

### 3.4 Fynd 4: CSP och säkerhetsheaders

**Beslut:** Accepterat med en modifiering: `style-src` får inte `'unsafe-inline'`. Vi tvingar SvelteKit att producera nonce-baserade styles.

#### 3.4.1 Implementering

**Centralt headers-middleware:**

```go
// apps/piholsterd/internal/api/middleware/headers.go
package middleware

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"
)

func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        nonce := genNonce() // 16 bytes, base64
        ctx := contextWithNonce(r.Context(), nonce)

        h := w.Header()
        h.Set("Content-Security-Policy",
            "default-src 'none'; "+
            "script-src 'self' 'nonce-"+nonce+"'; "+
            "style-src 'self' 'nonce-"+nonce+"'; "+
            "img-src 'self' data:; "+
            "connect-src 'self' wss://piholster.local wss://piholster.lan; "+
            "font-src 'self'; "+
            "frame-ancestors 'none'; "+
            "base-uri 'none'; "+
            "form-action 'self'; "+
            "report-uri /api/csp-report")
        h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
        h.Set("X-Content-Type-Options", "nosniff")
        h.Set("X-Frame-Options", "DENY")
        h.Set("Referrer-Policy", "no-referrer")
        h.Set("Permissions-Policy",
            "geolocation=(), camera=(), microphone=(), usb=(), payment=(), "+
            "accelerometer=(), gyroscope=(), magnetometer=(), midi=(), "+
            "fullscreen=(self), display-capture=()")
        h.Set("Cross-Origin-Opener-Policy", "same-origin")
        h.Set("Cross-Origin-Resource-Policy", "same-origin")
        h.Set("Cross-Origin-Embedder-Policy", "require-corp")

        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func genNonce() string {
    b := make([]byte, 16)
    rand.Read(b)
    return base64.StdEncoding.EncodeToString(b)
}
```

**SvelteKit-integration:**
- `svelte.config.js` aktiverar `csp: { mode: 'nonce' }`. SvelteKit injicerar `nonce`-attribut på `<script>` och `<style>` baserat på en placeholder.
- `index.html`-template har `<meta http-equiv="Content-Security-Policy" content="...{NONCE}...">` placeholder.
- Go-servern, vid serving av `index.html`, läser nonce från context och ersätter `{NONCE}` innan write.

**`/api/csp-report`-handler:**

```go
// apps/piholsterd/internal/api/csp_report.go
func cspReport(w http.ResponseWriter, r *http.Request) {
    if r.ContentLength > 16*1024 { http.Error(w, "too large", 413); return }
    var body json.RawMessage
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        http.Error(w, "bad json", 400); return
    }
    log.Warn().RawJSON("csp", body).Msg("csp violation")
    // Larma via Telegram om >5 violations / 5 min (rate-limited)
    cspAlerter.Tick()
    w.WriteHeader(204)
}
```

#### 3.4.2 Frontend-build-audit i CI

Granskningens R16:
```bash
# ci.yml frontend-build-step
if grep -RE 'https?://' apps/web/.svelte-kit/output/client/_app/ | grep -v 'piholster.local\|piholster.lan'; then
  echo "External URL detected in frontend build"; exit 1
fi
```

#### 3.4.3 HSTS-not

HSTS är OK på LAN eftersom vi alltid serverar över HTTPS. Vi `preload`:ar **inte** (vi finns inte i Chromiums preload-lista och vill inte). `includeSubDomains` är säkert eftersom `piholster.local` inte har subdomäner.

---

### 3.5 Fynd 5: Anti-rollback för uppdateringar

**Beslut:** Accepterat. Tre lager: `MIN_VERSION`, `released_at` i signatur, `revoked_versions.json`.

**Scope:** v1.0. Auto-update levereras inte i MVP (v0.1) — användaren flashar om SD-kortet manuellt. När auto-update aktiveras MÅSTE alla tre lager vara på plats; det ska inte gå att aktivera auto-update utan dem.

#### 3.5.1 Implementering

**Lager 1: Embedded `MIN_VERSION` i binären.**

```go
// apps/piholsterd/internal/version/version.go
package version

// Sätts vid build via -ldflags
var (
    Version    = "dev"          // ex. "1.2.3"
    ReleasedAt = "0"             // unix timestamp string
    MinVersion = "1.0.0"         // monotont ökande, hand-maintained per release
)
```

Build-flagga i `release-binary.yml`:
```bash
go build -ldflags "\
  -X piholster/internal/version.Version=$VERSION \
  -X piholster/internal/version.ReleasedAt=$RELEASED_AT \
  -X piholster/internal/version.MinVersion=$MIN_VERSION" \
  ./cmd/piholsterd
```

**Lager 2: Signed update-manifest.**

Update-mekanismen pollar `releases.piholster.net/manifest.json` (signerad med samma minisign-nyckel som binären). Manifestet:

```json
{
  "version": "1.4.0",
  "released_at": 1735689600,
  "min_version": "1.2.0",
  "binary_url": "https://releases.piholster.net/piholsterd-1.4.0-linux-armv7",
  "binary_sha256": "...",
  "binary_minisig_url": "https://releases.piholster.net/piholsterd-1.4.0-linux-armv7.minisig",
  "revoked_versions": ["1.0.3", "1.1.0", "1.1.1"],
  "revoked_min_version": "1.2.0"
}
```

`manifest.json.minisig` levereras separat. Klienten verifierar signatur **före** parsing.

**Lager 3: Anti-rollback-check.**

```go
// apps/piholsterd/internal/update/rollback.go
func (u *Updater) AcceptManifest(m *Manifest, current Manifest) error {
    if !semver.Less(current.Version, m.Version) {
        return errors.New("manifest version not greater than installed")
    }
    if semver.Less(m.Version, current.MinVersion) {
        return errors.New("manifest version below MIN_VERSION embedded in current build")
    }
    if m.ReleasedAt <= current.ReleasedAt {
        return errors.New("manifest released_at not greater than installed")
    }
    for _, rv := range m.RevokedVersions {
        if rv == m.Version {
            return errors.New("manifest version is in revoked list")
        }
    }
    if semver.Less(m.Version, m.RevokedMinVersion) {
        return errors.New("manifest version below revoked_min_version")
    }
    return nil
}
```

**Lager 4: `revoked_versions.json` lokalt.**

Klienten cachear det senaste signerade manifestet i `/var/lib/piholster/last-manifest.json`. Vid varje update-poll: om server-served manifest har **lägre** `revoked_min_version` än cachat: avvisa (revocations är monotont ökande, sänkning är ett angrepp).

**Lager 5: Update-poll med jitter.**

```go
// apps/piholsterd/internal/update/scheduler.go
func (u *Updater) Schedule() time.Duration {
    base := 24 * time.Hour
    jitter := time.Duration(rand.Int63n(int64(4 * time.Hour))) - 2*time.Hour
    return base + jitter
}
```

Plus: device identity hash blir HTTP `User-Agent`-suffix `PiHolster/1.4.0 (id:abc123)` så vi kan se aggregat utan att korrelera per IP.

#### 3.5.2 Test

```go
func TestRollbackRejected(t *testing.T) {
    cur := Manifest{Version: "1.4.0", ReleasedAt: 1000, MinVersion: "1.2.0"}
    older := &Manifest{Version: "1.3.0", ReleasedAt: 500, MinVersion: "1.2.0"}
    if err := u.AcceptManifest(older, cur); err == nil {
        t.Fatal("expected rollback to be rejected")
    }
}
```

---

## 4. Sammanställd MVP/v1.0-prioritering

| Fynd | MVP (v0.1) | v1.0 | Motivering |
|---|---|---|---|
| 1: Process-split ARP | **JA** | (klart redan) | RCE i HTTP-handler är sannolikt under en bug bounty inom första 30 dagarna. Kan inte släppas utan. |
| 2: Firstboot ordering | **JA** | (klart redan) | Race-condition existerar från sekund 1 av första boot. |
| 3: DNS rebinding | **JA** | (klart redan) | Trivial att exploit:a från publik webbsida; en blogpost krävs för att brännare hela installbasen. |
| 4: CSP/headers | **JA** | (klart redan) | XSS-skydd måste finnas innan första PR mergeas; lägga till efteråt missar redan exponerade endpoints. |
| 5: Anti-rollback | NEJ | **JA** | MVP har ingen auto-update. Måste levereras innan auto-update aktiveras. |

Utöver de 5 kritiska fynden blir följande recommendations från §5 i granskningen också MVP-blockers:
- **R7**: `govulncheck` + SHA-pinned actions + `gitleaks` i `ci.yml` (billigt; gör nu).
- **R8**: utökad systemd-hardening (specificerad i §3.1.2 ovan; gör nu).

Resten flyttas till en backlog-tabell i `docs/SECURITY-BACKLOG.md` (skapas separat) och mappas till v1.0/v1.1.

---

## 5. Hotmodell-uppdatering

ADR-001 §8.6 listade 5 hot. Granskningen lägger till 10 nya. Dessa accepteras i sin helhet och inarbetas i den kanoniska hotmodellen.

### 5.1 Komplett hotmodell efter ADR-002

| ID | Hot | Källa | Mitigering | Status |
|---|---|---|---|---|
| H-1 | Hostile LAN device brute-forces admin | ADR-001 | Argon2id + per-IP lockout + per-account global lockout (R6) + new-IP-alert | Mitigated |
| H-2 | Passive DNS observer | ADR-001 | DoT/DoH upstream, fail-closed | Mitigated |
| H-3 | Stolen SD card | ADR-001 | Encrypted query log; settings/devices accepted exposure; recycling-warning i wizard | Partially mitigated (accepted) |
| H-4 | Malicious upstream DNS | ADR-001 (out of scope) | Bundled multi-upstream rotation + DNSSEC opt-in + SPKI pinning (R15) | Partially mitigated (uppgraderad från "out of scope") |
| H-5 | Supply-chain på Go module | ADR-001 | go.sum + Dependabot + govulncheck + SLSA attestation + SBOM + SHA-pinned actions + gitleaks | Mitigated |
| **H-A** | **DNS rebinding från publik webbsida** | SR-001 §6.1 | Host-allowlist + Origin + CSRF (§3.3) | **Mitigated i v0.1** |
| **H-B** | **Kompromettead device på LAN (klass)** | SR-001 §6.1 | Process-split ARP + LAN-CIDR + ECH (futures) | **Partially mitigated; ECH i v1.1** |
| **H-C** | **Browser-extension i admin-användarens browser** | SR-001 §6.1 | Idle timeout + Telegram-alert vid login + WebAuthn passkey (v1.0) | **Partially mitigated i v0.1; full i v1.0** |
| **H-D** | **Side-channel via DNS-svarstider** | SR-001 §6.1 | Konstant svarstid via padding/jitter + EDNS0 padding | **v1.0** |
| **H-E** | **Supply-chain på frontend npm-deps** | SR-001 §6.1 | `pnpm --ignore-scripts` i CI + lockfile review + `pnpm audit` blocking | **Mitigated i v0.1** |
| **H-F** | **Pi flyttad mellan nätverk** | SR-001 §6.1 | Default-route-detect + tvinga relogin + alert | **v1.0** |
| **H-G** | **Tidsattack mot Argon2id-jämförelse** | SR-001 §6.1 | `subtle.ConstantTimeCompare` explicit i `auth/argon2.go` | **Mitigated i v0.1** |
| **H-H** | **Kallstartsfönster innan NTP-sync** | SR-001 §6.1 | `After=time-sync.target` på piholsterd.service (§3.2.1) | **Mitigated i v0.1** |
| **H-I** | **CSRF via WebSocket** | SR-001 §6.1 | WS Origin-check + CSRF-token i första WS-meddelande (§3.3.1 mek 4) | **Mitigated i v0.1** |
| **H-J** | **Subdomain-takeover av blocklist-källor** | SR-001 §6.1 | Blocklists pinnas till specifika commits/SHA256; uppdatering kräver review | **v1.0** |

### 5.2 Hot som flyttas från "out of scope" till "mitigated"

ADR-001 §8.6 markerade "Malicious upstream DNS" som out of scope. Det är inte längre acceptabelt. ADR-002 lyfter upp det till H-4 ovan med konkret mitigering (R15 i granskningen).

### 5.3 Ny hot-kategori: residual

Hot där vi accepterar en kvarvarande risk:
- **H-3 (stolen SD)**: kvarvarande risk att en angripare med fysisk SD-card-åtkomst lär sig hushållets WiFi-SSID. Acceptabelt; recycling-flow varnar tydligt.
- **H-J (blocklist-takeover)**: i MVP kör vi med pinned-SHA blocklists; risk är låg.

---

## 6. Teknisk skuld införd av ADR-002

För att hålla skulden synlig (jfr instructionen om Claude_handoff.MD-stilen):

1. **`piholster-arpd`-binärens cross-build-flow** är inte testad än — `release-binary.yml` måste utökas med en separat artefakt och egen minisign-signering. Skuld: ~0.5 dag.
2. **CSP nonce-injection i SvelteKit** kräver att `adapter-static`'s output passerar genom Go-templating — adapter-static producerar redan inline-styles utan nonce. Vi behöver en post-build-step som ersätter `style="..."` med nonce-baserade `<style nonce="...">`. Skuld: ~1 dag eller acceptera `'unsafe-inline'` på style-src under MVP och fixa till v1.0.
3. **Avahi programmatic publish** ersätter statiska service-filer; vi måste hantera att `avahi-publish` är en long-running process som kan dö. Skuld: tunn, men existerar.
4. **`revoked_versions.json`-publishing-flow** finns inte än; ska byggas innan auto-update aktiveras. Skuld: ~1 dag.

Dessa läggs i `docs/Claude_handoff.MD` (eller motsvarande PiHolster-skuld-fil) i samband med implementering.

---

## 7. Konsekvenser

**Positiva:**
- Privilege escalation från HTTP RCE till LAN-wide ARP-spoof är arkitektoniskt blockerad, inte mitigerad — angripare måste nu kompromettera två separata processer.
- Första-boot-fönstret är stängt på fyra olika lager; en angripare som lyckas race:a en av dem fastnar på de andra.
- DNS rebinding är blockerat på fem lager (Host, Origin, CSRF, CSP frame-ancestors, WS Origin) — en lyckad attack kräver att alla fem fallerar samtidigt.
- CSP-nonce + report-uri ger oss både prevention och detektion av XSS-försök.
- Anti-rollback gör att en kompromiterad release-server inte kan downgrad-attackera installbasen.

**Negativa / accepterade tradeoffs:**
- En extra binär att underhålla (`piholster-arpd`). Acceptabelt; den är ~200-400 LOC och rör sig sällan.
- CSP nonce-injection kräver en SvelteKit post-build-step. Acceptabelt.
- CSRF double-submit lägger en HTTP header på varje POST. Trivialt.
- Anti-rollback kräver disciplinerad version-management (manuell `MIN_VERSION`-bumpning per release). Acceptabelt; det är en text-rad i en CI-fil per release.
- systemd-hardening med `SystemCallFilter=@system-service` kan oavsiktligt blockera framtida funktioner. Vi accepterar att varje ny syscall-användning kräver explicit tillägg.

**Prestandaimpact:**
- Host-allowlist + Origin + CSRF middleware: ~5 µs per request på Pi 3. Försumbart.
- Headers middleware med nonce-genererning: ~10 µs (crypto/rand-call). Försumbart.
- IPC over Unix socket mellan piholsterd och piholster-arpd: ~50 µs round-trip. Försumbart för ARP-event-frekvens.

---

## 8. Approval

Approved by CTO, 2026-05-01. Supersedes ADR-001 §5.3, §7.2, §8.2, §8.4. Lägger till §8.7 i ADR-001-kanonen (HTTP security headers).

ADR-003 (encryption-at-rest, från ADR-001 §10) och ADR-004 (Telegram alerts, inklusive R11) följer härnäst — båda kan utvecklas oberoende av denna ADR.
