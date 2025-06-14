package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/russross/blackfriday/v2"
)

type article struct {
	MDName   string    // markdown filename
	Date     time.Time // parsed date
	DateStr  string    // date string
	Title    string    // extracted title
	HTMLPath string    // output html relative path
}

type theme struct {
	FontFamily      string
	FontColor       string
	BackgroundColor string
	MaxContentWidth string
}

func main() {
	watchDir := flag.String("watch", "", "directory to watch for changes")
	flag.Parse()

	if *watchDir == "" {
		executeOnce()
		return
	}

	// watch mode
	fmt.Printf("watching %s...\n", *watchDir)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("unable to initialize watcher: %v", err)
	}
	defer watcher.Close()

	// watch recursively
	err = filepath.WalkDir(*watchDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("unable to watch directory %s: %v", *watchDir, err)
	}

	// initial generate
	fmt.Println("generating...")
	executeOnceWithLogs()

	// debounce setup
	var mu sync.Mutex
	var timer *time.Timer
	reset := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(500*time.Millisecond, func() {
			fmt.Println("generating...")
			executeOnceWithLogs()
		})
	}

	// event loop
	for {
		select {
		case ev, ok := <-watcher.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				reset()
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("watcher error: %v", err)
		}
	}
}

func executeOnce() {
	count, err := runGeneration()
	if err != nil {
		log.Fatalf("generation error: %v", err)
	}
	fmt.Printf("generated %d files\n", count)
}

func executeOnceWithLogs() {
	count, err := runGeneration()
	if err != nil {
		log.Printf("generation error: %v", err)
	} else {
		fmt.Printf("generated %d files\n", count)
	}
}

// runGeneration performs a single site generation. Returns total HTML files produced.
func runGeneration() (int, error) {
	// load theme
	th, err := loadTheme("blog/themes/default.md")
	if err != nil {
		return 0, fmt.Errorf("unable to load theme: %w", err)
	}
	style := fmt.Sprintf(
		`body { font-family: %s; color: %s; background-color: %s; }`+"\n"+
			`.container { max-width: %s; margin: auto; }`,
		th.FontFamily, th.FontColor, th.BackgroundColor, th.MaxContentWidth,
	)

	// ensure output directories exist
	if err := os.MkdirAll("public/articles", 0755); err != nil {
		return 0, err
	}

	// process index.md
	mdIndex, err := os.ReadFile("blog/index.md")
	if err != nil {
		return 0, err
	}
	indexTitle := extractTitle(mdIndex)
	indexHTML := blackfriday.Run(mdIndex)

	// process articles list
	articles, warnings := loadArticles("blog/articles")
	for _, w := range warnings {
		log.Println("warning:", w)
	}

	// write article pages
	articleCount := writeArticles(articles, style)

	// write index.html (including articles list)
	listHTML := buildListHTML(articles)
	full := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s</title>
  <style>
%s
  </style>
</head>
<body>
  <div class="container">
%s
%s
  </div>
</body>
</html>`,
		indexTitle, style, indexHTML, listHTML,
	)
	if err := os.WriteFile("public/index.html", []byte(full), 0644); err != nil {
		return articleCount, err
	}

	// +1 for index
	return articleCount + 1, nil
}

func writeArticles(articles []article, style string) int {
	count := 0
	for _, a := range articles {
		mdPath := filepath.Join("blog/articles", a.MDName)
		md, err := os.ReadFile(mdPath)
		if err != nil {
			continue
		}
		htmlContent := blackfriday.Run(md)
		page := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s</title>
  <style>
%s
  </style>
</head>
<body>
  <div class="container">
%s
  </div>
</body>
</html>`,
			a.Title, style, htmlContent,
		)
		out := filepath.Join("public", a.HTMLPath)
		if err := os.WriteFile(out, []byte(page), 0644); err == nil {
			count++
		}
	}
	return count
}

func buildListHTML(articles []article) string {
	var buf bytes.Buffer
	if len(articles) > 0 {
		buf.WriteString("<h2>Articles</h2>\n<ul>\n")
		for _, a := range articles {
			buf.WriteString(fmt.Sprintf(
				"  <li>[%s] <a href=\"%s\">%s</a></li>\n",
				a.DateStr, a.HTMLPath, a.Title,
			))
		}
		buf.WriteString("</ul>\n")
	}
	return buf.String()
}

func loadArticles(dir string) ([]article, []string) {
	var arts []article
	var warns []string
	re := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.+)\.md$`)
	entries, err := os.ReadDir(dir)
	if err != nil {
		warns = append(warns, fmt.Sprintf("cannot read directory %s: %v", dir, err))
		return arts, warns
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		m := re.FindStringSubmatch(name)
		if m == nil {
			warns = append(warns, fmt.Sprintf("%s: filename does not match YYYY-MM-DD-name.md", name))
			continue
		}
		date, err := time.Parse("2006-01-02", m[1])
		if err != nil {
			warns = append(warns, fmt.Sprintf("%s: invalid date %s", name, m[1]))
			continue
		}
		title := extractTitleFromFile(filepath.Join(dir, name))
		htmlName := strings.TrimSuffix(name, ".md") + ".html"
		arts = append(arts, article{
			MDName:   name,
			Date:     date,
			DateStr:  m[1],
			Title:    title,
			HTMLPath: "articles/" + htmlName,
		})
	}
	sort.Slice(arts, func(i, j int) bool { return arts[i].Date.After(arts[j].Date) })
	return arts, warns
}

func extractTitleFromFile(path string) string {
	md, err := os.ReadFile(path)
	if err != nil {
		return "(no title)"
	}
	return extractTitle(md)
}

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
				}
			}
		}
	}
	return th, nil
}

func extractTitle(md []byte) string {
	for _, line := range bytes.Split(md, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("# ")) {
			return strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("# "))))
		}
	}
	return "(no title)"
}
