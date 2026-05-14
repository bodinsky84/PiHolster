import requests
import re

headers = {"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"}

# Try to find a working table page
urls = [
    "https://www.everysport.com/fotboll-herr/allsvenskan/77962", # This might be 2024
    "https://www.soccerway.com/national/sweden/allsvenskan/2026/regular-season/r81000/tables/", # Guessing ID
    "https://www.worldfootball.net/table/swe-allsvenskan-2026/",
    "https://www.transfermarkt.com/allsvenskan/tabelle/wettbewerb/SE1/saison_id/2025" # 2025 season ends in 2025, but maybe they mean 2026
]

for url in urls:
    print(f"Checking {url}...")
    try:
        r = requests.get(url, headers=headers, timeout=10)
        print(f"Status: {r.status_code}")
        if "Malm" in r.text:
            print("Found Malm!")
            # Extract teams
            teams = re.findall(r'title="([^"]+)"', r.text)
            print(f"Possible teams: {set(teams[:50])}")
            break
    except:
        pass
