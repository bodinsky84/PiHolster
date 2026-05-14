import requests
# Try to find Allsvenskan in SR API
url = "https://api.sr.se/api/v2/matchdata/series?format=json"
r = requests.get(url)
data = r.json()
for series in data.get('series', []):
    print(f"{series.get('id')}: {series.get('name')}")
