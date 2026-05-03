# Installationsguide för PiHolster

Den här guiden hjälper dig att installera PiHolster steg för steg. Du behöver inte ha någon teknisk bakgrund — följ bara stegen i ordning så kommer du igång.

---

## Vad du behöver

Innan du börjar, se till att du har följande:

- **Raspberry Pi 3, 3B, 3B+ eller nyare** — Modell A fungerar tyvärr inte eftersom den saknar Ethernet-port (den plats där du kopplar in en nätverkskabel).
- **MicroSD-kort, minst 8 GB** — Det lilla minneskort som fungerar som hårddisk i Pi:n.
- **Ethernetkabel** — Den vanliga nätverkskabeln med en liten plastkontakt i varje ände. WiFi stöds inte i den här versionen.
- **En dator** — Windows, Mac eller Linux fungerar alla bra. Du behöver den för att förbereda SD-kortet.

---

## Steg 1 — Ladda ner PiHolster

1. Gå till **https://github.com/piholster/piholster/releases/latest** i din webbläsare.
2. Hitta den senaste versionen och ladda ner filen som slutar på `.img.xz`. Det är programfilen som ska hamna på SD-kortet.

**Vill du vara extra säker? (valfritt)**

Bredvid nedladdningslänken finns ofta en lång rad med bokstäver och siffror — ett så kallat SHA256-kontrollvärde. Det är ett slags fingeravtryck för filen. Om du jämför detta värde med det som visas på din dator efter nedladdningen kan du vara säker på att filen inte har skadats på vägen. Det är frivilligt men kan vara bra om du vill vara extra noggrann.

---

## Steg 2 — Förbered SD-kortet

Du ska nu kopiera PiHolster till SD-kortet. Det enklaste sättet är med ett gratis program som heter **Raspberry Pi Imager**.

1. Ladda ner och installera Raspberry Pi Imager från **https://www.raspberrypi.com/software/**.
2. Sätt i SD-kortet i din dator (du kan behöva en adapter om din dator inte har en SD-kortplats).
3. Starta Raspberry Pi Imager.
4. Klicka på **"Välj OS"** och välj sedan **"Använd anpassad bild"** längst ned i listan.
5. Leta upp den `.img.xz`-fil du laddade ner i Steg 1 och välj den.
6. Klicka på **"Välj lagringsenhet"** och välj ditt SD-kort. Se till att du väljer rätt — allt på det valda kortet raderas.
7. Klicka på **"Skriv"** och vänta tills programmet är klart. Det kan ta några minuter.

**Tekniskt alternativ (för den som vet vad det innebär):**

```
xzcat piholster.img.xz | dd of=/dev/sdX bs=4M
```

Byt ut `/dev/sdX` mot rätt enhetsbeteckning för ditt SD-kort.

---

## Steg 3 — Starta PiHolster

Nu är det dags att sätta igång Pi:n.

1. Ta ut SD-kortet från datorn och sätt i det i Raspberry Pi:n — kortplatsen sitter på undersidan av Pi:n.
2. Koppla Ethernetkabeln mellan Pi:n och din router (den låda som ger dig internet hemma).
3. Koppla in strömkabeln i Pi:n. Det finns ingen strömknapp — den startar direkt.
4. Vänta tills den **gröna lampan lyser stadigt**. Det tar ungefär 60–90 sekunder. Under uppstarten blinkar lampan oregelbundet — det är normalt.

---

## Steg 4 — Öppna PiHolster

Nu ska du komma åt PiHolster via din webbläsare, precis som du öppnar en vanlig hemsida.

1. Öppna din webbläsare (till exempel Chrome, Firefox eller Safari).
2. Skriv in adressen **https://piholster.local** i adressfältet och tryck Enter.

**Du kommer troligtvis att se en säkerhetsvarning.** Det ser lite skrämmande ut men är helt normalt — förklaringen finns i säkerhetsavsnittet längre ned. Så här klickar du förbi den:

- **I Chrome:** Klicka på **"Avancerat"** och sedan på **"Fortsätt till piholster.local (osäkert)"**.
- **I Safari:** Klicka på **"Visa detaljer"** och sedan på **"besök den här webbplatsen"**. Bekräfta i rutan som dyker upp.

**Om piholster.local inte fungerar:**

- Prova **https://piholster.lan** istället.
- Fungerar inte det heller, logga in på din router och leta efter en lista över anslutna enheter. Pi:n heter troligtvis "piholster" i den listan. Bredvid namnet ser du en sifferserie som ser ut ungefär som `192.168.1.42` — det är Pi:ns IP-adress (IP-adress är ett slags hemadress på ditt nätverk). Skriv in den adressen i webbläsaren: **https://192.168.1.42**

---

## Steg 5 — Logga in och byt lösenord

