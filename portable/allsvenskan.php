<?php
/**
 * Allsvenskan 2026 - Ultimate Mobile Hub v6.0
 * Optimized for One.com | High Density Player Stats | 2026 Schedule & Standings
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

// 1. ALLSVENSKAN 2026 CLUBS & REGISTRY
$teams = [
    "AIK", "BK Häcken", "Djurgården", "GAIS", "Halmstads BK", "Hammarby",
    "Helsingborgs IF", "IF Brommapojkarna", "IF Elfsborg", "IFK Göteborg",
    "IFK Norrköping", "IFK Värnamo", "IK Sirius", "Malmö FF", "Mjällby AIF", "Östers IF"
];

$followed_team = isset($_COOKIE['followed_team']) ? $_COOKIE['followed_team'] : 'Malmö FF';
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
                'date' => date('j M', strtotime((string)$item->pubDate))
            ];
        }
    }
}
usort($all_news, function($a, $b) { return $b['ts'] - $a['ts']; });
$news = array_slice($all_news, 0, 20);

// 3. 2026 STANDINGS & FIXTURES (Projected 2026 Logic)
$standings = [
    ['pos'=>1, 'team'=>'Malmö FF', 's'=>10, 'v'=>8, 'o'=>1, 'f'=>1, 'p'=>25, 'form'=>['V','V','V','F','V']],
    ['pos'=>2, 'team'=>'Hammarby', 's'=>10, 'v'=>7, 'o'=>2, 'f'=>1, 'p'=>23, 'form'=>['V','O','V','V','F']],
    ['pos'=>3, 'team'=>'AIK', 's'=>10, 'v'=>6, 'o'=>3, 'f'=>1, 'p'=>21, 'form'=>['V','V','O','F','V']],
    ['pos'=>4, 'team'=>'Djurgården', 's'=>10, 'v'=>6, 'o'=>2, 'f'=>2, 'p'=>20, 'form'=>['F','V','V','V','O']],
    ['pos'=>5, 'team'=>'BK Häcken', 's'=>10, 'v'=>5, 'o'=>2, 'f'=>3, 'p'=>17, 'form'=>['V','F','O','V','V']],
    ['pos'=>6, 'team'=>'IF Elfsborg', 's'=>10, 'v'=>5, 'o'=>1, 'f'=>4, 'p'=>16, 'form'=>['F','V','F','V','V']],
    ['pos'=>7, 'team'=>'GAIS', 's'=>10, 'v'=>4, 'o'=>3, 'f'=>3, 'p'=>15, 'form'=>['V','O','V','F','F']],
    ['pos'=>8, 'team'=>'IFK Göteborg', 's'=>10, 'v'=>4, 'o'=>2, 'f'=>4, 'p'=>14, 'form'=>['F','F','V','V','O']],
    ['pos'=>9, 'team'=>'Mjällby AIF', 's'=>10, 'v'=>3, 'o'=>4, 'f'=>3, 'p'=>13, 'form'=>['O','V','F','O','V']],
    ['pos'=>10, 'team'=>'IK Sirius', 's'=>10, 'v'=>3, 'o'=>3, 'f'=>4, 'p'=>12, 'form'=>['F','V','O','F','F']],
    ['pos'=>11, 'team'=>'IFK Norrköping', 's'=>10, 'v'=>3, 'o'=>2, 'f'=>5, 'p'=>11, 'form'=>['F','F','F','V','V']],
    ['pos'=>12, 'team'=>'Helsingborgs IF', 's'=>10, 'v'=>2, 'o'=>4, 'f'=>4, 'p'=>10, 'form'=>['O','F','V','F','O']],
    ['pos'=>13, 'team'=>'Östers IF', 's'=>10, 'v'=>2, 'o'=>3, 'f'=>5, 'p'=>9, 'form'=>['F','O','F','F','V']],
    ['pos'=>14, 'team'=>'IF Brommapojkarna', 's'=>10, 'v'=>2, 'o'=>2, 'f'=>6, 'p'=>8, 'form'=>['F','F','V','F','F']],
    ['pos'=>15, 'team'=>'IFK Värnamo', 's'=>10, 'v'=>1, 'o'=>4, 'f'=>5, 'p'=>7, 'form'=>['O','F','F','O','F']],
    ['pos'=>16, 'team'=>'Halmstads BK', 's'=>10, 'v'=>1, 'o'=>2, 'f'=>7, 'p'=>5, 'form'=>['F','F','F','F','O']]
];

$fixtures = [
    ['date'=>'15 Maj', 'time'=>'19:00', 'home'=>'Malmö FF', 'away'=>'AIK', 'venue'=>'Eleda Stadion'],
    ['date'=>'16 Maj', 'time'=>'15:00', 'home'=>'Hammarby', 'away'=>'Djurgården', 'venue'=>'Tele2 Arena'],
    ['date'=>'16 Maj', 'time'=>'17:30', 'home'=>'IFK Göteborg', 'away'=>'GAIS', 'venue'=>'Gamla Ullevi'],
    ['date'=>'17 Maj', 'time'=>'15:00', 'home'=>'Helsingborgs IF', 'away'=>'Östers IF', 'venue'=>'Olympia'],
    ['date'=>'17 Maj', 'time'=>'15:00', 'home'=>'BK Häcken', 'away'=>'IF Elfsborg', 'venue'=>'Bravida Arena']
];

// 4. SQUAD & PLAYER INTEL (2026 Scouting)
$squad_db = [
    'Malmö FF' => [
        ['n'=>'Sebastian Nanasi', 'pos'=>'Vänstermittfältare', 'val'=>'€15.00m', 'buy'=>'€100k', 'age'=>24, 'spider'=>[95,90,92,40,82], 'xi'=>true],
        ['n'=>'Hugo Bolin', 'pos'=>'Offensiv Mittfältare', 'val'=>'€8.00m', 'buy'=>'Fri', 'age'=>23, 'spider'=>[85,94,88,45,78], 'xi'=>true],
        ['n'=>'Erik Botheim', 'pos'=>'Anfallare', 'val'=>'€4.50m', 'buy'=>'Fri', 'age'=>26, 'spider'=>[92,60,75,30,85], 'xi'=>true],
        ['n'=>'Pontus Jansson', 'pos'=>'Mittback', 'val'=>'€1.50m', 'buy'=>'Fri', 'age'=>35, 'spider'=>[20,50,75,95,90], 'xi'=>true]
    ],
    'AIK' => [
        ['n'=>'Ioannis Pittas', 'pos'=>'Anfallare', 'val'=>'€4.50m', 'buy'=>'€800k', 'age'=>30, 'spider'=>[96,50,65,30,85], 'xi'=>true],
        ['n'=>'Lamine Fanne', 'pos'=>'Mittfältare', 'val'=>'€6.00m', 'buy'=>'€50k', 'age'=>22, 'spider'=>[65,85,90,75,88], 'xi'=>true],
        ['n'=>'Sotirios Papagiannopoulos', 'pos'=>'Mittback', 'val'=>'€400k', 'buy'=>'Fri', 'age'=>35, 'spider'=>[15,40,65,92,94], 'xi'=>true]
    ],
    'Hammarby' => [
        ['n'=>'Nahir Besara', 'pos'=>'Offensiv Mittfältare', 'val'=>'€3.00m', 'buy'=>'Fri', 'age'=>35, 'spider'=>[88,96,90,30,72], 'xi'=>true],
        ['n'=>'Bazoumana Toure', 'pos'=>'Ytter', 'val'=>'€10.00m', 'buy'=>'€200k', 'age'=>20, 'spider'=>[94,80,85,35,90], 'xi'=>true],
        ['n'=>'Jusef Erabi', 'pos'=>'Anfallare', 'val'=>'€6.50m', 'buy'=>'Fri', 'age'=>23, 'spider'=>[92,55,70,45,95], 'xi'=>true]
    ]
];

$active_squad = isset($squad_db[$followed_team]) ? $squad_db[$followed_team] : [];

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover">
    <meta name="apple-mobile-web-app-capable" content="yes">
    <meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
    <meta name="theme-color" content="#020617">
    <title>Allsvenskan 2026 Scouting</title>
    <link href="https://cdn.jsdelivr.net/npm/remixicon@4.2.0/fonts/remixicon.css" rel="stylesheet">
    <style>
        :root { --primary: #38bdf8; --bg: #020617; --card: #0f172a; --text: #f8fafc; --muted: #94a3b8; --border: #1e293b; --green: #10b981; --gold: #fbbf24; --win: #10b981; --draw: #64748b; --loss: #ef4444; }
        * { box-sizing: border-box; -webkit-tap-highlight-color: transparent; }
        body { background: var(--bg); color: var(--text); font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding-bottom: 80px; }
        .app-header { background: rgba(15, 23, 42, 0.85); backdrop-filter: blur(12px); border-bottom: 1px solid var(--border); position: sticky; top: 0; z-index: 1000; padding: 15px 20px; }
        .container { max-width: 600px; margin: 0 auto; }
        h1 { margin: 0; font-size: 1.4rem; font-weight: 900; background: linear-gradient(to right, #38bdf8, #818cf8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }

        .bottom-nav { position: fixed; bottom: 0; left: 0; right: 0; background: #0f172a; border-top: 1px solid var(--border); display: flex; justify-content: space-around; padding: 12px 0; z-index: 1000; padding-bottom: calc(12px + env(safe-area-inset-bottom)); }
        .nav-item { color: var(--muted); text-decoration: none; display: flex; flex-direction: column; align-items: center; gap: 4px; font-size: 0.7rem; font-weight: 700; transition: 0.3s; }
        .nav-item.active { color: var(--primary); }
        .nav-item i { font-size: 1.5rem; }

        .card { background: var(--card); border-radius: 20px; border: 1px solid var(--border); padding: 18px; margin-bottom: 16px; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.4); }
        .section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 15px; }
        .section-title { font-size: 0.9rem; font-weight: 800; color: var(--primary); text-transform: uppercase; letter-spacing: 1px; }

        /* Table */
        .table { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
        .table th { text-align: left; color: var(--muted); padding: 8px 4px; font-size: 0.7rem; }
        .table td { padding: 12px 4px; border-bottom: 1px solid var(--border); }
        .pos { width: 25px; height: 25px; display: flex; align-items: center; justify-content: center; border-radius: 6px; font-weight: 800; font-size: 0.75rem; background: var(--border); }
        .pos-1 { background: var(--gold); color: #000; }
        .team-name { font-weight: 700; padding-left: 8px; }
        .form-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; margin-left: 2px; }
        .form-V { background: var(--win); } .form-O { background: var(--draw); } .form-F { background: var(--loss); }
        tr.active { background: rgba(56, 189, 248, 0.1); }

        /* Squad & Player */
        .player-card { display: flex; align-items: center; justify-content: space-between; padding: 12px 0; border-bottom: 1px solid var(--border); }
        .player-info { display: flex; align-items: center; gap: 12px; }
        .radar-box { width: 40px; height: 40px; }
        .radar-poly { fill: rgba(56, 189, 248, 0.2); stroke: var(--primary); stroke-width: 1.5; }
        .val-pill { background: rgba(16, 185, 129, 0.1); color: var(--green); padding: 4px 8px; border-radius: 6px; font-weight: 800; font-size: 0.8rem; }

        /* Schedule */
        .match-row { display: flex; align-items: center; gap: 15px; padding: 15px 0; border-bottom: 1px solid var(--border); }
        .match-date { width: 50px; text-align: center; border-right: 1px solid var(--border); padding-right: 10px; font-size: 0.7rem; font-weight: 800; color: var(--muted); }
        .match-teams { flex: 1; display: flex; flex-direction: column; gap: 4px; font-weight: 700; }

        /* Team Overlay */
        .overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.9); z-index: 2000; display: none; align-items: center; justify-content: center; backdrop-filter: blur(10px); }
        .overlay.active { display: flex; }
        .overlay-content { background: var(--card); width: 92%; max-width: 400px; border-radius: 28px; padding: 25px; border: 1px solid var(--border); max-height: 85vh; overflow-y: auto; }
        .team-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
        .team-btn { background: var(--border); border: none; color: #fff; padding: 14px; border-radius: 14px; font-size: 0.8rem; font-weight: 700; cursor: pointer; text-align: center; }
        .team-btn.selected { background: var(--primary); color: #000; box-shadow: 0 0 15px var(--primary); }

        .xi-badge { font-size: 0.6rem; background: var(--primary); color: #000; padding: 1px 4px; border-radius: 3px; margin-left: 5px; font-weight: 800; }

        .tab-content { display: none; animation: fadeIn 0.3s ease; }
        .tab-content.active { display: block; }
        @keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
    </style>
</head>
<body>

    <div class="app-header">
        <div class="container" style="display: flex; justify-content: space-between; align-items: center;">
            <h1>ALLSVENSKAN 2026</h1>
            <div onclick="toggleTeams()" style="background: var(--primary); color: #000; padding: 6px 14px; border-radius: 12px; font-size: 0.75rem; font-weight: 900; cursor: pointer; box-shadow: 0 4px 10px rgba(56, 189, 248, 0.4);">
                <?= $followed_team ?> <i class="ri-arrow-down-s-line"></i>
            </div>
        </div>
    </div>

    <div class="container">
        <!-- TAB: STANDINGS -->
        <div id="standings" class="tab-content active">
            <div class="card">
                <div class="section-title">Aktuell Tabell</div>
                <table class="table">
                    <thead><tr><th>#</th><th>LAG</th><th>S</th><th>P</th><th>FORM</th></tr></thead>
                    <tbody>
                        <?php foreach($standings as $s): ?>
                        <tr class="<?= $followed_team == $s['team'] ? 'active' : '' ?>">
                            <td><span class="pos pos-<?= $s['pos'] ?>"><?= $s['pos'] ?></span></td>
                            <td class="team-name"><?= $s['team'] ?></td>
                            <td><?= $s['s'] ?></td>
                            <td style="color: var(--primary); font-weight: 800;"><?= $s['p'] ?></td>
                            <td><?php foreach($s['form'] as $f) echo "<span class='form-dot form-$f'></span>"; ?></td>
                        </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- TAB: FIXTURES -->
        <div id="fixtures" class="tab-content">
            <div class="card">
                <div class="section-title">Spelschema</div>
                <?php foreach($fixtures as $f): ?>
                <div class="match-row">
                    <div class="match-date"><?= $f['date'] ?><br><small><?= $f['time'] ?></small></div>
                    <div class="match-teams">
                        <div style="display:flex; justify-content:space-between;"><span><?= $f['home'] ?></span> <span>-</span></div>
                        <div><?= $f['away'] ?></div>
                        <small style="color: var(--muted); font-weight: 400;"><i class="ri-map-pin-2-line"></i> <?= $f['venue'] ?></small>
                    </div>
                </div>
                <?php endforeach; ?>
            </div>
        </div>

        <!-- TAB: SQUAD -->
        <div id="squad" class="tab-content">
            <div class="card">
                <div class="section-header">
                    <div class="section-title">Truppen: <?= $followed_team ?></div>
                    <span style="font-size:0.7rem; color:var(--muted);">Totalvärde: ~€45m</span>
                </div>
                <?php if(empty($active_squad)): ?>
                    <p style="text-align:center; padding:20px; color:var(--muted);">Välj Malmö, AIK eller Hammarby för demo-trupp.</p>
                <?php else: ?>
                    <?php foreach($active_squad as $p):
                        $pts = []; $i = 0;
                        foreach($p['spider'] as $v) {
                            $a = $i * 72 * (M_PI / 180); $r = ($v / 100) * 18;
                            $pts[] = (20 + $r * cos($a)) . "," . (20 + $r * sin($a)); $i++;
                        }
                    ?>
                    <div class="player-card">
                        <div class="player-info">
                            <svg class="radar-box"><polygon points="<?= implode(' ', $pts) ?>" class="radar-poly" /></svg>
                            <div>
                                <div class="player-name"><?= $p['n'] ?> <?php if($p['xi']) echo "<span class='xi-badge'>START11</span>"; ?></div>
                                <div style="font-size: 0.7rem; color: var(--muted);"><?= $p['pos'] ?> • <?= $p['age'] ?> år</div>
                            </div>
                        </div>
                        <div style="text-align:right;">
                            <div class="val-pill"><?= $p['val'] ?></div>
                            <div style="font-size: 0.6rem; color: var(--gold); margin-top:3px;">Köp: <?= $p['buy'] ?></div>
                        </div>
                    </div>
                    <?php endforeach; ?>
                <?php endif; ?>
            </div>
        </div>

        <!-- TAB: NEWS -->
        <div id="news" class="tab-content">
            <div class="card">
                <div class="section-title">Live Nyheter</div>
                <?php foreach($news as $n): ?>
                <div style="margin-bottom: 20px; border-left: 3px solid var(--primary); padding-left: 15px;">
                    <div style="font-size:0.65rem; font-weight:800; color:var(--primary); margin-bottom:5px;"><?= $src ?> • <?= $n['date'] ?></div>
                    <a href="<?= $n['link'] ?>" target="_blank" style="text-decoration:none; color:#fff;"><h3 style="margin:0; font-size:1rem; line-height:1.4;"><?= $n['title'] ?></h3></a>
                </div>
                <?php endforeach; ?>
            </div>
        </div>
    </div>

    <!-- Navigation -->
    <nav class="bottom-nav">
        <a href="#" class="nav-item active" onclick="switchTab('standings', this)"><i class="ri-table-fill"></i><span>TABELL</span></a>
        <a href="#" class="nav-item" onclick="switchTab('fixtures', this)"><i class="ri-calendar-event-fill"></i><span>SCHEMA</span></a>
        <a href="#" class="nav-item" onclick="switchTab('squad', this)"><i class="ri-team-fill"></i><span>TRUPPEN</span></a>
        <a href="#" class="nav-item" onclick="switchTab('news', this)"><i class="ri-broadcast-fill"></i><span>NYHETER</span></a>
    </nav>

    <div id="teamOverlay" class="overlay">
        <div class="overlay-content">
            <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:25px;">
                <h2 style="margin:0; font-weight:900; color:#fff;">VÄLJ DITT LAG</h2>
                <i class="ri-close-circle-fill" onclick="toggleTeams()" style="font-size:2rem; color:var(--muted); cursor:pointer;"></i>
            </div>
            <div class="team-grid">
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
        function toggleTeams() { document.getElementById('teamOverlay').classList.toggle('active'); }
        function selectTeam(name) { window.location.href = '?team=' + encodeURIComponent(name); }
    </script>
</body>
</html>
