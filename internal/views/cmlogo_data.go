package views

import (
	_ "embed"
	"strings"
)

//go:embed cmlogo.txt
var cmLogoRaw string

// CMLogoText is the ASCII logo rendered into the /demo/cmlogo page. The fixed
// CMLogoCols/CMLogoRows dimensions feed logo-fit.js so the static <pre> scales
// to fill the band without measuring the text content on the client.
var (
	CMLogoText string
	CMLogoCols int
	CMLogoRows int
)

func init() {
	CMLogoText = strings.TrimRight(cmLogoRaw, "\n")
	lines := strings.Split(CMLogoText, "\n")
	CMLogoRows = len(lines)
	for _, ln := range lines {
		if len(ln) > CMLogoCols {
			CMLogoCols = len(ln)
		}
	}
}
