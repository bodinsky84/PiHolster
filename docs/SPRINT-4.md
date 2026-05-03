# Sprint 4 — RC-validering, GA-förberedelse och v0.1.0-release

**Sprintlängd:** 3 veckor
**Sprint-mål:** Validera att v0.1.0-rc1 är stabil nog för GA-release genom 7 dagars soak,
beta-testning med icke-tekniska användare, manuell verifiering av säkerhetskrav och
leverans av all dokumentation som krävs — samt tagga v0.1.0 GA.

Sprint 4 är INTE en feature-sprint. Inga nya funktioner byggs.
All kod som mergas under Sprint 4 är antingen testinfrastruktur, dokumentation eller
buggfixar som uppkommit under soak/beta. Feature-requests parkeras i v0.1.1-backloggen.

Alla US-19 till US-26 är GA-blockers. v0.1.0 får inte taggas förrän varje story
har slutförts och CTO + IT-säkerhet har gett skriftlig sign-off i GA-gate-PR:en.

---

## Beroendediagram

```
US-19 (7-dagars soak — CTO kör)
  |
  +---> [soak-period dag 1–7: parallellt med US-20, US-21, US-23, US-25, US-26]
  |
  +---> US-27 (GA release — blockas tills soak är klar)

US-20 (beta-testning, 3–5 användare)
  |
  +---> [buggfixar vid behov — mergas till rc-branch om kritiska]
  |
  +---> US-22 (RELEASE_NOTES — kända begränsningar från beta och soak)
  |
  +---> US-27

US-21 (WAL-konsistens och backup/restore)
  |
  +---> US-27

US-23 (SECURITY.md — M-05 dokumenterat)
  |
  +---> US-25 (IT-säk granskar M-04 och M-05 ihop)
  |
  +---> US-27

US-24 (TestHTTPSRedirect i smoke-suite)
  |
  +---> US-26 (smoke-tester på 3 Pi-enheter)
  |
  +---> US-27

US-25 (M-04 formell granskning av MANIFEST)
  |
  +---> US-27

US-26 (smoke-tester på 3 Pi 3-enheter)
  |
  +---> US-27

US-22 (RELEASE_NOTES.md)
  |
  +---> US-27
```

**Kritisk väg:** US-19 (soak dag 1–7) + US-26 (smoke 3 Pi) + US-25 (IT-säk M-04)
--> GA-gate --> US-27.

Soak-perioden styr tidplanen. Dag 8 kan inte inledas förrän dag 1–7 är avklarade
utan omstart eller kritisk avvikelse. Alla övriga stories (US-20 till US-26) ska
vara klara senast dag 14 så att GA-gate kan hållas dag 15–16 och US-27 körs dag 17–18.

Reservera dag 19–21 som buffert för: buggfixar från soak/beta, IT-säk-kö,
oförutsedda problem vid image-rebuild inför GA.

---

## User stories

---

### US-19 — 7-dagars soak-test på fysisk Pi 3

**Som** CTO
**vill jag** köra v0.1.0-rc1 kontinuerligt på min egen Pi 3 i 7 dagar med riktigt
DNS-flöde från mitt hemnätverk
**så att** vi med säkerhet vet att piholsterd inte minnesläcker, kraschar eller
kräver omstart under normal drift.

**Prioritet:** Hög — GA-blocker (CTO-krav)
**Estimat:** 7 dagar realtid (ingen dev-effort utöver setup, ~2 timmar)
**Ansvarig:** CTO (kör soaken), DevOps (setup av mätning)
**IT-säk review:** Ej krävd

**Acceptanskriterier:**

1. CTO:s Pi 3 (1 GB RAM) flashas med v0.1.0-rc1-imagen.
   Flash-datum och Pi-serienummer dokumenteras i ett kort soak-logg-dokument
   (`docs/soak-log-rc1.md` eller motsvarande, behöver ej mergas — CTO:s eget anteckningsblock).

2. Piholsterd tar emot riktigt DNS-flöde från minst en enhet på CTO:s hemnätverk
   under hela perioden. Minimikrav: >= 100 DNS-frågor per timme i snitt.