1. Du möts av en inloggningssida. Klicka på **Admin-menyn** och leta efter rubriken **"Startlösenord"** — det tillfälliga lösenordet visas där.
2. Skriv in lösenordet och logga in.
3. Byt lösenord direkt. Det tillfälliga lösenordet försvinner nästa gång Pi:n startas om.
4. Välj ett lösenord som du kommer ihåg, minst 8 tecken. Skriv upp det på ett säkert ställe om du är osäker på att du kommer ihåg det.

---

## Steg 6 — Peka nätverket till PiHolster

Det sista steget är att tala om för ditt nätverk att det ska använda PiHolster som DNS-server. DNS är en tjänst som översätter webbadresser (som `www.google.com`) till siffror som datorn förstår — och PiHolster filtrerar bort reklamannonser i det steget.

Du behöver Pi:ns IP-adress (se Steg 4 ovan om du inte noterat den).

**Ändra i routern (rekommenderas — gäller hela hemmet):**

1. Logga in på din router. Adressen är vanligtvis **http://192.168.1.1** eller **http://192.168.0.1** — titta på en etikett på undersidan av routern om du är osäker.
2. Leta efter inställningar för **DNS** eller **DHCP**. Exakt var det finns varierar beroende på routermodell — konsulta routerns manual eller tillverkarens webbplats om du inte hittar det.
3. Ange Pi:ns IP-adress som DNS-server.
4. Spara inställningen och starta om routern om det krävs.

**Vill du bara testa på din egen dator eller mobil först?**

Du kan ändra DNS-inställningen bara på en enskild enhet. Sök efter "ändra DNS Windows 11", "ändra DNS iPhone" eller motsvarande för din enhet för att hitta en guide anpassad för just ditt system.

---

## Felsökning

### 1. Lampan slutar blinka men lyser inte stadigt

SD-kortet sitter troligtvis inte ordentligt. Dra ut strömkabeln, ta ut och sätt tillbaka SD-kortet med ett bestämt tryck tills det klickar på plats, och koppla in strömkabeln igen.

### 2. Webbläsaren hittar inte piholster.local

Kontrollera att Ethernetkabeln är ordentligt inkopplad i både Pi:n och routern. WiFi stöds inte — kabeln måste användas. Prova också att starta om Pi:n genom att dra ut och sätta tillbaka strömkabeln.

### 3. Säkerhetsvarningen ser annorlunda ut än beskrivet

Det är normalt att varningens utseende varierar lite beroende på webbläsare och version. Det du letar efter är ett alternativ som heter "Avancerat", "Detaljer" eller liknande, följt av ett sätt att ändå fortsätta till sidan. Se säkerhetsavsnittet nedan för en förklaring till varför varningen dyker upp.

### 4. Jag har glömt lösenordet

Starta om Pi:n (dra ut och sätt tillbaka strömkabeln). Det tillfälliga startlösenordet är aktivt i 24 timmar efter omstart och visas i Admin-menyn under "Startlösenord". Logga in med det och byt sedan lösenord direkt.

### 5. DNS fungerar men jag ser fortfarande annonser

Vänta 5 minuter och ladda om sidan. Din dator och webbläsare sparar DNS-svar en stund för snabbare surfning — den lagrade informationen (cachen) måste hinna gå ut. Rensar du webbläsarens cache manuellt går det snabbare.

---

## Säkerhet

### Varför du alltid måste byta lösenord

Det tillfälliga startlösenordet är detsamma för alla som installerar PiHolster. Om du inte byter det kan vem som helst som är ansluten till ditt nätverk logga in och ändra dina inställningar. Byt alltid lösenord i Steg 5 innan du gör något annat.

### Vad är det där certifikatet egentligen?

När du surfar till `https://piholster.local` använder anslutningen kryptering för att skydda din trafik. Normalt bekräftar ett välkänt företag (ett certifikatutfärdare) att en sajt är äkta. PiHolster använder ett eget, så kallat **självsignerat certifikat** — det fungerar precis lika bra för kryptering, men ingen extern part har bekräftat det. Därför varnar webbläsaren. Eftersom du vet att du ansluter till din egen Pi hemma är det tryggt att klicka förbi varningen.

### Vad lagras och vart skickas det?

PiHolster sparar DNS-loggar — alltså en lista över vilka adresser som efterfrågats på ditt nätverk. Dessa loggar stannar på SD-kortet i din Pi och skickas **inte** till internet eller till molnet. Du kan se och rensa loggarna via Admin-menyn när du vill.

### Hur uppdaterar jag PiHolster?

En funktion för automatiska uppdateringar är under utveckling. Tills dess: håll ett öga på **https://github.com/piholster/piholster/releases** — när en ny version dyker upp upprepar du Steg 1–3 med den nya `.img.xz`-filen.

---

*Har du fastnat? Skapa ett ärende på GitHub: https://github.com/piholster/piholster/issues*
