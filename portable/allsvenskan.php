<?php
/**
 * Allsvenskan Aggregator - Portable Version for One.com
 */

function fetch_data($url) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true);
    curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36');
    curl_setopt($ch, CURLOPT_TIMEOUT, 10);
    $data = curl_exec($ch);
    curl_close($ch);
    return $data;
}

// Fetch News (RSS)
$news_xml = fetch_data("https://allsvenskan.se/feed/");
$news = [];
if ($news_xml) {
    $xml = simplexml_load_string($news_xml);
    foreach ($xml->channel->item as $item) {
        $news[] = [
            'title' => (string)$item->title,
            'link' => (string)$item->link,
            'desc' => strip_tags((string)$item->description),
            'date' => date('j M, H:i', strtotime((string)$item->pubDate))
        ];
        if (count($news) >= 10) break;
    }
}

// Fetch Stats (Cards)
$cards_html = fetch_data("https://www.worldfootball.net/players_yellow_red/swe-allsvenskan-2024/");
$cards = [];
if ($cards_html) {
    preg_match_all('/(?s)<tr>\s*<td>(\d+)<\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*>(\d+)<\/td>\s*<td[^>]*>(\d+)<\/td>\s*<td[^>]*>(\d+)<\/td>/', $cards_html, $matches, PREG_SET_ORDER);
    foreach ($matches as $m) {
        $cards[] = ['name' => $m[2], 'team' => $m[3], 'yellow' => $m[4], 'red' => $m[6]];
        if (count($cards) >= 10) break;
    }
}

// Fetch Matches
$matches_html = fetch_data("https://www.worldfootball.net/all_matches/swe-allsvenskan-2024/");
$matches_list = [];
if ($matches_html) {
    preg_match_all('/(?s)<tr>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*>[^<]*<\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*-\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>/', $matches_html, $matches, PREG_SET_ORDER);
    foreach ($matches as $m) {
        $matches_list[] = ['date' => $m[1], 'home' => $m[2], 'away' => $m[3], 'result' => $m[4]];
        if (count($matches_list) >= 10) break;
    }
}

// Fetch Stats (Scorers)
$stats_html = fetch_data("https://www.worldfootball.net/goalgetter/swe-allsvenskan-2024/");
$scorers = [];
if ($stats_html) {
    preg_match_all('/(?s)<tr>\s*<td>(\d+)<\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*>(\d+)/', $stats_html, $matches, PREG_SET_ORDER);
    foreach ($matches as $m) {
        $scorers[] = ['name' => $m[2], 'team' => $m[3], 'goals' => $m[4]];
        if (count($scorers) >= 10) break;
    }
}

// Fetch Table & Stats (WorldFootball as reliable source)
$table_html = fetch_data("https://www.worldfootball.net/table/swe-allsvenskan-2024/");
$table = [];
if ($table_html) {
    preg_match_all('/(?s)<tr>\s*<td[^>]*>(\d+)\.<\/td>\s*<td[^>]*><a[^>]*>([^<]+)<\/a><\/td>\s*<td[^>]*>(\d+)<\/td>\s*<td[^>]*>(\d+)<\/td>\s*<td[^>]*>(\d+)<\/td>\s*<td[^>]*>(\d+)<\/td>\s*<td[^>]*>([^<]+)<\/td>\s*<td[^>]*>([^<]+)<\/td>\s*<td[^>]*><b>(\d+)<\/b><\/td>/', $table_html, $matches, PREG_SET_ORDER);
    foreach ($matches as $m) {
        $table[] = [
            'pos' => $m[1],
            'team' => $m[2],
            'played' => $m[3],
            'wins' => $m[4],
            'draws' => $m[5],
            'losses' => $m[6],
            'goals' => $m[8],
            'points' => $m[9]
        ];
    }
}