3. **RAM-mätning:**
   - DevOps sätter upp ett enkelt mätskript som var 5:e minut skriver
     `free -m`-output till en loggfil på Pi:n via en lokalt schemalagd cron-job
     (eller motsvarande, utan att öppna inkommande portar).
   - Alternativt: `/api/status`-endpointen exponerar `ram_used_mb`; ett externt
     script pollar och sparar till CSV lokalt på CTO:s dator.
   - Kravet: RAM-användning vid mättidpunkterna visar ingen konsekvent uppåtgående
     trend (d.v.s. ingen minnesläcka). En ökning på <= 5 MB per 24 timmar är
     acceptabel.
   - CTO exporterar RAM-grafen (eller CSV-filen) och bifogar till GA-gate-PR:en.

4. **Stabilitetskrav:**
   - Noll planerade eller oplanerade omstarter av `piholsterd` under 7 dagar.
   - Noll `SIGSEGV`, `SIGABRT` eller OOM-kill-händelser i journalctl-loggen.
   - Om `piholsterd` kraschar eller systemet behöver startas om: soak-perioden
     börjar om från dag 0.

5. **Cert-renewal-path verifieras manuellt:**
   - Piholsterd svarar på `GET /api/tls-info` (eller motsvarande) med
     cert-giltighetstid.
   - CTO verifierar att cert-expiry-datum är korrekt (10 år från firstboot).
   - Om en cert-renewal-path är implementerad: CTO testar den manuellt och
     dokumenterar utfallet. Om den inte är implementerad: noteras som känd begränsning
     i RELEASE_NOTES (US-22).

6. Soak avslutas med att CTO skriver en kortfattad soak-rapport (3–5 meningar,
   OK/NOK per punkt) direkt som kommentar i GA-gate-PR:en.

---

### US-20 — Beta-testning med 3–5 icke-tekniska användare

**Som** produktteam
**vill jag** att minst 3 och upp till 5 icke-tekniska användare installerar PiHolster
med enbart INSTALL.md som stöd
**så att** vi vet att installationsguiden är tillräcklig och att vi fångar upp
användbarhetsproblem innan GA.

**Prioritet:** Hög — GA-blocker (CTO-krav)
**Estimat:** 3 dagar (rekrytering dag 1–2, körning dag 3–9, uppföljning dag 10–11)
**Ansvarig:** PM (koordinering), Senior Go-dev (teknisk support på avstånd vid showstopper)
**IT-säk review:** Ej krävd

**Acceptanskriterier:**

1. PM rekryterar 3–5 beta-testare. Krav på testare:
   - Inga kunskaper om Linux, kommandoraden eller nätverk.
   - Har tillgång till en Raspberry Pi 3 (eller 4) och ett hemnätverk.
   - Kan ge skriftlig feedback på svenska.

2. Beta-testare får enbart:
   - En länk till `.img.xz`-filen för v0.1.0-rc1.
   - En länk till `docs/INSTALL.md`.
   - En e-postadress eller Signal-nummer att kontakta vid showstopper (men
     utan att hjälpa dem igenom installationen steg-för-steg).

3. Beta-testare besvarar ett kortfattat feedbackformulär (max 10 frågor) efter
   installationsförsöket. Frågorna täcker minst:
   - Lyckades du installera? (Ja / Nej / Delvis)
   - Vilket steg var svårast eller oklart?
   - Fick du certifikatvarningen? Visste du vad du skulle göra?
   - Lyckades du konfigurera routern?
   - Har du några andra kommentarer?

4. Feedbacken sammanfattas av PM i ett kort dokument som bifogas GA-gate-PR:en.
   Varje identifierat problem klassificeras som:
   - **Showstopper:** blockerar GA (t.ex. INSTALL.md leder fel, image bootar inte).
   - **Bör fixas:** tas in i en rc2-patch om tid finns, annars noteras i RELEASE_NOTES.
   - **v0.1.1:** parkeras i backloggen.

5. Minst 2 av 3 beta-testare (eller minst 3 av 5) ska ha lyckats med en komplett
   installation utan hjälp från teamet. Om färre lyckas: PM och CTO beslutar om
   INSTALL.md behöver revideras innan GA.

6. Alla showstoppers är åtgärdade (och imagen eventuellt ombyggd som rc2) innan
   GA-gate.

---

