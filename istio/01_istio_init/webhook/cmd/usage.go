package cmd

import (
	"fmt"
	"github.com/logrusorgru/aurora"
)

var colorRender = []func(v any) string{
	func(v any) string {
		return aurora.BrightRed(v).String()
	},
	func(v any) string {
		return aurora.BrightGreen(v).String()
	},
	func(v any) string {
		return aurora.BrightYellow(v).String()
	},
	func(v any) string {
		return aurora.BrightBlue(v).String()
	},
	func(v any) string {
		return aurora.BrightMagenta(v).String()
	},
	func(v any) string {
		return aurora.BrightCyan(v).String()
	},
}

func blue(s string) string {
	return aurora.BrightBlue(s).String()
}

func green(s string) string {
	return aurora.BrightGreen(s).String()
}

func rainbow(s string) string {
	s0 := s[0]
	return colorRender[int(s0)%(len(colorRender)-1)](s)
}

func rpadx(s string, padding int) string {
	template := fmt.Sprintf("%%-%ds", padding)
	return rainbow(fmt.Sprintf(template, s))
}
