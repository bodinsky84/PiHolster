# Backup och återställning av PiHolster

PiHolster lagrar all konfiguration — betrodda enheter, ändrade enhetsnamn och
admin-sessioner — i en SQLite-databas på `/var/lib/piholster/piholster.db`.
Blocklistorna är inbyggda i imagen och behöver inte säkerhetskopieras.

## Vad som behöver säkerhetskopieras

| Fil | Sökväg | Innehåll |
|---|---|---|
| Konfigurationsdatabas | `/var/lib/piholster/piholster.db` | Betrodda enheter, enhetsnamn, admin-lösenord (hash) |

Certifikat och nycklar som genereras vid firstboot lagras i
`/etc/piholster/` och *behöver* säkerhetskopieras om du vill behålla
webbläsarens certifikats-undantag.

| Fil | Sökväg |
|---|---|
| TLS-certifikat | `/etc/piholster/cert.pem` |
| TLS-nyckel | `/etc/piholster/key.pem` |

---

## Ta backup via scp

Kör följande från din dator (byt ut `piholster.local` mot Pi:ns IP-adress om
mDNS inte fungerar):

```bash
# Skapa en lokal backup-katalog
mkdir -p ~/piholster-backup

# Kopiera databasen
scp piholster@piholster.local:/var/lib/piholster/piholster.db \
    ~/piholster-backup/piholster-$(date +%Y%m%d).db

# Kopiera TLS-certifikat och nyckel (valfritt)
scp piholster@piholster.local:/etc/piholster/cert.pem \
    piholster@piholster.local:/etc/piholster/key.pem \
    ~/piholster-backup/
```

> **Obs:** SSH är aktiverat under firstboot-fasen och inaktiveras därefter om
> du inte explicit aktiverar det. Kontrollera att SSH är aktiverat på Pi:n
> innan du försöker ansluta.

---

## Återställa backup

### Steg 1 — Stoppa piholsterd

```bash
ssh piholster@piholster.local "sudo systemctl stop piholsterd"
```

### Steg 2 — Kopiera backup-databasen tillbaka

```bash
scp ~/piholster-backup/piholster-YYYYMMDD.db \
    piholster@piholster.local:/var/lib/piholster/piholster.db
```

### Steg 3 — Sätt rätt ägarskap och rättigheter

```bash
ssh piholster@piholster.local \
    "sudo chown piholster:piholster /var/lib/piholster/piholster.db && \
     sudo chmod 600 /var/lib/piholster/piholster.db"
```

### Steg 4 — Starta piholsterd igen

```bash
ssh piholster@piholster.local "sudo systemctl start piholsterd"
```

### Steg 5 — Verifiera

Öppna `https://piholster.local` i din webbläsare och kontrollera att dina
betrodda enheter och enhetsnamn är tillbaka.

---

## Verifiera databasintegritet

Om du är osäker på om databasen är intakt efter ett strömavbrott kan du
kontrollera integriteten med:

```bash
ssh piholster@piholster.local \
    "sqlite3 /var/lib/piholster/piholster.db 'PRAGMA integrity_check;'"
```

Förväntat svar: `ok`

Om du får ett annat svar (t.ex. `*** in database main ***`) är databasen
korrupt och du behöver återställa från backup.

---

## Uppgradering till ny version

Det finns i nuläget ingen automatisk uppgraderingsväg. För att uppgradera:

1. Ta backup av databasen (se ovan).
2. Ta backup av TLS-certifikat och nyckel om du vill behålla dem.
3. Bränn om SD-kortet med den nya imagen.
4. Återställ databasen (se ovan).

> Backup-funktion via Web UI är planerad till v0.1.1.