### US-21 — Verifiera WAL-konsistens och backup/restore av piholster.db

**Som** PiHolster-installation
**vill jag** att SQLite-databasen överlever ett oväntat strömavbrott utan korruption
och att backup/restore-flödet fungerar
**så att** användare inte förlorar sin konfiguration vid ett strömavbrott.

**Prioritet:** Hög — GA-blocker (CTO-krav)
**Estimat:** 1,5 dagar
**Ansvarig:** Senior Go-dev
**IT-säk review:** Ej krävd

**Acceptanskriterier:**

1. **SQLite WAL-läge verifierat:**
   - Bekräfta att `piholster.db` öppnas med `PRAGMA journal_mode=WAL` och att
     `PRAGMA synchronous=NORMAL` (eller FULL) är satt.
   - Verifikationsmetod: anslut till databasen med `sqlite3` och kör
     `PRAGMA journal_mode; PRAGMA synchronous;` — dokumentera utfall i US-21:s PR.

2. **Strömavbrottstest:**
   - Flash en Pi med rc1-imagen.
   - Låt piholsterd köra i 5 minuter med aktiv DNS-trafik (skicka minst 50 frågor).
   - Simulera abrupt strömavbrott: dra strömkabeln utan `shutdown`.
   - Starta Pi:n igen.
   - Verifiera att `piholsterd` startar utan felmeddelanden relaterade till databas-korruption.
   - Verifiera att `sqlite3 /var/lib/piholster/piholster.db "PRAGMA integrity_check;"`
     returnerar `ok`.
   - Testet upprepas 3 gånger. Alla 3 måste passera.

3. **Backup-flöde:**
   - En backup-metod är dokumenterad i `docs/INSTALL.md` (eller ett separat
     `docs/BACKUP.md` om PM bedömer det lämpligare).
   - Minimikrav för backup: användaren kan kopiera `/var/lib/piholster/piholster.db`
     till sin dator via scp eller via Web UI:n om en backup-endpoint finns.
   - Om Web UI saknar backup-endpoint: dokumentera scp-metoden tydligt och notera
     "backup via UI" som v0.1.1-feature i RELEASE_NOTES (US-22).

4. **Restore-flöde:**
   - Dokumentera hur användaren återställer en backup-databas:
     stoppa piholsterd, ersätt `piholster.db`, starta piholsterd.
   - Testa flödet manuellt: ta backup, radera `piholster.db`, restore, verifiera
     att konfigurationen (blocklists, ev. inställningar) är intakt.

5. Testresultat och kommandoutput dokumenteras direkt i PR-beskrivningen.

---

### US-22 — RELEASE_NOTES.md med kända begränsningar och uppgraderingsguide

**Som** slutanvändare av v0.1.0
**vill jag** ha ett dokument som tydligt beskriver vad versionen kan, vad den
inte kan och hur jag uppgraderar i framtiden
**så att** mina förväntningar är korrekta och jag vet hur framtida uppdateringar kommer att gå till.

**Prioritet:** Hög — GA-blocker (CTO-krav)
**Estimat:** 1 dag
**Ansvarig:** PM (primär), Senior Go-dev (teknisk faktagranskning)
**IT-säk review:** Ej krävd

**Acceptanskriterier:**

1. Dokumentet lever i `RELEASE_NOTES.md` i repots rot och länkas från `README.md`.

2. **Versionsrubrik och datum:**
   ```
   ## v0.1.0 — [datum för GA-release]
   ```

3. **Vad som ingår i v0.1.0** — en kort punktlista (max 10 punkter) med de
   viktigaste funktionerna som levererats i Sprint 1–3.

4. **Kända begränsningar** — obligatoriska punkter:
   - `initial-password`-fil är mode 0640 i stället för 0400 (M-05). Förklaring:
     filen exponeras endast i RAM (tmpfs) och raderas vid första inloggning.
     Fixas i v0.1.1.
   - `LockPersonality=true` saknas i `piholster-firstboot.service` (L-03).
     Fixas i v0.1.1.
   - Ingen backup-funktion via Web UI (om det saknas efter US-21).
   - Ingen cert-renewal-funktion (om det saknas efter US-19:s verifiering).
   - WiFi stöds inte — Ethernet krävs.
   - Ingen one-click update-funktion. Uppgradering kräver ny SD-kortsbränning
     tills v1.0.
   - Ytterligare begränsningar som identifierats under soak (US-19) och beta (US-20)
     läggs till innan GA-gate.

