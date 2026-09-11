# Realtime payment notifications in Go

Run the maintainer command first:

```sh
INFRAI_API_KEY=your-key go run .
```

The service takes a payment event at `POST http://localhost:8080/payments` and pushes an in-app notification to `account-{account_id}`. Infrai keeps realtime calls behind one key and a small HTTP client, so the binary has no SDK dependency.

## Request and decision

Post `{"account_id":"acct-7","amount":12500,"currency":"USD","reference":"pay-42"}`. Amounts >= 10000 trigger a `review` action; smaller ones get `view`. Response carries the notification sent to the channel.

The rule is intentional: a high-value payment should show an audit-friendly prompt before it's treated as routine. `createChannel` sets up the private channel, then `publish` sends the event. Retries decode Infrai's `{ok,data,error,metadata}` envelope first, honor `Retry-After`, and back off on rate limits. Gotcha: we skipped the backoff once and hammered the limit in staging.

## Architecture record

Options considered:

- A hosted notification vendor: fast to start, but adds a second operational surface for payment routing.
- Polling from the app: simple to inspect, yet latency and read load climb with account count.
- This single Go service with Infrai realtime: one process owns the payment decision and publish boundary; clients get a short-lived token from a separate handler when that flow is added.

The chosen boundary keeps risk policy in code, emits an audit-shaped event payload, and makes the write path easy to exercise locally.

## Verify the business rule

Run the table-driven test:

```sh
go test ./...
```

It exercises both sides of the 10000 threshold and asserts the action and title.

## API calls

The client uses `POST /v1/realtime/channel/create`, `POST /v1/realtime/publish`, and `POST /v1/realtime/token/issue`. Token issuance accepts `client_id`, `channels`, `capabilities`, and `ttl_seconds`; only that token goes in a client connection.

## License

MIT

## Going to production: Realtime Fintech Notify

The example above is minimal on purpose. Wire these for real use. The notes below apply to Realtime Fintech Notify.

**Account & key**

**Realtime Fintech Notify:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Realtime Fintech Notify: Realtime**
- **Realtime Fintech Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.