# Sprint 3 — Image Build, Firstboot, Systemd-hardening, go:embed, Smoke-tester, Dokumentation

**Sprintlängd:** 2 veckor
**Sprint-mål:** Leverera en flashbar Raspberry Pi OS-image som startar säkert ur lådan,
serverar Web-UI som en enda Go-binär, klarar automatiserade smoke-tester och medföljer
en installations-guide som en icke-teknisk användare kan följa.

Sprint 3 är den sista sprinten innan release-kandidat v0.1.
Alla US-13 till US-18 är MVP-blockers. Ingenting mergas till `release/v0.1-rc1` förrän
varje story har IT-säk sign-off eller explicit undantag noterat i denna fil.

---

## Beroendediagram

```
US-13 (pi-gen image-pipeline)
  |
  +---> US-14 (firstboot-skript)
  |       |
  |       +---> US-15 (systemd-hardening)
  |               |
  |               +---> US-17 (smoke-test suite)
  |                       |
  |                       +---> US-18 (installationsdokumentation)
  |
US-16 (go:embed frontend)
  |
  +---> US-17 (smoke-test suite — kräver en enda deploybar binär)
```

**Kritisk väg:** US-13 -> US-14 -> US-15 -> US-17 -> US-18

US-13 måste ha en körbar pipeline senast dag 4 i sprinten — annars kan US-14 och US-15
inte verifieras på fysisk Pi.

US-16 (go:embed) kan utvecklas parallellt med US-13/14/15 och behöver vara klar
senast dag 8 för att blockera US-17.

US-17 (smoke-tester) kan inte påbörjas förrän US-15 och US-16 är klara. Reservera
minimum 2 dagars sprint-buffer för Pi-in-loop-körningar och oväntade hårdvarubuggar.

---

## User stories

---

### US-13 — Pi-gen stage-piholster med tre sub-stages

**Som** release-pipeline
**vill jag** kunna köra `image/build.sh` och få en signerad `.img.xz`-fil
**så att** en slutanvändare kan bränna SD-kortet och få en fullt konfigurerad PiHolster
utan att röra ett terminal-fönster.

**Prioritet:** Hög — blockar US-14, US-15, US-17, US-18
**Estimat:** 3 dagar
**Ansvarig:** DevOps / Senior Go-dev
**IT-säk review:** Krävs på sub-stage `02-harden` (ADR-002 §3.2, §4)

**Acceptanskriterier:**

1. `image/pi-gen` är ett pinnat git-submodul på ett känt-gott commit; commit-hash
   dokumenteras i `image/MANIFEST` tillsammans med base-image-SHA (Raspberry Pi OS Lite,
   Bookworm 64-bit).

2. `image/stage-piholster/` innehåller exakt tre sub-stages:

   **`00-install`**
   - Installerar systemberoenden: `avahi-daemon`, `iptables-persistent`,
     `unattended-upgrades` (security pocket only).
   - Kopierar den cross-kompilerade och minisign-signerade `piholsterd`-binären
     (hämtad från `release-binary.yml`-artefakten, verifierad med bundlad publik nyckel)
     till `/usr/local/bin/piholsterd`.
   - Kopierar `piholster-arpd`-binären till `/usr/local/bin/piholster-arpd`.
   - Skapar systemanvändare `piholster` (UID 999) och `piholster-arp` (UID 998) utan
     interaktiv shell (`/usr/sbin/nologin`).
   - Skapar datakatalog `/var/lib/piholster/` med ägare `piholster:piholster`, mode 0750.
   - Sätter capabilities: `setcap cap_net_bind_service=+ep /usr/local/bin/piholsterd`
     och `setcap cap_net_raw=+ep /usr/local/bin/piholster-arpd`.
   - Tar bort standardanvändaren `pi`. Det finns inget shell-konto i prod-imagen.
   - CI-check efter detta steg: `getcap piholsterd | grep -v cap_net_raw` passerar
     (piholsterd saknar cap_net_raw).

   **`01-firstboot`**
   - Installerar `piholster-firstboot.service` (oneshot, RemainAfterExit=yes).
   - Installerar firstboot-skriptet `/var/lib/piholster/firstboot.sh` (mode 0500).
   - Firstboot-servicen är aktiverad och körs på första boot (se US-14 för detaljerat
     beteende).

   **`02-harden`**
   - SSH inaktiverat som standard (`systemctl disable ssh`).
   - `iptables-persistent` levererar initial regelfil
     `image/stage-piholster/02-harden/files/iptables-initial.rules` med
     `INPUT DROP` som default-policy — inget inkommande utom loopback och
     ESTABLISHED/RELATED (se ADR-002 §3.2.1).
   - `sysctl`-hardening via `/etc/sysctl.d/99-piholster.conf`:
     `net.ipv4.tcp_syncookies=1`, `kernel.kptr_restrict=2`,
     `kernel.dmesg_restrict=1`.
   - `unattended-upgrades` konfigurerat för security-pocket enbart.
   - Avahi statiska service-filer är INTE installerade (publish sker programmatiskt
     efter firstboot per ADR-002 §3.2.1 Lager 3).
   - `avahi-daemon.conf` patch: `publish-hinfo=no`, `publish-workstation=no`,
     `publish-aaaa-on-ipv4=no`.

