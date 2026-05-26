// TFM2DB Validate Tool
// Validates a .tfm2db file structure and reports any issues.
//
// Usage: go run validate.go <input.tfm2db>
//
// This tool is part of the TFM2 Roster Mod project.
// https://github.com/eminyilmazz/tfm2-real-teams-and-rosters

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"regexp"
	"unicode/utf8"
)

const (
	TFM2Magic    = "TFM2"
	GzipMagic    = "\x1f\x8b\x08"
	HeaderSize   = 25
	MaxStrLength = 256
)

var (
	teamLogoRE = regexp.MustCompile(`^(?:custom:custom_team_logo/(\d+)|(\d+)_(\d+))$`)
	decoRE     = regexp.MustCompile(`^(?:#asset/|asset/|custom:|furniture_|wallpaper_|clean_|plain_|premium_|wide_window|basic_chair)`)
)

type ValidationResult struct {
	Errors   []string
	Warnings []string
	Info     []string
}

func (v *ValidationResult) addError(msg string) {
	v.Errors = append(v.Errors, msg)
}

func (v *ValidationResult) addWarning(msg string) {
	v.Warnings = append(v.Warnings, msg)
}

func (v *ValidationResult) addInfo(msg string) {
	v.Info = append(v.Info, msg)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("TFM2DB Validate Tool")
		fmt.Println("Usage: go run validate.go <input.tfm2db>")
		fmt.Println("\nValidates a .tfm2db file and reports issues.")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	result := &ValidationResult{}

	fmt.Printf("Validating %s...\n\n", inputPath)

	// Read file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	result.addInfo(fmt.Sprintf("File size: %d bytes", len(data)))

	// Validate header
	validateHeader(data, result)
	if len(result.Errors) > 0 {
		printResults(result)
		os.Exit(1)
	}

	// Find and validate gzip data
	gzipOffset := bytes.Index(data[4:], []byte(GzipMagic))
	if gzipOffset < 0 {
		result.addError("No gzip data found in file")
		printResults(result)
		os.Exit(1)
	}
	gzipOffset += 4
	result.addInfo(fmt.Sprintf("Gzip offset: %d", gzipOffset))

	// Validate header fields
	if gzipOffset >= 25 {
		validateHeaderFields(data, gzipOffset, result)
	}

	// Try decompression
	payload := validateDecompression(data[gzipOffset:], result)
	if payload == nil {
		printResults(result)
		os.Exit(1)
	}
	result.addInfo(fmt.Sprintf("Decompressed size: %d bytes", len(payload)))

	// Validate content
	validateContent(payload, result)

	// Print results
	printResults(result)

	if len(result.Errors) > 0 {
		os.Exit(1)
	}
}

func validateHeader(data []byte, result *ValidationResult) {
	if len(data) < 4 {
		result.addError("File too small - missing TFM2 header")
		return
	}

	magic := string(data[:4])
	if magic != TFM2Magic {
		result.addError(fmt.Sprintf("Invalid magic: expected 'TFM2', got '%s'", magic))
		return
	}
	result.addInfo("Magic: TFM2 ✓")

	if len(data) < HeaderSize {
		result.addWarning(fmt.Sprintf("Header smaller than expected: %d < %d bytes", len(data), HeaderSize))
	}
}

func validateHeaderFields(data []byte, gzipOffset int, result *ValidationResult) {
	// TFM2 header format (25 bytes):
	// 0-3:   Magic "TFM2"
	// 4:     Kind byte
	// 5-12:  Timestamp (u64 little-endian, milliseconds)
	// 13-20: Gzip length (u64 little-endian)
	// 21-24: CRC32 (u32 little-endian)

	kind := data[4]
	result.addInfo(fmt.Sprintf("Kind byte: %d", kind))

	timestamp := binary.LittleEndian.Uint64(data[5:13])
	result.addInfo(fmt.Sprintf("Timestamp: %d", timestamp))

	storedGzLen := binary.LittleEndian.Uint64(data[13:21])
	actualGzLen := len(data) - gzipOffset
	result.addInfo(fmt.Sprintf("Stored gzip length: %d", storedGzLen))
	result.addInfo(fmt.Sprintf("Actual gzip length: %d", actualGzLen))

	if storedGzLen != uint64(actualGzLen) {
		result.addWarning(fmt.Sprintf("Gzip length mismatch: stored %d != actual %d", storedGzLen, actualGzLen))
	}

	storedCRC := binary.LittleEndian.Uint32(data[21:25])
	actualCRC := crc32.ChecksumIEEE(data[gzipOffset:])
	result.addInfo(fmt.Sprintf("Stored CRC32: %d", storedCRC))
	result.addInfo(fmt.Sprintf("Actual CRC32: %d", actualCRC))

	if storedCRC != actualCRC {
		result.addWarning(fmt.Sprintf("CRC32 mismatch: stored %d != actual %d", storedCRC, actualCRC))
	}
}

