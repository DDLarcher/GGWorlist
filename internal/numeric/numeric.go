// Package numeric provides exhaustive digit combination generation.
// It generates all digit strings (0-9) of a given length, skipping any
// with 3 identical consecutive digits. Includes overflow-safe size estimation.
package numeric

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"wordlist-generator/internal/ui"
)

// countValid calculates the number of valid digit strings of length n
// (no 3 identical consecutive digits) using a recurrence relation:
// a(n) = strings ending in a single (non-paired) last digit
// b(n) = strings ending with a pair (last 2 same)
// a(1)=10, b(1)=0; a(n) = (a+b)*9; b(n) = a(n-1)
func countValid(n int) uint64 {
	if n <= 0 {
		return 0
	}
	if n == 1 {
		return 10
	}
	var a, b uint64 = 10, 0
	for i := 2; i <= n; i++ {
		na := (a + b) * 9
		nb := a
		a = na
		b = nb
	}
	return a + b
}

// CountValid is the exported wrapper around countValid.
func CountValid(n int) uint64 {
	return countValid(n)
}

// RunNonInteractive generates digit combinations and writes them to files.
// Uses stderr for progress output. No user interaction.
func RunNonInteractive(outDir string, maxBytes uint64, n int) {
	RunNonInteractiveWithProgress(outDir, maxBytes, n, nil)
}

// RunNonInteractiveWithProgress generates digit combinations with an optional
// progress callback. The progressFn is called periodically with the current
// word count and file number. If nil, progress goes to stderr.
func RunNonInteractiveWithProgress(outDir string, maxBytes uint64, n int, progressFn func(written uint64, files int)) {
	total := countValid(n)
	bytesPerWord := uint64(n + 1)

	var totalBytes uint64
	if total > 0 && bytesPerWord > 0 && total > ^uint64(0)/bytesPerWord {
		totalBytes = ^uint64(0)
	} else {
		totalBytes = total * bytesPerWord
	}

	fmt.Println()
	fmt.Println(ui.Colorize(ui.Yellow+ui.Bold, "Estimated output:"))
	fmt.Printf("  Valid combinations: %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", total)))
	fmt.Printf("  Storage needed:     %s\n", ui.Colorize(ui.Cyan, ui.FormatBytes(totalBytes)))
	fmt.Printf("  Per word:           %d bytes\n", bytesPerWord)

	fmt.Println()
	fmt.Println(ui.Colorize(ui.Yellow+ui.Bold, "Configuration"))
	fmt.Println(ui.Colorize(ui.Yellow, "-------------"))
	fmt.Printf("  Length:           %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", n)))
	fmt.Printf("  Charset:          %s\n", ui.Colorize(ui.Cyan, "0-9 (10 digits)"))
	fmt.Printf("  Total combos:     %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", total)))
	fmt.Printf("  Est. storage:     %s\n", ui.Colorize(ui.Cyan, ui.FormatBytes(totalBytes)))
	if maxBytes > 0 {
		fmt.Printf("  Max total size:   %s\n", ui.Colorize(ui.Cyan, ui.FormatBytes(maxBytes)))
	} else {
		fmt.Printf("  Max total size:   %s\n", ui.Colorize(ui.Cyan, "no cap"))
	}
	fmt.Println()
	fmt.Println(ui.Colorize(ui.Green, "Generating..."))

	start := time.Now()
	fw := ui.NewFileWriter(outDir)

	var written uint64
	var stopped bool
	progressEvery := uint64(1000000)
	if total < 1000000 {
		progressEvery = total / 100
		if progressEvery == 0 {
			progressEvery = 1
		}
	}

	digits := make([]byte, n)
	var generate func(pos int, last1, last2 byte)
	generate = func(pos int, last1, last2 byte) {
		if stopped {
			return
		}
		if pos == n {
			if err := fw.WriteLine(string(digits)); err != nil {
				stopped = true
				return
			}
			written++
			if written%progressEvery == 0 {
				if progressFn != nil {
					progressFn(written, fw.Files)
				} else {
					fmt.Fprintf(os.Stderr, "\rWritten: %d / %d (%.1f%%)  File #%d",
						written, total, float64(written)/float64(total)*100, fw.Files)
				}
			}
			if maxBytes > 0 && fw.CurSize > maxBytes {
				stopped = true
			}
			return
		}
		for d := byte(0); d < 10; d++ {
			if pos >= 2 && d == last1 && d == last2 {
				continue
			}
			digits[pos] = '0' + d
			generate(pos+1, d, last1)
			if stopped {
				return
			}
		}
	}

	generate(0, 0xFF, 0xFF)

	elapsed := time.Since(start)
	fmt.Fprintln(os.Stderr)
	fw.Close()

	if stopped {
		fmt.Println()
		if fw.WriteError != nil {
			ui.PrintError("Write error: " + fw.WriteError.Error())
		} else if maxBytes > 0 {
			ui.PrintError(fmt.Sprintf("Stopped at size cap (%s). Wrote %d of %d combinations.",
				ui.FormatBytes(maxBytes), written, total))
		}
	}
	ui.PrintSummary(written, fw.Files, elapsed, outDir)
}

