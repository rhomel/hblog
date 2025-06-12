package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/russross/blackfriday/v2"
)

func main() {
	// ensure output directory exists
	outDir := "public"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "unable to create output dir: %v\n", err)
		os.Exit(1)
	}

	// read source markdown
	srcPath := filepath.Join("blog", "index.md")
	md, err := ioutil.ReadFile(srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	// extract first "# " header for title
	title := "(no title)"
	for _, line := range bytes.Split(md, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("# ")) {
			title = strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("# "))))
			break
		}
	}

	// render full markdown to HTML
	bodyHTML := blackfriday.Run(md)

	// wrap in skeleton
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>%s</title>
</head>
<body>
%s
</body>
</html>`, title, bodyHTML)

	outPath := filepath.Join(outDir, "index.html")
	if err := ioutil.WriteFile(outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "unable to write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outPath)
}
