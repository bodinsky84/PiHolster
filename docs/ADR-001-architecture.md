# ADR-001: PiHolster Architecture Foundation

- **Status:** Accepted
- **Date:** 2026-05-01
- **Author:** CTO (PiHolster)
- **Decision scope:** Tech stack, repo layout, environments, CI/CD, image build, baseline security

---

## 1. Context

PiHolster is an open-source home network security appliance running on a Raspberry Pi 3+ (ARMv7, 1 GB RAM, 100 Mbit Ethernet). It is "Pi-hole but with better UX and a security focus". The hard constraints driving every decision below are:

- **Hardware floor:** Raspberry Pi 3 with 1 GB RAM. We must reserve at least 300 MB for the OS, DNS cache, and ARP scanner. The web stack therefore cannot be a Node/Electron-style memory hog.
- **User floor:** "Grandma installs it." No terminal access, no SSH onboarding, no manual DNS forwarder editing. Onboarding is plug-in-and-flash-SD-card.
- **Trust model:** PiHolster sees every DNS query in the home. A breach is catastrophic. We assume the device is on a hostile LAN (compromised IoT, malicious guest devices) and design accordingly.
- **Distribution:** A flashable Raspberry Pi OS image, not `apt install`. The user must never `sudo` anything.

This ADR locks in the foundations so we are not re-litigating them at week 12.

---

## 2. Decision Summary

| Area | Decision |
|---|---|
| Backend language | **Go 1.22** |
| Frontend framework | **SvelteKit (static adapter)** |
| DNS engine | **CoreDNS (embedded as a Go library)** |
| Data store | **SQLite (via `modernc.org/sqlite`, pure Go, no CGO)** |
| Local discovery | **mDNS/Avahi for `piholster.local`** |
| Reverse proxy | **None — Go binary serves HTTPS directly on :443** |
| Repo layout | **Monorepo, Turborepo + Go workspaces** |
| Environments | **dev (laptop) / test (Pi in CI) / prod (user device)** |
| CI/CD | **GitHub Actions, with a self-hosted ARM64 runner for image builds** |
| Image build | **pi-gen fork, fully scripted, reproducible, signed** |
| Admin auth | **Argon2id + per-device random initial password printed on packaging insert** |

---

## 3. Tech Stack

### 3.1 Backend: Go 1.22

**Decision:** Single Go binary called `piholsterd`.

**Why Go (and not Node, Python, or Rust):**
- **Memory:** A Go DNS server with embedded HTTP UI sits at ~40-80 MB RSS. Node-based DNS forwarders sit at 150+ MB; on a 1 GB Pi that is unacceptable once we add Chromium-rendered admin UIs in family use. Python (Flask/uvicorn) is acceptable on RAM but slow at DNS hot paths.
- **Distribution:** One static binary, cross-compiled with `GOOS=linux GOARCH=arm GOARM=7`. No interpreter, no `pip`, no `node_modules`. This is the single biggest reason — it makes our image trivially small and our update story trivially `systemctl restart`.
- **Concurrency:** Goroutines are exactly the model needed for "100 concurrent DNS lookups + ARP sweep + WebSocket dashboard". Threads-per-request would not fit in 1 GB.
- **Ecosystem fit:** CoreDNS, miekg/dns, gopacket (for ARP), and chi (HTTP router) are all first-class, all maintained.
- **Why not Rust:** Excellent fit technically but slower iteration speed for a small team. We can rewrite the DNS hot path in Rust in 2027 if profiling forces it. Premature.

**Consequence:** All backend code is Go. No Python helpers, no shell scripts in the request path. Setup scripts (image build only) may use bash.

### 3.2 Frontend: SvelteKit with `@sveltejs/adapter-static`

**Decision:** SvelteKit, statically pre-rendered, served as files by the Go binary.

