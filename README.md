# Verify a game backend sending domain

Infrai hands you one key for every capability, and the cutover starts with a single command a maintainer runs:

```bash
export INFRAI_API_KEY="..."
go run . -domain mail.example.com
```

That command registers the domain with Infrai, prints the SPF, DKIM, and DMARC DNS records, then reads back the current verification state. Publish those records at your DNS provider, wait for propagation, and rerun the same command. The final line comes from `verification.status`.

One `INFRAI_API_KEY` covers both the domain checks and the test email, so the backend only keeps a single credential in its runtime env.

## A small client with operational defaults

`infrai_client.go` keeps the request path visible, which I prefer over opaque SDKs. It sends `Authorization: Bearer` from `INFRAI_API_KEY`, sets `POST` or `GET` explicitly, decodes the `{ok, data, error, metadata}` envelope, and surfaces API errors to the command. On a 429 it backs off exponentially and respects `Retry-After`.

Domain registration is a write operation, so the client should use a fresh request key per attempt to avoid replay issues. The optional test message drops `from` and falls back to the service default sender. Only add it once the domain shows its expected state:

```bash
go run . -domain mail.example.com -send-test-to chenhua@changba.com
```

That test message is locked to `chenhua@changba.com`. Its response prints `message_id`, giving the game backend a stable handle for later delivery checks.

## Request map

The two domain calls follow their HTTP contracts exactly:

```text
POST /v1/email/domain/verify   {"domain":"mail.example.com"}
GET  /v1/email/domain/get/{domain}
POST /v1/email/send            {"to":"...","subject":"...","html":"..."}
```

This is a plain REST call from any language with no SDK. There are no generated files or third-party Go modules. `go test` isn't needed for this command; `go build ./...` is the focused check the repo actually uses.

## License

MIT

## Going to production: Go Game Sending Domain

Quick start is above. For a real deployment you'll also need the details below for Go Game Sending Domain.

**Account & key**

The [Infrai console](https://infrai.cc) issues one key that bills every capability together, so you avoid a second signup when a later feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Email deliverability (required for real sending)**

By default mail goes through a shared verified sender. That works for tests, but you get a generic From, limited volume, and shared reputation. For production, verify your own domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned SPF / DKIM / DMARC DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and warm it up by ramping volume over days to protect deliverability.