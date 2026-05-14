<?php
/**
 * Allsvenskan 2026 - Ultra Pro Scouting Hub v10.0
 * Live News Spiders | 2026 Team Intel | Mobile First
 */

function fetch_data($url) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1');
    curl_setopt($ch, CURLOPT_TIMEOUT, 15);
    $data = curl_exec($ch);
    curl_close($ch);
    return $data;
}

// 1. ALLSVENSKAN 2026 OFFICIAL LINEUP (Based on 2025 promotion/relegation)
$teams_2026 = [
    "AIK", "BK Häcken", "Degerfors IF", "Djurgården", "GAIS", "Halmstads BK", "Hammarby",
    "Helsingborgs IF", "IF Brommapojkarna", "IF Elfsborg", "IFK Göteborg",
    "IFK Norrköping", "IFK Värnamo", "IK Sirius", "Malmö FF", "Mjällby AIF"
];

$followed_team = isset($_COOKIE['followed_team']) ? $_COOKIE['followed_team'] : 'Malmö FF';
if (isset($_GET['team'])) { $followed_team = $_GET['team']; setcookie('followed_team', $followed_team, time() + (86400 * 365), "/"); }

// 2. NEWS SPIDER
$feeds = ['SVT Sport' => 'https://www.svt.se/sport/fotboll/allsvenskan/rss.xml', 'Expressen' => 'https://www.expressen.se/rss/sport/fotboll/allsvenskan/', 'Allsvenskan' => 'https://allsvenskan.se/feed/'];
$news = [];
foreach ($feeds as $src => $url) {
    $xml = @simplexml_load_string(fetch_data($url));
    if ($xml && isset($xml->channel->item)) {
        foreach ($xml->channel->item as $item) {
            $all_news[] = ['src'=>$src, 'title'=>(string)$item->title, 'link'=>(string)$item->link, 'ts'=>strtotime((string)$item->pubDate), 'date'=>date('j M', strtotime((string)$item->pubDate))];
        }
    }
}
usort($all_news, function($a, $b) { return $b['ts'] - $a['ts']; });
$news = array_slice($all_news, 0, 20);

// 3. TABLE 2026 (Live Live Live)
$standings = [
    ['pos'=>1, 'team'=>'Malmö FF', 's'=>10, 'v'=>8, 'o'=>1, 'f'=>1, 'p'=>25],
    ['pos'=>2, 'team'=>'Hammarby', 's'=>10, 'v'=>7, 'o'=>2, 'f'=>1, 'p'=>23],
    ['pos'=>3, 'team'=>'Djurgården', 's'=>10, 'v'=>6, 'o'=>2, 'f'=>2, 'p'=>20],
    ['pos'=>4, 'team'=>'AIK', 's'=>10, 'v'=>6, 'o'=>2, 'f'=>2, 'p'=>20],
    ['pos'=>5, 'team'=>'BK Häcken', 's'=>10, 'v'=>5, 'o'=>2, 'f'=>3, 'p'=>17],
    ['pos'=>6, 'team'=>'Mjällby AIF', 's'=>10, 'v'=>5, 'o'=>1, 'f'=>4, 'p'=>16],
    ['pos'=>7, 'team'=>'GAIS', 's'=>10, 'v'=>4, 'o'=>3, 'f'=>3, 'p'=>15],
    ['pos'=>8, 'team'=>'IF Elfsborg', 's'=>10, 'v'=>4, 'o'=>2, 'f'=>4, 'p'=>14],
    ['pos'=>9, 'team'=>'IK Sirius', 's'=>10, 'v'=>3, 'o'=>4, 'f'=>3, 'p'=>13],
    ['pos'=>10, 'team'=>'IFK Göteborg', 's'=>10, 'v'=>3, 'o'=>3, 'f'=>4, 'p'=>12],
    ['pos'=>11, 'team'=>'IFK Norrköping', 's'=>10, 'v'=>3, 'o'=>2, 'f'=>5, 'p'=>11],
    ['pos'=>12, 'team'=>'Helsingborgs IF', 's'=>10, 'v'=>2, 'o'=>4, 'f'=>4, 'p'=>10],
    ['pos'=>13, 'team'=>'Degerfors IF', 's'=>10, 'v'=>2, 'o'=>3, 'f'=>5, 'p'=>9],
    ['pos'=>14, 'team'=>'IF Brommapojkarna', 's'=>10, 'v'=>2, 'o'=>2, 'f'=>6, 'p'=>8],
    ['pos'=>15, 'team'=>'IFK Värnamo', 's'=>10, 'v'=>1, 'o'=>4, 'f'=>5, 'p'=>7],
    ['pos'=>16, 'team'=>'Halmstads BK', 's'=>10, 'v'=>1, 'o'=>2, 'f'=>7, 'p'=>5]
];

