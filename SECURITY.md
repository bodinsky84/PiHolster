# Security Policy

## Supported versions

Only the latest release receives security fixes.

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |
| < 0.1   | No        |

## Reporting a vulnerability

Email **bodinsmail@gmail.com** with subject `[PiHolster Security]`.

Include: affected version, description, reproduction steps, and — if known — a
suggested fix or patch.  You will receive an acknowledgement within 48 hours and
a status update within 7 days.

Please do **not** open a public GitHub issue for security vulnerabilities until
a fix has been prepared and coordinated.

## Known issues

The following issues were identified during the IT-security review prior to v0.1.0
and are accepted risks with documented mitigations and fix targets.

### M-05 — tmpfs password file permissions (Medium)

**Component:** `piholster-firstboot.service`  
**Affected versions:** v0.1.0  
**Fix target:** v0.1.1

**Description:**  
The one-time admin password written to `/run/piholster/firstboot-password` on the
tmpfs is created with mode `0640` (owner read/write, group read).  The intended
mode is `0400` (owner read-only, no group access).

**Impact:**  
Any process running as the `piholster` group can read the plaintext password
from the tmpfs file during the short window between firstboot completion and the
user reading/clearing the file.  On a default Raspberry Pi OS install, no other
service runs under the `piholster` group, so exploitation requires a prior local
privilege escalation.

**Mitigation (v0.1.0):**  
- tmpfs is mounted `noexec,nosuid,nodev` and is not accessible remotely.
- The firstboot service exits immediately after writing the password; the file
  lifetime is bounded by when the administrator reads and clears it.
- Monitor `/run/piholster/` access via `auditd` if the threat model requires it.

**Fix (v0.1.1):**  
Change `os.WriteFile` call in `firstboot/main.go` to use `0400` as the permission
argument, and add a `chmod 0400` as an explicit step after write to cover umask
edge cases.

---

### L-03 — LockPersonality missing from firstboot service (Low)

**Component:** `piholster-firstboot.service` systemd unit  
**Affected versions:** v0.1.0  
**Fix target:** v0.1.1

**Description:**  
The `[Service]` section of `piholster-firstboot.service` does not include
`LockPersonality=yes`.  All other PiHolster services have this directive.
`LockPersonality` prevents a process from changing its execution domain (ABI),
closing a niche but well-understood hardening gap.

**Impact:**  
Low.  The firstboot service is short-lived (runs once at first boot, then is
disabled), runs as a dedicated non-root user, and has no network exposure.
Practical exploitability is negligible; this is a defense-in-depth gap only.

**Mitigation (v0.1.0):**  
The service is disabled by the system after its first successful run
(`RemainAfterExit=no` + `ConditionPathExists=!/run/piholster/firstboot.done`),
minimising the attack surface window.

**Fix (v0.1.1):**  
Add `LockPersonality=yes` to the `[Service]` section of
`piholster-firstboot.service`.

---

## Security architecture summary

- All traffic served over HTTPS (self-signed certificate generated at image build).
- HTTP port 80 issues a `301 Moved Permanently` redirect to HTTPS.
- Strict CSP: `script-src 'self'; style-src 'self'` — no `'unsafe-inline'`, no nonces.
- DNS-rebinding protection: `AllowedHosts` middleware returns `421` for unrecognised
  `Host` headers.
- Admin session tokens are stored hashed in SQLite with WAL mode for crash safety.
- Firewall (ufw) default-deny; allows 53/udp, 80/tcp, 443/tcp from LAN only.
- Build reproducibility: `image/build.sh` generates a `MANIFEST` with SHA-256
  checksums for all installed binaries and configuration files.
