# Beta-feedback v0.1.0-rc1 (US-20)

PM använder detta dokument för:
1. **Inbjudan** — kopiera "Inbjudan till beta-testare" nedan och skicka till de 3–5
   rekryterade testarna tillsammans med länk till `.img.xz` och `docs/INSTALL-PI.md`.
2. **Feedbackformulär** — testaren fyller i sektionen "Feedback från testaren"
   och skickar tillbaka.
3. **Sammanställning** — PM sammanfattar alla svar i sektionen "PM-sammanställning"
   och bifogar dokumentet till GA-gate-PR:en.

Sprint 4 är RC-validering. Inga nya funktioner byggs som svar på beta-feedback —
problem klassificeras antingen som showstopper (måste fixas före GA), bör fixas
(rc2 om tid finns, annars noteras i `RELEASE_NOTES.md`) eller v0.1.1.

---

## Inbjudan till beta-testare

> Hej!
>
> Tack för att du vill testa PiHolster v0.1.0-rc1. PiHolster är en
> reklam- och spårningsblockerare som körs på en Raspberry Pi i ditt
> hemnätverk.
>
> Vi behöver din feedback för att se om installationsguiden är tillräckligt
> tydlig för någon som inte har Linux-vana. Försök att installera helt
> själv — skriv ner allt som var oklart, även små saker.
>
> **Det här behöver du:**
> - En Raspberry Pi 3 eller 4
> - Ett microSD-kort (8 GB eller större)
> - En kabel mellan Pi:n och routern
>
> **Det här ska du göra:**
> 1. Ladda ner image: `<länk till v0.1.0-rc1.img.xz>`
> 2. Följ guiden: `<länk till docs/INSTALL-PI.md>`
> 3. När du är klar (eller om du fastnar): fyll i formuläret nedan och
>    skicka tillbaka till `<PM:s e-postadress / Signal-nummer>`.
>
> **Showstopper-kontakt:** Om något är *helt* trasigt (image bootar inte,
> Pi:n är låst, du har gjort något allvarligt fel) — kontakta
> `<PM eller jourkontakt>`. För allt annat — bita ihop och skriv ner
> upplevelsen, det är precis det vi behöver.
>
> Tidsåtgång: ca 1 timme inklusive flash. Tack!

---

## Feedback från testaren

**Testarens namn:** ___________________________
**E-post / Signal:** ___________________________
**Datum:** YYYY-MM-DD
**Pi-modell:** Pi 3 / Pi 4 / annat: ___________

---

**1. Lyckades du installera PiHolster?**

- [ ] Ja, helt — kom hela vägen till inloggad webbsida
- [ ] Delvis — kom igenom flash men fastnade senare
- [ ] Nej — fastnade redan vid flash eller boot
- [ ] Avbröt — varför: ___________________________

**2. Vilket steg var svårast eller mest oklart?**

```
(skriv fritt — ett ord, en mening eller en hel paragraf — allt hjälper)
```

**3. Fick du certifikatvarningen i webbläsaren när du försökte logga in?**

- [ ] Ja, och jag förstod vad jag skulle göra (klicka "Avancerat → Fortsätt")
- [ ] Ja, men jag förstod inte — det kändes farligt
- [ ] Nej, jag fick aldrig se webbsidan
- [ ] Vet inte / kommer inte ihåg

**4. Lyckades du peka routerns DNS på Pi:n?**

- [ ] Ja, och blocket fungerar (reklam försvinner)
- [ ] Ja, men jag är osäker på om det fungerar
- [ ] Nej, hittade inte inställningen
- [ ] Nej, vågade inte ändra
- [ ] Hoppade över detta steg

**5. Hur lång tid tog hela installationen från start till inloggad webbsida?**

- [ ] Mindre än 30 minuter
- [ ] 30–60 minuter
- [ ] 1–2 timmar
- [ ] Mer än 2 timmar
- [ ] Hann inte slutföra

**6. Hur skulle du beskriva svårighetsgraden?**

- [ ] Lätt — guiden räckte
- [ ] Mellan — gick att lösa men krävde en del klurande
- [ ] Svår — jag är osäker på vad jag gjorde rätt och fel
- [ ] För svår — jag skulle inte våga göra detta utan hjälp

**7. Var det något du behövde googla eller fråga någon om?**

```
(t.ex. "vad betyder SSH", "hur ser man Pi:ns IP", "hur loggar man in på routern")
```

**8. Skulle du rekommendera PiHolster till en vän som inte är teknisk?**

- [ ] Ja, det går bra
- [ ] Ja, men bara om jag kan hjälpa till
- [ ] Nej, det är för svårt i nuläget
- [ ] Vet inte

**9. Är det något du saknade i installationsguiden?**

```
(t.ex. en bild på "Avancerade inställningar", en checklista för routern, …)
```

**10. Övrig feedback eller förslag?**

```
(allt går bra — beröm, klagomål, förslag, frågor)
```

---

## PM-sammanställning

PM fyller i denna sektion när alla testare har skickat in formuläret.
Bifogas GA-gate-PR:en.

### Översikt

| | Antal |
|---|---|
| Rekryterade testare | |
| Inkomna svar | |
| Lyckade installationer (fråga 1 = "Ja, helt") | |
| Delvis lyckade (fråga 1 = "Delvis") | |
| Misslyckade / avbrutna | |

**GA-tröskel (US-20 AC-5):** Minst 2 av 3, eller minst 3 av 5, ska ha lyckats
helt. Om färre — beslut tillsammans med CTO om INSTALL-PI.md behöver revideras
innan GA, eller om GA skjuts.

### Identifierade problem

Lista varje unikt problem som minst en testare flaggade. Klassificera enligt:

- **Showstopper** — blockerar GA. Måste fixas i rc2 innan GA-gate kan hållas.
- **Bör fixas** — försämrar upplevelsen. Tas in i rc2 om tid finns, annars noteras
  i `RELEASE_NOTES.md` som känd begränsning.
- **v0.1.1** — parkeras i backloggen, inte GA-blocker.

| # | Problem | Antal testare som flaggade | Klassificering | Beslut |
|---|---|---|---|---|
| 1 | t.ex. "Hittade inte routerns DHCP-inställning" | 3/5 | Bör fixas | Lägg till routergude-länkar i INSTALL-PI |
| 2 | … | | | |

### Showstoppers — status

| # | Problem | Åtgärd | Klar (Y/N) | rc2-PR # |
|---|---|---|---|---|
| | | | | |

Alla showstoppers måste vara markerade `Klar=Y` med länk till mergead PR innan
GA-gate kan hållas.

### PM-slutsats

> 3–5 meningar: räcker INSTALL-PI.md för en icke-teknisk användare?
> Vilka mönster syntes i feedbacken? Är vi redo för GA?
> Behövs revidering av INSTALL-PI.md innan GA-tagg?

**Beslut:** GA-redo / GA-blockerat tills [åtgärd]
**PM-signatur:** _____________________ (datum: YYYY-MM-DD)