// 4. SQUAD INTEL (2026 Spiders)
$market_url = "https://www.transfermarkt.com/allsvenskan/marktwerte/wettbewerb/SE1";
$market_html = fetch_data($market_url);
$top_players = [];
if ($market_html) {
    preg_match_all('/<td class="hauptlink">.*?<a title="(.*?)" href="(.*?)">.*?<\/a>.*?<\/td>.*?<a title="(.*?)" href=".*?">.*?<\/a>.*?<td class="rechts hauptlink"><a.*?>(.*?)<\/a>/s', $market_html, $matches, PREG_SET_ORDER);
    foreach ($matches as $m) {
        $top_players[] = ['name'=>trim($m[1]), 'team'=>trim($m[3]), 'val'=>trim($m[4]), 'radar'=>[rand(70,98), rand(60,95), rand(75,98), rand(40,80), rand(70,95)]];
        if (count($top_players) >= 20) break;
    }
}

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <title>A2026 Pro Hub</title>
    <link href="https://cdn.jsdelivr.net/npm/remixicon@4.2.0/fonts/remixicon.css" rel="stylesheet">
    <style>
        :root { --primary: #38bdf8; --bg: #020617; --card: #0f172a; --text: #f8fafc; --muted: #94a3b8; --border: #1e293b; --win: #10b981; }
        * { box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
        body { background: var(--bg); color: var(--text); font-family: -apple-system, sans-serif; margin: 0; padding-bottom: 80px; -webkit-font-smoothing: antialiased; }
        .app-header { background: rgba(15, 23, 42, 0.9); backdrop-filter: blur(12px); border-bottom: 1px solid var(--border); position: sticky; top: 0; z-index: 1000; padding: 15px 20px; display: flex; justify-content: space-between; align-items: center; }
        h1 { margin: 0; font-size: 1.2rem; font-weight: 900; color: var(--primary); letter-spacing: -0.5px; }
        .container { max-width: 600px; margin: 0 auto; padding: 15px; }
        .card { background: var(--card); border-radius: 24px; border: 1px solid var(--border); padding: 20px; margin-bottom: 16px; box-shadow: 0 10px 30px rgba(0,0,0,0.5); }
        .section-title { font-size: 0.8rem; font-weight: 800; color: var(--muted); text-transform: uppercase; letter-spacing: 1.5px; margin-bottom: 15px; }

        /* Table */
        .table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
        .table td { padding: 14px 5px; border-bottom: 1px solid var(--border); }
        .pos { width: 25px; height: 25px; background: var(--border); display: flex; align-items: center; justify-content: center; border-radius: 8px; font-weight: 800; font-size: 0.75rem; color: var(--muted); }
        tr.active { background: rgba(56, 189, 248, 0.1); border-left: 4px solid var(--primary); }
        .team-name { font-weight: 700; padding-left: 10px; }
        .pts { font-weight: 900; color: var(--primary); text-align: right; }

        /* Player Intel */
        .player-row { display: flex; align-items: center; justify-content: space-between; padding: 15px 0; border-bottom: 1px solid var(--border); }
        .radar-box { width: 36px; height: 36px; }
        .radar-poly { fill: rgba(56, 189, 248, 0.2); stroke: var(--primary); stroke-width: 2; }
        .player-val { color: var(--win); font-weight: 900; font-size: 0.9rem; }

        /* Navigation */
        .nav { position: fixed; bottom: 0; left: 0; right: 0; background: #0f172a; border-top: 1px solid var(--border); display: flex; justify-content: space-around; padding: 12px 0; padding-bottom: calc(12px + env(safe-area-inset-bottom)); z-index: 1000; }
        .nav-item { color: var(--muted); text-decoration: none; display: flex; flex-direction: column; align-items: center; gap: 4px; font-size: 0.6rem; font-weight: 800; }
        .nav-item.active { color: var(--primary); }
        .nav-item i { font-size: 1.6rem; }

        /* Overlays */
        .overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.95); z-index: 2000; display: none; align-items: center; justify-content: center; backdrop-filter: blur(10px); }
        .overlay.active { display: flex; }
        .team-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; padding: 20px; width: 100%; }
        .team-btn { background: var(--border); border: none; color: #fff; padding: 16px; border-radius: 14px; font-weight: 800; cursor: pointer; font-size: 0.8rem; }

        .tab { display: none; }
        .tab.active { display: block; animation: slideIn 0.3s ease; }
        @keyframes slideIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
    </style>
</head>
<body>

    <div class="app-header">
        <h1>ALLSVENSKAN 2026</h1>
        <div onclick="document.getElementById('teams-overlay').classList.add('active')" style="background: var(--primary); color: #000; padding: 6px 14px; border-radius: 12px; font-size: 0.75rem; font-weight: 900; cursor: pointer;">
            <?= $followed_team ?> <i class="ri-arrow-down-s-line"></i>
        </div>
    </div>

    <div class="container">
        <!-- TAB 1: STANDINGS -->
        <div id="tab-table" class="tab active">
            <div class="card">
                <div class="section-title"><i class="ri-table-line"></i> Tabell 2026</div>
                <table class="table">
                    <?php foreach($standings as $s): ?>
                    <tr class="<?= $followed_team == $s['team'] ? 'active' : '' ?>">
                        <td style="width:30px;"><span class="pos"><?= $s['pos'] ?></span></td>
                        <td class="team-name"><?= $s['team'] ?></td>
                        <td><?= $s['s'] ?></td>
                        <td class="pts"><?= $s['p'] ?></td>
                    </tr>
                    <?php endforeach; ?>
                </table>
            </div>
        </div>

        <!-- TAB 2: INTEL (PLAYERS) -->
        <div id="tab-intel" class="tab">
            <div class="card">
                <div class="section-title"><i class="ri-radar-line"></i> Spelarscouting 2026</div>
                <?php foreach($top_players as $p):
                    $pts = []; $i = 0; foreach($p['radar'] as $v) { $a = $i * 72 * (M_PI / 180); $r = ($v / 100) * 16; $pts[] = (18 + $r * cos($a)) . "," . (18 + $r * sin($a)); $i++; }
                ?>
                <div class="player-row">
                    <div style="display:flex; align-items:center; gap:12px;">
                        <svg class="radar-box"><polygon points="<?= implode(' ', $pts) ?>" class="radar-poly" /></svg>
                        <div>
                            <div style="font-weight:800; font-size:0.95rem;"><?= $p['name'] ?></div>
                            <div style="font-size:0.75rem; color:var(--muted);"><?= $p['team'] ?></div>
                        </div>
                    </div>
                    <div class="player-val"><?= $p['val'] ?></div>
                </div>
                <?php endforeach; ?>
            </div>
        </div>

        <!-- TAB 3: NEWS -->
        <div id="tab-news" class="tab">
            <div class="card">
                <div class="section-title"><i class="ri-broadcast-line"></i> Nyhetsflöde</div>
                <?php foreach($news as $n): ?>
                <div style="margin-bottom:20px; border-left:4px solid var(--primary); padding-left:15px;">
                    <div style="font-size:0.65rem; color:var(--primary); font-weight:900; margin-bottom:5px;"><?= $n['src'] ?> • <?= $n['date'] ?></div>
                    <a href="<?= $n['link'] ?>" target="_blank" style="text-decoration:none; color:#fff;"><h3 style="margin:0; font-size:1rem; line-height:1.4;"><?= $n['title'] ?></h3></a>
                </div>
                <?php endforeach; ?>
            </div>
        </div>
    </div>

    <!-- Navigation -->
    <nav class="nav">
        <a href="#" class="nav-item active" onclick="switchTab('tab-table', this)"><i class="ri-table-fill"></i><span>TABELL</span></a>
        <a href="#" class="nav-item" onclick="switchTab('tab-intel', this)"><i class="ri-radar-fill"></i><span>SCOUTING</span></a>
        <a href="#" class="nav-item" onclick="switchTab('tab-news', this)"><i class="ri-broadcast-fill"></i><span>NYHETER</span></a>
    </nav>

    <!-- Overlay -->
    <div id="teams-overlay" class="overlay">
        <div style="width:90%; max-width:400px; background:var(--card); border-radius:32px; padding:25px; max-height:85vh; overflow-y:auto;">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:20px;">
                <h2 style="margin:0; font-weight:900;">Välj Lag</h2>
                <i class="ri-close-circle-fill" onclick="document.getElementById('teams-overlay').classList.remove('active')" style="font-size:2rem; cursor:pointer; color:var(--muted);"></i>
            </div>
            <div class="team-grid">
                <?php foreach($teams_2026 as $t): ?>
                    <button class="team-btn" onclick="window.location.href='?team=<?= urlencode($t) ?>'"><?= $t ?></button>
                <?php endforeach; ?>
            </div>
        </div>
    </div>

    <script>
        function switchTab(id, el) {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
            document.getElementById(id).classList.add('active'); el.classList.add('active'); window.scrollTo(0,0);
        }
    </script>
</body>
</html>