5. **Uppgraderingsguide (v0.1.0 -> kommande versioner):**
   - Tydlig instruktion: "Det finns i nuläget ingen automatisk uppgraderingsväg.
     För att uppgradera till en ny version: ta backup av `piholster.db`,
     bränn om SD-kortet med ny image, restore databasen."
   - Hänvisning till backup/restore-instruktionen i INSTALL.md (eller BACKUP.md).

6. **Roadmap-utblick (kort):**
   - 2–3 meningar om vad v1.0 planeras innehålla (one-click update, schemalagd
     blocklist-uppdatering, historikvy). Exakta datum anges inte — "planerat ca 6
     månader efter v0.1.0 GA".

7. Dokumentet granskas av CTO innan GA-gate.

---

### US-23 — SECURITY.md med M-05 dokumenterat

**Som** säkerhetsmedveten administratör
**vill jag** hitta ett dokument som öppet redovisar projektets kända säkerhetsavvikelser
och vilken risknivå de har
**så att** jag kan göra ett informerat beslut om PiHolster passar min miljö.

**Prioritet:** Hög — GA-blocker (IT-säk-krav)
**Estimat:** 0,5 dag
**Ansvarig:** Senior Go-dev (primär), PM (struktur och ton)
**IT-säk review:** Krävs — IT-säk granskar M-05-formuleringen och riskbedömningen

**Acceptanskriterier:**

1. Dokumentet lever i `SECURITY.md` i repots rot och länkas från `README.md`.

2. **Avsnittet om M-05** ska innehålla:
   - **Fynd:** `initial-password`-fil är mode 0640 i stället för det härdade mode 0400.
   - **Kontext:** Filen lever i tmpfs (`/run/piholster/`), exponeras enbart i RAM
     och raderas automatiskt av `piholsterd` vid första lyckade inloggning.
     Gruppen `piholster` är den enda som kan läsa filen utöver root.
   - **Risknivå:** Låg. Kräver att en lokal process med piholster-grupptillhörighet
     komprometteras under firstboot-fönstret.
   - **Planerad åtgärd:** Ändras till mode 0400 i v0.1.1. Referens: L-03.
   - **Kringgåendet är godkänt av IT-säkerhet:** [IT-säks initialer och datum fylls
     i vid sign-off].

3. **Avsnittet om L-03** ska innehålla:
   - **Fynd:** `LockPersonality=true` saknas i `piholster-firstboot.service`.
   - **Kontext:** `piholsterd.service` och `piholster-arpd.service` har
     `LockPersonality=true`. Firstboot-tjänsten saknar det.
   - **Risknivå:** Mycket låg. Firstboot körs som oneshot vid varje boot och
     avslutas snabbt. Angreppsytan är minimal.
   - **Planerad åtgärd:** Läggs till i v0.1.1.

4. **Ansvarsfull avslöjandepolicy (Responsible Disclosure):**
   - E-postadress eller kontaktmetod för att rapportera säkerhetsproblem.
   - Förväntad svarstid: 5 arbetsdagar.
   - Policy: avslöjande sker 90 dagar efter rapportering om inte fix är deployad.

5. IT-säkerhet granskar hela SECURITY.md-dokumentet och godkänner M-05-formuleringen
   skriftligen (kommentar i PR) innan merge.

---

### US-24 — TestHTTPSRedirect i smoke-suite

**Som** smoke-test-suite
**vill jag** verifiera att HTTP-trafik på port 80 automatiskt omdirigeras till HTTPS
**så att** ingen framtida kodändring tyst tar bort HTTP->HTTPS-redirecten.

**Prioritet:** Hög — GA-blocker (IT-säk-krav)
**Estimat:** 0,5 dag
**Ansvarig:** Senior Go-dev
**IT-säk review:** Ej krävd (testinfrastruktur)

**Acceptanskriterier:**

1. Ett nytt test (`TestHTTPSRedirect`) läggs till i `tests/smoke/`-paketet
   (Go-testpaket eller bash-skript beroende på befintlig konvention i US-17).

