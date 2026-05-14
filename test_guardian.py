import requests
url = "https://www.theguardian.com/football/allsvenskan/table"
r = requests.get(url, headers={"User-Agent": "Mozilla/5.0"})
print(f"Status: {r.status_code}")
with open("guardian.html", "w") as f:
    f.write(r.text)
if "Malm" in r.text:
    print("Found Malm")
else:
    print("Not found")