func validateDecompression(gzipData []byte, result *ValidationResult) []byte {
	reader, err := gzip.NewReader(bytes.NewReader(gzipData))
	if err != nil {
		result.addError(fmt.Sprintf("Failed to create gzip reader: %v", err))
		return nil
	}
	defer reader.Close()

	payload, err := io.ReadAll(reader)
	if err != nil {
		result.addError(fmt.Sprintf("Failed to decompress: %v", err))
		return nil
	}

	result.addInfo("Decompression: OK ✓")
	return payload
}

func validateContent(payload []byte, result *ValidationResult) {
	// Count strings
	strings := scanStrings(payload)
	result.addInfo(fmt.Sprintf("Strings found: %d", len(strings)))

	// Count teams
	teamCount := 0
	for _, s := range strings {
		if teamLogoRE.MatchString(s) {
			teamCount++
		}
	}
	result.addInfo(fmt.Sprintf("Team logos found: %d", teamCount))

	if teamCount == 0 {
		result.addWarning("No team logos found - file may be corrupted or empty")
	} else if teamCount < 100 {
		result.addWarning(fmt.Sprintf("Only %d teams found (expected ~120)", teamCount))
	} else if teamCount > 200 {
		result.addWarning(fmt.Sprintf("Unusually many teams found: %d", teamCount))
	}

	// Count athlete-like entries
	athleteCount := countAthletes(payload, strings)
	result.addInfo(fmt.Sprintf("Athlete candidates: %d", athleteCount))

	if athleteCount == 0 {
		result.addWarning("No athletes found - file may be corrupted")
	} else if athleteCount < 500 {
		result.addWarning(fmt.Sprintf("Only %d athletes found (expected ~1200)", athleteCount))
	}

	// Check for common corruption patterns
	checkCorruption(payload, result)
}

func scanStrings(data []byte) []string {
	var result []string
	dataLen := len(data)

	for offset := 0; offset < dataLen-8; offset++ {
		length := binary.LittleEndian.Uint64(data[offset:])
		if length < 1 || length > MaxStrLength {
			continue
		}
		end := offset + 8 + int(length)
		if end > dataLen {
			continue
		}
		raw := data[offset+8 : end]
		if !utf8.Valid(raw) {
			continue
		}
		text := string(raw)
		if isPlausibleText(text) {
			result = append(result, text)
		}
	}
	return result
}

func isPlausibleText(text string) bool {
	if len(text) == 0 {
		return false
	}
	for _, r := range text {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}

func hasLetter(text string) bool {
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			return true
		}
	}
	return false
}

func countAthletes(data []byte, strings []string) int {
	count := 0
	dataLen := len(data)

	for offset := 0; offset < dataLen-8; offset++ {
		length := binary.LittleEndian.Uint64(data[offset:])
		if length < 2 || length > 32 {
			continue
		}
		end := offset + 8 + int(length)
		if end > dataLen {
			continue
		}
		raw := data[offset+8 : end]
		if !utf8.Valid(raw) {
			continue
		}
		text := string(raw)
		if !hasLetter(text) || decoRE.MatchString(text) {
			continue
		}

		// Check for stats before name
		preOffset := offset - 39*8
		if preOffset < 0 {
			continue
		}

		// Check if preceding values look like stats (small numbers)
		looksLikeStats := true
		for i := 0; i < 32; i++ {
			v := binary.LittleEndian.Uint64(data[preOffset+i*8:])
			if v > 1000 {
				looksLikeStats = false
				break
			}
		}
		if looksLikeStats {
			count++
		}
	}
	return count
}

func checkCorruption(payload []byte, result *ValidationResult) {
	// Check for runs of zeros (potential corruption)
	zeroRun := 0
	maxZeroRun := 0
	for _, b := range payload {
		if b == 0 {
			zeroRun++
			if zeroRun > maxZeroRun {
				maxZeroRun = zeroRun
			}
		} else {
			zeroRun = 0
		}
	}

	if maxZeroRun > 1000 {
		result.addWarning(fmt.Sprintf("Large run of zeros detected: %d bytes (possible corruption)", maxZeroRun))
	}

	// Check for null bytes in middle of data
	nullCount := 0
	for _, b := range payload {
		if b == 0 {
			nullCount++
		}
	}
	nullPercent := float64(nullCount) / float64(len(payload)) * 100
	result.addInfo(fmt.Sprintf("Null bytes: %.1f%%", nullPercent))

	if nullPercent > 50 {
		result.addWarning("High percentage of null bytes - file may be corrupted")
	}
}

func printResults(result *ValidationResult) {
	fmt.Println("=== VALIDATION RESULTS ===")
	fmt.Println()

	if len(result.Info) > 0 {
		fmt.Println("INFO:")
		for _, msg := range result.Info {
			fmt.Printf("  • %s\n", msg)
		}
		fmt.Println()
	}

	if len(result.Warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, msg := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", msg)
		}
		fmt.Println()
	}

	if len(result.Errors) > 0 {
		fmt.Println("ERRORS:")
		for _, msg := range result.Errors {
			fmt.Printf("  ✗ %s\n", msg)
		}
		fmt.Println()
	}

	if len(result.Errors) == 0 {
		fmt.Println("✓ Validation PASSED")
	} else {
		fmt.Println("✗ Validation FAILED")
	}
}
