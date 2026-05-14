import requests
sources = [
    "https://www.worldfootball.net/table/swe-allsvenskan-2024/",
    "https://www.svenskfotboll.se/serier-cuper/tabell/allsvenskan-herr/115477/",
    "https://www.fotbollskanalen.se/allsvenskan/tabell/",
    "https://www.aftonbladet.se/sportbladet/fotboll/sverige/allsvenskan/tabell",
    "https://www.soccerway.com/national/sweden/allsvenskan/2024/regular-season/r79958/tables/",
    "https://www.theguardian.com/football/allsvenskan/table"
]
for s in sources:
    try:
        r = requests.get(s, headers={"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"}, timeout=5)
        print(f"{s}: {r.status_code}, {len(r.text)} bytes, {'Malm' in r.text}")
    except Exception as e:
        print(f"{s}: Error {e}")
