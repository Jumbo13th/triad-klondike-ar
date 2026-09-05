# Triad: Klondike

A persistent four-faction resource-war game mode for Arma Reforger, built by the Triad
Tactics community. Players mine kolguyevite on Kolguyev, cut and sell it, buy gear and
land, and fight over the mines; the logical records (players, wallets, ledger, receipts,
audit) live in a local backend that the game server calls over the loopback interface.

- `Triad Klondike/` is the addon: Enforce Script under `scripts/`, prefabs, layouts,
  the string table and the Arland demo world. Load it in the Arma Reforger Workbench
  as a project.
- `backend/` is `klondiked`, the Go backend. Build and run instructions are in
  [backend/README.md](backend/README.md).
- `docs/` holds the rules and the technical design; start at
  [docs/README.md](docs/README.md).
- `specs/` holds one folder per feature (specification, plan, contracts, quickstart,
  handover). `.specify/memory/constitution.md` is the project's constitution.
- `tools/` has the development scripts: `backend.ps1` starts or restarts the backend,
  `generate-runtime-locales.ps1` regenerates the per-language string confs.

To play the current feature end to end, follow the quickstart of the newest folder
under `specs/`.
