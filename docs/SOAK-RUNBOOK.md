# Soak-runbook (US-19, v0.1.0 GA)

CTO-instruktion för 7-dagars soak-test av v0.1.0-rc1 på fysisk Pi 3. Soaken är
GA-blocker — om Pi:n startas om eller piholsterd kraschar börjar de 7 dagarna om.

Sprint 4 är RC-validering, inte feature-utveckling. Lägg inte till funktioner
under soaken — om något måste fixas: stoppa soaken, fixa, börja om.

---

## Förberedelser (~30 min, dag 0)

### 1. Flasha rc1-imagen

1. Verifiera SHA256 mot `.img.xz.sha256` innan flash:
   ```bash
   sha256sum -c v0.1.0-rc1.img.xz.sha256
   ```
2. Flasha med Raspberry Pi Imager till SD-kortet.
3. Boota Pi:n via Ethernet.

### 2. Dokumentera enhet

Skapa lokalt anteckningsblock (behöver inte committas — bifogas GA-gate-PR:en
som bilaga eller länk):

```
docs/soak-log-rc1.md  (eller motsvarande lokalt)

Pi-modell:        Raspberry Pi 3B+ (Rev 1.3)
Serienummer:      <SERIAL>
SD-kort:          <fabrikat / kapacitet>
rc1 image SHA256: <från .img.xz.sha256>
Flash-datum:      YYYY-MM-DD HH:MM
Soak start:       YYYY-MM-DD HH:MM
Soak slut (mål):  YYYY-MM-DD HH:MM (start + 7 dagar)
```

### 3. Installera soak-monitor

På Pi:n via SSH:

```bash
# Kopiera skriptet från ditt repo (kör från utvecklingsdatorn)
scp scripts/soak-monitor.sh pi@piholster.local:/tmp/

# På Pi:n
ssh pi@piholster.local
sudo install -m 0755 /tmp/soak-monitor.sh /usr/local/bin/soak-monitor.sh
sudo install -d -o root -g root -m 0755 /var/log/piholster

# Lägg till cron-jobbet (kör som root så VmRSS för piholsterd kan läsas)
echo '*/5 * * * * /usr/local/bin/soak-monitor.sh >> /var/log/piholster/soak.cron.log 2>&1' \
    | sudo tee /etc/cron.d/piholster-soak
sudo chmod 0644 /etc/cron.d/piholster-soak

# Verifiera första körningen direkt
sudo /usr/local/bin/soak-monitor.sh
cat /var/log/piholster/soak.csv
```

CSV ska visa header + 1 rad. Om `piholsterd_rss_mb` är `0` — kontrollera att
piholsterd faktiskt körs (`systemctl status piholsterd`) innan du fortsätter.

### 4. Peka routern på Pi:n som DNS

Sätt primär DNS i routerns DHCP till Pi:ns IP. Verifiera att riktiga enheter
i hemnätverket använder den:

```bash
# Vänta 10 minuter, kolla sedan att frågor faktiskt kommer in
ssh pi@piholster.local
sudo sqlite3 /var/lib/piholster/piholster.db \
    "SELECT COUNT(*) FROM query_log WHERE queried_at > datetime('now','-10 minutes');"
```

>= 5 ska räcka för att bekräfta att flödet fungerar. Om 0: routern pekar inte
rätt, eller DNS:en ger inget svar — felsök innan soaken startar.

### 5. Sätt extern monitoring (BB-12-mitigation)

På din utvecklingsdator (inte på Pi:n):

```bash
# Enkelt liveness-skript — varnar om piholster.local slutar svara på DNS
while true; do
    if ! dig +short +time=2 +tries=1 example.com @piholster.local >/dev/null; then
        echo "$(date): piholster.local DNS DEAD"
        # ev. notify-send / Pushover / e-post
    fi
    sleep 300  # 5 min
done
```

Stoppa skriptet när soaken är slut.

---

## Under soaken (dag 1–7, ~5 min/dag)

Daglig check (helst samma tid varje dag):

```bash
ssh pi@piholster.local

# 1. Kraschar?
sudo journalctl -u piholsterd --since "24 hours ago" | grep -iE 'sigsegv|sigabrt|oom-?kill|panic' || echo "OK: inga kraschar"

# 2. Omstarter?
sudo systemctl show piholsterd --property=ActiveEnterTimestamp,NRestarts
#   NRestarts=0 ska gälla under HELA soaken. Om != 0: soaken börjar om.

# 3. DNS-flöde senaste timmen (>=100 frågor)
sudo sqlite3 /var/lib/piholster/piholster.db \
    "SELECT COUNT(*) FROM query_log WHERE queried_at > datetime('now','-1 hour');"

# 4. Senaste RAM-mätning
tail -n 3 /var/log/piholster/soak.csv
```

Om något av punkt 1–2 fallerar: stoppa soaken, dokumentera utfall, fixa, börja om
från dag 0. Om punkt 3 är < 100 i snitt: justera så fler enheter använder Pi:n
som DNS (annars uppfyller soaken inte AC-2).

