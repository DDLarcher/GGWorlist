package ui

// This package provides shared UI utilities: ANSI colors, terminal prompts,
// menu selection, file writing with size limits, path validation, and
// the boxed summary output used by both mixer and numeric generators.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Configuration constants
const (
	MaxFileSize       = 1000 * 1000 * 1000 // 1 GB per output file
	MaxInputFileSize  = 100 * 1000 * 1000   // 100 MB max input word file
	progressInterval  = uint64(100000)      // update progress every N words
	largeOutputThreshold = uint64(100_000_000) // warn user if output exceeds this
	defaultMaxSpecLen = 4                   // default max special chars to combine
)

// Logo is the ASCII art banner displayed on startup
const Logo = `
   ___________      __            _____     __
  / ___/ ___/ | /| / /__  _______/ / (_)__ / /_
 / (_ / (_ /| |/ |/ / _ \/ __/ _  / / (_-</ __/
 \___/\___/ |__/|__/\___/_/  \_,_/_/_/___/\__/
                                                `

// ANSI color escape codes for terminal output
const (
	Reset       = "\x1b[0m"
	Red         = "\x1b[31m"
	Green       = "\x1b[32m"
	BrightGreen = "\x1b[92m"
	NeonGreen   = "\x1b[38;2;34;204;255m" // bright light blue text color
	NeonGreenBg = "\x1b[48;2;0;0;0m"       // black background
	White       = "\x1b[37m"
	Yellow      = "\x1b[33m"
	Blue        = "\x1b[34m"
	Magenta     = "\x1b[35m"
	Cyan        = "\x1b[36m"
	Bold        = "\x1b[1m"
)

// noColor controls whether ANSI color codes are stripped from output
var noColor = false

// SetNoColor enables or disables colored output globally.
func SetNoColor(v bool) {
	noColor = v
}

// Colorize wraps a string with an ANSI color code and reset.
// Returns the plain string if noColor is true.
func Colorize(c, s string) string {
	if noColor {
		return s
	}
	return c + s + Reset
}

// PrintLogo prints the ASCII art banner in cyan bold.
func PrintLogo() {
	fmt.Println(Colorize(Cyan+Bold, Logo))
}

// ProgressInterval returns the number of words between progress updates.
func ProgressInterval() uint64 { return progressInterval }

// LargeOutputThreshold returns the output count above which a memory warning is shown.
func LargeOutputThreshold() uint64 { return largeOutputThreshold }

// DefaultMaxSpecLen returns the default maximum number of special chars to combine.
func DefaultMaxSpecLen() int { return defaultMaxSpecLen }

// PrintHeader prints a section header with a divider line underneath.
func PrintHeader(s string) {
	fmt.Println()
	fmt.Println(Colorize(Yellow+Bold, s))
	fmt.Println(Colorize(Yellow, strings.Repeat("-", len(s))))
}

// PrintError prints an error message in red.
func PrintError(s string) {
	fmt.Println(Colorize(Red, s))
}

// PrintPrompt prints a user prompt in green without a newline.
func PrintPrompt(s string) {
	fmt.Print(Colorize(Green, s))
}

// ReadLine reads a line from the reader, trimming whitespace.
// Returns an error if input ends unexpectedly (EOF).
func ReadLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("input ended unexpectedly (EOF)")
	}
	return strings.TrimSpace(line), nil
}

