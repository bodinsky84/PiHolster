<?php
/**
 * Allsvenskan Aggregator - Final Optimized Version
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

// Allsvenskan 2024/2025 Teams (Official)
$all_teams = [
    "AIK", "BK Häcken", "Djurgården", "GAIS", "Halmstads BK", "Hammarby",
    "IF Brommapojkarna", "IF Elfsborg", "IFK Göteborg", "IFK Norrköping",
    "IFK Värnamo", "IK Sirius", "Kalmar FF", "Malmö FF", "Mjällby AIF", "Västerås SK"
];

// Fetch News from multiple sources
$feeds = [
    'Allsvenskan' => "https://allsvenskan.se/feed/",
    'Expressen' => "https://www.expressen.se/rss/sport/fotboll/allsvenskan/"
];

$all_news = [];
foreach ($feeds as $source => $url) {
    $xml_data = fetch_data($url);
    if ($xml_data) {
        $xml = @simplexml_load_string($xml_data);
        if ($xml) {
            foreach ($xml->channel->item as $item) {
                $all_news[] = [
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

usort($all_news, function($a, $b) { return $b['timestamp'] - $a['timestamp']; });
$news = array_slice($all_news, 0, 20);

// For the table, since Cloudflare blocks simple scrapers on most sports sites,
// we use a static initial set but the "Follow" logic works on everything.
$table = [
    ['pos'=>1, 'team'=>'Malmö FF', 'played'=>30, 'wins'=>19, 'draws'=>8, 'losses'=>3, 'gd'=>51, 'points'=>65],
    ['pos'=>2, 'team'=>'Hammarby', 'played'=>30, 'wins'=>16, 'draws'=>6, 'losses'=>8, 'gd'=>23, 'points'=>54],
    ['pos'=>3, 'team'=>'AIK', 'played'=>30, 'wins'=>17, 'draws'=>3, 'losses'=>10, 'gd'=>5, 'points'=>54],
    ['pos'=>4, 'team'=>'Djurgården', 'played'=>30, 'wins'=>16, 'draws'=>5, 'losses'=>9, 'gd'=>10, 'points'=>53],
    ['pos'=>5, 'team'=>'Mjällby AIF', 'played'=>30, 'wins'=>14, 'draws'=>8, 'losses'=>8, 'gd'=>11, 'points'=>50],
    ['pos'=>6, 'team'=>'GAIS', 'played'=>30, 'wins'=>14, 'draws'=>6, 'losses'=>10, 'gd'=>2, 'points'=>48],
    ['pos'=>7, 'team'=>'IF Elfsborg', 'played'=>30, 'wins'=>13, 'draws'=>6, 'losses'=>11, 'gd'=>8, 'points'=>45],
    ['pos'=>8, 'team'=>'BK Häcken', 'played'=>30, 'wins'=>12, 'draws'=>6, 'losses'=>12, 'gd'=>3, 'points'=>42],
    ['pos'=>9, 'team'=>'IK Sirius', 'played'=>30, 'wins'=>12, 'draws'=>5, 'losses'=>13, 'gd'=>1, 'points'=>41],
    ['pos'=>10, 'team'=>'IFK Göteborg', 'played'=>30, 'wins'=>8, 'draws'=>10, 'losses'=>12, 'gd'=>-10, 'points'=>34],
    ['pos'=>11, 'team'=>'IFK Norrköping', 'played'=>30, 'wins'=>9, 'draws'=>7, 'losses'=>14, 'gd'=>-21, 'points'=>34],
    ['pos'=>12, 'team'=>'IF Brommapojkarna', 'played'=>30, 'wins'=>8, 'draws'=>10, 'losses'=>12, 'gd'=>-7, 'points'=>34],
    ['pos'=>13, 'team'=>'IFK Värnamo', 'played'=>30, 'wins'=>7, 'draws'=>10, 'losses'=>13, 'gd'=>-10, 'points'=>31],
    ['pos'=>14, 'team'=>'Halmstads BK', 'played'=>30, 'wins'=>10, 'draws'=>3, 'losses'=>17, 'gd'=>-18, 'points'=>33],
    ['pos'=>15, 'team'=>'Kalmar FF', 'played'=>30, 'wins'=>8, 'draws'=>6, 'losses'=>16, 'gd'=>-20, 'points'=>30],
    ['pos'=>16, 'team'=>'Västerås SK', 'played'=>30, 'wins'=>6, 'draws'=>5, 'losses'=>19, 'gd'=>-17, 'points'=>23]
];

$followed_team = isset($_GET['team']) ? $_GET['team'] : '';

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Allsvenskan Hub - <?= $followed_team ?: 'Alla Lag' ?></title>
    <style>
        :root { --bg: #0f172a; --card: #1e293b; --text: #f8fafc; --muted: #94a3b8; --accent: #38bdf8; --border: #334155; --highlight: #2d6a4f; }
        body { background: var(--bg); color: var(--text); font-family: system-ui, sans-serif; margin: 0; padding: 10px; line-height: 1.5; }
        .container { max-width: 1200px; margin: 0 auto; }
        header { background: #020617; padding: 20px; border-radius: 12px; margin-bottom: 2rem; display: flex; flex-direction: column; gap: 1rem; }
        h1 { color: var(--accent); margin: 0; font-size: 1.8rem; }
        .team-selector { width: 100%; max-width: 300px; background: var(--card); color: var(--text); border: 1px solid var(--border); padding: 12px; border-radius: 8px; font-size: 1rem; }
        .grid { display: grid; grid-template-columns: 1.5fr 1fr; gap: 1.5rem; }
        @media (max-width: 1000px) { .grid { grid-template-columns: 1fr; } }
        .card { background: var(--card); padding: 1.5rem; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.2); }
        h2 { border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; color: var(--accent); margin-top: 0; font-size: 1.3rem; }
        .table-wrapper { overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
        th, td { text-align: left; padding: 12px 8px; border-bottom: 1px solid var(--border); }
        th { color: var(--muted); text-transform: uppercase; font-size: 0.7rem; }
        tr.highlight { background: var(--highlight); }
        .points { font-weight: bold; color: var(--accent); }
        .news-item { margin-bottom: 1.5rem; border-bottom: 1px solid var(--border); padding-bottom: 1rem; }
        .news-item a { text-decoration: none; color: inherit; }
        .news-item h3 { margin: 0 0 5px 0; font-size: 1.1rem; color: var(--text); }
        .news-item a:hover h3 { color: var(--accent); }
        .meta { display: flex; gap: 10px; font-size: 0.7rem; margin-bottom: 5px; }
        .source { background: #334155; padding: 2px 8px; border-radius: 99px; color: #fff; font-weight: bold; }
        .date { color: var(--muted); }
        .news-item p { font-size: 0.9rem; color: var(--muted); margin: 8px 0 0 0; }
        .btn-reset { color: var(--muted); text-decoration: none; font-size: 0.8rem; margin-top: 5px; display: inline-block; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <div>
                <h1>Allsvenskan Hub</h1>
                <p style="margin: 5px 0 0 0; color: var(--muted); font-size: 0.9rem;">Senaste nytt och aktuell tabell</p>
            </div>
            <form method="GET">
                <select name="team" class="team-selector" onchange="this.form.submit()">
                    <option value="">Följ ett specifikt lag...</option>
                    <?php foreach($all_teams as $t_name): ?>
                        <option value="<?= $t_name ?>" <?= $followed_team == $t_name ? 'selected' : '' ?>><?= $t_name ?></option>
                    <?php endforeach; ?>
                </select>
                <?php if($followed_team): ?>
                    <br><a href="index.php" class="btn-reset">Visa alla nyheter &times;</a>
                <?php endif; ?>
            </form>
        </header>

        <div class="grid">
            <div class="card">
                <h2>Tabell 2024</h2>
                <div class="table-wrapper">
                    <table>
                        <thead>
                            <tr><th>#</th><th>Lag</th><th>S</th><th>V</th><th>O</th><th>F</th><th>+/-</th><th>P</th></tr>
                        </thead>
                        <tbody>
                            <?php foreach($table as $t): ?>
                            <tr class="<?= ($followed_team && strpos($t['team'], $followed_team) !== false) ? 'highlight' : '' ?>">
                                <td><?= $t['pos'] ?></td>
                                <td><strong><?= $t['team'] ?></strong></td>
                                <td><?= $t['played'] ?></td>
                                <td><?= $t['wins'] ?></td>
                                <td><?= $t['draws'] ?></td>
                                <td><?= $t['losses'] ?></td>
                                <td><?= $t['gd'] ?></td>
                                <td class="points"><?= $t['points'] ?></td>
                            </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>
            </div>

            <div class="card">
                <h2>Nyhetsflöde <?= $followed_team ? " för $followed_team" : "" ?></h2>
                <?php
                $count = 0;
                foreach($news as $n):
                    $show = true;
                    if ($followed_team) {
                        // Better matching logic for team names in titles/desc
                        $search = strtolower($followed_team);
                        if (strpos(strtolower($n['title'] . $n['desc']), $search) === false) {
                            $show = false;
                        }
                    }
                    if ($show):
                        $count++;
                ?>
                <div class="news-item">
                    <div class="meta">
                        <span class="source"><?= $n['source'] ?></span>
                        <span class="date"><?= $n['date'] ?></span>
                    </div>
                    <a href="<?= $n['link'] ?>" target="_blank">
                        <h3><?= $n['title'] ?></h3>
                    </a>
                    <p><?= $n['desc'] ?></p>
                </div>
                <?php
                    endif;
                endforeach;
                if ($count == 0):
                    echo "<div style='padding: 20px; text-align: center; color: var(--muted);'>";
                    echo "<p>Inga specifika nyheter hittades för <strong>$followed_team</strong> just nu.</p>";
                    echo "<p><a href='index.php' style='color: var(--accent)'>Se alla nyheter istället</a></p>";
                    echo "</div>";
                endif;
                ?>
            </div>
        </div>
    </div>
</body>
</html>
