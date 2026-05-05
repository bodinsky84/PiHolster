#!/usr/bin/env bash
# soak-analyze.sh — analyze a soak.csv produced by soak-monitor.sh and decide
# whether the v0.1.0 7-day soak (US-19) passes the GA-gate criteria.
#
# Acceptance threshold (docs/SPRINT-4.md US-19 AC-3):
#   piholsterd RSS may grow at most 5 MB per 24 h on average over the period.
#
# Usage:
#   bash scripts/soak-analyze.sh /path/to/soak.csv
#   bash scripts/soak-analyze.sh /path/to/soak.csv --json   (machine-readable)
#
# Exit codes:
#   0  PASS — soak meets US-19 AC-3 and AC-4 criteria
#   1  NOK  — at least one criterion fails (memory leak, restarts, or short)
#   2  input error (file missing, malformed)

set -uo pipefail

CSV="${1:-}"
JSON_MODE=0
if [[ "${2:-}" == "--json" ]]; then
    JSON_MODE=1
fi

if [[ -z "${CSV}" ]]; then
    echo "Usage: $0 <soak.csv> [--json]" >&2
    exit 2
fi
if [[ ! -f "${CSV}" ]]; then
    echo "soak-analyze: '${CSV}' does not exist" >&2
    exit 2
fi

# Validate header — must match what soak-monitor.sh writes.
expected_header="ts,uptime_s,mem_total_mb,mem_available_mb,mem_used_mb,piholsterd_rss_mb,arpd_rss_mb,piholsterd_pid,arpd_pid,dns_queries_24h"
actual_header="$(head -n 1 "${CSV}")"
if [[ "${actual_header}" != "${expected_header}" ]]; then
    echo "soak-analyze: CSV header mismatch" >&2
    echo "  expected: ${expected_header}" >&2
    echo "  got:      ${actual_header}" >&2
    exit 2
fi

# Sample count (excluding header).
n_samples="$(($(wc -l < "${CSV}") - 1))"
if (( n_samples < 100 )); then
    echo "soak-analyze: only ${n_samples} samples — soak is too short to draw conclusions" >&2
    echo "  expected ~2016 samples for a 7-day run at 5-min intervals" >&2
    exit 1
fi

