package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/russross/blackfriday/v2"
)

type article struct {
	Date     time.Time
	DateStr  string
	Title    string
	HTMLPath string
}

type theme struct {
	FontFamily      string
	FontColor       string
	BackgroundColor string
	MaxContentWidth string
}

func main() {
	// load theme
	themeFile := "blog/themes/default.md"
	th, err := loadTheme(themeFile)
	if err != nil {
		log.Fatalf("unable to load theme %s: %v", themeFile, err)
	}

	// build CSS style from theme
	style := fmt.Sprintf(`body { font-family: %s; color: %s; background-color: %s; }`+"\n"+`.container { max-width: %s; margin: auto; }`,
		th.FontFamily, th.FontColor, th.BackgroundColor, th.MaxContentWidth)

	// ensure output directories exist
	if err := os.MkdirAll("public/articles", 0755); err != nil {
		log.Fatalf("unable to create output dir: %v", err)
	}

	// --- process blog/index.md ---
	indexSrc := "blog/index.md"
	mdIndex, err := ioutil.ReadFile(indexSrc)
	if err != nil {
		log.Fatalf("unable to read %s: %v", indexSrc, err)
	}
	indexTitle := extractTitle(mdIndex)
	indexHTML := blackfriday.Run(mdIndex)

	// --- process blog/articles/*.md ---
	articles, warnings := loadArticles("blog/articles", style)
	for _, w := range warnings {
		log.Println("warning:", w)
	}

	// sort newest first
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Date.After(articles[j].Date)
	})

	// build articles list section
	var listBuf bytes.Buffer
	if len(articles) > 0 {
		listBuf.WriteString("<h2>Articles</h2>\n<ul>\n")
		for _, a := range articles {
			listBuf.WriteString(fmt.Sprintf(
				"  <li>[%s] <a href=\"%s\">%s</a></li>\n",
				a.DateStr, a.HTMLPath, a.Title,
			))
		}
		listBuf.WriteString("</ul>\n")
	}

	// --- write public/index.html ---
	indexOut := "public/index.html"
	fullIndex := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset=\"utf-8\">
  <title>%s</title>
  <style>
%s
  </style>
</head>
<body>
  <div class="container">%s%s  </div>
</body>
</html>`,
		indexTitle, style, indexHTML, listBuf.String())

	if err := ioutil.WriteFile(indexOut, []byte(fullIndex), 0644); err != nil {
		log.Fatalf("unable to write %s: %v", indexOut, err)
	}

	fmt.Printf("Generated %s with %d articles\n", indexOut, len(articles))
}

// loadArticles reads markdown files from dir, returns valid articles and warnings.
func loadArticles(dir, style string) ([]article, []string) {
	var arts []article
	var warns []string
	re := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)\.md$`)

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		warns = append(warns, fmt.Sprintf("cannot read directory %s: %v", dir, err))
		return arts, warns
	}

	for _, fi := range entries {
		if fi.IsDir() {
			continue
		}
		name := fi.Name()
		m := re.FindStringSubmatch(name)
		if m == nil {
			warns = append(warns, fmt.Sprintf("%s: filename does not match YYYY-MM-DD-name.md", name))
			continue
		}
		dateStr := m[1]
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			warns = append(warns, fmt.Sprintf("%s: invalid date %s", name, dateStr))
			continue
		}

		srcPath := filepath.Join(dir, name)
		md, err := ioutil.ReadFile(srcPath)
		if err != nil {
			warns = append(warns, fmt.Sprintf("%s: unable to read: %v", name, err))
			continue
		}
		title := extractTitle(md)

		htmlName := strings.TrimSuffix(name, ".md") + ".html"
		htmlPath := "articles/" + htmlName
		outPath := filepath.Join("public", htmlPath)

		// write individual article file
		articleHTML := blackfriday.Run(md)
		full := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset=\"utf-8\">
  <title>%s</title>
  <style>
%s
  </style>
</head>
<body>
  <div class="container">%s
  </div>
</body>
</html>`, title, style, articleHTML)

		if err := ioutil.WriteFile(outPath, []byte(full), 0644); err != nil {
			warns = append(warns, fmt.Sprintf("%s: write error: %v", htmlName, err))
			continue
		}

		arts = append(arts, article{
			Date:     date,
			DateStr:  dateStr,
			Title:    title,
			HTMLPath: htmlPath,
		})
	}

	return arts, warns
}

// loadTheme parses a theme markdown, extracting properties under "# Properties".
func loadTheme(path string) (theme, error) {
	var th theme
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return th, err
	}
	lines := bytes.Split(content, []byte("\n"))
	inProps := false
	re := regexp.MustCompile(`^-\s*([a-z-]+):\s*(.+)$`)
	for _, line := range lines {
		trim := strings.TrimSpace(string(line))
		if strings.EqualFold(trim, "# Properties") {
			inProps = true
			continue
		}
		if inProps {
			if strings.HasPrefix(trim, "# ") {
				break
			}
			m := re.FindStringSubmatch(trim)
			if m != nil {
				key := m[1]
				val := m[2]
				switch key {
				case "font-family":
					th.FontFamily = val
				case "font-color":
					th.FontColor = val
				case "background-color":
					th.BackgroundColor = val
				case "max-content-width":
					th.MaxContentWidth = val
				}
			}
		}
	}
	return th, nil
}

// extractTitle finds the first "# " header or returns "(no title)".
func extractTitle(md []byte) string {
	for _, line := range bytes.Split(md, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("# ")) {
			return strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("# "))))
		}
	}
	return "(no title)"
}