3. `image/build.sh` hämtar signerade binärer från GitHub Release, verifierar
   minisign-signatur, och avbryter med exitkod 1 om verifieringen misslyckas.

4. Den färdiga imagen är < 2 GB okomprimerat. Komprimerad `.img.xz` är < 600 MB.

5. `image/MANIFEST` uppdateras automatiskt av `build.sh` med: pi-gen commit,
   base-image SHA256, piholsterd version, piholster-arpd version, build timestamp.

6. GitHub Actions workflow `release-image.yml` kör `build.sh` på den self-hosted
   ARM64-runnern (Pi 4) och laddar upp `.img.xz`, `.img.xz.sha256` och
   `.img.xz.minisig` till GitHub Release.

7. IT-säkerhet granskar `02-harden`-steget och `iptables-initial.rules` innan merge.

---

### US-14 — Firstboot-skript: lösenord, TLS-cert och device identity key

**Som** ny PiHolster-enhet som startar för första gången
**vill jag** automatiskt generera ett slumpmässigt admin-lösenord, ett självsignerat
TLS-certifikat och en unik device identity key
**så att** ingen enhet har samma hemligheter och ingen konfiguration behöver göras manuellt.

**Prioritet:** Hög — blockar US-15, US-17
**Estimat:** 2,5 dagar
**Ansvarig:** Senior Go-dev
**IT-säk review:** Krävs (generering av kryptografiskt material, ADR-002 §3.2)

**Acceptanskriterier:**

1. Firstboot-skriptet körs som `piholster-firstboot.service` (Type=oneshot,
   RemainAfterExit=yes) med ordering:
   ```
   After=network-online.target time-sync.target
   Before=piholster-arpd.service piholsterd.service piholster-avahi-publish.service
   ```
   Om firstboot-tjänsten avslutas med exitkod != 0 startar varken `piholster-arpd`
   eller `piholsterd` (täcker ADR-002 §3.2.1 Lager 2).

2. **Admin-lösenord:**
   - Genereras med `openssl rand` eller Go-ekvivalent med `/dev/urandom`-källa.
   - 24 tecken, base32-alfabet utan tvetydiga tecken (I, O, 1, 0 exkluderade),
     vilket ger >= 120 bits entropi.
   - Skrivs till `/run/piholster/initial-password` (tmpfs, mode 0600, ägare
     `piholster`). Filen finns bara i RAM och försvinner vid omstart.
   - Lösenordet hashas med Argon2id (parametrar: memory=64MB, iterations=3,
     parallelism=4) och skrivs in i `users`-tabellen i SQLite som admin-kontot.
   - Klartext-lösenordet skrivs ALDRIG till disk utanför tmpfs-sökvägen.
   - Filen raderas automatiskt av `piholsterd` vid första lyckade inloggning.
   - Lösenordet visas som en QR-kod via ACT-LED (framtid) — i Sprint 3 räcker det
     att lösenordet finns läsbart i `/run/piholster/initial-password` för
     installationsguiden att hänvisa till.