2. **Testlogik:**
   ```
   GET http://<PI_IP>:80/
   Förväntat: HTTP 301 eller 302
   Location-header: https://<PI_IP>/  (eller https://piholster.local/)
   ```
   Om svaret är något annat än 301/302, eller om `Location`-headern saknas:
   testet misslyckas.

3. Testet läggs till i det befintliga CI-jobbet `smoke-test` i
   `release-image.yml` och körs som en del av den ordinarie smoke-sviten.
   Det är ett nytt mandatory test — ett misslyckat `TestHTTPSRedirect` ska
   markera hela smoke-jobbet som Failed och blockera image-upload, precis som
   övriga smoke-tester.

4. PR inkluderar ett kort bevis på att testet faktiskt kan misslyckas: antingen
   ett screenshot av ett misslyckat test-run i CI, eller en lokal körning mot
   en Pi där redirecten temporärt är inaktiverad.

---

### US-25 — M-04 formell IT-säkerhetsgranskning av MANIFEST-generering

**Som** IT-säkerhet
**vill jag** granska och formellt godkänna att `image/MANIFEST` genereras korrekt
och innehåller tillräcklig information för att verifiera image-integriteten
**så att** vi kan stå bakom att v0.1.0-imagen är verifierbar och att supply chain
är dokumenterad.

**Prioritet:** Hög — GA-blocker (IT-säk-krav)
**Estimat:** 0,5 dag (dev-effort: inga kodändringar förväntade — enbart granskning
och dokumentation)
**Ansvarig:** IT-säkerhet (granskning), DevOps (svar på frågor och eventuella åtgärder)
**IT-säk review:** Denna story IS IT-säk-granskningen

**Acceptanskriterier:**

1. IT-säkerhet granskar `image/build.sh` och det genererade `image/MANIFEST`-formatet
   från ett rc1-bygge. Granskningen ska täcka:
   - Är pi-gen commit-hash med och korrekt?
   - Är base-image SHA256 med och verifierbart?
   - Är piholsterd- och piholster-arpd-versioner med?
   - Är build-timestamp med?
   - Genereras MANIFEST automatiskt eller finns risk för manuella misstag?

2. Om granskningen hittar brister: dessa åtgärdas av DevOps och en ny
   MANIFEST genereras från ett rc1-bygge (eller rc2 om image-rebuild krävs).

3. IT-säkerhet dokumenterar utfallet skriftligen:
   - Om godkänt: kommentar i GA-gate-PR:en med texten "M-04 verifierat —
     MANIFEST-generering godkänd. [initialer + datum]".
   - Om ej godkänt: konkret lista på vad som saknas, med deadline för åtgärd.

4. Ingen kod mergas specifikt för denna story (om inga brister hittas).
   Om brister hittas: DevOps öppnar en separata bugg-PR med fix, IT-säk granskar om.

---

### US-26 — Smoke-tester på minst 3 fysiska Pi 3-enheter med olika SD-kort

**Som** IT-säkerhet och release-ansvarig
**vill jag** att smoke-sviten körs på minst 3 olika Pi 3-enheter med olika SD-kort
**så att** vi vet att release-kandidaten inte bara fungerar på vår specifika test-Pi
utan är reproducerbar på ny hårdvara.

**Prioritet:** Hög — GA-blocker (IT-säk-krav)
**Estimat:** 2 dagar (logistik + körning + dokumentation)
**Ansvarig:** DevOps (primär), IT-säkerhet (verifiering av utfall)
**IT-säk review:** IT-säk verifierar att testerna körts korrekt och godkänner utfallet

**Acceptanskriterier:**

1. Tre fysiska Raspberry Pi 3-enheter identifieras och dokumenteras:
   - Enhet A: befintlig test-Pi (känd sedan Sprint 3)
   - Enhet B: en annan Pi 3, helst annan revision (t.ex. 3B vs 3B+)
   - Enhet C: en tredje Pi 3

   SD-korten ska vara av olika fabrikat eller ålder (d.v.s. inte tre identiska
   kort från samma batch).

2. Varje enhet flashas med v0.1.0-rc1-imagen (exakt samma `.img.xz`-fil,
   SHA256 verifierat före flash).

