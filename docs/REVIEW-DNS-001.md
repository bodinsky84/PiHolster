# CODE REVIEW: DNS-001
# PiHolster — DNS-kärna (US-02 + US-03)
# Granskare: Ghost Reviewer
# Datum: 2026-05-01
# Filer: blocklist.go, upstream.go, server.go, blocklist_test.go, main.go, go.mod, ads.txt

---

## BESLUT: GODKAND MED KOMMENTARER

Koden är i grunden solid och uppfyller de kritiska kraven i ADR-001 och ADR-002.
Inga blockers som hindrar merge, men tre punkter bör åtgärdas i ett snabbt follow-up.

---

## 1. Kritiska fel — blockers för merge

Inga blockers hittades.

Fail-closed DoH är korrekt implementerat. Dual-upstream med SERVFAIL-fallback
fungerar som avsett. Blocklist är trådsäker. Graceful shutdown inom 5s finns på
plats i main.go:46.

---

## 2. Förbättringar — bör fixas men blockerar inte merge

### 2a. go.mod: modernc.org/sqlite är en oanvänd beroende
Fil: go.mod, rad 7

modernc.org/sqlite är listad som direkt beroende men används ingenstans i de
granskade filerna. Det är ett tungt beroende (CGO-fri men stor) och dess
närvaro bryter mot ADR-002 om minimal attack surface.

Om SQLite planeras för en framtida feature — lägg till det i den PRen.
Om det är ett misstag — ta bort det nu.

    go mod tidy

Principbrott: Enkelhet (princip 1) — varje beroende motiverar sin existens.

### 2b. server.go:47-73 — Start() har en race: bind-fel missas i praktiken
Fil: server.go, rad 66-73

Start() returnerar direkt om ingen kanal har ett värde inom samma goroutine-
schemaläggningscykel. Det finns ingen garanti att ListenAndServe hinner försöka
binda porten innan select { default: return nil } körs.

Praktisk konsekvens: om porten är upptagen kan Start() returnera nil och main.go
tror att servern är uppe. Felet dyker upp i bakgrundsgoroutinen men fångas aldrig.

Korrekt lösning är att använda NotifyStartedFunc-callbacken som miekg/dns
exponerar, eller en kort sleep som är ett code smell. Bättre:

    s.udpServer = &mdns.Server{
        Addr:              addr,
        Net:               "udp",
        Handler:           mux,
        NotifyStartedFunc: func() { close(udpReady) },
    }

och sedan blockera tills båda kanalerna signalerar start — inte tills ett fel
dyker upp. Bind-felet skickas fortfarande via ListenAndServe-returvärdet.

Principbrott: Korrekthet (princip 2) — koden påstår att den rapporterar
startfel, men gör det inte tillförlitligt.

### 2c. server.go:77-88 — Shutdown returnerar bara ett av två fel
Fil: server.go, rad 85-88

Om både udpErr och tcpErr är satta returneras bara udpErr. tcpErr tappas
tyst. Även om det sällan spelar roll i praktiken är det en felhanteringsbrist.

Enkel fix med errors.Join (Go 1.20+):

    return errors.Join(udpErr, tcpErr)

Principbrott: Korrekthet (princip 2).

### 2d. upstream.go:83 — Response body utan storleksbegränsning motiveras men siffran är godtycklig
Fil: upstream.go, rad 83

io.LimitReader(httpResp.Body, 65535) är bra — det finns en gräns. Men 65535
bytes är det absoluta max för ett DNS-meddelande (RFC 1035 §2.3.4), vilket
innebär att ett meddelande precis på gränsen inte ger ett fel men ett som är
65536 bytes kapas tyst och Unpack misslyckas med ett kryptiskt fel.

Lägg till en konstant med kommentar:

    // maxDNSMsgSize is the maximum size of a DNS message per RFC 1035 §2.3.4.
    const maxDNSMsgSize = 65535

och om Unpack returnerar fel efter LimitReader — logga att det kan bero på
att svaret kapades. Det gör felsökning enklare.

Principbrott: Läsbarhet (princip 4) — den magiska siffran saknar kontext i koden.

### 2e. main.go:39 — DNS_PORT loggas men kan vara tom sträng
Fil: main.go, rad 39

    slog.Info("DNS server listening", "port", os.Getenv("DNS_PORT"))

Om DNS_PORT inte är satt är det tomma strängen som loggas, men servern lyssnar
faktiskt på 5300 (defaultvärdet sätts i NewServer). Loggraden är vilseledande.

Logga istället serverns faktiska port. Enklast: exportera port från Server
eller läs env-variabeln en gång i main och skicka ner den.

Principbrott: Korrekthet (princip 2) — loggen ljuger om vilken port som används.

---

## 3. Positivt

- blocklist.go är textboksexempel på korrekt RWMutex-användning. Lock vid
  skrivning (rad 78), RLock vid läsning (rad 28, 93). Inga onödiga låsningar.

- FQDN trailing dot hanteras på rätt ställe i both IsBlocked (blocklist.go:27)
  och handle (server.go:100). Dubbel normalisering är ofarlig och defensiv.

- Fail-closed är genuint korrekt: upstream.go:52 returnerar servfail(req), nil
  — inte ett error — vilket innebär att server.go:114 inte triggar HandleFailed
  utan skickar ett välformat SERVFAIL. Det är exakt rätt beteende.

- Blocklist-testerna är exemplariska. TestIsNotBlocked:71 testar explicit att
  subdomäner INTE blockeras — ett edge case som ofta glöms. TestLoadFromReaderMerges
  verifierar att laddning är additiv, inte ersättande.

- Inga hårdkodade secrets. Inga CGO-importer. log/slog används konsekvent.
  ADR-002 uppfylls.

- ads.txt använder korrekt hosts-format och parsas utan problem av LoadFromReader.

- Graceful shutdown i main.go:41-52 är korrekt strukturerad med signal.Notify
  och 5s context-timeout som matchar kravet i ADR-001.

---

## Sammanfattning av åtgärder

| # | Fil | Rad | Prioritet | Åtgärd |
|---|-----|-----|-----------|--------|
| 2a | go.mod | 7 | HOG | Ta bort oanvänt sqlite-beroende |
| 2b | server.go | 47-73 | HOG | Använd NotifyStartedFunc för tillförlitlig startdetektering |
| 2c | server.go | 85-88 | MEDEL | errors.Join för att inte tappa tcpErr |
| 2d | upstream.go | 83 | LAG | Namnge magic number, förbättra felmeddelande vid kapning |
| 2e | main.go | 39 | MEDEL | Logga faktisk port, inte env-variabeln |
