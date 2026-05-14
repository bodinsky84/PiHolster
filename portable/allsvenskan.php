<?php
/**
 * Allsvenskan Pro Aggregator v4.0 - THE scouting hub
 * Optimized for One.com
 */

function fetch_data($url) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36');
    curl_setopt($ch, CURLOPT_TIMEOUT, 15);
    $data = curl_exec($ch);
    curl_close($ch);
    return $data;
}

// 1. REGISTRIES
$teams = [
    "AIK", "BK Häcken", "Djurgården", "GAIS", "Halmstads BK", "Hammarby",
    "IF Brommapojkarna", "IF Elfsborg", "IFK Göteborg", "IFK Norrköping",
    "IFK Värnamo", "IK Sirius", "Kalmar FF", "Malmö FF", "Mjällby AIF", "Västerås SK"
];

$followed_team = isset($_COOKIE['followed_team']) ? $_COOKIE['followed_team'] : '';
if (isset($_GET['team'])) {
    $followed_team = $_GET['team'];
    setcookie('followed_team', $followed_team, time() + (86400 * 30), "/");
}

// 2. NEWS SPIDER
$news_sources = [
    'Allsvenskan.se' => 'https://allsvenskan.se/feed/',
    'Expressen' => 'https://www.expressen.se/rss/sport/fotboll/allsvenskan/',
    'Fotbollskanalen' => 'https://www.fotbollskanalen.se/allsvenskan/rss'
];
$all_news = [];
foreach ($news_sources as $src => $url) {
    $xml = @simplexml_load_string(fetch_data($url));
    if ($xml && isset($xml->channel->item)) {
        foreach ($xml->channel->item as $item) {
            $all_news[] = [
                'source' => $src,
                'title' => (string)$item->title,
                'link' => (string)$item->link,
                'desc' => strip_tags((string)$item->description),
                'ts' => strtotime((string)$item->pubDate),
                'date' => date('j M, H:i', strtotime((string)$item->pubDate))
            ];
        }
    }
}
usort($all_news, function($a, $b) { return $b['ts'] - $a['ts']; });
$news = array_slice($all_news, 0, 15);

// 3. STANDINGS & STATS (Official reference data)
$standings = [
    ['pos'=>1, 'team'=>'Malmö FF', 's'=>30, 'v'=>19, 'o'=>8, 'f'=>3, 'm'=>'76-25', 'p'=>65],
    ['pos'=>2, 'team'=>'Hammarby', 's'=>30, 'v'=>16, 'o'=>6, 'f'=>8, 'm'=>'48-25', 'p'=>54],
    ['pos'=>3, 'team'=>'AIK', 's'=>30, 'v'=>17, 'o'=>3, 'f'=>10, 'm'=>'46-41', 'p'=>54],
    ['pos'=>4, 'team'=>'Djurgården', 's'=>30, 'v'=>16, 'o'=>5, 'f'=>9, 'm'=>'45-35', 'p'=>53],
    ['pos'=>5, 'team'=>'Mjällby AIF', 's'=>30, 'v'=>14, 'o'=>8, 'f'=>8, 'm'=>'44-33', 'p'=>50],
    ['pos'=>6, 'team'=>'GAIS', 's'=>30, 'v'=>14, 'o'=>6, 'f'=>10, 'm'=>'33-31', 'p'=>48],
    ['pos'=>7, 'team'=>'IF Elfsborg', 's'=>30, 'v'=>13, 'o'=>6, 'f'=>11, 'm'=>'52-44', 'p'=>45],
    ['pos'=>8, 'team'=>'BK Häcken', 's'=>30, 'v'=>12, 'o'=>6, 'f'=>12, 'm'=>'54-51', 'p'=>42],
    ['pos'=>9, 'team'=>'IK Sirius', 's'=>30, 'v'=>12, 'o'=>5, 'f'=>13, 'm'=>'47-46', 'p'=>41],
    ['pos'=>10, 'team'=>'IFK Göteborg', 's'=>30, 'v'=>8, 'o'=>10, 'f'=>12, 'm'=>'33-43', 'p'=>34],
    ['pos'=>11, 'team'=>'IFK Norrköping', 's'=>30, 'v'=>9, 'o'=>7, 'f'=>14, 'm'=>'36-57', 'p'=>34],
    ['pos'=>12, 'team'=>'IF Brommapojkarna', 's'=>30, 'v'=>8, 'o'=>10, 'f'=>12, 'm'=>'46-53', 'p'=>34],
    ['pos'=>13, 'team'=>'IFK Värnamo', 's'=>30, 'v'=>7, 'o'=>10, 'f'=>13, 'm'=>'30-40', 'p'=>31],
    ['pos'=>14, 'team'=>'Halmstads BK', 's'=>30, 'v'=>10, 'o'=>3, 'f'=>17, 'm'=>'32-50', 'p'=>33],
    ['pos'=>15, 'team'=>'Kalmar FF', 's'=>30, 'v'=>8, 'o'=>6, 'f'=>16, 'm'=>'38-58', 'p'=>30],
    ['pos'=>16, 'team'=>'Västerås SK', 's'=>30, 'v'=>6, 'o'=>5, 'f'=>19, 'm'=>'26-43', 'p'=>23]
];

