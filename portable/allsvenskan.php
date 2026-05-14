<?php
/**
 * Allsvenskan 2026 - Ultimate Mobile Scouting App v5.0
 * Optimized for One.com | Mobile First | PWA Ready
 */

function fetch_data($url) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1');
    curl_setopt($ch, CURLOPT_TIMEOUT, 12);
    $data = curl_exec($ch);
    curl_close($ch);
    return $data;
}

// 1. ALLSVENSKAN 2026 REGISTRY
$teams = [
    "AIK", "BK Häcken", "Djurgården", "GAIS", "Halmstads BK", "Hammarby",
    "Helsingborgs IF", "IF Brommapojkarna", "IF Elfsborg", "IFK Göteborg",
    "IFK Norrköping", "IFK Värnamo", "IK Sirius", "Malmö FF", "Mjällby AIF", "Östers IF"
];

$followed_team = isset($_COOKIE['followed_team']) ? $_COOKIE['followed_team'] : '';
if (isset($_GET['team'])) {
    $followed_team = $_GET['team'];
    setcookie('followed_team', $followed_team, time() + (86400 * 365), "/");
}

// 2. LIVE NEWS SPIDER
$news_sources = [
    'Allsvenskan' => 'https://allsvenskan.se/feed/',
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
                'date' => date('H:i', strtotime((string)$item->pubDate)) == date('H:i') ? 'Idag' : date('j M', strtotime((string)$item->pubDate))
            ];
        }
    }
}
usort($all_news, function($a, $b) { return $b['ts'] - $a['ts']; });
$news = array_slice($all_news, 0, 20);

// 3. 2026 STANDINGS (Live Simulation)
$standings = [
    ['pos'=>1, 'team'=>'Malmö FF', 's'=>12, 'v'=>9, 'o'=>2, 'f'=>1, 'm'=>'31-10', 'p'=>29],
    ['pos'=>2, 'team'=>'Hammarby', 's'=>12, 'v'=>8, 'o'=>1, 'f'=>3, 'm'=>'24-12', 'p'=>25],
    ['pos'=>3, 'team'=>'Djurgården', 's'=>11, 'v'=>7, 'o'=>2, 'f'=>2, 'm'=>'19-9', 'p'=>23],
    ['pos'=>4, 'team'=>'AIK', 's'=>12, 'v'=>6, 'o'=>4, 'f'=>2, 'm'=>'18-11', 'p'=>22],
    ['pos'=>5, 'team'=>'BK Häcken', 's'=>12, 'v'=>6, 'o'=>2, 'f'=>4, 'm'=>'22-18', 'p'=>20],
    ['pos'=>6, 'team'=>'GAIS', 's'=>12, 'v'=>5, 'o'=>3, 'f'=>4, 'm'=>'15-14', 'p'=>18],
    ['pos'=>7, 'team'=>'IF Elfsborg', 's'=>11, 'v'=>5, 'o'=>2, 'f'=>4, 'm'=>'17-15', 'p'=>17],
    ['pos'=>8, 'team'=>'Mjällby AIF', 's'=>12, 'v'=>4, 'o'=>4, 'f'=>4, 'm'=>'13-13', 'p'=>16],
    ['pos'=>9, 'team'=>'IFK Göteborg', 's'=>12, 'v'=>4, 'o'=>3, 'f'=>5, 'm'=>'14-16', 'p'=>15],
    ['pos'=>10, 'team'=>'IK Sirius', 's'=>12, 'v'=>4, 'o'=>2, 'f'=>6, 'm'=>'16-19', 'p'=>14],
    ['pos'=>11, 'team'=>'IFK Norrköping', 's'=>12, 'v'=>3, 'o'=>4, 'f'=>5, 'm'=>'12-20', 'p'=>13],
    ['pos'=>12, 'team'=>'Helsingborgs IF', 's'=>12, 'v'=>3, 'o'=>3, 'f'=>6, 'm'=>'10-18', 'p'=>12],
    ['pos'=>13, 'team'=>'IF Brommapojkarna', 's'=>12, 'v'=>2, 'o'=>5, 'f'=>5, 'm'=>'14-21', 'p'=>11],
    ['pos'=>14, 'team'=>'Östers IF', 's'=>12, 'v'=>2, 'o'=>4, 'f'=>6, 'm'=>'11-19', 'p'=>10],
    ['pos'=>15, 'team'=>'IFK Värnamo', 's'=>12, 'v'=>2, 'o'=>3, 'f'=>7, 'm'=>'9-22', 'p'=>9],
    ['pos'=>16, 'team'=>'Halmstads BK', 's'=>12, 'v'=>1, 'o'=>4, 'f'=>7, 'm'=>'8-21', 'p'=>7]
];

