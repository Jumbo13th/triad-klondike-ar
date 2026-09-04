# Handover: Backend Boundary (feature 001)

State on 2026-09-05: implemented, every quickstart scenario played on the local
dedicated server with two clients. Open only what is listed under "Not done".

Changes:
- Go backend `backend/` (klondiked): health, connect, wallet, commands, audit; SQLite;
  contract "1"; operator list applied at every start; command delay flag for the demo
  world; one log line per decision
- Game mode `TK_GameMode`, `TK_BackendComponent` (readiness poll, connect on audit
  success, retry after outage, commands, receipts), `TK_PlayerComponent` (owner RPCs),
  `TK_CockpitMenu` (operator panel, player balance)
- Development identity: `DevIdentityFromName` in `$profile:TK_Backend.json`, name
  derived, same-name joiners numbered
- Receipt delivery settled by `target_online` on the compensate envelope
- Demo world `worlds/Arland/KlondikeDemo.ent`, prefabs, mission header, menu and
  input overrides, panel layouts, localization EN+RU
- Tooling: `tools/backend.ps1`, `.github/workflows/backend.yml`; setup notes in `backend/README.md` and the quickstart prerequisites
- Constitution v1.2.0 (Conflict as UI reference, User Interface section, audit done
  criteria, backend logging)

Workbench files to double-check (agent-authored, plan.md "Workbench steps"):
- `Prefabs/MP/Modes/TK_GameMode.et`, `Prefabs/Characters/Core/TK_PlayerController.et`
- `worlds/Arland/KlondikeDemo.ent`, `KlondikeDemo_Layers/a_systems.layer`,
  `KlondikeDemo_Layers/spawns.layer`, `Missions/KlondikeDemo.conf`
- `Configs/System/chimeraMenus.conf`, `Configs/System/chimeraInputCommon.conf`
- `UI/Cockpit/CockpitPanel.layout`, `UI/Cockpit/CockpitTextLine.layout`
- `Language/tk_localization.st` and the 13 runtime confs, `addon.gproj`

GUID audit (T037, re-run 2026-09-05): every new GUID and object id is absent from the
vanilla game data and appears once in the addon (twice where a conf, script or gproj
references it). The only vanilla matches are the inherited prefab IDs of
`GameMode_Base` and `DefaultPlayerControllerMP` and the RespawnSystem component id,
which derived prefabs keep by engine convention. Both conf overrides carry the base
game's GUID on purpose.

Review of PR #1 (2026-09-05, T047): the code and document corrections are listed in
tasks.md. Not played yet on the local server: the dropped connect answer for a player
who left, `target_ambiguous`, the empty-target refusal, the receipt push gated on
dispatch-time presence, the re-parented spawn points (the vanilla `SpawnPoint_USSR.et`
GUID is not in the script dump, so both factions derive from `SpawnPoint_Base.et`
with the faction prefab's own overrides; re-parent in Workbench if preferred), and
the string-table author fields. Play Scenario B and D2 once more after the merge.

Quickstart results (details under each scenario in quickstart.md):
- A readiness: passed 2026-09-03 (Workbench)
- B balance, credit, receipt, insufficient funds: passed 2026-09-04
- C retry after unreachable, server kill in three forms: passed 2026-09-04;
  step 4 covered by the backend test
- D busy, identity refusal, development identity: passed 2026-09-04
- D2 join during outage, connect retry: passed 2026-09-04
- E runtime regulation: passed 2026-09-04
- F Russian: passed 2026-09-03
- Receipt delivery (T046): passed 2026-09-04

Accepted trade-offs (operator, 2026-09-04):
- A receipt marked delivered at creation is lost if the game server dies before
  pushing it; the ledger row stands
- A request the engine could not dispatch before its timeout is never sent; the retry
  answers accepted, not already_applied; one ledger row either way

Not done, for the cockpit redo against the Conflict screens:
- Panel looks the wallet up itself before the first send
- One Refresh for the whole panel, lookup labelled as lookup
- Retry hidden unless an operation is pending
- Offline targets reachable by identity in an obvious way; case-insensitive name match
- Receipts kept on the player component or shown in a notification feed, so a receipt
  delivered before the panel opens is not lost from view
- "Older" disabled while an audit page is in flight; two quick presses append the
  same page twice today
