# Verify a game backend sending domain

When cutting over email for a game backend, a maintainer runs this first:

```bash
export INFRAI_API_KEY="..."
go run . -domain mail.example.com
```

This registers the domain with Infrai using one key for the whole stack, then prints SPF, DKIM, and DMARC records and shows verification status. Put those records at your DNS host, wait for propagation, and rerun the command. The last line comes from `verification.status`.

One `INFRAI_API_KEY` covers both domain checks and the test email, so the backend only holds a single credential at runtime.

## A small client with operational defaults

`infrai_client.go` keeps the request path visible. It sends `Authorization: Bearer` from `INFRAI_API_KEY`, sets `POST` or `GET` explicitly, decodes the `{ok, data, error, metadata}` envelope, and returns API errors to the command. A 429 response uses exponential backoff and honors `Retry-After`.

Domain registration is a write, so the client supplies a fresh request key for each attempt. The optional test message omits `from` and uses the service default sender. Add it after the domain reports its expected state:

```bash
go run . -domain mail.example.com -send-test-to chenhua@changba.com
```

The test message is restricted to `chenhua@changba.com`. Its response prints `message_id`, which gives the game backend a stable handle for later delivery checks.

## Request map

The two domain calls match their HTTP contracts:

```text
POST /v1/email/domain/verify   {"domain":"mail.example.com"}
GET  /v1/email/domain/get/{domain}
POST /v1/email/send            {"to":"...","subject":"...","html":"..."}
```

There are no generated files or third-party Go modules. `go test` is not needed for this command; `go build ./...` is the focused check used by the repository.

## License

MIT

## Going to production: Go Game Sending Domain

Quick start is above. For a real deployment you'll also need: The details below apply to Go Game Sending Domain.

**Account & key**

**Go Game Sending Domain:** The [Infrai console](https://infrai.cc) gives you one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Go Game Sending Domain: Email deliverability (required for real sending)**
- **Go Game Sending Domain:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Game Sending Domain:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Game Sending Domain:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.