**Why:**
- **Bundle size:** A SvelteKit static build is ~30-60 KB JS for our scope. React equivalent is ~150 KB. Over a slow home WiFi, a non-technical user sees this as "the app feels instant" vs "the app hangs". This matters for the Grandma-mode promise.
- **No SSR runtime:** We do not want a Node runtime on the Pi. Static export means the Go binary serves `index.html` and a hashed `_app/` directory and we are done.
- **Three UI modes:** Grandma / Advanced / Admin map cleanly to three SvelteKit routes (`/`, `/advanced`, `/admin`) with shared components. Reactive stores handle the live device-list updates.
- **Why not React:** Bigger bundles, more boilerplate, no real benefit at our scope. We are not building Facebook.
- **Why not htmx + server-rendered Go templates:** Tempting for simplicity, but the live-updating device list and the WebSocket-driven alerts are painful in htmx. Svelte's reactivity wins here.

### 3.3 DNS Engine: CoreDNS as a library

**Decision:** Embed CoreDNS as a Go library. Custom plugins for blocklist matching and query logging. miekg/dns is the underlying DNS protocol library (CoreDNS already depends on it).

**Why:**
- **Battle-tested:** CoreDNS is the default DNS for Kubernetes. It has handled adversarial input at scale for years.
- **Plugin model:** We write `plugin/blocklist`, `plugin/queryjournal`, `plugin/familymode` as in-tree plugins. We do not fork CoreDNS; we vendor it and register additional plugins via the import block.
- **DoH/DoT upstream:** CoreDNS's `forward` plugin already supports `tls://` and `https://` upstreams. Default upstreams: Quad9 (`9.9.9.9` over DoT) and Cloudflare (`1.1.1.1` over DoH), user-selectable, with a hard rule that the resolver MUST NOT downgrade to plaintext if both fail (return SERVFAIL instead — fail closed).
- **Why not write our own:** DNS is full of edge cases (EDNS0, DNSSEC, truncation, IPv6 PTR). We will get this wrong. CoreDNS will not.
- **Why not dnsmasq:** Configuration is text-file-driven and painful to mutate from a UI without race conditions.

### 3.4 Data Store: SQLite (pure Go driver)

**Decision:** SQLite, accessed via `modernc.org/sqlite` (pure-Go, no CGO).

**Why:**
- **Single file, embedded:** No external service. Backup is `cp piholster.db`. Restore is the same.
- **Pure-Go driver:** Eliminates CGO from our build. CGO complicates cross-compilation and would force us to build the ARM image on an ARM machine. Without CGO, `GOOS=linux GOARCH=arm GOARM=7 go build` from any laptop produces a Pi-ready binary.
- **Performance:** WAL mode, ~10K writes/sec on a Pi 3. We will write at most ~50 query-log rows/sec under heavy use. Comfortable.
- **Schema:** Three core tables — `devices` (mac, ip, hostname, first_seen, last_seen, trusted), `query_log` (rolling 7-day window, auto-vacuumed nightly), `settings` (key/value). Migrations via `golang-migrate`, embedded with `go:embed`.
- **Why not Postgres:** Massive overkill, eats 200 MB RAM, requires a service.
- **Why not BoltDB:** No SQL means we re-implement filtering and aggregation. The admin UI needs `SELECT ... GROUP BY domain ORDER BY count DESC LIMIT 50` — give me SQL.

### 3.5 Local Network Discovery: mDNS via Avahi

**Decision:** `piholster.local` resolves via Avahi (already in Raspberry Pi OS). We do not bundle our own mDNS responder. We publish `_http._tcp` and `_https._tcp` services on boot.

**Why:** Avahi is reliable on the local LAN, works on macOS/iOS out of the box, works on modern Windows 10+, and Android via Chrome's "Cast" stack. For Android holdouts we expose `piholster.lan` as a fallback hostname via our own DNS server (since we *are* the DNS server, this is free).

### 3.6 No external reverse proxy

