# Triad: Klondike Documentation

- `RULES.md` — canonical player-observable rules, numbered by section.
- `LORE.md` — canonical lore.
- `TECHNICAL-DESIGN.md` — engine and authority constraints behind the rules. Not player-facing.
- `LORE-DOSSIER-EN.pdf`, `LORE-DOSSIER-RU.pdf` — lore editions built from `lore-dossier/`.

## Building the PDFs

From the repository root, with Chrome or Edge installed:

```powershell
pwsh -NoProfile -File .\docs\lore-dossier\build-lore-pdfs.ps1
```

The script runs the layout check (page overflow, resolved contents) and writes both PDFs
to `docs/`.

Development setup (backend, dedicated server, checks): [DEVELOPMENT.md](DEVELOPMENT.md).
