
We are creating a basic markdown static site generator in Golang.

## structure of the project

The project has the following initial directories and empty files:

- README.md
- Makefile
- blog/index.md
- cmd/main.go
- public/.gitkeep

For each new project command, update the README to include usage of the command.

Create a basic project that supports the following commands:

## make

Simply running `make` should run the Go site generator.

For now the site generator should read the empty blog/index.md and generate a
skeleton public/index.html file.

