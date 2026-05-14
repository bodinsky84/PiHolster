<?php
/**
 * Allsvenskan Pro Hub - Advanced Player Intel & Spiders
 * Includes: Market Value, Acquisition Fees, Live News & Standings
 */

function fetch_data($url) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36');
    curl_setopt($ch, CURLOPT_TIMEOUT, 15);
    $data = curl_exec($ch);
    curl_close($ch);
    return $data;
}

$team_map = [
    "AIK" => 271, "BK Häcken" => 1109, "Djurgården" => 1044, "GAIS" => 369,
    "Halmstads BK" => 1011, "Hammarby" => 1059, "IF Brommapojkarna" => 1092,
    "IF Elfsborg" => 1065, "IFK Göteborg" => 429, "IFK Norrköping" => 703,
    "IFK Värnamo" => 2772, "IK Sirius" => 7945, "Kalmar FF" => 1473,
    "Malmö FF" => 496, "Mjällby AIF" => 2719, "Västerås SK" => 1108
];

$followed_team = isset($_GET['team']) ? $_GET['team'] : '';

// 1. SPIDER: Market Values (Top 25)
$market_url = "https://www.transfermarkt.com/allsvenskan/marktwerte/wettbewerb/SE1";
$market_html = fetch_data($market_url);
$top_players = [];
if ($market_html) {
    preg_match_all('/<td class="hauptlink">.*?<a title="(.*?)" href="(.*?)">.*?<\/a>.*?<\/td>.*?<a title="(.*?)" href=".*?">.*?<\/a>.*?<td class="rechts hauptlink"><a.*?>(.*?)<\/a>/s', $market_html, $matches, PREG_SET_ORDER);
    foreach ($matches as $m) {
        $top_players[] = [
            'name' => trim($m[1]),
            'url'  => "https://www.transfermarkt.com" . $m[2],
            'team' => trim($m[3]),
            'value' => trim($m[4])
        ];
        if (count($top_players) >= 25) break;
    }
}

// 2. SPIDER: Squad Intel (Market Value + Est. Purchase Fee)
$squad = [];
if ($followed_team && isset($team_map[$followed_team])) {
    $id = $team_map[$followed_team];
    $squad_url = "https://www.transfermarkt.com/team/startseite/verein/$id";
    $squad_html = fetch_data($squad_url);
    if ($squad_html) {
        // Extract player info including joined-from fee if available in title tags
        preg_match_all('/<td class="posrela">.*?<a title="(.*?)".*?<\/td>.*?<td class="hauptlink">.*?<a href="(.*?)">(.*?)<\/a>.*?<td class="rechts hauptlink">(.*?)<\/td>/s', $squad_html, $matches, PREG_SET_ORDER);
        foreach ($matches as $m) {
            $intel = $m[1]; // Joined from... fee: ...
            $fee = "Okänd";
            if (preg_match('/fee: (.*?)$/', $intel, $fee_match)) {
                $fee = $fee_match[1];
            }

            $squad[] = [
                'name'  => trim($m[3]),
                'url'   => "https://www.transfermarkt.com" . $m[2],
                'value' => trim($m[4]),
                'intel' => $intel,
                'fee'   => $fee
            ];
        }
    }
}

// 3. SPIDER: Live News
$feeds = ['Allsvenskan' => "https://allsvenskan.se/feed/", 'Expressen' => "https://www.expressen.se/rss/sport/fotboll/allsvenskan/"];
$news = [];
foreach ($feeds as $source => $url) {
    $xml_data = fetch_data($url);
    if ($xml_data) {
        $xml = @simplexml_load_string($xml_data);
        if ($xml) {
            foreach ($xml->channel->item as $item) {
                $news[] = [
                    'source' => $source,
                    'title' => (string)$item->title,
                    'link' => (string)$item->link,
                    'desc' => strip_tags((string)$item->description),
                    'timestamp' => strtotime((string)$item->pubDate),
                    'date' => date('j M, H:i', strtotime((string)$item->pubDate))
                ];
            }
        }
    }
}
usort($news, function($a, $b) { return $b['timestamp'] - $a['timestamp']; });
$news = array_slice($news, 0, 15);

