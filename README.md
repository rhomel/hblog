# Static Markdown Site Generator

A minimal Go-based generator that converts Markdown sources in `blog/` into HTML in `public/`.

## Usage

To generate the site (reads blog/index.md → public/index.html), run:

```
make
```

Generated HTML files appear under the `public/` directory.

## Dependencies

This project uses the following third-party Go module:

* github.com/russross/blackfriday/v2 – Markdown processor

### Setup

From the project root, execute the following commands in your shell:

1. Initialize a Go module (replace the module path as needed)

   ```
   go mod init example.com/static-site-generator
   ```

2. Add the Markdown library

   ```
   go get github.com/russross/blackfriday/v2
   ```

3. Tidy up dependencies

   ```
   go mod tidy
   ```

After setting up, build or run with:

```
make
```