3. Hela smoke-sviten från US-17 (inklusive det nya `TestHTTPSRedirect` från US-24)
   körs mot varje enhet. Testerna körs manuellt om CI-infrastrukturen inte stöder
   multi-Pi-körning automatiskt.

4. Testresultat dokumenteras per enhet i en tabell:

   | Test | Enhet A | Enhet B | Enhet C |
   |---|---|---|---|
   | Boot-tid (s) | | | |
   | Firstboot-fönster (filtered -> open) | | | |
   | DNS-latens median (ms) | | | |
   | RAM idle (MB) | | | |
   | Web UI HTTP 200 | | | |
   | API /api/status dns_ok | | | |
   | Capability-kontroll | | | |
   | TestHTTPSRedirect | | | |

   Tabellen bifogas GA-gate-PR:en.

5. Alla 3 enheter måste klara hela smoke-sviten utan undantag för att GA-gate
   ska kunna godkännas. Om en enhet misslyckas på ett enskilt test:
   - DevOps utreder om det är ett miljöproblem (t.ex. dåligt SD-kort) eller
     ett reproducerbart fel i imagen.
   - Om reproducerbart: buggen fixas, imagen byggs om som rc2 och alla 3 enheter
     flashas och körs igen.

6. IT-säkerhet signerar utfallet skriftligen i GA-gate-PR:en.

---

### US-27 — v0.1.0 GA release: tagg, GitHub Release och image-upload

**Som** produktteam
**vill jag** tagga v0.1.0 och publicera en officiell GitHub Release med den
signerade imagen
**så att** v0.1.0 GA är officiellt tillgänglig för allmänheten.

**Prioritet:** Hög — sprintens slutmål
**Estimat:** 0,5 dag
**Ansvarig:** DevOps (teknisk körning), PM (koordinering av GA-gate)
**IT-säk review:** Implicit via GA-gate (se nedan)

**Acceptanskriterier:**

1. **GA-gate är passerad** (se separat sektion nedan). US-27 får inte påbörjas
   förrän CTO och IT-säkerhet har gett skriftlig sign-off i GA-gate-PR:en.

2. **Ny image-build från taggad commit:**
   - En ny `release-image.yml`-körning triggas från den taggade commiten (inte
     från rc1-imagen som använts under testning).
   - Smoke-sviten körs automatiskt som en del av image-build-pipelinen och
     måste vara grön.
   - Imagen verifieras: SHA256 och minisign-signatur stämmer.

3. **Git-tagg:**
   ```
   git tag -s v0.1.0 -m "v0.1.0 GA"
   git push origin v0.1.0
   ```
   Taggen signeras med projektets GPG-nyckel om en sådan finns; annars annoterad
   tagg utan signering (notera avsaknaden i RELEASE_NOTES som känd begränsning
   om relevant).

4. **GitHub Release skapas** med:
   - Titel: `v0.1.0`
   - Body: innehållet från `RELEASE_NOTES.md` (v0.1.0-avsnittet).
   - Bifogade filer: `.img.xz`, `.img.xz.sha256`, `.img.xz.minisig`.

5. **README.md uppdateras** med "Latest release: v0.1.0" och länk till GitHub Release.

6. **Fira.** Sprint 4 är klar.

---

## GA-gate — Exakt vad CTO och IT-säkerhet måste sign-off:a

GA-gate är en dedikerad PR (`release/v0.1.0-ga`) som öppnas av PM när alla
US-19 till US-26 är klara. Ingen kod mergas i denna PR — den fungerar som ett
samlingsdokument för all sign-off.

**PR:en ska innehålla:**

1. Länk till soak-logg och RAM-graf från US-19.
2. Sammanfattning av beta-feedback och åtgärdslista från US-20.
3. Testresultat från WAL-konsistens och backup/restore (US-21).
4. Länk till `RELEASE_NOTES.md` (US-22) och `SECURITY.md` (US-23) i mergebart skick.
5. Smoke-testresultat-tabellen från US-26 (alla 3 enheter).
6. IT-säks skriftliga utfall av M-04-granskningen (US-25).

**CTO sign-off — krävs på:**

