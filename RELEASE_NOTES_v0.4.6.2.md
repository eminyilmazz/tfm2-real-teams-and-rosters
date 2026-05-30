# v0.4.6.2 - Language, LCP, Logos, and Random Extras

This release packages the May 30 validated database with the language/origin fixes, LCP/APAC rearrangement, custom-logo fixes, and the load-tested random extra-player cleanup.

## Changelog

- Updated player language/origin data using the decoded dynamic language-vector layout instead of unsafe fixed-offset language writes.
- Rearranged the LCP/APAC team layout and moved the corresponding players, role/stat rows, and logos with their teams.
- Restored Deep Cross Gaming and VARREL YOUTH to custom logo references, bringing the package back to 120 custom-logo team refs and 0 default-logo refs.
- Preserved the NRG square logo fix and the sourced head-coach pass from the previous release line.
- Applied the load-tested random extra-player limiting pass by physically removing surplus visible player rows that were proven to import successfully in game.
- Restored BNK FEARX to seven visible players: `Clear`, `Raptor`, `VicLa`, `Taeyoon`, `Kellin`, `REMIND`, and `POP`.
- Left the known unsafe overflow/gap deletion layer out of the release after it failed in-game import testing.
- Updated the release preflight rule so `database_pack.info` may change version metadata, but its `description` field must remain unchanged.

## Validation

- Packaged database SHA256: `3c153033167e0fc02bef0aea0d5332835ca5c643e8a51f3015b166c8674f5d68`
- Import package kind: `1`
- Gzip offset: `25`
- Parsed teams: `120`
- Embedded custom logo PNG blocks: `120`
- Teams using custom logo refs: `120`
- Teams using default logo refs: `0`
- Parsed version-5 stat athletes: `910`
- Visible roster rows: `777`
- BNK FEARX visible roster count: `7`
- Strict release validator passed with the known large zero-run warning.

## Notes

- This is intentionally based on the in-game-green Stage46 package, not the broader Stage43/Probe07 deletion layer.
- Twelve boundary slots still retain larger generated-player blocks because that final overflow/gap deletion layer was the known import-failing layer.