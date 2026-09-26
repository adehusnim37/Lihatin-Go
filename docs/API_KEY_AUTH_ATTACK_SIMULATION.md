# API key authentication attack simulation

Run on 2026-09-26 against the local Redis-backed authentication gate and Gin
middleware. No production API or third-party host received test traffic.

Run the repeatable checks with:

```sh
go test ./middleware -run 'TestAPIKeyAuth' -count=1 -v
go test ./internal/pkg/auth -run '^$' -bench '^BenchmarkValidateAPISecretKey$' -benchtime=20x -count=1
```

The test runner needs permission to create a local Redis Unix socket. A test
reported as `SKIP` is **not** evidence that its scenario passed.

| Scenario | Local observation | Meaning |
| --- | --- | --- |
| 15 sequential failures, one IP and key ID | 10 admitted, 5 rejected | The 11th and later attempts get HTTP 429. The HTTP test also verifies `Retry-After`. |
| 125 different key IDs, one IP | 120 admitted, 5 rejected | Rotating key IDs bypasses the per-pair failure counter until the IP-wide cap. |
| 120 different key IDs from each of two IPs | 240 admitted | Limits are per IP; distributed sources scale the permitted work. |
| 40 simultaneous failures, one IP and key ID | All 40 admitted before failures were recorded | The failure counter is checked before authentication and incremented afterward. Concurrent attempts can overshoot the nominal 10-failure threshold; the 120/minute IP cap remains the upper bound for one IP during that window. |
| Missing API key through real Gin middleware | First 10 requests return 401; requests 11–12 return 429 with `Retry-After` | The HTTP response matches the guard's counters. |
| Forged `X-Forwarded-For` from untrusted peer | Ignored in Gin test | This requires production `TRUSTED_PROXY_CIDRS` to contain only real proxies. |
| Redis unavailable | Guard returns an error instead of admitting; HTTP test returns 503 | Authentication fails closed, trading availability for protection. |
| Header longer than 256 bytes | HTTP 400 before repository lookup | Oversized credentials avoid the database and PBKDF2 path. |

On the local Apple M3 Pro, 20 wrong-secret PBKDF2-SHA512 checks averaged
about **21 ms each**. This measures the KDF only, not database latency or the
deployed server's CPU. Unknown well-formed key IDs also execute a dummy KDF,
so rotating IDs can consume CPU even without a real key.

## Assessment

The current guard slows simple sequential guessing, but it is **not a complete
DoS defense**. A burst of parallel requests can pass before failure counters
are updated. Multiple source IPs scale the admitted KDF work. A reverse proxy
misconfiguration can also merge all users into one IP bucket or let attackers
spoof their IP. The per-IP 120/minute cap is shared by legitimate clients
behind the same NAT/proxy.

Additional paths checked in code but not exercised as a live attack:

- **Stolen valid key:** authentication succeeds; the account's shared tier
  limit (50/hour standard or 100/10 minutes premium) and the key's optional
  total-use limit apply after permission checks. A stolen key is still usable
  until revoked or restricted; throttling does not replace rotation.
- **Known key ID used to deny a client behind the same NAT:** ten intentionally
  wrong secrets from that IP can temporarily block the real holder's requests
  for that key ID from the same IP. A different source IP is unaffected.
- **Requests to other routes or network-layer floods:** the API-key guard only
  covers routes using `AuthRepositoryAPIKeyMiddleware`; it does not protect
  unrelated endpoints or stop traffic before it reaches the Go process.
- **Multiple app instances:** the Redis counters are shared when every
  instance uses the same Redis database, but this was not load-tested.

Priorities before treating the API as hardened against sustained attacks:

1. Add an edge limit at the reverse proxy/CDN, including a short-burst and
   connection/concurrency cap, before requests reach Go and the database.
2. Add a short-burst or in-flight cap before PBKDF2 in the application. Tune it
   using real premium-client burst traffic to avoid accidental throttling.
3. Monitor authentication failures, 429s, CPU, database latency, and source IP
   distribution. Use these measurements to tune limits and detect distributed
   attempts.
4. Verify the production proxy's actual source IP and set
   `TRUSTED_PROXY_CIDRS` narrowly. Never trust arbitrary forwarded headers.

This is a bounded simulation, not a proof against every attack. It does not
cover live network saturation, a multi-node deployment, a CDN/WAF, or the
production database. OWASP recommends layered controls and tuning limits to
endpoint cost and real traffic:

- https://cheatsheetseries.owasp.org/cheatsheets/Bot_Management_and_Anti-Automation_Cheat_Sheet.html
- https://api-security.owasp.org/editions/2023/en/0xa4-unrestricted-resource-consumption/