3. **Självsignerat TLS-certifikat:**
   - Genereras med Go:s `crypto/x509`-paket (eller openssl som fallback om
     piholsterd inte är igång under firstboot).
   - RSA 4096-bitars nyckel eller P-256 ECDSA (P-256 föredras: mindre cert,
     snabbare handshake på Pi 3).
   - Giltighetstid: 3650 dagar (10 år; användaren ska inte behöva förnya manuellt).
   - Subject: `CN=piholster.local, O=PiHolster`.
   - SubjectAltName: `piholster.local`, `piholster.lan`, lokal IP (detekteras
     via default-route-interface).
   - Cert lagras: `/var/lib/piholster/tls/server.crt` (mode 0644).
   - Nyckel lagras: `/var/lib/piholster/tls/server.key` (mode 0640, ägare
     `piholster:piholster`).
   - Om cert-filerna redan existerar vid firstboot-körning: hoppa över generering
     (idempotens).

4. **Device identity key:**
   - 32 bytes, kryptografiskt slumpmässiga, genererade med `crypto/rand`.
   - Lagras i `/var/lib/piholster/device-id.key` (mode 0400, ägare `piholster`).
   - Används som rot för framtida per-rad AES-GCM-kryptering av query_log
     (ADR-001 §8.5, ADR-003 scope).
   - Används som suffix i HTTP User-Agent för update-polling:
     `PiHolster/0.1.0 (id:<sha256(device-id.key)[:8]>)` (hash, inte råvärde).

5. **LED-feedback (ADR-002 §3.2.1 Lager 4):**
   - Firstboot start: snabbblinkande ACT-LED (100 ms on / 100 ms off via
     `trigger=timer`).
   - Firstboot klar: LED fast on (`trigger=default-on`).
   - Felet om firstboot misslyckas: 3 långa blinkar, 3 korta blinkar
     (Morse-liknande SOS-mönster via bash-loop på `/sys/class/leds/led0/`).

6. **Klockvänta:**
   - Firstboot-tjänsten har `After=time-sync.target`.
   - Om `time-sync.target` inte nåtts inom 120 sekunder efter boot: logga varning
     men fortsätt ändå (Pi 3 utan RTC kan ha en långsam första NTP-sync på
     tunna nätverk).

7. Firstboot-skriptet är idempotent: att köra det två gånger på en redan
   initialiserad enhet gör ingenting och avslutar med exitkod 0.

8. Enhetstester (Go) täcker: lösenordsentropi-kontroll (minst 120 bits),
   cert-generering producerar giltig x509, device-id är 32 bytes och unik
   per körning.

9. IT-säkerhet granskar PR innan merge.

---

### US-15 — Systemd-hardening: iptables boot-sekvens, service-kedja, klockväntan

**Som** PiHolster-system
**vill jag** att iptables default-DROPpar allt inkommande trafik tills firstboot är klar,
att systemd-tjänsterna startar i korrekt ordning, och att `piholsterd` aldrig startar
utan att klockan är synkroniserad
**så att** firstboot-fönstret är stängt (ADR-002 Fynd 2) och tjänsterna inte startar i
ett osäkert tillstånd.

**Prioritet:** Hög — MVP-blocker per ADR-002
**Estimat:** 2 dagar
**Ansvarig:** Senior Go-dev + DevOps
**IT-säk review:** Krävs (firewall-regler och service-kedja)

**Acceptanskriterier:**

1. **iptables default-DROP under boot:**
   - `iptables-persistent` laddar `iptables-initial.rules` (installerat av US-13
     `02-harden`) vid varje boot, INNAN firstboot-tjänsten körs.
   - Regelfilen implementerar: INPUT DROP (default), FORWARD DROP (default),
     OUTPUT ACCEPT (default).
   - Tilåtna inkommande undantag i initial-filen: `-i lo` (loopback),
     `-m conntrack --ctstate ESTABLISHED,RELATED`, ICMP echo-request
     (rate-limitad till 5/sec).
   - Inga portar (53, 80, 443) är öppna förrän firstboot-skriptet explicit
     lägger till dem och kör `iptables-save > /etc/iptables/rules.v4`.

