# Security policy

Pigura parses untrusted media (via libvips, libheif and ffmpeg) and can expose files through share links,
so please report vulnerabilities privately.

**Report via GitHub:** *Security* tab → *Report a vulnerability* (private advisory). Please do not open public
issues for security problems. Include affected version, reproduction steps and impact.

Hardening notes for operators:

- Set `PIGURA_TOKEN` (long random string) unless Pigura is only reachable on `127.0.0.1`.
- Put a TLS reverse proxy in front for anything beyond your LAN, and list its hostname in `PIGURA_ALLOWED_HOSTS`.
- Mount your library read-only and run as a non-root user (see `docker-compose.yml`, `deploy/pigura.service`).
- Expose only the public share listener (`PIGURA_PUBLIC_ADDR`) to the internet, never the main one.