**Decision:** The Go binary listens on :80 (redirect to HTTPS), :443 (admin UI), and :53 (DNS). No nginx, no Caddy.

**Why:**
- One process to supervise. One log stream.
- Go's `net/http` plus `crypto/tls` is production-grade.
- TLS cert: self-signed root, generated on first boot, rotated annually. The setup wizard walks the user through trusting the cert on their phone (or accepting the warning, since this is purely LAN traffic). We do **not** use Let's Encrypt — it requires a public domain and exposes the device's existence externally.

---

## 4. Repository Structure (Monorepo)

```
piholster/
├── README.md
├── LICENSE                      (GPL-3.0 — see 4.1)
├── go.work                      (Go workspaces: backend + plugins)
├── turbo.json                   (Turborepo pipeline orchestration)
├── package.json                 (workspaces: web, docs)
├── docs/
│   ├── ADR-001-architecture.md  (this file)
│   └── adr/                     (future ADRs)
├── apps/
│   ├── piholsterd/              (Go: main daemon binary)
│   │   ├── cmd/piholsterd/main.go
│   │   ├── internal/
│   │   │   ├── api/             (HTTP + WebSocket handlers)
│   │   │   ├── auth/            (Argon2id, session tokens)
│   │   │   ├── dns/             (CoreDNS embedding + custom plugins)
│   │   │   ├── arp/             (gopacket ARP scanner)
│   │   │   ├── alerts/          (Telegram bot client)
│   │   │   ├── store/           (SQLite + migrations)
│   │   │   └── config/          (settings + environment)
│   │   ├── migrations/*.sql
│   │   └── go.mod
│   └── web/                     (SvelteKit frontend)
│       ├── src/routes/
│       │   ├── +layout.svelte
│       │   ├── +page.svelte                  (Grandma mode)
│       │   ├── advanced/+page.svelte
│       │   └── admin/+page.svelte
│       ├── src/lib/             (shared components, stores, API client)
│       ├── svelte.config.js     (adapter-static)
│       └── package.json
├── packages/
│   ├── blocklists/              (curated lists, versioned)
│   │   ├── ads.txt
│   │   ├── trackers.txt
│   │   ├── malware.txt
│   │   └── update.go            (CLI to refresh from upstream sources)
│   └── eslint-config/           (shared frontend lint rules)
├── image/
│   ├── pi-gen/                  (submodule, pinned to a commit)
│   ├── stage-piholster/         (our custom pi-gen stage)
│   │   ├── 00-install/
│   │   ├── 01-firstboot/
│   │   └── 02-harden/
│   └── build.sh
├── scripts/
│   ├── dev.sh                   (run backend + frontend on laptop)
│   └── release.sh               (tag, build, sign, publish image)
└── .github/workflows/
    ├── ci.yml                   (lint + test, every PR)
    ├── release-binary.yml       (cross-compile on tag)
    └── release-image.yml        (build .img on tag, self-hosted ARM64)
```

### 4.1 License: GPL-3.0

**Decision:** GPL-3.0. Reason: this is a privacy/security tool. Copyleft prevents a vendor from shipping a closed-source fork that quietly weakens DNS protections. Pi-hole uses EUPL-1.2; we go further. Frontend (`apps/web`) is also GPL-3.0 — no MIT carveout, because the frontend is the auth UI and forks must also be open.

### 4.2 Why monorepo

- The backend serves the frontend's static build directly via `go:embed`. They ship as one binary. They must release in lockstep. A polyrepo creates version-skew bugs we will not catch until a user reports them.
- Turborepo gives us caching across `pnpm`, `go test`, and `go build`.
- Go workspaces (`go.work`) let `apps/piholsterd` import from a future `apps/piholster-cli` without `replace` directives.

---

## 5. Environment Strategy

Three environments. No more.

### 5.1 dev (developer laptop)