2. **Firstboot öppnar portar som sista steg:**
   Sista sektion i `firstboot.sh`:
   ```bash
   iptables -A INPUT -p udp --dport 53   -j ACCEPT
   iptables -A INPUT -p tcp --dport 53   -j ACCEPT
   iptables -A INPUT -p tcp --dport 80   -j ACCEPT
   iptables -A INPUT -p tcp --dport 443  -j ACCEPT
   iptables -A INPUT -p udp --dport 5353 -j ACCEPT
   iptables-save > /etc/iptables/rules.v4
   ```

3. **Systemd service-kedja (Requires/After):**

   Exakt dessa relationer måste finnas och verifieras med `systemctl show`:

   | Tjänst | After | Requires | Before |
   |---|---|---|---|
   | `piholster-firstboot.service` | `network-online.target time-sync.target` | — | `piholster-arpd.service piholsterd.service piholster-avahi-publish.service` |
   | `piholster-arpd.service` | `network-online.target piholster-firstboot.service` | `piholster-firstboot.service` | `piholsterd.service` |
   | `piholsterd.service` | `network-online.target time-sync.target piholster-firstboot.service piholster-arpd.service` | `piholster-firstboot.service piholster-arpd.service` | — |
   | `piholster-avahi-publish.service` | `piholsterd.service` | `piholsterd.service` | — |

4. **Type=notify för `piholster-arpd` och `piholsterd`:**
   - Båda tjänsterna skickar `sd_notify(READY=1)` när de är redo att ta emot anslutningar.
   - `piholsterd` ansluter till arpd:s Unix socket och väntar på heartbeat innan
     den skickar READY=1 (se US-09 spec).
   - Om `piholsterd` inte har fått heartbeat från arpd inom 30 sekunder loggas
     ett kritiskt fel och `piholsterd` avbryter sin startup.

5. **Klockväntan:**
   - `piholsterd.service` har `After=time-sync.target`.
   - TLS-certifikat-validering i `piholsterd` kontrollerar att systemklockan
     är > 2024-01-01 innan HTTPS-servern startar. Om klockan är uppenbart felaktig
     (t.ex. Unix epoch) loggas kritiskt fel och startup avbryts.

6. **Verifieringstest (Pi-in-loop, körs av US-17):**
   - Flash färsk image, boot, anslut omedelbart:
     `nmap -p 53,80,443 $PI_IP` returnerar `filtered` på alla portar.
   - Vänta 90 sekunder:
     `nmap -p 53,80,443 $PI_IP` returnerar `open` på alla portar.
   - Kontrollera att `piholsterd` inte startade före `piholster-firstboot`:
     `journalctl -u piholsterd --since boot` visar att startordningen är korrekt.

7. **Systemd-unit-filer för alla fyra tjänster** finns i
   `image/stage-piholster/01-firstboot/files/` och `00-install/files/` och
   installeras av pi-gen-steget.

8. IT-säkerhet granskar iptables-regler och service-beroenden innan merge.

---

### US-16 — go:embed: SvelteKit-bygget inbäddat i Go-binären

**Som** deployad piholsterd-binär
**vill jag** att hela Web-UI:n är inbäddad i binären
**så att** det räcker med en enda fil att distribuera — ingen separat `static/`-katalog,
inga sökvägar att konfigurera.

**Prioritet:** Hög — MVP-blocker (ADR-001 §3.2 och §6.2 kräver `go:embed` för release)
**Estimat:** 1,5 dagar
**Ansvarig:** Senior Go-dev (fullstack)
**IT-säk review:** Inte krävd för embedding i sig, men ADR-002 §3.4.1 nonce-injection
måste vara implementerad och granskas av CTO.

**Acceptanskriterier:**

1. SvelteKit byggs till en statisk output i `apps/web/.svelte-kit/output/` via
   `pnpm --filter web build` med `@sveltejs/adapter-static`.

2. Go-koden bäddar in SvelteKit-outputen via `go:embed`:
   ```go
   //go:embed all:web/dist
   var webFS embed.FS
   ```
   `web/dist` pekar på en kopia av SvelteKit-outputen som kopieras dit under build
   (Makefile/Turbo-task `pnpm build && cp -r apps/web/.svelte-kit/output/client apps/piholsterd/web/dist`).

