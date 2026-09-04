# Development setup

How the local setup for developing Klondike is designed: one machine runs Workbench,
the backend and, for identity-dependent tests, a dedicated server.

## Backend

The backend is started and checked by the agent, not by hand:

```text
pwsh -File tools/backend.ps1            # reuse a healthy backend, else build and start
pwsh -File tools/backend.ps1 -Status
pwsh -File tools/backend.ps1 -Restart -Announce 2     # unknown-version state for a test
pwsh -File tools/backend.ps1 -Restart -Delay 7s       # slow-answer state
pwsh -File tools/backend.ps1 -Stop
```

It listens on `127.0.0.1:8471`, which is the default the game writes into
`$profile:TK_Backend.json`. The first start writes `backend/data/klondiked.json` with an
empty operator list; put the operator's player identity there (the server log prints it
at connect) and restart, or every cockpit command is refused as unauthorized.

## Dedicated server

The server loads the addon from disk, not from the workshop: `-addonsDir` names the
directory that contains the `Triad Klondike` folder and `-addons` names the addon id,
`TriadKlondike`. The profile directory is `dsprofile`. In the config JSON,
`game.scenarioId` is the demo world, `{<mission GUID>}Missions/KlondikeDemo.conf`, with
the GUID from `Triad Klondike/Missions/KlondikeDemo.conf.meta`; `maxPlayers` is at
least 2 (operator and player); for a local server `visible` is false and BattlEye is
off, because addons loaded from disk are unsigned. Cap the frame rate (`-maxFPS 60`) so
an idle server does not burn a core.

## Identity

Only a dedicated server audits players, so only there does a player get the stable
identity every wallet command is keyed by. Workbench play shows the boundary state and
refuses everything identity-keyed; that is the designed behaviour, not a fault.

## Continuous integration

`.github/workflows/backend.yml` runs `gofmt`, `go vet`, `go build` and `go test` for
every push or pull request that touches `backend/`. Run the same four locally before
handing over.