// PromptMenu displays a single-select menu and returns the chosen index.
// If the user presses enter without typing, the defaultIdx is returned.
func PromptMenu(reader *bufio.Reader, title string, options []string, defaultIdx int) int {
	PrintHeader(title)
	for i, opt := range options {
		marker := " "
		if i == defaultIdx {
			marker = Colorize(Yellow, "*")
		}
		fmt.Printf("  %s %s) %s\n", marker, Colorize(White, fmt.Sprintf("%d", i+1)), opt)
	}
	fmt.Println()
	PrintPrompt(fmt.Sprintf("Select an option [1-%d] (default %d): ", len(options), defaultIdx+1))
	line, err := ReadLine(reader)
	if err != nil {
		PrintError(err.Error())
		os.Exit(1)
	}
	if line == "" {
		return defaultIdx
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(options) {
		PrintError(fmt.Sprintf("Invalid choice, using default (%d).", defaultIdx+1))
		return defaultIdx
	}
	return n - 1
}

// PromptMultiMenu displays a multi-select menu and returns the chosen indices.
// Users can enter comma-separated numbers, "all", or press enter for defaults.
func PromptMultiMenu(reader *bufio.Reader, title string, options []string, defaultIdxs []int) []int {
	PrintHeader(title)
	defaultSet := make(map[int]bool)
	for _, i := range defaultIdxs {
		defaultSet[i] = true
	}
	for i, opt := range options {
		marker := " "
		if defaultSet[i] {
			marker = Colorize(Yellow, "*")
		}
		fmt.Printf("  %s %s) %s\n", marker, Colorize(White, fmt.Sprintf("%d", i+1)), opt)
	}
	fmt.Println()
	PrintPrompt("Select options (comma-separated, e.g. 1,3, or 'all', default = starred): ")
	line, err := ReadLine(reader)
	if err != nil {
		PrintError(err.Error())
		os.Exit(1)
	}
	line = strings.ToLower(line)
	if line == "" {
		return defaultIdxs
	}
	if line == "all" {
		idxs := make([]int, len(options))
		for i := range options {
			idxs[i] = i
		}
		return idxs
	}
	var idxs []int
	for _, p := range strings.Split(line, ",") {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err == nil && n >= 1 && n <= len(options) {
			idxs = append(idxs, n-1)
		}
	}
	if len(idxs) == 0 {
		PrintError("No valid choices, using defaults.")
		return defaultIdxs
	}
	return idxs
}

// ChooseOutputDir prompts the user for an output directory path.
// Loops until a valid path is entered (rejects path traversal and system dirs).
func ChooseOutputDir(reader *bufio.Reader) string {
	for {
		fmt.Println()
		PrintPrompt("Output directory [default: output]: ")
		line, err := ReadLine(reader)
		if err != nil {
			PrintError(err.Error())
			os.Exit(1)
		}
		if line == "" {
			line = "output"
		}
		if err := ValidateOutputPath(line); err != nil {
			PrintError(err.Error() + " Try again.")
			continue
		}
		return filepath.Clean(line)
	}
}

// ValidateOutputPath checks that a path is safe for output.
// It rejects paths containing "..", paths outside CWD/home, and
// Windows system directories.
func ValidateOutputPath(path string) error {
	clean := filepath.Clean(path)
	// Reject path traversal attempts
	if strings.Contains(clean, "..") {
		return errors.New("path contains '..' (path traversal not allowed)")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return fmt.Errorf("could not resolve path: %w", err)
	}
	// Only allow output inside the current working directory or home directory
	cwd, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	safe := false
	if cwd != "" && strings.HasPrefix(abs, cwd) {
		safe = true
	}
	if home != "" && strings.HasPrefix(abs, home) {
		safe = true
	}
	if !safe {
		return fmt.Errorf("output path must be inside your working directory (%s) or home directory (%s)", cwd, home)
	}
	// On Windows, also block writes to system directories
	if runtime.GOOS == "windows" {
		winSystem := strings.ToLower(filepath.VolumeName(abs))
		if winSystem == `c:` {
			lower := strings.ToLower(abs)
			for _, f := range []string{`c:\windows`, `c:\program files`, `c:\program files (x86)`, `c:\system volume information`} {
				if strings.HasPrefix(lower, f) {
					return fmt.Errorf("output to system directory is not allowed: %s", abs)
				}
			}
		}
	}
	return nil
}

// ChooseMaxGB prompts the user for a maximum output size in GB.
// Returns 0 if the user leaves it empty (no cap).
func ChooseMaxGB(reader *bufio.Reader) uint64 {
	fmt.Println()
	PrintPrompt("Max total output in GB (empty = no cap): ")
	line, err := ReadLine(reader)
	if err != nil {
		PrintError(err.Error())
		os.Exit(1)
	}
	if line == "" {
		return 0
	}
	gb, err := strconv.ParseFloat(line, 64)
	if err != nil || gb <= 0 {
		PrintError("Invalid value, using no cap.")
		return 0
	}
	return uint64(gb * float64(1000*1000*1000))
}

// ConfirmDir creates the output directory and warns if existing files
// would be overwritten. Returns false if the user aborts.
func ConfirmDir(reader *bufio.Reader, outDir string) bool {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		PrintError("Error creating output directory: " + err.Error())
		return false
	}
	// Check for existing wordlist files that would be overwritten
	entries, err := os.ReadDir(outDir)
	if err != nil {
		return true
	}
	existing := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "wordlist_") && strings.HasSuffix(e.Name(), ".txt") {
			existing++
		}
	}
	if existing > 0 {
		fmt.Println()
		PrintError(fmt.Sprintf("Found %d existing wordlist_*.txt file(s) in '%s'.", existing, outDir))
		PrintPrompt("They will be OVERWRITTEN. Continue? (y/n) [default: n]: ")
		confirm, err := ReadLine(reader)
		if err != nil {
			return false
		}
		confirm = strings.ToLower(confirm)
		if confirm != "y" && confirm != "yes" {
			fmt.Println("Aborted by user.")
			return false
		}
	}
	return true
}