3. HTTP-servern serverar de inbäddade filerna med korrekt Content-Type och
   cache-headers. `index.html` serveras för alla okända sökvägar
   (SPA-fallback / history-mode routing).

4. **CSP nonce-injection (ADR-002 §3.4.1):**
   - Go-servern läser `web/dist/index.html` ur `embed.FS`.
   - Per request genereras ett 16-byte nonce (base64) och skrivs in i `index.html`
     genom att ersätta platshållaren `%%CSP_NONCE%%` med rätt värde.
   - SvelteKit-bygget är konfigurerat med `csp: { mode: 'nonce' }` i `svelte.config.js`.
   - `SecurityHeaders`-middlewaret sätter CSP-headern med samma nonce-värde.
   - CI-check: `grep -r "unsafe-inline" apps/web/.svelte-kit/output/` returnerar
     ingenting (inga inline-styles utan nonce).
   - Teknisk skuld från Sprint 2 (BB-03) är därmed löst.

5. `piholsterd`-binären startar och serverar Web-UI utan att det finns någon
   `static/`-katalog på disk. Verifieras med: `rm -rf /tmp/testdir && cp piholsterd
   /tmp/testdir/ && cd /tmp/testdir && ./piholsterd --dev` och bekräfta att
   `localhost:8080` svarar med index.html.

6. Frontendbygg-storlekscheck i CI kvarstår: total output < 500 KB JS.

7. CI-check för externa URL:er kvarstår (ADR-002 §3.4.2):
   ```bash
   if grep -RE 'https?://' apps/web/.svelte-kit/output/client/_app/ \
     | grep -v 'piholster.local\|piholster.lan'; then
     echo "External URL detected"; exit 1
   fi
   ```

8. `release-binary.yml` kör `pnpm --filter web build` som ett setup-steg INNAN
   `go build`, så att `web/dist`-innehållet är aktuellt i den signerade binären.

---

### US-17 — Smoke-test suite: boot, DNS, RAM, Web UI

**Som** release-pipeline
**vill jag** att en automatiserad svit av smoke-tester körs mot en fysisk Pi 3 med
färsk image
**så att** vi med säkerhet vet att v0.1-release-kandidaten uppfyller de kvantitativa
MVP-kraven och inte regrederar dem framöver.

**Prioritet:** Hög — MVP gate
**Estimat:** 2 dagar
**Ansvarig:** DevOps + Senior Go-dev
**IT-säk review:** Inte krävd (testinfrastruktur, ej produktion)

**Acceptanskriterier:**

1. Smoke-testerna körs som ett dedikerat CI-jobb `smoke-test` i
   `.github/workflows/release-image.yml`, gated till `release/*`-brancher och
   taggade commits.

2. Miljö: en fysisk Raspberry Pi 3 (1 GB RAM, ARMv7) är registrerad som
   self-hosted GitHub-runner. Innan varje smoke-test-körning flashas en färsk image
   via `rpi-imager` CLI i ett pre-step.

3. **Test 1 — Boot-tid:**
   - Mät tid från att strömmen sätts på tills `GET https://piholster.local/`
     svarar med HTTP 200.
   - Krav: < 90 sekunder.
   - Mätmetod: CI-script poller med `curl --retry 60 --retry-delay 2` och
     registrerar timestamp för första lyckade svar.

4. **Test 2 — Firstboot-fönster (iptables):**
   - Omedelbart efter flash och start (inom 5 sekunder):
     `nmap -p 53,80,443 $PI_IP` — alla portar ska vara `filtered`.
   - Efter 90 sekunder (firstboot borde vara klar):
     `nmap -p 53,80,443 $PI_IP` — alla portar ska vara `open`.

5. **Test 3 — DNS-latens:**
   - Skicka 100 DNS-frågor (mix av blockerade och ej blockerade domäner)
     via `dig @$PI_IP` med 1 sekunders intervall.
   - Beräkna median-latens.
   - Krav: median < 20 ms.
   - Mätmetod: `dig +stats`-output parsas av CI-script.

