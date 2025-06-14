package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
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
	MDName   string
	Date     time.Time
	DateStr  string
	Title    string
	HTMLPath string
}

// broadcaster for SSE reload events
var clientsMu sync.Mutex
var clients = make(map[chan string]struct{})

func main() {
	// allow "-serve" with no value to default to localhost:8888
	for i, arg := range os.Args {
		if arg == "-serve" {
			os.Args[i] = "-serve=localhost:8888"
		}
	}

	watchDir := flag.String("watch", "", "directory to watch for changes")
	serveAddr := flag.String("serve", "localhost:8888", "address to serve HTTP ("+"localhost:8888)")
	flag.Parse()

	// if serve only
	if *serveAddr != "" && *watchDir == "" {
		startServer(*serveAddr)
		return
	}

	// watch mode with optional serve
	if *watchDir != "" {
		// start HTTP if needed
		if *serveAddr != "" {
			go startServer(*serveAddr)
		}

		fmt.Printf("watching %s...\n", *watchDir)
		watchAndGenerate(*watchDir)
		return
	}

	// no flags: single generation
	executeOnce()
}

func startServer(addr string) {
	mux := http.NewServeMux()
	// static files
	fileHandler := http.FileServer(http.Dir("public"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".html") || r.URL.Path == "/" {
			// load file
			path := filepath.Join("public", r.URL.Path)
			if r.URL.Path == "/" {
				path = filepath.Join("public", "index.html")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			// inject reload script before </head>
			script := `<script>
			var es = new EventSource('/_sse');
			es.onmessage = function(e) { if (e.data === 'reload') window.location.reload(); };
			</script>`
			out := bytes.Replace(data, []byte("</head>"), []byte(script+"</head>"), 1)
			w.Write(out)
			return
		}
		// other assets
		fileHandler.ServeHTTP(w, r)
	})
	// SSE endpoint
	mux.HandleFunc("/_sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		ch := make(chan string)
		clientsMu.Lock()
		clients[ch] = struct{}{}
		clientsMu.Unlock()

		notify := w.(http.CloseNotifier).CloseNotify()
		go func() {
			<-notify
			clientsMu.Lock()
			delete(clients, ch)
			clientsMu.Unlock()
		}()

		for msg := range ch {
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})

	log.Printf("serving on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func broadcast(msg string) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func watchAndGenerate(dir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("unable to init watcher: %v", err)
	}
	defer watcher.Close()
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	// initial gen
	log.Println("generating...")
	executeOnceWithBroadcast()

	// debounce
	var mu sync.Mutex
	var timer *time.Timer
	reset := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(500*time.Millisecond, func() {
			log.Println("generating...")
			executeOnceWithBroadcast()
		})
	}

	for {
		select {
		case ev := <-watcher.Events:
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				reset()
			}
		case err := <-watcher.Errors:
			log.Printf("watch error: %v", err)
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

func executeOnceWithBroadcast() {
	count, err := runGeneration()
	if err != nil {
		log.Printf("generation error: %v", err)
	} else {
		fmt.Printf("generated %d files\n", count)
		broadcast("reload")
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
		`
	body { font-family: %s; color: %s; background-color: %s; }
	.container { max-width: %s; margin: auto; }
	.container p { line-height: %s }
		`,
		th.FontFamily, th.FontColor, th.BackgroundColor, th.MaxContentWidth, th.ArticleLineHeight,
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
	listHTML := buildListHTML(articles, th.TextDeEmphasize)
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

func buildListHTML(articles []article, dateColor string) string {
	var buf bytes.Buffer
	if len(articles) > 0 {
		buf.WriteString("<h2>Articles</h2>\n<ul>\n")
		for _, a := range articles {
			// date without brackets, colored de‑emphasized
			buf.WriteString(fmt.Sprintf(
				"  <li><span style=\"color: %s\">%s</span> <a href=\"%s\">%s</a></li>\n",
				dateColor, a.DateStr, a.HTMLPath, a.Title,
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

func extractTitle(md []byte) string {
	for _, line := range bytes.Split(md, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("# ")) {
			return strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("# "))))
		}
	}
	return "(no title)"
}
