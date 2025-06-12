package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	// ensure output directory exists
	outDir := "public"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "unable to create output dir: %v\n", err)
		os.Exit(1)
	}

	// read source markdown (content not yet used)
	srcPath := filepath.Join("blog", "index.md")
	if _, err := ioutil.ReadFile(srcPath); err != nil {
		fmt.Fprintf(os.Stderr, "unable to read %s: %v\n", srcPath, err)
		os.Exit(1)
	}

	// produce a simple skeleton
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Blog Index</title>
</head>
<body>
  <h1>Blog Index</h1>
  <p>Source: %s</p>
</body>
</html>`, srcPath)

	outPath := filepath.Join(outDir, "index.html")
	if err := ioutil.WriteFile(outPath, []byte(html), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "unable to write %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s\n", outPath)
}