6. **Test 4 — RAM vid idle:**
   - Vänta 5 minuter efter boot (systemet ska ha stabiliserat sig).
   - Hämta minnesanvändning via `ssh piholster@$PI_IP free -m` (SSH måste vara
     aktiverat på testenheten — separat test-image-variant med SSH aktiverat,
     INTE produktionsimagen).
   - Alternativt: exponera `/api/status` med ett `ram_used_mb`-fält.
   - Krav: < 300 MB använt RAM vid idle.

7. **Test 5 — Web UI svarar:**
   - `GET https://piholster.local/` returnerar HTTP 200.
   - Response-body innehåller `<html`.
   - Inga externa resurser laddas (kontrolleras med Playwright i headless-läge
     om tillgängligt; annars enbart HTTP 200-kontroll i MVP).

8. **Test 6 — API-hälsa:**
   - `GET https://piholster.local/api/status` returnerar HTTP 200 och ett
     JSON-objekt med `dns_ok: true`.

9. **Test 7 — Capability-kontroll:**
   - `ssh piholster-runner@$PI_IP getcap /usr/local/bin/piholsterd`
     returnerar ENBART `cap_net_bind_service` — `cap_net_raw` ska INTE finnas.
   - `ssh piholster-runner@$PI_IP getcap /usr/local/bin/piholster-arpd`
     returnerar `cap_net_raw`.

10. Alla tester är implementerade i `tests/smoke/` som ett Go-testpaket eller
    ett bash-skript beroende på vad som är enklast att underhålla. Testvärden
    (PI_IP, trösklar) är konfigurerbara via miljövariabler.

11. Om ett enskilt test misslyckas markeras hela smoke-test-jobbet som Failed
    och release-imagen laddas INTE upp till GitHub Release. Felmeddelandet
    inkluderar vilket test och vilket faktiskt uppmätt värde som orsakade felet.

---

### US-18 — Installationsdokumentation: "bränn SD-kort och sätt in"

**Som** slutanvändare utan teknisk erfarenhet
**vill jag** ha en steg-för-steg-guide som tar mig från "jag köpte en Raspberry Pi"
till "PiHolster skyddar mitt hemnätverk"
**så att** jag kan installera PiHolster på under 10 minuter utan att behöva öppna
ett terminalfönster.

**Prioritet:** Medium — krävs för v0.1 release men blockar inte teknisk implementation
**Estimat:** 1,5 dagar
**Ansvarig:** Senior Go-dev (primär) + PM (review)
**IT-säk review:** Krävs på säkerhetsavsnittet (lösenordshantering, certifikatvarning)

**Acceptanskriterier:**

1. Dokumentet lever i `docs/INSTALL.md` och länkas från `README.md`.

2. **Vad du behöver** — lista utan jargong:
   - Raspberry Pi 3, 3B, 3B+, 4 eller 5 (modell A räcker inte — kräver Ethernet)
   - MicroSD-kort, minst 8 GB
   - Ethernetkabel (WiFi stöds inte i MVP)
   - En dator för att bränna SD-kortet (Windows, macOS eller Linux)

