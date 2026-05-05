#!/usr/bin/env bash
# soak-monitor.sh — sample memory and process state for the v0.1.0 soak (US-19).
#
# Runs once per invocation. Designed to be triggered every 5 minutes via cron:
#
#     */5 * * * * /usr/local/bin/soak-monitor.sh >> /var/log/piholster/soak.cron.log 2>&1
#
# Writes one CSV row per call to /var/log/piholster/soak.csv. Header is written
# automatically on first invocation.
#
# Columns:
#   ts                 RFC3339 timestamp (local time)
#   uptime_s           Pi uptime in seconds
#   mem_total_mb       /proc/meminfo MemTotal
#   mem_available_mb   /proc/meminfo MemAvailable (preferred over MemFree)
#   mem_used_mb        total - available
#   piholsterd_rss_mb  RSS of the piholsterd process, 0 if not running
#   arpd_rss_mb        RSS of the piholster-arpd process, 0 if not running
#   piholsterd_pid     PID of piholsterd, 0 if not running
#   arpd_pid           PID of piholster-arpd, 0 if not running
#   dns_queries_24h    optional, requires sqlite3 + read access to piholster.db
#
# Exit code is always 0 unless the log directory cannot be created. Cron should
# never see a failure for transient issues (process restarts, db locked, etc.).

set -u

LOG_DIR="${SOAK_LOG_DIR:-/var/log/piholster}"
LOG_FILE="${LOG_DIR}/soak.csv"
DB_PATH="${DB_PATH:-/var/lib/piholster/piholster.db}"

mkdir -p "${LOG_DIR}" || {
    echo "soak-monitor: cannot create ${LOG_DIR}" >&2
    exit 1
}

if [[ ! -f "${LOG_FILE}" ]]; then
    echo "ts,uptime_s,mem_total_mb,mem_available_mb,mem_used_mb,piholsterd_rss_mb,arpd_rss_mb,piholsterd_pid,arpd_pid,dns_queries_24h" > "${LOG_FILE}"
fi

ts="$(date --iso-8601=seconds)"

# Uptime in seconds (integer).
uptime_s="$(awk '{print int($1)}' /proc/uptime)"

# Memory: read MemTotal and MemAvailable from /proc/meminfo (kB).
mem_total_kb="$(awk '/^MemTotal:/ {print $2}' /proc/meminfo)"
mem_avail_kb="$(awk '/^MemAvailable:/ {print $2}' /proc/meminfo)"
mem_total_mb=$(( mem_total_kb / 1024 ))
mem_avail_mb=$(( mem_avail_kb / 1024 ))
mem_used_mb=$(( mem_total_mb - mem_avail_mb ))

# Process RSS in MB. pgrep -x matches exact name. /proc/<pid>/status VmRSS is in kB.
rss_mb_for() {
    local name="$1"
    local pid
    pid="$(pgrep -x "${name}" | head -n 1)"
    if [[ -z "${pid}" ]]; then
        echo "0 0"
        return
    fi
    local rss_kb
    rss_kb="$(awk '/^VmRSS:/ {print $2}' "/proc/${pid}/status" 2>/dev/null)"
    if [[ -z "${rss_kb}" ]]; then
        echo "0 ${pid}"
        return
    fi
    echo "$(( rss_kb / 1024 )) ${pid}"
}

read -r piholsterd_rss piholsterd_pid < <(rss_mb_for piholsterd)
read -r arpd_rss arpd_pid < <(rss_mb_for piholster-arpd)

# DNS query count over the last 24 h. Best-effort — empty if sqlite3 missing
# or db unreadable. Soak still passes/fails based on RAM, not on this number.
dns_queries_24h=""
if command -v sqlite3 >/dev/null 2>&1 && [[ -r "${DB_PATH}" ]]; then
    dns_queries_24h="$(sqlite3 -readonly "${DB_PATH}" \
        "SELECT COUNT(*) FROM query_log WHERE queried_at > datetime('now','-1 day');" \
        2>/dev/null || true)"
fi

printf "%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n" \
    "${ts}" \
    "${uptime_s}" \
    "${mem_total_mb}" \
    "${mem_avail_mb}" \
    "${mem_used_mb}" \
    "${piholsterd_rss}" \
    "${arpd_rss}" \
    "${piholsterd_pid}" \
    "${arpd_pid}" \
    "${dns_queries_24h}" \
    >> "${LOG_FILE}"
