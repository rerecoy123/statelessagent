// Package cli provides shared formatting helpers for CLI output.
package cli

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color constants.
const (
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Red    = "\033[31m"
	Cyan   = "\033[36m"
	Dim    = "\033[2m"
	Bold   = "\033[1m"
	Reset  = "\033[0m"
)

// Box width is the inner content width (between the border characters).
const boxWidth = 40

// Margin is the left indent for all branded output.
const margin = "  "

// ANSI 256-color red gradient — bright to dark, one per logo line.
var redGradient = []string{
	"\033[38;5;196m", // #ff1a1a bright red
	"\033[38;5;196m", // #f01515
	"\033[38;5;160m", // #e01010
	"\033[38;5;160m", // #d00c0c
	"\033[38;5;124m", // #bf0808
	"\033[38;5;124m", // #af0505
	"\033[38;5;124m", // #9e0404
	"\033[38;5;88m",  // #8e0303
	"\033[38;5;88m",  // #7d0202
	"\033[38;5;88m",  // #6d0202
	"\033[38;5;52m",  // #5c0101
	"\033[38;5;52m",  // #4c0101
}

// ShortenHome replaces $HOME prefix with ~.
func ShortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// FormatNumber adds comma separators (1234 -> "1,234").
func FormatNumber(n int) string {
	if n < 0 {
		return "-" + FormatNumber(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return FormatNumber(n/1000) + "," + fmt.Sprintf("%03d", n%1000)
}

// Banner prints the large STATELESS AGENT ASCII art logo with red gradient
// and tagline. Used by `same init`.
func Banner(version string) {
	logo := []string{
		"  \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557 \u2588\u2588\u2588\u2588\u2588\u2557 \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2557     \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557",
		"  \u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d\u255a\u2550\u2550\u2588\u2588\u2554\u2550\u2550\u255d\u2588\u2588\u2554\u2550\u2550\u2588\u2588\u2557\u255a\u2550\u2550\u2588\u2588\u2554\u2550\u2550\u255d\u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d\u2588\u2588\u2551     \u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d\u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d\u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d",
		"  \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557   \u2588\u2588\u2551   \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2551   \u2588\u2588\u2551   \u2588\u2588\u2588\u2588\u2588\u2557  \u2588\u2588\u2551     \u2588\u2588\u2588\u2588\u2588\u2557  \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557",
		"  \u255a\u2550\u2550\u2550\u2550\u2588\u2588\u2551   \u2588\u2588\u2551   \u2588\u2588\u2554\u2550\u2550\u2588\u2588\u2551   \u2588\u2588\u2551   \u2588\u2588\u2554\u2550\u2550\u255d  \u2588\u2588\u2551     \u2588\u2588\u2554\u2550\u2550\u255d  \u255a\u2550\u2550\u2550\u2550\u2588\u2588\u2551\u255a\u2550\u2550\u2550\u2550\u2588\u2588\u2551",
		"  \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2551   \u2588\u2588\u2551   \u2588\u2588\u2551  \u2588\u2588\u2551   \u2588\u2588\u2551   \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2551\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2551",
		"  \u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d   \u255a\u2550\u255d   \u255a\u2550\u255d  \u255a\u2550\u255d   \u255a\u2550\u255d   \u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d",
		"           \u2588\u2588\u2588\u2588\u2588\u2557  \u2588\u2588\u2588\u2588\u2588\u2588\u2557 \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2557   \u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557",
		"          \u2588\u2588\u2554\u2550\u2550\u2588\u2588\u2557\u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d \u2588\u2588\u2554\u2550\u2550\u2550\u2550\u255d\u2588\u2588\u2588\u2588\u2557  \u2588\u2588\u2551\u255a\u2550\u2550\u2588\u2588\u2554\u2550\u2550\u255d",
		"          \u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2551\u2588\u2588\u2551  \u2588\u2588\u2588\u2557\u2588\u2588\u2588\u2588\u2588\u2557  \u2588\u2588\u2554\u2588\u2588\u2557 \u2588\u2588\u2551   \u2588\u2588\u2551",
		"          \u2588\u2588\u2554\u2550\u2550\u2588\u2588\u2551\u2588\u2588\u2551   \u2588\u2588\u2551\u2588\u2588\u2554\u2550\u2550\u255d  \u2588\u2588\u2551\u255a\u2588\u2588\u2557\u2588\u2588\u2551   \u2588\u2588\u2551",
		"          \u2588\u2588\u2551  \u2588\u2588\u2551\u255a\u2588\u2588\u2588\u2588\u2588\u2588\u2554\u255d\u2588\u2588\u2588\u2588\u2588\u2588\u2588\u2557\u2588\u2588\u2551 \u255a\u2588\u2588\u2588\u2588\u2551   \u2588\u2588\u2551",
		"          \u255a\u2550\u255d  \u255a\u2550\u255d \u255a\u2550\u2550\u2550\u2550\u2550\u255d \u255a\u2550\u2550\u2550\u2550\u2550\u2550\u255d\u255a\u2550\u255d  \u255a\u2550\u2550\u2550\u255d   \u255a\u2550\u255d",
	}

	fmt.Println()
	for i, line := range logo {
		color := redGradient[i%len(redGradient)]
		fmt.Printf("%s%s%s\n", color, line, Reset)
	}
	fmt.Println()
	fmt.Printf("  %sEvery AI session starts from zero.%s %s%sNot anymore.%s\n",
		Dim, Reset, Bold, Red, Reset)
	fmt.Println()
	fmt.Printf("  %sSAME%s %s\u2014 Stateless Agent Memory Engine v%s%s\n",
		Bold, Reset, Dim, version, Reset)
}

// Header prints a small heavy-border box with a title. Used by `same status` and `same doctor`.
func Header(title string) {
	fmt.Println()
	heavyTop := margin + "\u250f" + strings.Repeat("\u2501", boxWidth) + "\u2513"
	heavyBottom := margin + "\u2517" + strings.Repeat("\u2501", boxWidth) + "\u251b"

	content := "  " + title
	padded := padRight(content, boxWidth)

	fmt.Printf("%s%s%s\n", Cyan, heavyTop, Reset)
	fmt.Printf("%s%s\u2503%s\u2503%s\n", Cyan, margin, padded, Reset)
	fmt.Printf("%s%s%s\n", Cyan, heavyBottom, Reset)
}

// Section prints a section divider line: ── Name ─────────────────
func Section(name string) {
	prefix := "\u2500\u2500 " + name + " "
	remaining := boxWidth + 2 - runeLen(prefix)
	if remaining < 0 {
		remaining = 0
	}
	rule := prefix + strings.Repeat("\u2500", remaining)
	fmt.Printf("\n%s%s%s%s%s\n\n", margin, Cyan, rule, Reset, "")
}

// Box prints a light-border box around content lines.
func Box(lines []string) {
	lightTop := margin + "\u250c" + strings.Repeat("\u2500", boxWidth) + "\u2510"
	lightBottom := margin + "\u2514" + strings.Repeat("\u2500", boxWidth) + "\u2518"

	fmt.Println()
	fmt.Println(lightTop)
	for _, line := range lines {
		content := "  " + line
		padded := padRight(content, boxWidth)
		fmt.Printf("%s\u2502%s\u2502\n", margin, padded)
	}
	fmt.Println(lightBottom)
}

// Footer prints the branded footer in dim text.
func Footer() {
	fmt.Printf("\n%s%sstatelessagent.com \u00b7 sgx-labs/statelessagent%s\n\n", margin, Dim, Reset)
}

// padRight pads s with spaces to exactly width characters.
// If s is longer than width, it is truncated.
func padRight(s string, width int) string {
	n := runeLen(s)
	if n >= width {
		r := []rune(s)
		return string(r[:width])
	}
	return s + strings.Repeat(" ", width-n)
}

// runeLen counts the display width in runes.
func runeLen(s string) int {
	return len([]rune(s))
}