3. **Steg-för-steg-guide (max 6 steg):**

   **Steg 1: Ladda ner imagen**
   - Länk till senaste `.img.xz` på GitHub Releases.
   - Instruktion att verifiera SHA256 (kommandoradsalternativ) eller
     "om du inte vet vad det här betyder, hoppa över det — det är valfritt".
   - Storlek på filen (uppskattning) och förväntad nedladdningstid.

   **Steg 2: Bränn SD-kortet**
   - Primär metod: Raspberry Pi Imager (gratis, grafiskt, Windows/Mac/Linux).
   - Skärmdump eller bildtext som visar de tre klicken: välj imagen, välj SD-kortet,
     klicka Skriv.
   - Alternativ för avancerade användare: `xzcat piholster.img.xz | dd of=/dev/sdX bs=4M`.

   **Steg 3: Sätt i och starta**
   - Sätt i SD-kortet i Pi:n.
   - Koppla Ethernetkabeln till routern.
   - Koppla in strömmen.
   - Vänta tills den gröna lampan lyser stadigt (ca 60–90 sekunder).

   **Steg 4: Hitta PiHolster i webbläsaren**
   - Öppna `https://piholster.local` på en enhet i samma nätverk.
   - Förväntat certifikatvarning — förklaring i vanlig svenska varför det dyker upp
     och att det är säkert att klicka "Avancerat" -> "Fortsätt".
   - Om `piholster.local` inte fungerar: testa `https://piholster.lan` eller
     hitta Pi:ns IP-adress i routerns enhetslist.

   **Steg 5: Logga in och byt lösenord**
   - Det initiala lösenordet finns på etiketten under Pi:n (packaging insert —
     utanför Sprint 3 scope, men INSTALL.md dokumenterar att det finns i
     `/run/piholster/initial-password` om etiketten saknas, åtkomst via Admin UI).
   - Instruktion att byta lösenord direkt.
   - Förklaring att lösenordet bara visas en gång.

   **Steg 6: Konfigurera din router**
   - Ändra din routers DNS-server till Pi:ns IP-adress.
   - Skärmdumpar eller generell instruktion för de vanligaste routermärkena
     (Asus, TP-Link, Netgear).
   - Alternativ: DHCP-override på enhetsnivå för att testa utan att ändra routern.

4. **Felsökning** — sektion med de 5 vanligaste problemen:
   - Gröna lampan slutar blinka men lyser aldrig stadigt
   - `piholster.local` hittas inte
   - Certifikatvarningen ger ett annat felmeddelande än förväntat
   - Jag råkade byta lösenordet till något jag inte minns
   - DNS fungerar men ett site som var blockerat fungerar fortfarande

5. **Säkerhetsavsnittet:**
   - Varför PiHolster kräver ett eget lösenord (aldrig "admin/admin").
   - Vad det självsignerade certifikatet är och varför det inte är en säkerhetsrisk.
   - Vad data som lagras (DNS-loggar stannar på Pi:n, inget skickas till molnet).
   - IT-säk granskar detta avsnitt.

6. Dokumentet är skrivet på svenska och testas av en person utan teknisk bakgrund
   (t.ex. en familjemedlem till en av teammedlemmarna). Feedbacken dokumenteras och
   om den leder till ändringar: de görs innan merge.

---

## Sprint-backlog sammanfattning

| ID    | Story                                         | Prioritet | Estimat   | Ansvarig              | IT-säk |
|-------|-----------------------------------------------|-----------|-----------|-----------------------|--------|
| US-13 | Pi-gen image-pipeline (3 sub-stages)          | Hög       | 3 dagar   | DevOps + Go-dev       | Ja     |
| US-14 | Firstboot: lösenord, TLS-cert, identity key   | Hög       | 2,5 dagar | Senior Go-dev         | Ja     |
| US-15 | Systemd-hardening: iptables, kedja, klocka    | Hög       | 2 dagar   | Go-dev + DevOps       | Ja     |
| US-16 | go:embed frontend i Go-binären                | Hög       | 1,5 dagar | Senior Go-dev         | Nej*   |
| US-17 | Smoke-test suite (Pi-in-loop)                 | Hög       | 2 dagar   | DevOps + Go-dev       | Nej    |
| US-18 | Installationsdokumentation                    | Medium    | 1,5 dagar | Go-dev + PM           | Ja**   |

*US-16 CTO-review krävs på nonce-injection-implementationen.
**IT-säk granskar enbart säkerhetsavsnittet i INSTALL.md.

**Total estimering:** 12,5 dagar (~2 veckors sprint).
Buffert: 1,5 dag reserveras för oväntade Pi-in-loop-buggar och IT-säk-kö.
Bokas in: IT-säk review dag 6–8 i sprinten (US-13, US-14, US-15 ska ha PRs redo dag 6).

---

## Definition of Done (Sprint 3)

En story är klar när ALLA punkter nedan är uppfyllda:

1. Koden är mergad till `develop` via godkänd PR (minst 1 peer review).

2. IT-säkerhet har granskat och sign-off:at PR för US-13 (`02-harden`),
   US-14 (kryptomaterial), US-15 (firewall + service-kedja) och
   säkerhetsavsnittet i US-18.