| Krav | Story | Verifiering |
|---|---|---|
| 7-dagars soak utan omstart, ingen minnesläcka | US-19 | RAM-graf + soak-rapport i PR |
| Minst 2 av 3 (eller 3 av 5) beta-testare lyckades | US-20 | Beta-sammanfattning i PR |
| WAL-konsistens klarar 3 strömavbrott | US-21 | Testresultat i PR |
| RELEASE_NOTES är korrekt och fullständig | US-22 | CTO läser och godkänner |
| Cert-renewal-path verifierad eller känd begränsning dokumenterad | US-19 §5 | Soak-rapport |

**IT-säkerhet sign-off — krävs på:**

| Krav | Story | Verifiering |
|---|---|---|
| M-05 dokumenterat i SECURITY.md med korrekt riskbedömning | US-23 | PR-kommentar |
| M-04 verifierat: MANIFEST-generering godkänd | US-25 | PR-kommentar |
| Smoke-sviten (inkl. TestHTTPSRedirect) grön på 3 Pi 3-enheter | US-24 + US-26 | Tabell i PR |

**Sign-off-format:**

CTO och IT-säkerhet skriver en kommentar i GA-gate-PR:en med exakt:
```
GA APPROVED — [CTO/IT-säk] — [datum]
Alla krav i GA-gate är uppfyllda. OK att tagga v0.1.0.
```

US-27 får inte påbörjas förrän BÅDA kommentarerna finns i PR:en.

---

## Sprint-backlog sammanfattning

| ID    | Story                                              | Prioritet | Estimat       | Ansvarig                  | IT-säk |
|-------|----------------------------------------------------|-----------|---------------|---------------------------|--------|
| US-19 | 7-dagars soak-test (CTO kör, Pi 3)                 | Hög       | 7 d realtid   | CTO + DevOps (setup)      | Nej    |
| US-20 | Beta-testning, 3–5 icke-tekniska användare         | Hög       | 3 d + 9 d     | PM + Go-dev (standby)     | Nej    |
| US-21 | WAL-konsistens och backup/restore                  | Hög       | 1,5 dagar     | Senior Go-dev             | Nej    |
| US-22 | RELEASE_NOTES.md                                   | Hög       | 1 dag         | PM + Go-dev (faktagranskning) | Nej |
| US-23 | SECURITY.md (M-05 dokumenterat)                    | Hög       | 0,5 dag       | Go-dev + PM               | Ja     |
| US-24 | TestHTTPSRedirect i smoke-suite                    | Hög       | 0,5 dag       | Senior Go-dev             | Nej    |
| US-25 | M-04 formell granskning av MANIFEST-generering     | Hög       | 0,5 dag       | IT-säk + DevOps (svar)    | Ja*    |
| US-26 | Smoke-tester på 3 Pi 3-enheter, olika SD-kort      | Hög       | 2 dagar       | DevOps + IT-säk (verify)  | Ja     |
| US-27 | v0.1.0 GA release (tagg, GitHub Release, upload)   | Hög       | 0,5 dag       | DevOps + PM               | Implicit** |

*US-25 är i sig en IT-säk-granskningsuppgift.
**US-27 kräver att GA-gate (CTO + IT-säk) är godkänd.

**Tidplan:**
- Dag 1–2: US-20 rekrytering, US-24 implementeras, US-23 skrivs, soak-setup (DevOps)
- Dag 1–7: US-19 soak körs (CTO, autonomt)
- Dag 3–9: US-20 beta-körning (testare installerar självständigt)
- Dag 3–5: US-21 WAL-test, US-22 skrivs
- Dag 5–6: US-25 IT-säk granskar MANIFEST
- Dag 6–8: US-26 tre Pi-körningar
- Dag 8–10: US-20 feedback sammanfattas, buggfixar vid behov
- Dag 10–14: US-23 IT-säk review, eventuella rc2-fixar
- Dag 14: GA-gate PR öppnas
- Dag 15–16: CTO och IT-säk sign-off
- Dag 17–18: US-27 GA release
- Dag 19–21: Buffert (buggfixar, IT-säk-kö, image-rebuild)

---

## Definition of Done (Sprint 4)

Sprinten är klar när ALLA punkter nedan är uppfyllda:

