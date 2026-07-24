package ui

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

const (
	MaxFileSize       = 1000 * 1000 * 1000
	MaxInputFileSize  = 100 * 1000 * 1000
	progressInterval  = uint64(100000)
	largeOutputThreshold = uint64(100_000_000)
	defaultMaxSpecLen = 4
)

const Logo = `
   ___________      __            _____     __
  / ___/ ___/ | /| / /__  _______/ / (_)__ / /_
 / (_ / (_ /| |/ |/ / _ \/ __/ _  / / (_-</ __/
 \___/\___/ |__/|__/\___/_/  \_,_/_/_/___/\__/
                                               `

const (
	Reset      = "\x1b[0m"
	Red        = "\x1b[31m"
	Green      = "\x1b[32m"
	BrightGreen = "\x1b[92m"
	NeonGreen  = "\x1b[38;2;34;204;255m"
	NeonGreenBg = "\x1b[48;2;0;0;0m"
	White      = "\x1b[37m"
	Yellow     = "\x1b[33m"
	Blue       = "\x1b[34m"
	Magenta    = "\x1b[35m"
	Cyan       = "\x1b[36m"
	Bold       = "\x1b[1m"
)

var noColor = false

func SetNoColor(v bool) {
	noColor = v
}

func Colorize(c, s string) string {
	if noColor {
		return s
	}
	return c + s + Reset
}

func PrintLogo() {
	fmt.Println(Colorize(Cyan+Bold, Logo))
}

func ProgressInterval() uint64 { return progressInterval }
func LargeOutputThreshold() uint64 { return largeOutputThreshold }
func DefaultMaxSpecLen() int { return defaultMaxSpecLen }

func PrintHeader(s string) {
	fmt.Println()
	fmt.Println(Colorize(Yellow+Bold, s))
	fmt.Println(Colorize(Yellow, strings.Repeat("-", len(s))))
}

func PrintError(s string) {
	fmt.Println(Colorize(Red, s))
}

func PrintPrompt(s string) {
	fmt.Print(Colorize(Green, s))
}

func ReadLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("input ended unexpectedly (EOF)")
	}
	return strings.TrimSpace(line), nil
}

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

func ValidateOutputPath(path string) error {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return errors.New("path contains '..' (path traversal not allowed)")
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return fmt.Errorf("could not resolve path: %w", err)
	}
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

func ConfirmDir(reader *bufio.Reader, outDir string) bool {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		PrintError("Error creating output directory: " + err.Error())
		return false
	}
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

func PadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

type FileWriter struct {
	Dir        string
	Num        int
	File       *os.File
	BW         *bufio.Writer
	CurSize    uint64
	Files      int
	WriteError error
}

func NewFileWriter(dir string) *FileWriter {
	return &FileWriter{Dir: dir, Num: 1, Files: 0}
}

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
	fw.BW = bufio.NewWriterSize(f, 4*1024*1024)
	fw.CurSize = 0
	fw.Files++
	return nil
}

func (fw *FileWriter) WriteLine(line string) error {
	if fw.WriteError != nil {
		return fw.WriteError
	}
	if err := fw.EnsureOpen(); err != nil {
		fw.WriteError = err
		return err
	}
	lineBytes := uint64(len(line) + 1)
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

func PrintSummary(written uint64, files int, elapsed time.Duration, outDir string) {
	fmt.Println()
	fmt.Println()

	rate := "N/A"
	if elapsed.Seconds() > 0 {
		rate = fmt.Sprintf("%.0f words/sec", float64(written)/elapsed.Seconds())
	}

	stats := [][2]string{
		{"Words written", fmt.Sprintf("%d", written)},
		{"Files created", fmt.Sprintf("%d", files)},
		{"Elapsed time", elapsed.String()},
		{"Generation rate", rate},
		{"Output directory", outDir},
	}

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

	maxFileNameLen := 20 + 2 + len(FormatBytes(0))
	if valueWidth < maxFileNameLen {
		valueWidth = maxFileNameLen
	}

	innerWidth := labelWidth + 2 + valueWidth
	g := NeonGreen + NeonGreenBg
	gb := NeonGreen + NeonGreenBg + Bold

	top := Colorize(g, " ┌" + strings.Repeat("─", innerWidth) + "┐")
	bottom := Colorize(g, " └" + strings.Repeat("─", innerWidth) + "┘")
	bar := Colorize(g, "│")

	padRow := func(content string) string {
		return bar + " " + PadRight(content, innerWidth) + " " + bar
	}

	fmt.Println(top)
	fmt.Println(padRow(Colorize(gb, "Generation Complete")))
	fmt.Println(padRow(""))

	for _, s := range stats {
		label := PadRight(s[0], labelWidth)
		value := s[1]
		row := label + "  " + Colorize(gb, value)
		fmt.Println(padRow(row))
	}

	fmt.Println(padRow(""))
	fmt.Println(padRow(Colorize(gb, "Output Files")))
	fmt.Println(padRow(""))

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