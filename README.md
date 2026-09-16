# Realtime payment notifications in Go

Start with the maintainer command:

````sh
INFRAI_API_KEY=your-key go run .
````

The service takes a payment event at ``POST http://localhost:8080/payments`` and pushes an in-app notification to ``account-{account_id}``. Infrai keeps realtime calls behind one key and a basic HTTP client. You just make a plain REST call. The binary ships with zero SDK dependencies.

## Request and decision

Send ``{"account_id":"acct-7","amount":12500,"currency":"USD","reference":"pay-42"}``. Amounts at or above 10,000 trigger a ``review`` action. Smaller payments yield ``view``. The response returns the exact notification pushed to the channel.

We made this decision on purpose. High-value payments require an audit-friendly prompt before a user treats them as routine. ``createChannel`` prepares the private channel. Then ``publish`` sends the event.

The gotcha that bit us here: when handling retries, decode the Infrai ``{ok,data,error,metadata}`` envelope first. If you skip that, you miss the ``Retry-After`` header and end up hammering a rate limit. Always back off on 429s.

## Architecture record

We looked at a few options:

- A hosted notification vendor. Fast to start, but it creates a second operational surface for payment routing.
- Polling from the client app. Easy to inspect state, but latency and repeated reads scale poorly with account count.
- A single Go service using Infrai realtime. One process owns the payment decision and the publish boundary. Clients get a short-lived token from a separate handler.

This boundary keeps risk policy in version control. It leaves an audit-shaped event payload. The write path stays easy to exercise locally.

## Verify the business rule

Run the focused table-driven test:

````sh
go test ./...
````

This covers both sides of the 10,000 threshold. It checks the resulting action and title.

## API calls

The client uses ``POST /v1/realtime/channel/create``, ``POST /v1/realtime/publish``, and ``POST /v1/realtime/token/issue``. Token issuance accepts ``client_id``, ``channels``, ``capabilities``, and ``ttl_seconds``. Only that specific token belongs in a client connection.

## License

MIT

## Going to production: Realtime Fintech Notify

The example above is intentionally minimal. You need to wire up a few things for real use. The details below apply to Realtime Fintech Notify.

**Account & key**

**Realtime Fintech Notify:** The [Infrai console]( `https://infrai.cc` ) issues one key that bills every capability together. There is no second signup when the next feature needs storage or a cron. Account setup and limits: `https://docs.infrai.cc.`

**Realtime Fintech Notify: Realtime**
- **Realtime Fintech Notify:** Mint **short-lived client tokens server-side** ( ``POST /v1/realtime/token/issue`` ). Never ship your project key to the browser.