- Backend: `go run ./apps/piholsterd` listens on :5300 (DNS) and :8080 (HTTP). Non-privileged ports so no sudo.
- Frontend: `pnpm --filter web dev`, Vite on :5173, proxies `/api` and `/ws` to :8080.
- DNS resolver: an in-memory upstream stub returning fixed answers, so devs are not hammering Quad9 in tests.
- ARP scanner: mocked. Real ARP requires a real LAN interface; we replay captured pcap files instead.
- Data: ephemeral SQLite at `./tmp/piholster.dev.db`, recreated on `make reset`.

### 5.2 test (CI + Pi-in-loop)

- Unit + integration tests run on `ubuntu-latest` with the pure-Go SQLite driver.
- Smoke tests run on a **physical Pi 3 in our office**, registered as a self-hosted GitHub runner, gated to `release/*` branches and tags. The Pi runs a fresh image per test, flashed via `rpi-imager` CLI in a pre-step.
- Scope of Pi-in-loop tests: boot in <90s, DNS query latency <20ms median, RAM <300 MB at idle, web UI loads.

### 5.3 prod (user's Pi)

- Single signed `.img.xz` distributed via GitHub Releases with SHA256 + minisign signature.
- The binary on the Pi runs as a dedicated `piholster` user (not root). Privileged ports :53, :80, :443 obtained via `setcap cap_net_bind_service,cap_net_raw=+ep /usr/local/bin/piholsterd` at install time.
- systemd-managed service with `Restart=on-failure`, `ProtectSystem=strict`, `NoNewPrivileges=true`, `PrivateTmp=true`.
- Auto-update: opt-in (default ON for security patches, default OFF for feature releases). Updates are signed binaries downloaded from `releases.piholster.net`, verified with the bundled minisign public key, atomically swapped, then `systemctl restart`.

---

## 6. CI/CD Strategy (GitHub Actions)

Three workflows. Each does one thing.

### 6.1 `ci.yml` — every PR and push to main

Steps, in order, fail fast:
1. `pnpm install --frozen-lockfile` and `go mod download`.
2. **Lint:** `golangci-lint run` (with `gosec`, `errcheck`, `revive`) and `pnpm --filter web lint`.
3. **Unit tests:** `go test -race -cover ./...` (target ≥70% coverage on `internal/dns`, `internal/auth`, `internal/store` — non-negotiable for security-critical code) and `pnpm --filter web test`.
4. **Integration tests:** spin up the daemon in a container, fire 1000 DNS queries through it, assert blocklist + DoT upstream behavior.
5. **Frontend build:** `pnpm --filter web build`, then assert the output is <500 KB total — if it grows past this we have regressed and the PR fails.
6. **Cross-compile check:** `GOOS=linux GOARCH=arm GOARM=7 go build` to catch architecture-specific breakage on every PR.

### 6.2 `release-binary.yml` — on git tag `v*`

- Cross-compile `piholsterd` for `linux/arm/v7`, `linux/arm64`, and `linux/amd64` (the last for our own Docker dev environment).
- Embed frontend build via `go:embed`.
- Sign each binary with minisign (key in GitHub Encrypted Secrets, never logged, never echoed).
- Upload to the GitHub Release.

### 6.3 `release-image.yml` — on git tag `v*`, after `release-binary.yml` succeeds

- Runs on a **self-hosted ARM64 runner** (a dedicated Pi 4 in the office, network-isolated, ephemeral build user). pi-gen does not run reliably under emulation for our needs.
- Builds the `.img` via `image/build.sh`, which:
  1. Pulls the latest signed `piholsterd` binary from step 6.2.
  2. Runs pi-gen with our custom stage.
  3. Compresses to `.img.xz`.
  4. Generates `.img.xz.sha256` and `.img.xz.minisig`.
- Uploads all three artifacts to the GitHub Release.
- Posts the release URL to a Discord/Matrix webhook for community visibility.

### 6.4 No CD to production

Releases are pulled by users, not pushed. Auto-update on the device is **client-initiated** with verification — we never get to push code to a user's network without their consent. This is a hard rule.