// Run is the interactive entry point for the numeric generator.
// It collects configuration via prompts, shows estimates, and generates.
func Run(reader *bufio.Reader, outDir string, maxBytes uint64) {
	ui.PrintHeader("Numeric Generator Mode")
	fmt.Println("Generates ALL digit combinations (0-9) of a given length,")
	fmt.Println("skipping any with 3 identical consecutive digits.")
	fmt.Println()

	ui.PrintPrompt("String length (1-14 recommended): ")
	line, err := ui.ReadLine(reader)
	if err != nil {
		ui.PrintError(err.Error())
		return
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 {
		ui.PrintError("Invalid length. Exiting.")
		return
	}
	if n > 16 {
		ui.PrintError("Length > 16 would produce exabyte-scale output. Exiting.")
		return
	}

	total := countValid(n)
	bytesPerWord := uint64(n + 1)

	var totalBytes uint64
	if total > 0 && bytesPerWord > 0 && total > ^uint64(0)/bytesPerWord {
		totalBytes = ^uint64(0)
	} else {
		totalBytes = total * bytesPerWord
	}

	fmt.Println()
	fmt.Println(ui.Colorize(ui.Yellow+ui.Bold, "Estimated output:"))
	fmt.Printf("  Valid combinations: %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", total)))
	fmt.Printf("  Storage needed:     %s\n", ui.Colorize(ui.Cyan, ui.FormatBytes(totalBytes)))
	fmt.Printf("  Per word:           %d bytes (digit + newline)\n", bytesPerWord)

	if maxBytes > 0 && totalBytes > maxBytes {
		fmt.Println()
		ui.PrintError(fmt.Sprintf("WARNING: Estimated %s exceeds your cap of %s!", ui.FormatBytes(totalBytes), ui.FormatBytes(maxBytes)))
		fmt.Println(ui.Colorize(ui.Red, "The generator will STOP when it hits the size cap."))
		fmt.Println()
		ui.PrintPrompt("Continue anyway? (y/n) [default: n]: ")
		confirm, err := ui.ReadLine(reader)
		if err != nil {
			return
		}
		confirm = strings.ToLower(confirm)
		if confirm != "y" && confirm != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	fmt.Println()
	fmt.Println(ui.Colorize(ui.Yellow+ui.Bold, "Configuration"))
	fmt.Println(ui.Colorize(ui.Yellow, "-------------"))
	fmt.Printf("  Length:           %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", n)))
	fmt.Printf("  Charset:          %s\n", ui.Colorize(ui.Cyan, "0-9 (10 digits)"))
	fmt.Printf("  Total combos:     %s\n", ui.Colorize(ui.Cyan, fmt.Sprintf("%d", total)))
	fmt.Printf("  Est. storage:     %s\n", ui.Colorize(ui.Cyan, ui.FormatBytes(totalBytes)))
	if maxBytes > 0 {
		fmt.Printf("  Max total size:   %s\n", ui.Colorize(ui.Cyan, ui.FormatBytes(maxBytes)))
	} else {
		fmt.Printf("  Max total size:   %s\n", ui.Colorize(ui.Cyan, "no cap"))
	}
	fmt.Printf("  Max file size:    %s\n", ui.Colorize(ui.Cyan, "1 GB per file"))
	fmt.Println()
	fmt.Println(ui.Colorize(ui.Green, "Generating..."))

	start := time.Now()
	fw := ui.NewFileWriter(outDir)

	var written uint64
	var stopped bool
	progressEvery := uint64(1000000)
	if total < 1000000 {
		progressEvery = total / 100
		if progressEvery == 0 {
			progressEvery = 1
		}
	}

	digits := make([]byte, n)
	var generate func(pos int, last1, last2 byte)
	generate = func(pos int, last1, last2 byte) {
		if stopped {
			return
		}
		if pos == n {
			if err := fw.WriteLine(string(digits)); err != nil {
				stopped = true
				return
			}
			written++
			if written%progressEvery == 0 {
				fmt.Fprintf(os.Stderr, "\rWritten: %d / %d (%.1f%%)  File #%d",
					written, total, float64(written)/float64(total)*100, fw.Files)
			}
			if maxBytes > 0 && fw.CurSize > maxBytes {
				stopped = true
			}
			return
		}
		for d := byte(0); d < 10; d++ {
			if pos >= 2 && d == last1 && d == last2 {
				continue
			}
			digits[pos] = '0' + d
			generate(pos+1, d, last1)
			if stopped {
				return
			}
		}
	}

	generate(0, 0xFF, 0xFF)

	elapsed := time.Since(start)
	fmt.Fprintln(os.Stderr)
	fw.Close()

	if stopped {
		fmt.Println()
		if fw.WriteError != nil {
			ui.PrintError("Write error: " + fw.WriteError.Error())
			ui.PrintError(fmt.Sprintf("Aborted. Wrote %d of %d combinations before failure.", written, total))
		} else if maxBytes > 0 {
			ui.PrintError(fmt.Sprintf("Stopped at size cap (%s). Wrote %d of %d combinations.",
				ui.FormatBytes(maxBytes), written, total))
		}
	}
	ui.PrintSummary(written, fw.Files, elapsed, outDir)
}