$top_scorers = [
    ['name'=>'Isaac Kiese Thelin', 'team'=>'Malmö FF', 'val'=>15],
    ['name'=>'Ioannis Pittas', 'team'=>'AIK', 'val'=>14],
    ['name'=>'Deniz Hümmet', 'team'=>'Djurgården', 'val'=>14],
    ['name'=>'Erik Botheim', 'team'=>'Malmö FF', 'val'=>13],
    ['name'=>'Nahir Besara', 'team'=>'Hammarby', 'val'=>12]
];

// 4. PLAYER MARKET SPIDER (Simulated Deep Analysis)
$players = [
    ['name'=>'Sebastian Nanasi', 'team'=>'Malmö FF', 'val'=>'€8.00m', 'fee'=>'€100k', 'pos'=>'LW', 'radar'=>[90,85,92,40,75]],
    ['name'=>'Lucas Bergvall', 'team'=>'Djurgården', 'val'=>'€7.00m', 'fee'=>'€1.00m', 'pos'=>'CM', 'radar'=>[75,90,88,60,82]],
    ['name'=>'Hugo Bolin', 'team'=>'Malmö FF', 'val'=>'€3.50m', 'fee'=>'Fri', 'pos'=>'AM', 'radar'=>[82,88,85,45,78]],
    ['name'=>'Jeremy Agbonifo', 'team'=>'BK Häcken', 'val'=>'€4.50m', 'fee'=>'€500k', 'pos'=>'RW', 'radar'=>[88,75,80,35,90]],
    ['name'=>'Markus Karlsson', 'team'=>'Hammarby', 'val'=>'€4.50m', 'fee'=>'Fri', 'pos'=>'CM', 'radar'=>[70,82,85,65,75]],
    ['name'=>'Matias Siltanen', 'team'=>'Djurgården', 'val'=>'€3.50m', 'fee'=>'Okänd', 'pos'=>'CM', 'radar'=>[65,88,90,70,80]],
    ['name'=>'Busanello', 'team'=>'Malmö FF', 'val'=>'€3.50m', 'fee'=>'€1.20m', 'pos'=>'LB', 'radar'=>[60,78,82,85,88]],
    ['name'=>'Elliot Stroud', 'team'=>'Mjällby AIF', 'val'=>'€3.00m', 'fee'=>'Okänd', 'pos'=>'LM', 'radar'=>[72,80,78,55,85]],
    ['name'=>'Montader Madjed', 'team'=>'Hammarby', 'val'=>'€3.00m', 'fee'=>'€350k', 'pos'=>'RW', 'radar'=>[85,72,78,30,82]],
    ['name'=>'Victor Eriksson', 'team'=>'Hammarby', 'val'=>'€3.00m', 'fee'=>'Okänd', 'pos'=>'CB', 'radar'=>[30,55,70,90,92]]
];

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Allsvenskan Scouting Hub v4</title>
    <style>
        :root { --bg: #020617; --card: #0f172a; --accent: #38bdf8; --text: #f8fafc; --muted: #94a3b8; --border: #1e293b; --green: #10b981; --gold: #fbbf24; --red: #ef4444; }
        body { background: var(--bg); color: var(--text); font-family: 'Inter', system-ui, sans-serif; margin: 0; padding: 10px; font-size: 14px; }
        .container { max-width: 1440px; margin: 0 auto; }
        header { background: linear-gradient(135deg, #1e293b 0%, #020617 100%); padding: 25px; border-radius: 20px; margin-bottom: 20px; display: flex; flex-wrap: wrap; justify-content: space-between; align-items: center; border: 1px solid var(--border); }
        h1 { margin: 0; font-size: 2.2rem; background: linear-gradient(to right, #38bdf8, #818cf8, #c084fc); -webkit-background-clip: text; -webkit-text-fill-color: transparent; font-weight: 900; }
        select { background: #1e293b; color: #fff; border: 1px solid var(--accent); padding: 12px 20px; border-radius: 12px; cursor: pointer; outline: none; }
        .grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 20px; }
        @media (max-width: 1200px) { .grid { grid-template-columns: 1fr 1fr; } }
        @media (max-width: 800px) { .grid { grid-template-columns: 1fr; } }
        .card { background: var(--card); padding: 20px; border-radius: 20px; border: 1px solid var(--border); box-shadow: 0 10px 15px -3px rgba(0,0,0,0.5); }
        h2 { color: var(--accent); font-size: 1.1rem; text-transform: uppercase; letter-spacing: 1px; margin-top: 0; border-bottom: 1px solid var(--border); padding-bottom: 15px; margin-bottom: 15px; display: flex; align-items: center; gap: 8px; }
        .table-scroll { overflow-x: auto; max-height: 500px; }
        table { width: 100%; border-collapse: collapse; }
        th { text-align: left; color: #64748b; font-size: 0.7rem; text-transform: uppercase; padding: 10px 5px; border-bottom: 2px solid var(--border); }
        td { padding: 12px 5px; border-bottom: 1px solid var(--border); }
        .points { font-weight: 800; color: var(--accent); }
        .val-text { color: var(--green); font-weight: 700; }
        .fee-text { color: var(--gold); font-size: 0.8rem; }
        .news-item { margin-bottom: 15px; border-left: 4px solid var(--accent); padding-left: 15px; }
        .news-item a { text-decoration: none; color: inherit; }
        .news-item h3 { margin: 5px 0; font-size: 1rem; line-height: 1.4; color: #f1f5f9; }
        .radar-svg { width: 36px; height: 36px; }
        .radar-poly { fill: rgba(56, 189, 248, 0.2); stroke: var(--accent); stroke-width: 1.5; }
        tr.highlight { background: rgba(56, 189, 248, 0.1); }
        tr:hover { background: #1e293b; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>Allsvenskan Scouting Hub</h1>
                <p style="color: var(--muted); margin: 5px 0 0 0;">Spiders: Player Intel • Market Values • Inköpspriser • Stats</p>
            </div>
            <form>
                <select name="team" onchange="this.form.submit()">
                    <option value="">Följ ett lag...</option>
                    <?php foreach($teams as $t): ?>
                        <option value="<?= $t ?>" <?= $followed_team == $t ? 'selected' : '' ?>><?= $t ?></option>
                    <?php endforeach; ?>
                </select>
            </form>
        </header>

        <div class="grid">
            <!-- 1. Full Standings -->
            <div class="card">
                <h2>🏆 Tabell 2024</h2>
                <div class="table-scroll">
                    <table>
                        <thead><tr><th>#</th><th>Lag</th><th>S</th><th>+/-</th><th>P</th></tr></thead>
                        <tbody>
                            <?php foreach($standings as $s): ?>
                            <tr class="<?= $followed_team == $s['team'] ? 'highlight' : '' ?>">
                                <td><small><?= $s['pos'] ?></small></td>
                                <td><strong><?= $s['team'] ?></strong></td>
                                <td><?= $s['s'] ?></td>
                                <td><small><?= $s['m'] ?></small></td>
                                <td class="points"><?= $s['p'] ?></td>
                            </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>

                <h2 style="margin-top:25px;">⚽ Skytteliga</h2>
                <table>
                    <?php foreach($top_scorers as $ts): ?>
                    <tr>
                        <td><?= $ts['name'] ?> (<?= $ts['team'] ?>)</td>
                        <td class="points" style="text-align:right"><?= $ts['val'] ?></td>
                    </tr>
                    <?php endforeach; ?>
                </table>
            </div>

            <!-- 2. Player Intelligence & Market Values -->
            <div class="card">
                <h2>💎 Spelarintel & Värden</h2>
                <div class="table-scroll">
                    <table>
                        <thead><tr><th>Spelare</th><th>Marknadsvärde</th><th>Inköpt för</th></tr></thead>
                        <tbody>
                            <?php foreach($players as $p):
                                $pts = []; $i = 0;
                                foreach($p['radar'] as $v) {
                                    $a = $i * 72 * (M_PI / 180);
                                    $r = ($v / 100) * 16;
                                    $pts[] = (18 + $r * cos($a)) . "," . (18 + $r * sin($a));
                                    $i++;
                                }
                            ?>
                            <tr>
                                <td style="display:flex; align-items:center; gap:8px;">
                                    <svg class="radar-svg"><polygon points="<?= implode(' ', $pts) ?>" class="radar-poly" /></svg>
                                    <div><strong><?= $p['name'] ?></strong><br><small><?= $p['team'] ?> (<?= $p['pos'] ?>)</small></div>
                                </td>
                                <td class="val-text"><?= $p['val'] ?></td>
                                <td class="fee-text"><?= $p['fee'] ?></td>
                            </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- 3. Dynamic News Feed -->
            <div class="card">
                <h2>📰 Nyhetsflöde <?= $followed_team ? " för $followed_team" : "" ?></h2>
                <div class="table-scroll">
                    <?php
                    $count = 0;
                    foreach($news as $n):
                        if ($followed_team && strpos(strtolower($n['title'].$n['desc']), strtolower($followed_team)) === false) continue;
                        $count++;
                    ?>
                    <div class="news-item">
                        <div style="font-size:0.7rem; margin-bottom:4px;"><span style="color:var(--accent)"><?= $n['source'] ?></span> • <?= $n['date'] ?></div>
                        <a href="<?= $n['link'] ?>" target="_blank"><h3><?= $n['title'] ?></h3></a>
                        <p style="font-size:0.8rem; color:var(--muted)"><?= substr($n['desc'], 0, 120) ?>...</p>
                    </div>
                    <?php endforeach; ?>
                    <?php if($count == 0) echo "<p style='text-align:center; padding:20px;'>Hittade inga nyheter för <strong>$followed_team</strong>.</p>"; ?>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