// ValidateInputPath checks that an input file path is safe and within size limits.
// Rejects path traversal, directories, and files larger than MaxInputFileSize.
func ValidateInputPath(path string) error {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return errors.New("path contains '..' (path traversal not allowed)")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return errors.New("path is a directory, not a file")
	}
	if info.Size() > MaxInputFileSize {
		return fmt.Errorf("file is too large (%s, max %s). Use a smaller word list",
			FormatBytes(uint64(info.Size())), FormatBytes(MaxInputFileSize))
	}
	return nil
}

// FormatBytes converts a byte count to a human-readable string (B, KB, MB, GB).
func FormatBytes(b uint64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// PadRight right-pads a string with spaces to the given width.
func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// FileWriter manages output to wordlist files, automatically splitting
// into new files when the size limit (1 GB) is reached.
type FileWriter struct {
	Dir        string    // output directory path
	Num        int       // current file number (1-based)
	File       *os.File  // currently open file handle
	BW         *bufio.Writer // buffered writer for performance
	CurSize    uint64    // current file size in bytes
	Files      int       // total number of files created
	WriteError error     // first write error encountered, if any
}

// NewFileWriter creates a FileWriter for the given directory.
func NewFileWriter(dir string) *FileWriter {
	return &FileWriter{Dir: dir, Num: 1, Files: 0}
}

// EnsureOpen opens a new file if none is currently open.
// Files are named wordlist_001.txt, wordlist_002.txt, etc.
func (fw *FileWriter) EnsureOpen() error {
	if fw.File != nil {
		return nil
	}
	name := fmt.Sprintf("wordlist_%03d.txt", fw.Num)
	path := filepath.Join(fw.Dir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	fw.File = f
	fw.BW = bufio.NewWriterSize(f, 4*1024*1024) // 4 MB buffer
	fw.CurSize = 0
	fw.Files++
	return nil
}

// WriteLine writes a single line followed by a newline to the current file.
// If the current file would exceed MaxFileSize, it closes it and opens a new one.
// Any write error is stored in WriteError and returned on subsequent calls.
func (fw *FileWriter) WriteLine(line string) error {
	if fw.WriteError != nil {
		return fw.WriteError
	}
	if err := fw.EnsureOpen(); err != nil {
		fw.WriteError = err
		return err
	}
	lineBytes := uint64(len(line) + 1) // +1 for newline
	// Check if we need to split to a new file
	if fw.CurSize+lineBytes > MaxFileSize && fw.CurSize > 0 {
		fw.BW.Flush()
		fw.File.Close()
		fw.Num++
		fw.File = nil
		if err := fw.EnsureOpen(); err != nil {
			fw.WriteError = err
			return err
		}
	}
	if _, err := fw.BW.WriteString(line); err != nil {
		fw.WriteError = err
		return err
	}
	if err := fw.BW.WriteByte('\n'); err != nil {
		fw.WriteError = err
		return err
	}
	fw.CurSize += lineBytes
	return nil
}

// Close flushes the buffer and closes the file. Captures any flush error.
func (fw *FileWriter) Close() {
	if fw.BW != nil {
		if err := fw.BW.Flush(); err != nil && fw.WriteError == nil {
			fw.WriteError = err
		}
	}
	if fw.File != nil {
		fw.File.Sync()
		fw.File.Close()
	}
}

// PrintSummary prints a boxed summary of the generation results,
// including stats (words, files, elapsed, rate) and a list of output files.
// Uses a single color (neon green on black) for the entire box.
func PrintSummary(written uint64, files int, elapsed time.Duration, outDir string) {
	fmt.Println()
	fmt.Println()

	// Calculate the generation rate
	rate := "N/A"
	if elapsed.Seconds() > 0 {
		rate = fmt.Sprintf("%.0f words/sec", float64(written)/elapsed.Seconds())
	}

	// Build the stats rows
	stats := [][2]string{
		{"Words written", fmt.Sprintf("%d", written)},
		{"Files created", fmt.Sprintf("%d", files)},
		{"Elapsed time", elapsed.String()},
		{"Generation rate", rate},
		{"Output directory", outDir},
	}

	// Calculate column widths for alignment
	labelWidth := 0
	valueWidth := 0
	for _, s := range stats {
		if len(s[0]) > labelWidth {
			labelWidth = len(s[0])
		}
		if len(s[1]) > valueWidth {
			valueWidth = len(s[1])
		}
	}

	// Ensure the box is wide enough for file listing rows
	maxFileNameLen := 20 + 2 + len(FormatBytes(0))
	if valueWidth < maxFileNameLen {
		valueWidth = maxFileNameLen
	}

	innerWidth := labelWidth + 2 + valueWidth
	g := NeonGreen + NeonGreenBg   // regular green on black
	gb := NeonGreen + NeonGreenBg + Bold // bold green on black

	// Build box border pieces
	top := Colorize(g, " ┌" + strings.Repeat("─", innerWidth) + "┐")
	bottom := Colorize(g, " └" + strings.Repeat("─", innerWidth) + "┘")
	bar := Colorize(g, "│")

	// padRow pads content to fill the box width with borders
	padRow := func(content string) string {
		return bar + " " + PadRight(content, innerWidth) + " " + bar
	}

	// Print the header section
	fmt.Println(top)
	fmt.Println(padRow(Colorize(gb, "Generation Complete")))
	fmt.Println(padRow(""))

	// Print each stat row
	for _, s := range stats {
		label := PadRight(s[0], labelWidth)
		value := s[1]
		row := label + "  " + Colorize(gb, value)
		fmt.Println(padRow(row))
	}

	// Print the file listing section
	fmt.Println(padRow(""))
	fmt.Println(padRow(Colorize(gb, "Output Files")))
	fmt.Println(padRow(""))

	// List all wordlist files with their sizes
	entries, _ := os.ReadDir(outDir)
	fileCount := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "wordlist_") && strings.HasSuffix(e.Name(), ".txt") {
			info, err := os.Stat(filepath.Join(outDir, e.Name()))
			if err != nil {
				continue
			}
			row := PadRight(e.Name(), 20) + "  " + PadRight(FormatBytes(uint64(info.Size())), valueWidth-22)
			fmt.Println(padRow(row))
			fileCount++
		}
	}
	if fileCount == 0 {
		fmt.Println(padRow("(no files found)"))
	}

	fmt.Println(bottom)
}