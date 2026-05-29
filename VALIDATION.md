# Validation Guardrails

Run this before pushing any database change:

```powershell
./tools/preflight.ps1
```

The same preflight runs in GitHub Actions on pull requests and pushes that touch the database, package metadata, tools, or the workflow.

## What CI Checks

- `tfm2_teams_and_rosters.tfm2db` is a direct import package: kind byte `1`, gzip offset `25`.
- Header gzip length and CRC32 match the actual compressed payload.
- The database decompresses successfully.
- Exactly `120` team rows are parsed.
- Exactly `120` embedded custom logo PNG blocks are present.
- Custom logo references resolve to embedded logo ids.
- NRG must use a `custom:custom_team_logo/N` reference.
- Only the current intentional default-logo teams are allowed:
  - `Deep Cross Gaming`
  - `VARREL YOUTH`
- At least `1100` athlete-like rows are found.
- `database_pack.info` must not change.

## Release Rules

- Do not edit `database_pack.info`; Steam behaves unpredictably when that file changes.
- Do not create a GitHub release until the import file has been tested in-game.
- Keep release commits small and explain whether the edit is names-only, logo-ref-only, exact-length logo payload replacement, or broader data generation.

## Logo Rules

- Custom logo id is read from the `u64` immediately before the embedded PNG length, not from PNG block order.
- Prefer text-only `team_logo` ref fixes when the correct embedded id already exists.
- If replacing a logo payload, keep the embedded PNG block length unchanged unless the hidden logo table is fully revalidated.
- For different-sized source images, convert/resize/pad to an exact-length valid PNG payload.
- Treat new default `X_Y` logo refs as suspicious unless they are explicitly added to the allowlist with a reason.

## Roster Rules

- Visible roster blocks are the game-facing truth; `contract_team_id` alone does not determine the roster screen.
- Prefer names-only edits when repairing display mapping problems.
- Broad numeric rewrites require extra suspicion and in-game testing, especially contracts, hidden stats, ages, and face/profile fields.
- Source-data gaps should be reported as source gaps instead of forced into mismatches.

## Lessons From The Stage 21-25 Fixes

- A visually plausible roster pass can still hide contiguous boundary errors; audit every team against source data.
- Div2 roster surfaces can shift by one block, and boundary teams may use hidden rows.
- One fix can create a collision when two selected teams read the same visible surface; stop rotating names once that happens and map the real surface.
- `database_pack.info` churn is not harmless metadata churn for Steam Workshop workflows.
- Variable-length logo payload replacement is risky because later logo offsets can be interpreted differently by the game.
- Square logo canvases are safer for in-game square logo slots than wide raw wordmarks.

## Ideal Next Layer

The current guardrails catch structural mistakes and known logo regressions. The ideal next layer is a checked-in roster oracle/manifest that lets CI compare every team roster against source data, the way Stage 23 was audited locally. That would turn the all-team roster audit into a repeatable PR check instead of a one-off workspace script.