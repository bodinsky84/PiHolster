# Sprint 1 — Projektstruktur, DNS-kärna, Web-UI-skelett

**Sprintlängd:** 2 veckor  
**Sprint-mål:** En körbar prototyp på Raspberry Pi 3 där DNS-blockering fungerar lokalt
och en tom men navigerbar web-UI kan nås från LAN.

---

## User stories

### US-01 — Repo och projektstruktur

**Som** ny bidragsgivare  
**vill jag** kunna klona repot och förstå strukturen på under fem minuter  
**så att** jag snabbt kan börja bidra utan att behöva fråga teamet.

**Prioritet:** Hög (blockar allt annat)  
**Estimat:** 0,5 dag  
**Ansvarig:** Senior utvecklare

**Acceptanskriterier:**
- Repot innehåller mapparna `dns/`, `network/`, `ui/`, `notifications/`, `image/`, `docs/`, `tests/`
- `README.md` beskriver vad projektet är, hur man sätter upp dev-miljö och hur man kör tester
- `.gitignore` täcker Python-artefakter, node_modules, `.env`-filer och systemloggar
- En `Makefile` eller `justfile` med minst: `make dev`, `make test`, `make lint`
- `CONTRIBUTING.md` finns och refererar till `CHANGE-PROCESS.md`

---

### US-02 — DNS-server med grundläggande blocklist

**Som** hemmaanvändare  
**vill jag** att min Raspberry Pi fungerar som DNS-server och blockerar annonser och trackers  
**så att** min surfning är renare och snabbare utan att jag behöver konfigurera något.

**Prioritet:** Hög  
**Estimat:** 3 dagar  
**Ansvarig:** Senior utvecklare

**Acceptanskriterier:**
- dnsmasq eller Unbound är installerat och startar automatiskt vid boot
- En initialblocklist (minst StevenBlack hosts-format) är hämtad och laddad vid start
- DNS-uppslag mot blocklistan returnerar NXDOMAIN eller 0.0.0.0 för träffar
- Legitima domäner (google.com, github.com) löses korrekt
- Blocklist uppdateras vid start om nätverksanslutning finns
- En lokal enhet (laptop/mobil) kan peka sin DNS mot Pi:n och få blockering
- Loggar skrivs till `/var/log/piholster/dns.log` med roterande loggar (max 50 MB)

---

### US-03 — DNS-over-HTTPS som standard

**Som** hemmaanvändare  
**vill jag** att min DNS-trafik är krypterad som standard  
**så att** min internetleverantör inte kan se vilka domäner jag slår upp.

**Prioritet:** Hög  
**Estimat:** 1 dag  
**Ansvarig:** Senior utvecklare

**Acceptanskriterier:**
- Upstream DNS-uppslag går via DoH (Cloudflare 1.1.1.1 eller Quad9 som standard)
- Konfigurationsfil anger upstream-URL så att Admin kan byta provider
- Fallback till DoT om DoH-endpoint är nådd av timeout (max 3 s)
- `dig` mot lokal Pi visar korrekt svar och upstream-träffen syns i log som `DoH`
- Inga okrypterade uppslag görs mot port 53 externt vid normal drift

---

### US-04 — Web-UI-skelett: grundnavigering

**Som** användare  
**vill jag** kunna öppna en webbsida på `http://piholster.local` och se tre lägen att välja mellan  
**så att** jag hamnar i rätt vy för min kunskapsnivå.

**Prioritet:** Hög  
**Estimat:** 2 dagar  
**Ansvarig:** Senior utvecklare

**Acceptanskriterier:**
- En webbserver (FastAPI eller Flask) startar automatiskt vid boot och lyssnar på port 80
- `http://piholster.local` svarar (mDNS via avahi)
- Startsidan visar tre alternativ: "Mormor", "Avancerat", "Admin"
- Mormor-vyn visar minst: ett trafikljus (grön/gul/röd status), antal blockerade annonser idag och antal enheter online
- Avancerat-vyn innehåller platshållare med text "Kommer i Sprint 2"
- Admin-vyn visar ett lösenordsformulär (autentisering implementeras Sprint 2, just nu räcker 200 OK)
- Sidan är responsiv och fungerar i mobilwebbläsare (375 px bredd)

---

### US-05 — Grundläggande CI-pipeline

**Som** code reviewer  
**vill jag** att varje PR automatiskt körs genom linting och enhetstester  
**så att** jag kan lita på att inget uppenbart fel mergas in.

**Prioritet:** Medium  
**Estimat:** 1 dag  
**Ansvarig:** DevOps

**Acceptanskriterier:**
- GitHub Actions-workflow körs på varje PR mot `main` och `develop`
- Workflow kör: lint (ruff eller flake8 för Python, eslint för JS), enhetstester, bygger Docker-image utan fel
- Röd pipeline blockerar merge (branch protection rule aktiverad på `main`)
- Workflow-tid under 5 minuter för en ren build
- Testrapport publiceras som PR-kommentar (GitHub Actions summary räcker)

---

### US-06 — Lokal Docker-baserad dev-miljö

**Som** utvecklare  
**vill jag** kunna köra hela stacken lokalt med ett kommando  
**så att** jag inte behöver en fysisk Raspberry Pi för att testa ändringar i UI eller DNS-logik.

**Prioritet:** Medium  
**Estimat:** 1 dag  
**Ansvarig:** DevOps

**Acceptanskriterier:**
- `docker compose up` startar DNS-tjänsten och web-UI
- DNS-container exponerar port 5353 lokalt (undviker konflikt med systemets DNS)
- Web-UI är nåbar på `http://localhost:8080`
- En `docker-compose.override.yml.example` finns för lokal konfiguration
- README beskriver hur man kör och stoppar dev-miljön

---

## Sprint-backlog sammanfattning

| ID    | Story                              | Prioritet | Estimat  | Ansvarig          |
|-------|------------------------------------|-----------|----------|-------------------|
| US-01 | Repo och projektstruktur           | Hög       | 0,5 dag  | Senior dev        |
| US-02 | DNS-server med blocklist           | Hög       | 3 dagar  | Senior dev        |
| US-03 | DNS-over-HTTPS som standard        | Hög       | 1 dag    | Senior dev        |
| US-04 | Web-UI-skelett: grundnavigering    | Hög       | 2 dagar  | Senior dev        |
| US-05 | Grundläggande CI-pipeline          | Medium    | 1 dag    | DevOps            |
| US-06 | Lokal Docker-baserad dev-miljö     | Medium    | 1 dag    | DevOps            |

**Total estimering:** 8,5 dagar (~2 veckors sprint med marginal för review och buggar)

---

## Definition of Done (per story)

1. Koden är mergad till `develop` via godkänd PR (minst 1 review)
2. CI-pipeline passerar grönt
3. Acceptanskriterierna är manuellt verifierade av en annan teammedlem
4. Inga öppna säkerhetsanmärkningar från IT-säkerhets-review
5. `ROADMAP.md` eller `SPRINT-1.md` uppdateras om scope förändrats

---

## Blockers att bevaka

- Val av DNS-backend (dnsmasq vs Unbound) — CTO beslutar senast dag 1
- mDNS-stöd (`piholster.local`) kräver avahi-daemon — verifiera att det fungerar utan konflikt med befintliga tjänster i dev-miljö
- GitHub Actions ARM64-runner behövs för image-byggen i senare sprintar — DevOps utreder
