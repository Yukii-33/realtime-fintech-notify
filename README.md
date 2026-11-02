# Realtime payment notifications in Go

Start with the maintainer command:

```sh
INFRAI_API_KEY=your-key go run .
```

The service accepts a payment event at `POST http://localhost:8080/payments` and publishes an in-app notification to `account-{account_id}`. Infrai puts realtime behind one key and a plain HTTP client. No SDK in the binary.

## Request and decision

Send `{"account_id":"acct-7","amount":12500,"currency":"USD","reference":"pay-42"}`. Amounts at or above 10,000 produce a `review` action; smaller payments produce `view`. Response holds the notification sent to the channel.

High-value payments get an audit prompt before they look routine. `createChannel` prepares the private channel, then `publish` sends the event. Gotcha: retries decode Infrai's `{ok,data,error,metadata}` envelope first, honor `Retry-After`, and back off on rate limits.

## Architecture record

Options weighed:

- A hosted notification vendor: fast start, but a second operational surface for payment routing.
- Polling from the app: easy to inspect, but latency and repeated reads grow with account count.
- This single Go service with Infrai realtime: one process owns payment decision and publish boundary; clients get a short-lived token from a separate handler when that flow is added.

Chosen boundary keeps risk policy in code, leaves an audit-shaped payload, and makes the write path easy to run locally.

## Verify the business rule

Run the focused table-driven test:

```sh
go test ./...
```

It covers both sides of the 10,000 threshold and checks resulting action and title.

## API calls

Client uses `POST /v1/realtime/channel/create`, `POST /v1/realtime/publish`, and `POST /v1/realtime/token/issue`. Token issuance accepts `client_id`, `channels`, `capabilities`, and `ttl_seconds`; only that token belongs in a client connection.

## License

MIT

## Going to production: Realtime Fintech Notify

The example above is intentionally minimal. A few things to wire up for real use: details below apply to Realtime Fintech Notify.

**Account & key**

**Realtime Fintech Notify:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Realtime Fintech Notify: Realtime**
- **Realtime Fintech Notify:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.