# ADR-002 Tillägg — BB-07: CSP Nonce-injection med adapter-static

**Status:** Accepterad  
**Datum:** 2026-05-03  
**Blocker:** BB-07  
**Ersätter delvis:** ADR-002 §3.4.1 (komplettering, ej upphävning)

## Kontext

ADR-002 §3.4.1 kräver CSP utan `'unsafe-inline'` med per-request nonce.
SvelteKit `adapter-static` genererar statiska filer — inbyggt nonce-stöd kräver SSR
(adapter-node), vilket bryter ADR-001 §6.2 (single binary, go:embed, ingen Node runtime).

## Alternativ som övervägdes

| Alt | Beskrivning                                      | Förkastningsorsak                              |
|-----|--------------------------------------------------|------------------------------------------------|
| A   | Byt till adapter-node (SSR)                      | Bryter ADR-001 §6.2, kräver Node.js på Pi     |
| B   | Post-process index.html med nonce-platshållare   | Brittleness vid SvelteKit-uppgraderingar       |
| C   | Acceptera unsafe-inline på style-src till v1.0   | Reservalternativ — se nedan                    |
| D   | inlineStyleThreshold: 0, eliminera inline styles | Valt — se beslut                               |

## Beslut: Alternativ D

Sätt `vitePlugin: { config: { build: { ... } } }` och `inlineStyleThreshold: 0` i
`svelte.config.js`. SvelteKit genererar då inga inline `<style>`-taggar i HTML-output.
`script-src` och `style-src` kan vara fullt strikta utan `'unsafe-inline'` eller nonce.

ADR-002 §3.4.1 uppfylls utan dispens. Ingen post-processing krävs.

## Konsekvenser

- Prestanda: En extra CSS-fil per sida. På lokal LAN-trafik (Pi 3, privat IP) är
  RTT-kostnaden negligerbar (< 2 ms). Omprövas om extern åtkomst tillkommer i v1.1.
- Underhåll: Ingen nonce-logik i Go-servern, inga regex på HTML-output.
- Reserv: Om specifik komponent strukturellt kräver inline styles eskaleras till
  Alternativ C (unsafe-inline på style-src, ej script-src) med nytt ADR-tillägg.

## Åtgärd

- [ ] Sätt `inlineStyleThreshold: 0` i `svelte.config.js`
- [ ] Verifiera CSP-headers i Go-server returnerar strikt policy utan unsafe-inline
- [ ] Stäng BB-07 efter grön byggverifiering