# All math is done in awk to avoid float-handling pain in pure bash.
# We compute:
#   - first/last timestamps (epoch seconds), duration in hours
#   - piholsterd_rss min/max/mean
#   - linear regression slope of piholsterd_rss against time-in-hours
#   - count of samples where piholsterd_pid changed (a proxy for restarts)
#   - count of samples where piholsterd_pid==0 (process was missing)
read -r duration_h \
        rss_min rss_max rss_mean \
        slope_mb_per_day \
        pid_changes pid_zero_samples \
        first_ts last_ts \
        first_pid last_pid \
        < <(awk -F, '
NR == 1 { next }
{
    # parse RFC3339 timestamp via "date -d" is too slow per-row; instead approximate
    # by trusting that uptime_s is monotonic within a single boot. For elapsed
    # hours we use uptime difference; for a long soak with no restarts this is
    # exactly equivalent to wallclock elapsed.
    ts[NR-1] = $1
    uptime[NR-1] = $2 + 0
    rss[NR-1] = $6 + 0
    pid[NR-1] = $8 + 0
    n = NR - 1
}
END {
    # Duration in hours, derived from uptime delta of the first vs last sample.
    # If the Pi rebooted during the soak, uptime is non-monotonic — handle below.
    duration_s = uptime[n] - uptime[1]
    duration_h = duration_s / 3600.0

    # Linear regression: rss = a + b * t, where t is hours since first sample.
    # Closed form: b = (n*Sxy - Sx*Sy) / (n*Sxx - Sx*Sx).
    sumX = 0; sumY = 0; sumXY = 0; sumXX = 0
    rmin = rss[1]; rmax = rss[1]; rsum = 0
    pid_changes = 0
    pid_zero = 0
    last_seen_pid = pid[1]
    for (i = 1; i <= n; i++) {
        t_h = (uptime[i] - uptime[1]) / 3600.0
        sumX  += t_h
        sumY  += rss[i]
        sumXY += t_h * rss[i]
        sumXX += t_h * t_h
        if (rss[i] < rmin) rmin = rss[i]
        if (rss[i] > rmax) rmax = rss[i]
        rsum += rss[i]
        if (pid[i] == 0) pid_zero++
        if (pid[i] != 0 && pid[i] != last_seen_pid && last_seen_pid != 0) {
            pid_changes++
        }
        if (pid[i] != 0) last_seen_pid = pid[i]
    }
    rmean = rsum / n
    denom = (n * sumXX - sumX * sumX)
    if (denom > 0) {
        slope_per_h = (n * sumXY - sumX * sumY) / denom
    } else {
        slope_per_h = 0
    }
    slope_per_day = slope_per_h * 24.0

    printf "%.2f %d %d %.2f %.3f %d %d %s %s %d %d\n",
        duration_h, rmin, rmax, rmean,
        slope_per_day, pid_changes, pid_zero,
        ts[1], ts[n], pid[1], pid[n]
}
' "${CSV}")

# Decide verdict.
verdict_memory="PASS"
if awk "BEGIN { exit !(${slope_mb_per_day} > 5.0) }"; then
    verdict_memory="NOK"
fi

verdict_stability="PASS"
if (( pid_changes > 0 )); then
    verdict_stability="NOK"
fi
# pid_zero_samples are tolerated up to a single 5-min sampling gap (i.e. <= 1 sample).
if (( pid_zero_samples > 1 )); then
    verdict_stability="NOK"
fi

verdict_duration="PASS"
duration_short=0
if awk "BEGIN { exit !(${duration_h} < 168.0) }"; then
    verdict_duration="NOK"
    duration_short=1
fi

overall="PASS"
if [[ "${verdict_memory}" != "PASS" || "${verdict_stability}" != "PASS" || "${verdict_duration}" != "PASS" ]]; then
    overall="NOK"
fi

if (( JSON_MODE == 1 )); then
    cat <<EOF
{
  "verdict": "${overall}",
  "memory": {
    "verdict": "${verdict_memory}",
    "rss_min_mb": ${rss_min},
    "rss_max_mb": ${rss_max},
    "rss_mean_mb": ${rss_mean},
    "slope_mb_per_day": ${slope_mb_per_day},
    "threshold_mb_per_day": 5.0
  },
  "stability": {
    "verdict": "${verdict_stability}",
    "pid_changes": ${pid_changes},
    "samples_without_process": ${pid_zero_samples}
  },
  "duration": {
    "verdict": "${verdict_duration}",
    "hours": ${duration_h},
    "samples": ${n_samples},
    "first_ts": "${first_ts}",
    "last_ts": "${last_ts}"
  }
}
EOF
else
    cat <<EOF
=== Soak analysis report (US-19) ===

Input:    ${CSV}
Samples:  ${n_samples}
Period:   ${first_ts} -> ${last_ts}  (${duration_h} h)

Memory (piholsterd RSS):
  min:    ${rss_min} MB
  max:    ${rss_max} MB
  mean:   ${rss_mean} MB
  slope:  ${slope_mb_per_day} MB/day  (threshold: 5.0 MB/day)
  -> ${verdict_memory}

Stability (piholsterd process):
  PID changes:        ${pid_changes}    (expected: 0)
  Samples no process: ${pid_zero_samples}    (expected: 0–1)
  -> ${verdict_stability}

Duration:
  ${duration_h} h elapsed  (expected: >= 168 h for 7-day soak)
  -> ${verdict_duration}

Overall: ${overall}
EOF
fi

if [[ "${overall}" == "PASS" ]]; then
    exit 0
else
    exit 1
fi
