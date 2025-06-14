// theme.go
package main

import (
	"bytes"
	"os"
	"regexp"
	"strings"
)

// theme holds site‑wide styling properties.
type theme struct {
	FontFamily        string
	FontColor         string
	BackgroundColor   string
	MaxContentWidth   string
	TextDeEmphasize   string // color for de‑emphasized text, e.g. dates
	ArticleLineHeight string // CSS line-height for article content
}

// loadTheme reads a Markdown theme file and extracts its properties.
// It looks for a “# Properties” section and parses lines like:
//   - font-family: Helvetica
//   - font-color: #333333
//   - text-de-emphasize: #676767
//   - article-line-height: 1.5
func loadTheme(path string) (theme, error) {
	var th theme
	content, err := os.ReadFile(path)
	if err != nil {
		return th, err
	}
	lines := bytes.Split(content, []byte("\n"))
	inProps := false
	re := regexp.MustCompile(`^\-\s*([a-z\-]+):\s*(.+)$`)

	for _, line := range lines {
		trim := strings.TrimSpace(string(line))
		if strings.EqualFold(trim, "# Properties") {
			inProps = true
			continue
		}
		if inProps {
			// stop at next header
			if strings.HasPrefix(trim, "# ") {
				break
			}
			if m := re.FindStringSubmatch(trim); m != nil {
				switch m[1] {
				case "font-family":
					th.FontFamily = m[2]
				case "font-color":
					th.FontColor = m[2]
				case "background-color":
					th.BackgroundColor = m[2]
				case "max-content-width":
					th.MaxContentWidth = m[2]
				case "text-de-emphasize":
					th.TextDeEmphasize = m[2]
				case "article-line-height":
					th.ArticleLineHeight = m[2]
				}
			}
		}
	}
	return th, nil
}
