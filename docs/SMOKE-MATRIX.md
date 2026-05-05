# Smoke-matris för 3 Pi-enheter (US-26)

DevOps-runbook för att köra smoke-sviten mot 3 fysiska Pi 3-enheter med olika
SD-kort. Resultatet bifogas GA-gate-PR:en. Alla 3 enheter måste klara hela sviten
för att GA-gate ska kunna passera.

Sprint 4 är RC-validering. Smoke-sviten ändras inte under denna körning. Om
ett test misslyckas och det är en bugg i imagen — bygg rc2, flasha alla 3 igen,
kör om hela matrisen.

---

## Förberedelser

### Hårdvaruinventering (dag 1, BB-14)

| Enhet | Modell / Rev | Serienummer | SD-kort (fabrikat / kapacitet) | Verifierad finnas |
|-------|--------------|-------------|-------------------------------|-------------------|
| A     |              |             |                               |                   |
| B     |              |             |                               |                   |
| C     |              |             |                               |                   |

Kraven från US-26 AC-1:
- Minst 3 fysiska Pi 3-enheter (3B vs 3B+ blandning föredras).
- 3 olika SD-kort — inte 3 identiska från samma batch.
- Pi 4 godtas som ersättning för max en enhet om Pi 3 saknas (CTO-beslut, BB-14).

### Image-verifiering

```bash
sha256sum -c v0.1.0-rc1.img.xz.sha256
```

Använd **exakt samma** `.img.xz`-fil till alla 3 flashar. Om imagen byggs om
som rc2 — kör om hela matrisen från noll, inkl. tabellen nedan.

### Flasha varje enhet

Flasha enhet A, B och C i tur och ordning med Raspberry Pi Imager. Boota varje
Pi via Ethernet och vänta 90 s på att firstboot är klar.

---

## Köra smoke-sviten

Smoke-suiten körs från en utvecklingsdator, inte på Pi:n. Den läser `PI_IP`
från miljön och använder `tests/smoke`-paketet.

```bash
cd tests/smoke

# Enhet A
PI_IP=192.168.X.A go test -v -count=1 ./... 2>&1 | tee ../../smoke-results-A.log

# Enhet B
PI_IP=192.168.X.B go test -v -count=1 ./... 2>&1 | tee ../../smoke-results-B.log

# Enhet C
PI_IP=192.168.X.C go test -v -count=1 ./... 2>&1 | tee ../../smoke-results-C.log
```

`-count=1` tvingar Go att inte cacha resultat mellan körningar. `tee` sparar
output så IT-säk kan granska loggarna i efterhand.

Valfria miljövariabler:
- `SMOKE_TIMEOUT` — per-test HTTP-timeout (default 30 s)
- `BOOT_TIMEOUT` — max-väntetid för `TestBootTime` (default 90 s)

---

## Resultat-tabell

Fyll i en rad per Pi efter varje körning. PASS = testet exit 0, FAIL = exit non-zero.
Notera rådata (latens, RAM-tal, etc.) i kolumnen där det är meningsfullt.

| Test                              | Krav / kommentar                          | Enhet A | Enhet B | Enhet C |
|-----------------------------------|-------------------------------------------|---------|---------|---------|
| TestBootTime                      | Pi svarar på SSH + DNS inom BOOT_TIMEOUT | _ s     | _ s     | _ s     |
| TestFirewallPreFirstboot          | Inga öppna portar utöver 22/53/80/443     | PASS / FAIL | PASS / FAIL | PASS / FAIL |
| TestHTTPSRedirect (US-24)         | HTTP 80 -> 301 med Location: https://     | PASS / FAIL | PASS / FAIL | PASS / FAIL |
| TestDNSLatency                    | Median <= acceptansgräns (se test)        | _ ms    | _ ms    | _ ms    |
| TestRAMUsage                      | RAM idle < gräns (se test)                | _ MB    | _ MB    | _ MB    |
| TestWebUIResponds                 | GET / -> HTTP 200, HTML innehåller titel  | PASS / FAIL | PASS / FAIL | PASS / FAIL |
| TestAPIHealth                     | /api/status -> 200 + dns_running=true     | PASS / FAIL | PASS / FAIL | PASS / FAIL |

Manuell verifiering (utöver Go-testerna):

| Kontroll                                  | Hur                                                  | Enhet A | Enhet B | Enhet C |
|-------------------------------------------|------------------------------------------------------|---------|---------|---------|
| Capability `cap_net_bind_service` på piholsterd | `getcap /usr/local/bin/piholsterd`             | PASS / FAIL | PASS / FAIL | PASS / FAIL |
| Capability `cap_net_raw` på piholster-arpd | `getcap /usr/local/bin/piholster-arpd`              | PASS / FAIL | PASS / FAIL | PASS / FAIL |
| Firstboot-fönstret stängs (filtered -> open) | `nmap -p 53,80,443 PI_IP` före och efter firstboot | _ s     | _ s     | _ s     |

---

## Beslutslogik vid FAIL

Per US-26 AC-5:

1. **Hittas ett enskilt FAIL på en enhet?** DevOps utreder om det är miljö
   (dåligt SD-kort, kabel, router) eller imagen.
2. **Miljöproblem?** Notera i tabellen, byt komponent, kör om bara den enheten.
   Räknas som PASS efter omkörning.
3. **Reproducerbart fel i imagen?** Stoppa hela matrisen. Fixa, bygg rc2,
   flasha alla 3 igen, kör hela matrisen om från noll.
4. **Inkonsekventa resultat (FAIL ibland, PASS ibland)?** Behandlas som
   reproducerbart fel — flaky tester får inte gå till GA.

---

## IT-säk sign-off

När alla 3 enheter visar PASS i alla rader:

```
US-26 verifierat — smoke-sviten grön på 3 Pi 3-enheter
[IT-säks initialer + datum]
Loggar bilagda: smoke-results-A.log, smoke-results-B.log, smoke-results-C.log
```

Klistras som kommentar i GA-gate-PR:en. Alla 3 loggfiler bifogas PR:en eller
laddas upp som GitHub Actions artifacts.
