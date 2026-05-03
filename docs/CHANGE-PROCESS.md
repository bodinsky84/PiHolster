# PiHolster — Change Process

Dokumentet beskriver hur alla kodändringar, hotfixes och releaser hanteras i PiHolster-projektet.
Processen gäller alla teammedlemmar och bidragsgivare.

---

## Branchstrategi

Projektet använder en förenklad Gitflow med tre permanenta branchnivåer:

```
main          — Alltid deploybar, speglar senaste stabila release
develop       — Integration-branch, speglar pågående sprint
feature/*     — Kortlivade feature-branches, en per story/bugfix
hotfix/*      — Kortlivade branches för kritiska produktionsrättningar
```

Ingen direkt commit till `main` eller `develop` är tillåten — allt går via Pull Request.

---

## Normalflöde: Feature eller bugfix

### Steg 1 — Skapa branch

Brancha alltid från `develop`, inte från `main`.

Namnkonvention:

```
feature/US-02-dns-blocklist
feature/US-04-web-ui-skeleton
bugfix/fix-arp-scan-timeout
```

```
git checkout develop
git pull origin develop
git checkout -b feature/US-02-dns-blocklist
```

### Steg 2 — Implementera och commita

- Commitmeddelanden ska följa Conventional Commits: `feat:`, `fix:`, `chore:`, `docs:`, `test:`
- Håll commits atomära — en logisk förändring per commit
- Kör `make lint` och `make test` lokalt innan push

Exempel:
```
feat(dns): add StevenBlack blocklist loader
fix(dns): handle timeout when upstream DoH is unreachable
test(dns): add unit tests for blocklist parser
```

### Steg 3 — Öppna Pull Request

- PR öppnas mot `develop` (aldrig direkt mot `main`)
- PR-titeln ska matcha story-ID och kort beskrivning: `[US-02] DNS-server med grundläggande blocklist`
- PR-beskrivningen ska innehålla:
  - Vad som ändrats och varför
  - Hur man testar manuellt (steg-för-steg)
  - Länk till relaterad issue eller story
  - Skärmdump eller log-utdrag om det är ett UI- eller beteendeändring

### Steg 4 — Code review

- Minst **1 godkänd review** krävs för merge till `develop`
- Minst **2 godkända reviews** krävs för merge till `main`
- IT-säkerhet granskar alla PRs som rör: DNS-logik, autentisering, nätverksskanning, Telegram-integration
- Reviewer-ansvar:
  - Code reviewer: kodkvalitet, logik, testäckning
  - IT-säkerhet: attackyta, exponerade endpoints, hantering av hemliga uppgifter

Reviewkommentarer kategoriseras:
- `[BLOCK]` — Måste åtgärdas innan merge
- `[SUGGEST]` — Rekommendation, kan ignoreras med motivering
- `[NIT]` — Stil eller namngivning, valfritt

### Steg 5 — CI måste vara grön

Alla följande kontroller måste passera:
- Lint (ruff/eslint)
- Enhetstester
- Docker-image bygger utan fel

Röd CI = ingen merge, oavsett reviews.

### Steg 6 — Merge och cleanup

- Squash merge till `develop` om PRn innehåller många WIP-commits
- Merge commit till `develop` om historiken är ren och meningsfull
- Branchen raderas efter merge
- Uppdatera story-status i sprint-dokumentet

---

## Hotfix-flöde: Kritiskt produktionsfel

Används när ett fel i `main` kräver omedelbar rättning utan att vänta på pågående sprint.

```
git checkout main
git pull origin main
git checkout -b hotfix/dns-crash-on-empty-blocklist
```

- Hotfix-PR öppnas mot **både** `main` och `develop` parallellt
- Kräver 2 godkända reviews (inklusive IT-säkerhet om säkerhetsrelaterat)
- Efter merge till `main`: tagga en patch-release omedelbart (se Releaseprocess)
- Hotfix mergas också till `develop` för att undvika regression

---

## Releaseprocess

### Releasekandidat

1. Skapa branch `release/v0.1.0` från `develop`
2. Inga nya features — enbart bugfixar och dokumentation
3. IT-säkerhet genomför säkerhetsgenomgång av hela releasen
4. CTO godkänner releasen skriftligt (PR-kommentar eller issue-kommentar)

### Taggning och publicering

```
git checkout main
git merge release/v0.1.0 --no-ff
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin main --tags
```

DevOps ansvarar för:
- Bygga Raspberry Pi OS-image via CI
- Publicera image och checksumma till GitHub Releases
- Uppdatera `CHANGELOG.md` med vad som är nytt, fixat och borttaget

### Versionsnumrering

Projektet följer Semantic Versioning: `MAJOR.MINOR.PATCH`

| Typ           | Exempel | När                                              |
|---------------|---------|--------------------------------------------------|
| Patch         | v0.1.1  | Bugfix utan ny funktionalitet                    |
| Minor         | v0.2.0  | Ny funktion, bakåtkompatibel                     |
| Major         | v1.0.0  | Bryta API eller stor arkitekturförändring        |

---

## Hantering av hemligheter

- Inga API-nycklar, lösenord eller tokens committas till repot — någonsin
- `.env`-filer ligger i `.gitignore`
- Telegram bot-token och liknande konfigureras via miljövariabler eller en separat konfigurationsfil utanför repot
- Om en hemlighet oavsiktligt committas: kontakta IT-säkerhet omedelbart, rotera nyckeln, och rensa git-historiken med `git filter-repo`

---

## Eskalering och beslut

| Situation                              | Vem beslutar         |
|----------------------------------------|----------------------|
| Teknikval (DNS-backend, ramverk)       | CTO                  |
| Säkerhetsfrågor och risker             | IT-säkerhet          |
| Sprint-scope och prioritering          | PM                   |
| Release go/no-go                       | CTO + IT-säkerhet    |
| Merge-konflikt mellan reviews          | CTO avgör            |

Blockers eskaleras till PM samma dag de identifieras — inte i slutet av sprinten.
