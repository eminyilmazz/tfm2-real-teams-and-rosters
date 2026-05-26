# TFM2 Real-World Rosters 2026

A custom roster mod for **Teamfight Manager 2** featuring real League of Legends esports players, teams, and logos from the 2026 competitive season.

![Version](https://img.shields.io/badge/version-v0.4.1-blue)
![TFM2 Version](https://img.shields.io/badge/TFM2-v0.4.1-green)
![License](https://img.shields.io/badge/license-MIT-lightgrey)

## What This Mod Does

This mod replaces the default TFM2 roster database with real-world data:

- **🏆 120 Teams** - Real esports organizations from LCK, LPL, LEC, LCS, CBLOL, LLA, PCS, LJL, VCS, and more
- **👤 1,197 Players** - Real pro players with their in-game names
- **📊 Realistic Stats** - Player statistics derived from Oracle's Elixir competitive data
- **🎨 Team Logos** - Custom logo references for major esports organizations

### Leagues Included

| Region | League | Teams |
|--------|--------|-------|
| Korea | LCK | Gen.G, T1, HLE, DK, KT, BNK, DRX, HJ, NS, DNS |
| China | LPL | BLG, JDG, TES, EDG, WBG, IG, LNG, WE, OMG, LGD |
| Europe | LEC | G2, FNC, VIT, KC, GX, NAVI, LR, KOI |
| North America | LCS | C9, TL, FLY, NRG, DIG, SEN, SR |
| Brazil | CBLOL | LOUD, paiN, FURIA, RED |
| And more... | | |

## Installation

### Quick Install (Recommended)

1. Download `tfm2_teams_and_rosters.tfm2db` from the [Releases](../../releases) page
2. Rename it to `custom_database.tfm2db`
3. Copy to: `%APPDATA%\TeamSamoyed\TeamfightManager2\data\`
4. Create a file named `custom_db_enabled.flag` in the same folder (can be empty)
5. Restart TFM2

### Manual Path
```
C:\Users\<YourName>\AppData\Roaming\TeamSamoyed\TeamfightManager2\data\custom_database.tfm2db
```

## What's Included vs. What's Missing

### ✅ Working Features

| Feature | Status | Notes |
|---------|--------|-------|
| Team Names | ✅ Working | All 120 teams renamed |
| Team Logos | ✅ Working | Custom logo references (requires logo pack) |
| Player Names | ✅ Working | Real IGNs for all players |
| Player Stats | ✅ Working | Derived from Oracle's Elixir data |
| Position Skills | ✅ Working | Based on actual roles played |

### ⚠️ Known Limitations

| Feature | Status | Reason |
|---------|--------|--------|
| **Player Ages** | ❌ Not Working | Could not get this field working |
| **Potential & Hidden Stats** | ⚠️ Partial | Some hidden fields could not be reliably modified |
| **Coach Names** | ⚠️ Partial | Some coaches may be missing or have default names |
| **Stadium Names** | ⚠️ Unchanged | Kept as game defaults |
| **Transfer History** | ❌ Not Included | Would require more research |

### ⚠️ Data Freshness

- **Data is from early 2026** - May not include latest transfers and roster changes
- New recruits from Summer split may be missing

### Data Quality Notes

- **Stats are approximations**: Real competitive performance is complex. We derived stats using formulas based on KDA, CS/min, vision score, damage share, and other metrics from Oracle's Elixir.
- **League strength calibration**: Players from stronger leagues (LCK, LPL) have slightly boosted stats to reflect competition level.
- **Academy/Challenger players**: May have more estimated stats due to less data availability.
- **Role proficiency**: Based on games played per position, not necessarily player preference.

## For Modders: Create Your Own Roster

This project includes Go tools to unpack, edit, and repack tfm2db files.

### Prerequisites

- [Go 1.21+](https://go.dev/dl/) installed
- A base tfm2db file to start from

### Tool Usage

```bash
# 1. Unpack a tfm2db file to editable CSVs
go run unpack.go "input.tfm2db" "output_folder"

# 2. Edit the CSV files in output_folder:
#    - teams.csv    (team names, logos)
#    - athletes.csv (player names, stats)

# 3. Repack back into a tfm2db file
go run repack.go "output_folder" "modded.tfm2db"

# 4. Validate your modded file
go run validate.go "modded.tfm2db"
```

### CSV Field Reference

#### teams.csv
| Field | Editable | Notes |
|-------|----------|-------|
| team_name | ✅ Yes | Same length only |
| team_logo | ✅ Yes | `custom:custom_team_logo/N` or `X_Y` format |
| stadium_name | ✅ Yes | Same length only |
| manager_name | ✅ Yes | Same length only |

#### athletes.csv
| Field | Editable | Notes |
|-------|----------|-------|
| name | ✅ Yes | Player name |
| last_hit, skill_avoid, etc. | ✅ Yes | Stats (0-100 typical) |
| team_id | ✅ Yes | Contract field |
| face | ✅ Yes | Portrait index |

## Data Sources

- **Player/Team Data**: [Oracle's Elixir](https://oracleselixir.com/) - 2026 early/Spring split
- **Team Logos**: Referenced from existing TFM2 custom logo format
- **Base File Format**: Reverse-engineered from TFM2 v0.4.1

## Technical Details

### TFM2DB File Format

```
Offset  Size  Description
0       4     Magic bytes "TFM2"
4       1     Kind byte (1 = roster)
5       8     Timestamp (u64, milliseconds since epoch)
13      8     Gzip payload length (u64)
21      4     CRC32 of gzip payload
25      ...   Gzip-compressed Rust bincode data
```

### Athlete Data Structure

Each athlete has 39 u64 values before their name string:
- Index 0: Version
- Index 1-20: Visible stats
- Index 21-31: Hidden stats
- Index 32-36: Contract fields
- Index 37: Face index

## Disclaimer & Credits

### AI Assistance Disclosure

This project was developed with significant assistance from **GitHub Copilot (Claude Opus 4.5)**. The AI helped with:
- Reverse-engineering the tfm2db binary format
- Writing the Go unpacking/repacking tools
- Generating stat conversion formulas
- Debugging file corruption issues
- Writing this documentation

### Software Used

- **Python 3.12** - Initial tooling and data processing
- **Go 1.21** - Final clean tools for community use
- **Oracle's Elixir** - Competitive data source

### Legal

This mod is unofficial and not affiliated with Team Samoyed (TFM2 developers) or Riot Games (League of Legends). All team names, player names, and logos are property of their respective owners. This is a fan project for personal use.

### Known Issues

1. **Some players may have incorrect stats** - The formula is an approximation
2. **Academy players have less accurate data** - Fewer games to analyze
3. **Data from early 2026** - Some teams (FeelsStrongMan Los Ratones) no longer exist, latest transfers not included
4. **Some fields not editable** - Ages, potential, and some hidden stats could not be modified

## Contributing

Found an issue? Have better data? Pull requests welcome!

1. Fork this repository
2. Edit the source CSVs or improve the tools
3. Test your changes in-game
4. Submit a PR with description of changes

## License

MIT License - See [LICENSE](LICENSE) for details.

---

*Last updated: May 2026*