---

## 7. Image Build Strategy

### 7.1 Base: pi-gen, pinned

- We fork pi-gen at a known-good commit and pin via git submodule. We do not track upstream master — surprise upstream changes can ship a vulnerability. We rebase quarterly, deliberately.
- Base distribution: Raspberry Pi OS Lite (Bookworm, 64-bit on Pi 3+). Lite means no desktop, ~400 MB image. The user never sees it; this is the right tradeoff.

### 7.2 Custom pi-gen stage `stage-piholster`

Three sub-stages:

**`00-install`**
- Install dependencies: `avahi-daemon`, `iptables-persistent`, `unattended-upgrades` (limited to security pocket only).
- Drop the signed `piholsterd` binary into `/usr/local/bin/`.
- Drop the systemd unit, the `piholster` user (UID 999), and the data dir `/var/lib/piholster/`.
- Remove the default `pi` user. There is no user shell account on the prod image.

**`01-firstboot`**
- A oneshot systemd unit `piholster-firstboot.service` runs once on first boot:
  - Generates the self-signed TLS cert.
  - Generates the per-device random admin password (24-char, base32, no ambiguous chars).
  - Writes the password to `/var/lib/piholster/initial-admin-password.txt` (mode 0400, owned by `piholster`). The setup wizard reads it and forces a password change on first login. The file is deleted after first successful login.
  - Generates a SQLite-encrypted device identity key.
  - Disables itself.

**`02-harden`**
- Disable SSH by default. (Power users can enable via Admin UI; it requires the admin password.)
- Configure `iptables` to drop all inbound except :53/udp, :53/tcp, :80/tcp, :443/tcp, :5353/udp (mDNS), and ICMP.
- `sysctl` hardening: `net.ipv4.tcp_syncookies=1`, `kernel.kptr_restrict=2`, `kernel.dmesg_restrict=1`.
- `unattended-upgrades` on, security pocket only.

### 7.3 Reproducibility

- The pi-gen submodule is pinned. The base image SHA is recorded in `image/MANIFEST`. `apt` packages are pinned by version where security-relevant. We will not achieve bit-for-bit reproducibility (apt indexes change), but we achieve "two builds from the same tag produce functionally identical devices", which is what we need for incident response.

---

## 8. Security Principles (baseline — extended in ADR-002)

These are the non-negotiable defaults. Every future ADR must respect them or supersede them explicitly.

### 8.1 Admin password

- **Never** a hardcoded default. There is no "admin/admin" or "admin/piholster". A device shipped with a default password is a botnet recruit waiting to happen.
- **Per-device random initial password:** generated at first boot (see `01-firstboot`). 24 characters, 120 bits of entropy.
- **Hashing:** Argon2id with parameters `memory=64 MB, iterations=3, parallelism=2`. Tuned to take ~500 ms on a Pi 3, which prices brute-force attacks against an exfiltrated DB out of reach.
- **Storage:** hash + per-user random salt in the `users` table. The plaintext is never written to disk after the firstboot file is deleted.
- **Forced rotation:** the user MUST change the initial password on first login. The "remember me" cookie is not issued until they do.
- **Lockout:** 5 failed attempts -> 60s exponential lockout per source IP. Rate limit is enforced server-side; the frontend's UX hint is advisory only.

### 8.2 Sessions

- Server-side session tokens (32 random bytes, base64). Stored in SQLite, indexed, with `expires_at`. We do not use JWT — there is no need for stateless tokens on a single-node device, and JWT's footguns (algorithm confusion, no easy revocation) are not worth it.
- Cookie attributes: `HttpOnly`, `Secure`, `SameSite=Strict`, `Path=/admin` for admin sessions.
- Idle timeout: 30 min (Admin), 8 h (Advanced), no expiry (Grandma mode is unauthenticated read-only of basic stats).

### 8.3 Network exposure

