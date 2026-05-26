// TFM2DB Repack Tool
// Applies CSV edits to a tfm2db file and creates a new modded file.
//
// Usage: go run repack.go <directory> <output.tfm2db>
//
// The directory should contain:
//   - header.bin (original TFM2 header)
//   - payload.bin (decompressed payload)
//   - teams.csv (edited team data - optional)
//   - athletes.csv (edited athlete data - optional)
//
// This tool is part of the TFM2 Roster Mod project.
// https://github.com/eminyilmazz/tfm2-real-teams-and-rosters

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const HeaderSize = 25

var athleteStatFields = []string{
	"last_hit", "skill_avoid", "skill_hit", "positioning", "control_speed",
	"concentration", "mental", "judgement", "order", "roaming",
	"aggressive", "ego", "top", "jungle", "mid", "bottom", "support",
	"like_champion", "dislike_champion", "language",
}

var athleteHiddenFields = []string{
	"potential", "stamina_recovery_max", "stamina_cost_per_set_min",
	"stamina_cost_per_set_max", "stress_sensitivity", "condition_baseline",
	"condition_amplitude", "condition_period", "condition_phase",
	"match_impact_sensitivity", "stamina_recovery_min",
}

var athleteContractFields = []string{
	"team_id", "end_date", "weekly_salary", "transfer_fee", "start_date",
}