// Standings
$table = [
    ['pos'=>1, 'team'=>'Malmö FF', 'p'=>30, 'pts'=>65],
    ['pos'=>2, 'team'=>'Hammarby', 'p'=>30, 'pts'=>54],
    ['pos'=>3, 'team'=>'AIK', 'p'=>30, 'pts'=>54],
    ['pos'=>4, 'team'=>'Djurgården', 'p'=>30, 'pts'=>53],
    ['pos'=>5, 'team'=>'Mjällby AIF', 'p'=>30, 'pts'=>50],
    ['pos'=>6, 'team'=>'GAIS', 'p'=>30, 'pts'=>48],
    ['pos'=>7, 'team'=>'IF Elfsborg', 'p'=>30, 'pts'=>45],
    ['pos'=>8, 'team'=>'BK Häcken', 'p'=>30, 'pts'=>42]
];

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Allsvenskan Intel Hub</title>
    <style>
        :root { --bg: #0f172a; --card: #1e293b; --text: #f8fafc; --accent: #38bdf8; --border: #334155; --gold: #f59e0b; --green: #10b981; }
        body { background: var(--bg); color: var(--text); font-family: 'Inter', system-ui, sans-serif; margin: 0; padding: 10px; }
        .container { max-width: 1400px; margin: 0 auto; }
        header { background: #020617; padding: 25px; border-radius: 16px; margin-bottom: 25px; display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; border: 1px solid var(--border); }
        h1 { margin: 0; font-size: 2rem; background: linear-gradient(to right, #38bdf8, #818cf8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
        select { background: #1e293b; color: #fff; border: 1px solid var(--accent); padding: 12px 20px; border-radius: 10px; font-size: 1rem; cursor: pointer; outline: none; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 20px; }
        @media (max-width: 800px) { .grid { grid-template-columns: 1fr; } }
        .card { background: var(--card); padding: 20px; border-radius: 16px; border: 1px solid var(--border); box-shadow: 0 10px 15px -3px rgba(0,0,0,0.3); }
        h2 { color: var(--accent); font-size: 1.2rem; margin-top: 0; display: flex; align-items: center; gap: 10px; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th { text-align: left; color: #94a3b8; font-size: 0.75rem; text-transform: uppercase; padding: 12px 5px; border-bottom: 2px solid var(--border); }
        td { padding: 14px 5px; border-bottom: 1px solid var(--border); font-size: 0.9rem; }
        .val { color: var(--green); font-weight: 700; }
        .fee { color: var(--gold); font-size: 0.8rem; }
        .news-item { margin-bottom: 15px; border-left: 4px solid var(--accent); padding-left: 15px; }
        .news-item a { text-decoration: none; color: inherit; }
        .news-item h3 { margin: 5px 0; font-size: 1.05rem; line-height: 1.4; }
        .source-tag { font-size: 0.65rem; background: #334155; padding: 2px 8px; border-radius: 50px; color: #38bdf8; }
        tr:hover { background: #2d3748; }
        .scroll { max-height: 600px; overflow-y: auto; padding-right: 5px; }
        ::-webkit-scrollbar { width: 6px; }
        ::-webkit-scrollbar-thumb { background: var(--border); border-radius: 10px; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>Allsvenskan Intel Hub</h1>
                <p style="color: #94a3b8; margin: 5px 0 0 0;">Pro Spiders: Live Marknadsvärden, Inköpspriser & Nyheter</p>
            </div>
            <form method="GET">
                <select name="team" onchange="this.form.submit()">
                    <option value="">Välj ett lag för truppanalys...</option>
                    <?php foreach($team_map as $name => $id): ?>
                        <option value="<?= $name ?>" <?= $followed_team == $name ? 'selected' : '' ?>><?= $name ?></option>
                    <?php endforeach; ?>
                </select>
            </form>
        </header>

        <div class="grid">
            <!-- 1. Dyrast i Ligan -->
            <div class="card">
                <h2>🏆 Mest värdefulla (Ligan)</h2>
                <div class="scroll">
                    <table>
                        <thead><tr><th>Spelare</th><th>Lag</th><th>Värde</th></tr></thead>
                        <tbody>
                            <?php foreach($top_players as $p): ?>
                            <tr>
                                <td><a href="<?= $p['url'] ?>" target="_blank" style="color:inherit; text-decoration:none;"><strong><?= $p['name'] ?></strong></a></td>
                                <td><small><?= $p['team'] ?></small></td>
                                <td class="val"><?= $p['value'] ?></td>
                            </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- 2. Lag Trupp med Inköpspris -->
            <div class="card">
                <?php if($followed_team): ?>
                    <h2>🛡️ Truppanalys: <?= $followed_team ?></h2>
                    <div class="scroll">
                        <table>
                            <thead><tr><th>Spelare</th><th>Värde</th><th>Inköpt för</th></tr></thead>
                            <tbody>
                                <?php foreach($squad as $s): ?>
                                <tr>
                                    <td><a href="<?= $s['url'] ?>" target="_blank" style="color:inherit; text-decoration:none;"><?= $s['name'] ?></a></td>
                                    <td class="val"><?= $s['value'] ?></td>
                                    <td class="fee"><?= $s['fee'] ?></td>
                                </tr>
                                <?php endforeach; ?>
                            </tbody>
                        </table>
                    </div>
                <?php else: ?>
                    <h2>📊 Tabell (Topp 8)</h2>
                    <table>
                        <thead><tr><th>#</th><th>Lag</th><th>S</th><th>P</th></tr></thead>
                        <tbody>
                            <?php foreach($table as $t): ?>
                            <tr><td><?= $t['pos'] ?></td><td><strong><?= $t['team'] ?></strong></td><td><?= $t['p'] ?></td><td class="val"><?= $t['pts'] ?></td></tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                    <p style="margin-top: 20px; color: #94a3b8; font-size: 0.9rem; text-align: center;">Välj ett lag ovan för att se detaljerad truppstatistik och inköpspriser.</p>
                <?php endif; ?>
            </div>

            <!-- 3. Nyhetsflöde -->
            <div class="card">
                <h2>📰 Senaste Intel</h2>
                <div class="scroll">
                    <?php
                    $count = 0;
                    foreach($news as $n):
                        if ($followed_team && strpos(strtolower($n['title'].$n['desc']), strtolower($followed_team)) === false) continue;
                        $count++;
                    ?>
                    <div class="news-item">
                        <div><span class="source-tag"><?= $n['source'] ?></span> <small style="color: #64748b;"><?= $n['date'] ?></small></div>
                        <a href="<?= $n['link'] ?>" target="_blank"><h3><?= $n['title'] ?></h3></a>
                        <p><?= substr($n['desc'], 0, 120) ?>...</p>
                    </div>
                    <?php endforeach; ?>
                    <?php if($count == 0) echo "<p>Inga specifika nyheter för tillfället.</p>"; ?>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
