# v0.4.4.4 - Roster Mapping and NRG Logo Fix

This release is a focused fix release on top of `v0.4.4.3`.

## Changelog

- Fixed roster display mapping issues found after the `v0.4.4.3` import package.
- Corrected the remaining contiguous mismatch block around slots `48-59`, including Estral, INTZ, and JP Div 1 teams.
- Preserved the confirmed CN2/EU2/NA2/SA2 roster display fixes.
- Fixed NRG using a default logo by assigning it to `custom:custom_team_logo/86`.
- Replaced NRG's first wide logo payload with a square `250x250` embedded PNG converted from `nrg_logo_square.webp`.
- Added GitHub Actions validation for PRs, pushes, version tags, published releases, and manual workflow runs.
- Added strict database guardrails for package kind, gzip offset, CRC, team count, embedded logo count, NRG logo ref, default-logo allowlist, and `database_pack.info` protection.

## Validation

- Packaged database SHA256: `f4aa4ce5ea37c260a9d76d4850dacacba0bc41ffd845ceb3e23236da778cc29b`
- `database_pack.info` SHA256 unchanged: `2abec509cb369a633ced735c03dd4837216e2543a6b72c15cff194031e26d491`
- Import package kind: `1`
- Gzip offset: `25`
- Parsed teams: `120`
- Embedded custom logo PNG blocks: `120`
- Teams using default logo refs: `2`
  - `Deep Cross Gaming`
  - `VARREL YOUTH`
- NRG logo ref: `custom:custom_team_logo/86`
- NRG embedded logo dimensions: `250x250`

## Notes

- The local all-team roster audit reached `119/120` source-backed core roster matches.
- The remaining source-data gap is CFO Academy, which has no local Oracle `.data` roster rows.
- No `database_pack.info` change is included in this release.