3. CI-pipeline är grön: lint, enhetstester, cross-kompilering, frontendstorlek,
   extern-URL-check, capability-check (`cap_net_raw` saknas i piholsterd).

4. Smoke-testerna i US-17 passerar mot en fysisk Pi 3 med färsk image
   (boot < 90s, DNS < 20ms median, RAM < 300MB idle, Web UI svarar).

5. Den färdiga `.img.xz` är verifierbar: SHA256 stämmer, minisign-signatur
   valideras med den bundlade publika nyckeln.

6. `image/MANIFEST` är uppdaterad med pi-gen commit, base-image SHA, binärversioner
   och build-timestamp.

7. Inga `[BLOCK]`-kommentarer i öppna PR-reviews.

8. Inga hemligheter (device-id, admin-lösenord, bot-tokens) är commitade i klartext.

9. SPRINT-3.md (denna fil) är uppdaterad om scope har förändrats under sprinten
   (PM-ansvar).

10. **Release-kandidat-gate:** CTO och IT-säkerhet co-sign att imagen är redo för
    taggning som `v0.1.0-rc1`. Sign-off dokumenteras som en kommentar i
    release-PR:en på GitHub.

---

## Blockers att bevaka

**BB-06 — Self-hosted ARM64-runner tillgänglighet**
Pi-gen (US-13) och smoke-tester (US-17) kräver den self-hosted Pi 4-runnern.
Om den är offline eller i konflikt med parallella jobb blockeras hela sprint-slutet.
DevOps verifierar runner-status dag 1 och skapar en backup-plan (t.ex. QEMU-emulering
som fallback för image-build, ej för smoke-tester).
Eskalering: CTO beslutar om fallback-strategi senast dag 2.

**BB-07 — CSP nonce + SvelteKit adapter-static (teknisk skuld från BB-03)**
Sprint 2 accepterade `'unsafe-inline'` på `style-src` som temporär lösning.
ADR-002 §3.4 kräver nonce-baserade styles. US-16 löser detta, men det kräver att
SvelteKit-bygget faktiskt producerar nonce-kompatibel output.
Om `adapter-static` motstår nonce-integration: CTO beslutar senast dag 3 om vi
accepterar `'unsafe-inline'` till v1.0 (formellt undantag dokumenteras i ADR-002)
eller om vi byter adapter.
Risk: hög. Reservera 0,5 dag extra för US-16 om nonce-integreringen är krånglig.

**BB-08 — IT-säkerhets tillgänglighet**
US-13, US-14 och US-15 kräver alla IT-säk-review. Tre separata PRs kan skapa kö.
PM bokar ett blockat review-fönster dag 6–7. Om IT-säk är otillgänglig dag 6:
eskalera till CTO som avgör om sprint-deadline kan hållas eller om release-kandidat
skjuts en vecka.

**BB-09 — Raspberry Pi 3-lagret för Pi-in-loop**
Smoke-testerna (US-17) kräver fysisk Pi 3 (1 GB RAM, ARMv7). Om vår test-Pi
havererar under sprinten har vi ingen omedelbar ersättning.
Mitigation: säkerställ att en reserv-Pi 3 finns på kontoret och är förkonfigurerad
med CI-runner-credentials.

**BB-10 — iptables-persistent ARM-kompatibilitet**
`iptables-persistent` och `iptables-nft` vs `iptables-legacy` kan ge oväntade
beteenden på Raspberry Pi OS Bookworm (som föredrar nftables backend).
DevOps testar dag 1 att `iptables-initial.rules` faktiskt laddas vid boot och
appliceras korrekt. Om inte: antingen `iptables-legacy-save`-variant eller
migrering av regler till `nft`-syntax.
Eskalering: DevOps rapporterar status dag 2. CTO beslutar om omskrivning krävs.

**BB-11 — Installationsguide behöver icke-teknisk testare (US-18)**
Definitionen av Done för US-18 kräver att en person utan teknisk bakgrund
testkör guiden. PM koordinerar detta med en familjemedlem eller vän senast dag 12.
Om ingen testare är tillgänglig innan merge: noteras som ett öppet acceptance-kriterium
med datum för uppföljning.
```