- The admin UI binds **only** to LAN interfaces. We detect the default route's interface at boot and bind to that. We never bind to a public-facing interface. If the user wants remote access, ADR-002 will introduce a Tailscale/WireGuard integration — not a port forward.
- Outbound: only DNS upstreams (DoT/DoH), Telegram API (`api.telegram.org`), and update server (`releases.piholster.net`). Everything else is blocked by egress firewall. This is enforced by `iptables` OUTPUT rules, not just by application code.
- Telemetry: **off by default. No exceptions.** Opt-in only, anonymous, aggregate, and the user sees the exact JSON payload before it is sent the first time.

### 8.4 Update integrity

- All binaries and images are minisign-signed. The Pi verifies the signature with a public key embedded in the image at build time. A signature failure aborts the update and surfaces a UI warning.
- The signing key lives in a hardware token (YubiKey) held by two CTOs. Releases require two-of-two signature. (Yes, this is heavy for an MVP. It is also the right answer for a project that controls home DNS.)

### 8.5 Logging

- DNS query logs are private to the device, default 7-day retention, user-configurable down to "off" or up to 30 days.
- Logs are **never** sent off-device unless the user explicitly downloads them.
- The `query_log` table is encrypted at rest using SQLCipher-equivalent (we use SQLite + per-row AES-GCM with a key derived from the device identity key, since `modernc.org/sqlite` does not include SQLCipher; see ADR-003 for the full crypto design).

### 8.6 Threat model in scope for v1

- Hostile devices on the LAN trying to brute-force admin: mitigated by lockout + Argon2id.
- Network attacker passively observing DNS: mitigated by DoT/DoH upstream.
- Stolen SD card: mitigated by encrypted query logs (see 8.5). Settings and device list are NOT encrypted in v1 — we accept that an attacker with physical SD-card access learns your device list, on the rationale that they could see the same data by walking through your house.
- Malicious upstream DNS: out of scope. The user picks the upstream; we recommend Quad9 by default.
- Supply-chain attack on a Go module: mitigated by `go.sum`, Dependabot, and a vendored `go.mod` review on every PR.

---

## 9. Consequences

**Positive:**
- One Go binary, one SQLite file. The on-device footprint is tiny and the operational model is simple enough that a community contributor can understand the whole system in an afternoon.
- Cross-compilation without CGO means any contributor on any laptop can produce a Pi-ready binary in 10 seconds.
- The static frontend means we will never have a Node runtime on the Pi, which permanently caps an entire class of memory regressions.
- GPL-3.0 + signed releases + no telemetry creates a credibility moat that proprietary competitors cannot match.

**Negative / accepted tradeoffs:**
- SvelteKit is less common than React in the Linux/sysadmin contributor pool. We accept a slightly smaller frontend contributor base in exchange for the bundle-size win.
- Embedding CoreDNS means we inherit its update cadence; a CoreDNS CVE forces a PiHolster release. We accept this; the alternative (rolling our own DNS) is worse.
- Self-hosted ARM runner is operational overhead. We accept this for image-build correctness.
- Two-of-two release signing slows hotfixes. We accept this; key compromise is worse than a 4-hour delay.

---

## 10. Open Questions (deferred to later ADRs)

- **ADR-002:** Remote access. Tailscale built-in, WireGuard manual, or none? Leaning Tailscale.
- **ADR-003:** Encryption-at-rest design for query logs (key derivation, rotation).
- **ADR-004:** Telegram alert architecture vs alternatives (ntfy.sh, Signal). Telegram is convenient but is itself a privacy tradeoff.
- **ADR-005:** Multi-Pi clustering (high-availability DNS for households with two Pis). Not for v1.
- **ADR-006:** Internationalization. Grandma mode in Swedish first, English second, then community translations.

---

## 11. Approval

Approved by CTO, 2026-05-01. Supersedes nothing. All subsequent technical decisions reference this ADR by number.