?>
<!DOCTYPE html>
<html lang="sv">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Allsvenskan Hub</title>
    <style>
        :root { --bg: #0f172a; --card: #1e293b; --text: #f8fafc; --muted: #94a3b8; --accent: #38bdf8; --border: #334155; }
        body { background: var(--bg); color: var(--text); font-family: system-ui, sans-serif; margin: 0; padding: 20px; line-height: 1.5; }
        .container { max-width: 1200px; margin: 0 auto; }
        h1 { color: var(--accent); font-size: 2.5rem; }
        .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem; }
        @media (max-width: 900px) { .grid { grid-template-columns: 1fr; } }
        .card { background: var(--card); padding: 1.5rem; border-radius: 12px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); margin-bottom: 2rem; }
        h2 { border-bottom: 1px solid var(--border); padding-bottom: 0.5rem; color: var(--muted); }
        table { width: 100%; border-collapse: collapse; font-size: 0.9rem; }
        th, td { text-align: left; padding: 10px; border-bottom: 1px solid var(--border); }
        th { color: var(--muted); }
        .points { font-weight: bold; color: var(--accent); }
        .news-item { margin-bottom: 1.5rem; border-bottom: 1px solid var(--border); padding-bottom: 1rem; }
        .news-item a { text-decoration: none; color: inherit; }
        .news-item h3 { margin: 0 0 5px 0; font-size: 1.1rem; color: var(--text); transition: 0.2s; }
        .news-item a:hover h3 { color: var(--accent); }
        .date { font-size: 0.75rem; color: var(--muted); display: block; }
        .news-item p { font-size: 0.9rem; color: var(--muted); margin: 5px 0 0 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Allsvenskan</h1>
        <div class="grid">
            <div class="card">
                <h2>Tabell</h2>
                <table>
                    <thead>
                        <tr><th>#</th><th>Lag</th><th>S</th><th>V</th><th>O</th><th>F</th><th>Mål</th><th>P</th></tr>
                    </thead>
                    <tbody>
                        <?php foreach($table as $t): ?>
                        <tr>
                            <td><?= $t['pos'] ?></td>
                            <td><strong><?= $t['team'] ?></strong></td>
                            <td><?= $t['played'] ?></td>
                            <td><?= $t['wins'] ?></td>
                            <td><?= $t['draws'] ?></td>
                            <td><?= $t['losses'] ?></td>
                            <td><?= $t['goals'] ?></td>
                            <td class="points"><?= $t['points'] ?></td>
                        </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>
            <div class="card">
                <h2>Skytteliga</h2>
                <table>
                    <thead>
                        <tr><th>Spelare</th><th>Lag</th><th>Mål</th></tr>
                    </thead>
                    <tbody>
                        <?php foreach($scorers as $s): ?>
                        <tr>
                            <td><?= $s['name'] ?></td>
                            <td><?= $s['team'] ?></td>
                            <td class="points"><?= $s['goals'] ?></td>
                        </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>

            <div class="card">
                <h2>Senaste Matcherna</h2>
                <table>
                    <thead>
                        <tr><th>Datum</th><th>Match</th><th>Resultat</th></tr>
                    </thead>
                    <tbody>
                        <?php foreach($matches_list as $m): ?>
                        <tr>
                            <td><small><?= $m['date'] ?></small></td>
                            <td><?= $m['home'] ?> - <?= $m['away'] ?></td>
                            <td class="points"><?= $m['result'] ?></td>
                        </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>

            <div class="card">
                <h2>Kortliga</h2>
                <table>
                    <thead>
                        <tr><th>Spelare</th><th>Lag</th><th>Gula</th><th>Röda</th></tr>
                    </thead>
                    <tbody>
                        <?php foreach($cards as $c): ?>
                        <tr>
                            <td><?= $c['name'] ?></td>
                            <td><?= $c['team'] ?></td>
                            <td><?= $c['yellow'] ?></td>
                            <td class="points" style="color: #ef4444;"><?= $c['red'] ?></td>
                        </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            </div>

            <div class="card">
                <h2>Senaste Nyheterna</h2>
                <?php foreach($news as $n): ?>
                <div class="news-item">
                    <a href="<?= $n['link'] ?>" target="_blank">
                        <span class="date"><?= $n['date'] ?></span>
                        <h3><?= $n['title'] ?></h3>
                    </a>
                    <p><?= $n['desc'] ?></p>
                </div>
                <?php endforeach; ?>
            </div>
        </div>
    </div>
</body>
</html>
