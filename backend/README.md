# klondiked

The local backend of Triad: Klondike. It runs on the same machine as the dedicated
game server, listens on the loopback interface only, and is the system of record for
every logical record: players, wallets, ledger, receipts, audit, and the runtime
configuration the game polls. The HTTP contract is in
`../specs/001-backend-boundary/contracts/backend-http.md`.

## Build and test

```text
go build ./cmd/klondiked
go test ./...
```

## Run

```text
klondiked -listen 127.0.0.1:8471 -data ./data
```

| Flag | Default | Meaning |
|---|---|---|
| `-listen` | `127.0.0.1:8471` | listen address; a non-loopback address is refused |
| `-data` | `data` | directory for `klondike.db` and the `klondiked.json` seed |
| `-announce-contract` | the built-in version | contract version announced by `/v1/health`; demo-world control |
| `-delay` | `0` | delay every answer, for example `7s`; demo-world control |

`klondiked.json` is read once, when the database is created:

```json
{ "operators": ["<player uuid>"], "answer_timeout_s": 5, "recheck_interval_s": 15 }
```

Later changes to the two runtime values come from the in-game cockpit and are recorded
with a new configuration revision.
