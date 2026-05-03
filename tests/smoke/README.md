# Smoke tests — PiHolster

Integration tests that run against a physical Pi 3 over the network.

## Prerequisites

- Go 1.22+
- Pi running `piholsterd` with port 80 (HTTP) and 53 (DNS) accessible from this machine
- `go mod download` run inside `tests/smoke/`

## Environment variables

| Variable           | Required | Default | Description                              |
|--------------------|----------|---------|------------------------------------------|
| `PI_IP`            | yes      | —       | IP address of the target Pi              |
| `SMOKE_TIMEOUT`    | no       | `30s`   | Per-test HTTP/DNS client timeout         |
| `BOOT_TIMEOUT`     | no       | `90s`   | Max wait time in TestBootTime            |
| `SKIP_FIREWALL_TEST` | no     | —       | Set to `1` to skip TestFirewallPreFirstboot |

## Running

```sh
cd tests/smoke
go mod download

PI_IP=192.168.1.50 go test -v -timeout 120s ./...
```

With custom timeouts:

```sh
PI_IP=192.168.1.50 SMOKE_TIMEOUT=15s BOOT_TIMEOUT=120s go test -v -timeout 150s ./...
```

Skip the firewall test when boot timing is uncertain:

```sh
PI_IP=192.168.1.50 SKIP_FIREWALL_TEST=1 go test -v -timeout 120s ./...
```

## Tests

| Test                      | What it checks                                                  |
|---------------------------|-----------------------------------------------------------------|
| `TestBootTime`            | Polls `/api/health` every 2 s, fails if no 200 within BOOT_TIMEOUT |
| `TestFirewallPreFirstboot`| TCP connect to :80 must not time out (silent-drop detection)    |
| `TestDNSLatency`          | 20 DNS queries via miekg/dns, median must be <= 20 ms          |
| `TestRAMUsage`            | Reads `ram_used_mb` from `/api/health`, skips if field absent   |
| `TestWebUIResponds`       | GET `/` must return 200 with `<html` in body                    |
| `TestAPIHealth`           | GET `/api/health` must return 200 with `status:ok` or `dns_running:true` |

## Notes

- All tests share a single `http.Client` with `InsecureSkipVerify=true` to handle self-signed TLS certificates.
- If `PI_IP` is not set, `TestMain` exits with code 0 (no failure) so CI pipelines without a Pi are unaffected.