---

## Cert-renewal-verifiering (en gång under soaken, dag 1–6)

Per US-19 AC-5. Verifiera att TLS-certets giltighetstid är 10 år från firstboot:

```bash
ssh pi@piholster.local
sudo openssl x509 -in /var/lib/piholster/tls/cert.pem -noout -dates
# notBefore: <firstboot-datum>
# notAfter:  <firstboot-datum + 10 år>
```

PiHolster v0.1.0 har **ingen automatisk cert-renewal**. Detta noteras som känd
begränsning i `RELEASE_NOTES.md` (US-22) — inget extra steg behövs.

Om det finns en `/api/tls-info`-endpoint i en framtida version: testa den och
dokumentera utfallet. För v0.1.0 räcker `openssl x509 -dates` ovan.

---

## Avslutning (dag 7+, ~30 min)

### 1. Stoppa cron och samla in data

```bash
ssh pi@piholster.local
sudo rm /etc/cron.d/piholster-soak

# Kopiera CSV till utvecklingsdatorn
exit
scp pi@piholster.local:/var/log/piholster/soak.csv ./soak-rc1.csv
```

### 2. Generera RAM-graf

Snabbaste vägen — Python + matplotlib (eller Excel/Numbers/LibreOffice Calc):

```python
# soak_plot.py
import csv, matplotlib.pyplot as plt
from datetime import datetime

ts, rss = [], []
with open('soak-rc1.csv') as f:
    for row in csv.DictReader(f):
        ts.append(datetime.fromisoformat(row['ts']))
        rss.append(int(row['piholsterd_rss_mb']))

plt.figure(figsize=(12, 4))
plt.plot(ts, rss, '-')
plt.title('piholsterd RSS over 7-day soak (v0.1.0-rc1)')
plt.xlabel('time'); plt.ylabel('RSS (MB)')
plt.grid(True, alpha=0.3)
plt.tight_layout()
plt.savefig('soak-rc1-ram.png', dpi=120)
```

Verifiera kravet: **<=5 MB ökning per 24 timmar**. Snabb sanity-check:

```bash
# Första och sista mätning
head -n 2 soak-rc1.csv | tail -n 1
tail -n 1 soak-rc1.csv
# (sista_rss - första_rss) / antal_dagar bör vara <= 5
```

Om trenden är konsekvent uppåt med > 5 MB/dygn: minnesläcka — soaken NOK.

#### Automatiserad analys

`scripts/soak-analyze.sh` läser CSV:n, kör linjär regression mot
piholsterd-RSS, kontrollerar PID-byten och ger ett samlat PASS/NOK.
Använd det som beslutsstöd istället för head/tail-checken ovan:

```bash
bash scripts/soak-analyze.sh ./soak-rc1.csv
# eller maskinläsbart:
bash scripts/soak-analyze.sh ./soak-rc1.csv --json
```

Exit-kod 0 = PASS, 1 = NOK, 2 = inputfel (CSV saknas/format fel).

### 3. Skriv soak-rapport

3–5 meningar, OK/NOK per AC, klistras direkt som kommentar i GA-gate-PR:en.
Mall:

```
SOAK-RAPPORT v0.1.0-rc1 — CTO — YYYY-MM-DD

Period:           YYYY-MM-DD HH:MM -> YYYY-MM-DD HH:MM (7 d, X h)
Hårdvara:         Pi 3B+ <serial>, SD-kort <fabrikat>
DNS-flöde:        snitt N frågor/timme (krav >=100) — OK / NOK
RAM-trend:        +N MB över 7 dygn (krav <=35 MB) — OK / NOK
Stabilitet:       NRestarts=0, inga SIGSEGV/SIGABRT/OOM — OK / NOK
Cert-utgångsdatum: YYYY-MM-DD (10 år från firstboot) — OK
Cert-renewal:     ingen automatik (känd begränsning i RELEASE_NOTES) — OK

Bilaga: soak-rc1-ram.png + soak-rc1.csv
Slutsats: GA APPROVED / GA BLOCKED
```

Bifoga PNG + CSV till GA-gate-PR:en.

---

## Felsökning

### Cron skriver inte till soak.csv
- `sudo systemctl status cron` — körs cron-tjänsten?
- `sudo cat /var/log/piholster/soak.cron.log` — felmeddelanden?
- Kör skriptet manuellt: `sudo /usr/local/bin/soak-monitor.sh && tail -n 1 /var/log/piholster/soak.csv`

### piholsterd_rss_mb är 0 i alla rader
- piholsterd körs inte: `systemctl status piholsterd`
- Eller: cron körs som annan user och kan inte läsa `/proc/<pid>/status`. Lös: cron-jobbet ska köras som root (det är default i `/etc/cron.d/`).

### dns_queries_24h är tom
- sqlite3 saknas: `sudo apt-get install -y sqlite3`
- DB-sökväg fel: kör `DB_PATH=/var/lib/piholster/piholster.db /usr/local/bin/soak-monitor.sh`
- Detta blockerar inte soaken — kolumnen är best-effort.