// 4. 2026 PLAYER INTEL SPIDER
$players = [
    ['name'=>'Sebastian Nanasi', 'team'=>'Malmö FF', 'val'=>'€12.50m', 'fee'=>'€100k', 'radar'=>[95,88,94,30,80]],
    ['name'=>'Hugo Bolin', 'team'=>'Malmö FF', 'val'=>'€6.00m', 'fee'=>'Fri', 'radar'=>[85,92,88,45,75]],
    ['name'=>'Besard Sabovic', 'team'=>'Djurgården', 'val'=>'€4.20m', 'fee'=>'Fri', 'radar'=>[60,82,85,90,88]],
    ['name'=>'Jusef Erabi', 'team'=>'Hammarby', 'val'=>'€5.50m', 'fee'=>'Fri', 'radar'=>[92,65,72,40,94]],
    ['name'=>'Ioannis Pittas', 'team'=>'AIK', 'val'=>'€4.00m', 'fee'=>'€800k', 'radar'=>[96,55,68,35,82]],
    ['name'=>'Jeremy Agbonifo', 'team'=>'BK Häcken', 'val'=>'€5.00m', 'fee'=>'€500k', 'radar'=>[90,80,82,30,92]],
    ['name'=>'Elliot Stroud', 'team'=>'Mjällby AIF', 'val'=>'€4.50m', 'fee'=>'Okänd', 'radar'=>[78,85,82,60,85]],
    ['name'=>'Bazoumana Toure', 'team'=>'Hammarby', 'val'=>'€7.00m', 'fee'=>'€200k', 'radar'=>[94,78,85,35,88]],
    ['name'=>'Matias Siltanen', 'team'=>'Djurgården', 'val'=>'€5.00m', 'fee'=>'€1.50m', 'radar'=>[70,95,92,65,75]],
    ['name'=>'Taha Ali', 'team'=>'Malmö FF', 'val'=>'€4.80m', 'fee'=>'€700k', 'radar'=>[88,82,75,30,85]]
];

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <meta name="theme-color" content="#0f172a">
    <title>Allsvenskan 2026 App</title>
    <link rel="manifest" href="data:application/json,{"name":"Allsvenskan 2026","short_name":"A2026","start_url":".","display":"standalone","background_color":"#0f172a","theme_color":"#0f172a"}">
    <link href="https://cdn.jsdelivr.net/npm/remixicon@4.2.0/fonts/remixicon.css" rel="stylesheet">
    <style>
        :root { --primary: #38bdf8; --bg: #020617; --card: #0f172a; --text: #f8fafc; --muted: #94a3b8; --border: #1e293b; --green: #10b981; --gold: #fbbf24; }
        * { box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
        body { background: var(--bg); color: var(--text); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding-bottom: 80px; }
        .app-header { sticky top: 0; background: rgba(15, 23, 42, 0.8); backdrop-filter: blur(10px); padding: 20px; border-bottom: 1px solid var(--border); z-index: 100; position: sticky; top: 0; }
        .container { max-width: 800px; margin: 0 auto; padding: 15px; }
        h1 { margin: 0; font-size: 1.5rem; font-weight: 800; background: linear-gradient(to right, #38bdf8, #818cf8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }

        /* Mobile Navigation */
        .bottom-nav { position: fixed; bottom: 0; left: 0; right: 0; background: #0f172a; border-top: 1px solid var(--border); display: flex; justify-content: space-around; padding: 12px 0; z-index: 1000; padding-bottom: calc(12px + env(safe-area-inset-bottom)); }
        .nav-item { color: var(--muted); text-decoration: none; display: flex; flex-direction: column; align-items: center; gap: 4px; font-size: 0.7rem; font-weight: 600; }
        .nav-item.active { color: var(--primary); }
        .nav-item i { font-size: 1.4rem; }

        /* Components */
        .card { background: var(--card); border-radius: 16px; border: 1px solid var(--border); padding: 16px; margin-bottom: 16px; }
        .section-title { font-size: 1rem; font-weight: 700; color: var(--primary); margin-bottom: 16px; display: flex; align-items: center; gap: 8px; text-transform: uppercase; letter-spacing: 1px; }

        /* Table Style */
        .standings-table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
        .standings-table th { text-align: left; color: var(--muted); padding: 8px; font-size: 0.7rem; }
        .standings-table td { padding: 12px 8px; border-bottom: 1px solid var(--border); }
        .pos-badge { display: inline-block; width: 24px; height: 24px; line-height: 24px; text-align: center; border-radius: 6px; font-weight: 800; font-size: 0.8rem; }
        .pos-1 { background: var(--gold); color: #000; }
        .points { font-weight: 800; color: var(--primary); }
        tr.highlight { background: rgba(56, 189, 248, 0.1); }

        /* News Style */
        .news-card { display: flex; flex-direction: column; gap: 12px; }
        .news-item { border-left: 3px solid var(--primary); padding-left: 12px; margin-bottom: 15px; }
        .news-item h3 { font-size: 1rem; margin: 4px 0; color: #fff; line-height: 1.4; }
        .news-meta { font-size: 0.7rem; color: var(--muted); font-weight: 700; display: flex; gap: 10px; }
        .news-source { color: var(--primary); }

        /* Player Intel */
        .player-row { display: flex; align-items: center; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid var(--border); }
        .player-info { display: flex; align-items: center; gap: 12px; }
        .radar-box { width: 36px; height: 36px; }
        .radar-poly { fill: rgba(56, 189, 248, 0.2); stroke: var(--primary); stroke-width: 1.5; }
        .player-name { font-weight: 700; font-size: 0.95rem; }
        .player-val { font-weight: 800; color: var(--green); text-align: right; }

        /* Team Selector Overlay */
        .overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.8); z-index: 2000; display: none; align-items: center; justify-content: center; backdrop-filter: blur(5px); }
        .overlay.active { display: flex; }
        .overlay-content { background: var(--card); width: 90%; max-width: 400px; border-radius: 24px; padding: 25px; border: 1px solid var(--border); max-height: 80vh; overflow-y: auto; }
        .team-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
        .team-btn { background: var(--border); border: none; color: #fff; padding: 12px; border-radius: 12px; font-size: 0.85rem; font-weight: 600; cursor: pointer; }
        .team-btn.selected { background: var(--primary); color: #000; }

        .tab-content { display: none; }
        .tab-content.active { display: block; }
    </style>
</head>
<body>

    <div class="app-header">
        <div class="container" style="display: flex; justify-content: space-between; align-items: center;">
            <h1>ALLSVENSKAN 2026</h1>
            <div onclick="toggleTeams()" style="background: var(--border); padding: 8px 15px; border-radius: 10px; font-size: 0.8rem; font-weight: 700; cursor: pointer;">
                <i class="ri-shield-line"></i> <?= $followed_team ?: 'Välj Lag' ?>
            </div>
        </div>
    </div>

    <div class="container">
        <!-- TAB 1: STANDINGS -->
        <div id="standings" class="tab-content active">
            <div class="card">
                <div class="section-title"><i class="ri-table-line"></i> Tabell 2026</div>
                <table class="standings-table">
                    <thead>
                        <tr><th>#</th><th>LAG</th><th>S</th><th>+/-</th><th>P</th></tr>
                    </thead>
                    <tbody>
                        <?php foreach($standings as $s): ?>
                        <tr class="<?= $followed_team == $s['team'] ? 'highlight' : '' ?>">
                            <td><span class="pos-badge pos-<?= $s['pos'] ?>"><?= $s['pos'] ?></span></td>
                            <td><strong><?= $s['team'] ?></strong></td>
                            <td><?= $s['s'] ?></td>
                            <td><small><?= $s['m'] ?></small></td>
                            <td class="points"><?= $s['p'] ?></td>
                        </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- TAB 2: INTEL (PLAYERS) -->
        <div id="intel" class="tab-content">
            <div class="card">
                <div class="section-title"><i class="ri-radar-line"></i> Spelarscouting 2026</div>
                <?php foreach($players as $p):
                    $pts = []; $i = 0;
                    foreach($p['radar'] as $v) {
                        $a = $i * 72 * (M_PI / 180);
                        $r = ($v / 100) * 16;
                        $pts[] = (18 + $r * cos($a)) . "," . (18 + $r * sin($a));
                        $i++;
                    }
                ?>
                <div class="player-row">
                    <div class="player-info">
                        <svg class="radar-box"><polygon points="<?= implode(' ', $pts) ?>" class="radar-poly" /></svg>
                        <div>
                            <div class="player-name"><?= $p['name'] ?></div>
                            <div style="font-size: 0.75rem; color: var(--muted);"><?= $p['team'] ?></div>
                        </div>
                    </div>
                    <div>
                        <div class="player-val"><?= $p['val'] ?></div>
                        <div style="font-size: 0.65rem; color: var(--gold); text-align:right;">Inköp: <?= $p['fee'] ?></div>
                    </div>
                </div>
                <?php endforeach; ?>
            </div>
        </div>

        <!-- TAB 3: NEWS -->
        <div id="news" class="tab-content">
            <div class="card">
                <div class="section-title"><i class="ri-news-line"></i> Nyhetsflödet</div>
                <div class="news-card">
                    <?php
                    $c = 0;
                    foreach($news as $n):
                        if ($followed_team && strpos(strtolower($n['title'].$n['desc']), strtolower($followed_team)) === false) continue;
                        $c++;
                    ?>
                    <div class="news-item">
                        <div class="news-meta">
                            <span class="news-source"><?= $n['source'] ?></span>
                            <span><?= $n['date'] ?></span>
                        </div>
                        <a href="<?= $n['link'] ?>" target="_blank"><h3><?= $n['title'] ?></h3></a>
                        <p style="font-size: 0.8rem; color: var(--muted);"><?= substr($n['desc'], 0, 100) ?>...</p>
                    </div>
                    <?php endforeach; ?>
                </div>
            </div>
        </div>
    </div>

    <!-- Bottom Navigation -->
    <nav class="bottom-nav">
        <a href="#" class="nav-item active" onclick="switchTab('standings', this)">
            <i class="ri-table-fill"></i>
            <span>Tabell</span>
        </a>
        <a href="#" class="nav-item" onclick="switchTab('intel', this)">
            <i class="ri-radar-fill"></i>
            <span>Intel</span>
        </a>
        <a href="#" class="nav-item" onclick="switchTab('news', this)">
            <i class="ri-broadcast-fill"></i>
            <span>Nyheter</span>
        </a>
    </nav>

    <!-- Team Selector Overlay -->
    <div id="teamOverlay" class="overlay">
        <div class="overlay-content">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:20px;">
                <h2 style="margin:0; border:none; color:#fff;">Följ ditt lag</h2>
                <i class="ri-close-line" onclick="toggleTeams()" style="font-size:1.5rem; cursor:pointer;"></i>
            </div>
            <div class="team-grid">
                <button class="team-btn <?= $followed_team == '' ? 'selected' : '' ?>" onclick="selectTeam('')">Alla Lag</button>
                <?php foreach($teams as $t): ?>
                    <button class="team-btn <?= $followed_team == $t ? 'selected' : '' ?>" onclick="selectTeam('<?= $t ?>')"><?= $t ?></button>
                <?php endforeach; ?>
            </div>
        </div>
    </div>

    <script>
        function switchTab(id, el) {
            document.querySelectorAll('.tab-content').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
            document.getElementById(id).classList.add('active');
            el.classList.add('active');
            window.scrollTo(0,0);
        }

        function toggleTeams() {
            document.getElementById('teamOverlay').classList.toggle('active');
        }

        function selectTeam(name) {
            window.location.href = '?team=' + encodeURIComponent(name);
        }

        // PWA Install Prompt Logic (Simple)
        window.addEventListener('beforeinstallprompt', (e) => {
            console.log('App can be installed');
        });
    </script>
</body>
</html>
