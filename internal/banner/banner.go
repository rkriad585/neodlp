package banner

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"neodlp/internal/version"
)

const width = 56

func padRight(s string, n int) string {
	w := utf8.RuneCountInString(s)
	if w >= n {
		return s
	}
	return s + strings.Repeat(" ", n-w)
}

func line(parts ...string) string {
	inner := strings.Join(parts, "")
	return "│" + padRight(inner, width-2) + "│"
}

func String() string {
	v := version.Version
	c := version.Commit

	titlePad := width - 2 - utf8.RuneCountInString(" neodlp ")
	left := titlePad / 2
	right := titlePad - left

	top := "╭" + strings.Repeat("─", left) + " neodlp " + strings.Repeat("─", right) + "╮"
	bot := "╰" + strings.Repeat("─", width-2) + "╯"

	lines := []string{
		top,
		line("      Author : RK Riad Khan"),
		line("      Version: " + v),
		line("      Commit : " + c),
		line("      GitHub : rkriad585/neodlp"),
		bot,
	}
	return strings.Join(lines, "\n")
}

func Print() {
	fmt.Println(String())
}