1. US-19 (soak): 7 dagar avklarade utan omstart. RAM-graf bilagd GA-gate-PR:en.
   CTO:s soak-rapport skriven.

2. US-20 (beta): Minst 2 av 3 (eller 3 av 5) beta-testare lyckades installera
   självständigt. Alla showstoppers åtgärdade. Beta-sammanfattning bilagd GA-gate-PR:en.

3. US-21 (WAL): 3 av 3 strömavbrottstester passerade. Backup/restore-flöde
   dokumenterat och testat.

4. US-22 (RELEASE_NOTES): Dokumentet finns i repots rot, granskat av CTO,
   innehåller alla kända begränsningar.

5. US-23 (SECURITY.md): Dokumentet finns i repots rot, IT-säk har sign-off:at
   M-05-formuleringen.

6. US-24 (TestHTTPSRedirect): Testet är mergedat till `tests/smoke/` och ingår
   i CI-smoke-jobbet. Kan misslyckas demonstrerat.

7. US-25 (M-04): IT-säk har skriftligen godkänt MANIFEST-generering i GA-gate-PR:en.

8. US-26 (3 enheter): Smoke-sviten grön på alla 3 Pi 3-enheter. Tabell bilagd
   GA-gate-PR:en. IT-säk har sign-off:at.

9. GA-gate-PR: Öppnad, alla 8 punkter ovan dokumenterade, CTO och IT-säk har
   skrivit "GA APPROVED" med datum.

10. US-27 (GA release): v0.1.0-taggen finns på GitHub. GitHub Release publicerad
    med `.img.xz`, `.img.xz.sha256` och `.img.xz.minisig`. README uppdaterad.

11. Inga `[BLOCK]`-kommentarer i öppna PR-reviews.

12. SPRINT-4.md (denna fil) är uppdaterad om scope förändrats under sprinten (PM-ansvar).

---

## Blockers att bevaka

**BB-12 — Soak-perioden styr hela tidplanen**
Om soak-perioden (US-19) måste börja om p.g.a. krasch eller omstart skjuts
GA-gate med samma antal dagar. Det finns ingen workaround — CTO-kravet är 7
sammanhängande dagar. PM håller koll på soak-status dagligen under dag 1–7
och eskalerar omedelbart om något händer.
Mitigation: DevOps sätter upp extern monitoring (t.ex. ett enkelt ping-skript
på CTO:s dator) som skickar notis om piholster.local slutar svara.

**BB-13 — Rekrytering av beta-testare (US-20)**
3–5 icke-tekniska testare måste rekryteras dag 1–2. Om PM inte hittar tillräckligt
med testare: eskalera till CTO dag 2. Alternativ: familjemedlemmar till teammedlemmar,
kollegor på andra avdelningar, vänner till PM.
Om vi inte hittar 3 testare: CTO beslutar om vi kan gå vidare med 2 testare
(notera undantaget i GA-gate-PR:en) eller om GA skjuts.

**BB-14 — Tillgänglighet av 3 fysiska Pi 3-enheter (US-26)**
Vi behöver 3 Pi 3-enheter och 3 olika SD-kort. Om enhet B eller C saknas dag 1:
PM eskalerar till CTO som avgör om en Pi 4 kan godtas som ersättning för en av
enheterna (Pi 4 är inte identisk hårdvara men acceptabel som pragmatisk lösning).
DevOps bekräftar hårdvaruinventering dag 1.

**BB-15 — IT-säks tillgänglighet för US-23 och US-25**
Båda uppgifterna kräver IT-säk. PM bokar ett dedikerat review-fönster dag 5–8.
Om IT-säk är otillgänglig: eskalera till CTO dag 5. GA-gate kan inte hållas
dag 14 om IT-säk-granskning inte är klar dag 10.

**BB-16 — Buggfixar från soak/beta kan kräva image-rebuild**
Om soak eller beta avslöjar ett kritiskt fel som kräver kodändring: imagen måste
byggas om som rc2, och US-26 (smoke-tester på 3 enheter) måste köras igen
med den nya imagen. Detta kan förlänga sprinten med 2–3 dagar.
PM och CTO beslutar tillsammans om buggfixar är showstoppers (kräver rc2) eller
kan parkeras i v0.1.1 och noteras i RELEASE_NOTES.