const (
	AthleteStatVersionIndex   = 0
	AthleteStatStartIndex     = 1
	AthleteHiddenStartIndex   = 21
	AthleteContractStartIndex = 32
	AthleteFaceIndex          = 37
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("TFM2DB Repack Tool")
		fmt.Println("Usage: go run repack.go <directory> <output.tfm2db>")
		fmt.Println("\nApplies CSV edits to payload.bin and creates a new .tfm2db file.")
		fmt.Println("\nRequired files in directory:")
		fmt.Println("  - header.bin  (original TFM2 header)")
		fmt.Println("  - payload.bin (decompressed payload)")
		fmt.Println("\nOptional files (edits):")
		fmt.Println("  - teams.csv    (team name/logo edits)")
		fmt.Println("  - athletes.csv (player stat edits)")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputPath := os.Args[2]

	fmt.Printf("Repacking from %s -> %s\n\n", inputDir, outputPath)

	// Read header
	headerPath := filepath.Join(inputDir, "header.bin")
	header, err := os.ReadFile(headerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading header.bin: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Read header: %d bytes\n", len(header))

	// Read payload
	payloadPath := filepath.Join(inputDir, "payload.bin")
	payload, err := os.ReadFile(payloadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading payload.bin: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Read payload: %d bytes\n", len(payload))

	// Convert payload to mutable slice
	modifiedPayload := make([]byte, len(payload))
	copy(modifiedPayload, payload)

	// Apply team edits
	teamsPath := filepath.Join(inputDir, "teams.csv")
	if _, err := os.Stat(teamsPath); err == nil {
		teamEdits, err := applyTeamEdits(modifiedPayload, teamsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying team edits: %v\n", err)
			os.Exit(1)
		}
		if teamEdits > 0 {
			fmt.Printf("Applied %d team edits\n", teamEdits)
		}
	}

	// Apply athlete edits
	athletesPath := filepath.Join(inputDir, "athletes.csv")
	if _, err := os.Stat(athletesPath); err == nil {
		athleteEdits, err := applyAthleteEdits(modifiedPayload, athletesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error applying athlete edits: %v\n", err)
			os.Exit(1)
		}
		if athleteEdits > 0 {
			fmt.Printf("Applied %d athlete stat edits\n", athleteEdits)
		}
	}

	// Compress
	fmt.Println("Compressing...")
	var compressedBuf bytes.Buffer
	gzWriter, _ := gzip.NewWriterLevel(&compressedBuf, gzip.BestSpeed)
	gzWriter.Write(modifiedPayload)
	gzWriter.Close()
	compressed := compressedBuf.Bytes()
	fmt.Printf("Compressed: %d -> %d bytes\n", len(modifiedPayload), len(compressed))

	// Build new header
	newHeader := make([]byte, len(header))
	copy(newHeader, header)

	// Update timestamp
	binary.LittleEndian.PutUint64(newHeader[5:13], uint64(time.Now().UnixMilli()))

	// Update gzip length
	binary.LittleEndian.PutUint64(newHeader[13:21], uint64(len(compressed)))

	// Update CRC32
	binary.LittleEndian.PutUint32(newHeader[21:25], crc32.ChecksumIEEE(compressed))

	// Write output file
	output := append(newHeader, compressed...)
	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nWrote %s (%d bytes)\n", outputPath, len(output))
	fmt.Println("Done!")
}

func applyTeamEdits(payload []byte, csvPath string) (int, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Skip BOM if present
	buf := make([]byte, 3)
	file.Read(buf)
	if buf[0] != 0xEF || buf[1] != 0xBB || buf[2] != 0xBF {
		file.Seek(0, 0)
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}

	if len(records) < 2 {
		return 0, nil
	}

	// Build column index
	headerRow := records[0]
	colIndex := make(map[string]int)
	for i, name := range headerRow {
		colIndex[strings.TrimSpace(name)] = i
	}

	editCount := 0
	for _, row := range records[1:] {
		// Team name edit
		if nameOffset, ok := getIntColumn(row, colIndex, "team_name_offset"); ok {
			if newName, ok := getStringColumn(row, colIndex, "team_name"); ok {
				if editLPString(payload, nameOffset, newName) {
					editCount++
				}
			}
		}

		// Team logo edit
		if logoOffset, ok := getIntColumn(row, colIndex, "team_logo_offset"); ok {
			if newLogo, ok := getStringColumn(row, colIndex, "team_logo"); ok {
				if editLPString(payload, logoOffset, newLogo) {
					editCount++
				}
			}
		}

		// Stadium name edit
		if stadiumOffset, ok := getIntColumn(row, colIndex, "stadium_name_offset"); ok {
			if newStadium, ok := getStringColumn(row, colIndex, "stadium_name"); ok {
				if editLPString(payload, stadiumOffset, newStadium) {
					editCount++
				}
			}
		}

		// Manager name edit
		if managerOffset, ok := getIntColumn(row, colIndex, "manager_name_offset"); ok {
			if newManager, ok := getStringColumn(row, colIndex, "manager_name"); ok {
				if editLPString(payload, managerOffset, newManager) {
					editCount++
				}
			}
		}
	}

	return editCount, nil
}

func applyAthleteEdits(payload []byte, csvPath string) (int, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Skip BOM if present
	buf := make([]byte, 3)
	file.Read(buf)
	if buf[0] != 0xEF || buf[1] != 0xBB || buf[2] != 0xBF {
		file.Seek(0, 0)
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return 0, err
	}

	if len(records) < 2 {
		return 0, nil
	}

	// Build column index
	headerRow := records[0]
	colIndex := make(map[string]int)
	for i, name := range headerRow {
		colIndex[strings.TrimSpace(name)] = i
	}

	editCount := 0
	for _, row := range records[1:] {
		nameOffset, ok := getIntColumn(row, colIndex, "name_offset")
		if !ok {
			continue
		}

		// Calculate base offset for stats (39 u64s before name)
		preOffset := nameOffset - 39*8
		if preOffset < 0 {
			continue
		}

		// Apply stat edits
		for i, field := range athleteStatFields {
			if val, ok := getUint64Column(row, colIndex, field); ok {
				offset := preOffset + (AthleteStatStartIndex+i)*8
				if writeU64(payload, offset, val) {
					editCount++
				}
			}
		}

		// Apply hidden stat edits
		for i, field := range athleteHiddenFields {
			if val, ok := getUint64Column(row, colIndex, field); ok {
				offset := preOffset + (AthleteHiddenStartIndex+i)*8
				if writeU64(payload, offset, val) {
					editCount++
				}
			}
		}

		// Apply contract edits
		for i, field := range athleteContractFields {
			if val, ok := getUint64Column(row, colIndex, field); ok {
				offset := preOffset + (AthleteContractStartIndex+i)*8
				if writeU64(payload, offset, val) {
					editCount++
				}
			}
		}

		// Apply face edit
		if face, ok := getUint64Column(row, colIndex, "face"); ok {
			offset := preOffset + AthleteFaceIndex*8
			if writeU64(payload, offset, face) {
				editCount++
			}
		}

		// Name edit (text field)
		if newName, ok := getStringColumn(row, colIndex, "name"); ok {
			if editLPString(payload, nameOffset, newName) {
				editCount++
			}
		}
	}

	return editCount, nil
}

func getIntColumn(row []string, colIndex map[string]int, name string) (int, bool) {
	idx, ok := colIndex[name]
	if !ok || idx >= len(row) {
		return 0, false
	}
	val := strings.TrimSpace(row[idx])
	if val == "" {
		return 0, false
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}
	return n, true
}

func getUint64Column(row []string, colIndex map[string]int, name string) (uint64, bool) {
	idx, ok := colIndex[name]
	if !ok || idx >= len(row) {
		return 0, false
	}
	val := strings.TrimSpace(row[idx])
	if val == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func getStringColumn(row []string, colIndex map[string]int, name string) (string, bool) {
	idx, ok := colIndex[name]
	if !ok || idx >= len(row) {
		return "", false
	}
	return row[idx], true
}

func editLPString(data []byte, offset int, newText string) bool {
	if offset < 0 || offset+8 > len(data) {
		return false
	}

	oldLength := binary.LittleEndian.Uint64(data[offset:])
	oldEnd := offset + 8 + int(oldLength)
	if oldEnd > len(data) {
		return false
	}

	// Only edit if same length (to avoid offset shifts)
	newBytes := []byte(newText)
	if len(newBytes) != int(oldLength) {
		fmt.Printf("Warning: String at offset %d has different length (%d vs %d), skipping to avoid corruption\n",
			offset, oldLength, len(newBytes))
		return false
	}

	// Write new text
	copy(data[offset+8:oldEnd], newBytes)
	return true
}

func writeU64(data []byte, offset int, value uint64) bool {
	if offset < 0 || offset+8 > len(data) {
		return false
	}
	binary.LittleEndian.PutUint64(data[offset:], value)
	return true
}
