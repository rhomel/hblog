# Define common variables
BLOG_DIR := ./blog
BIN := bin/hblog

.PHONY: all build run watch serve

all: run

build:
	go build -o $(BIN) cmd/main.go cmd/theme.go

run: build
	$(BIN)

watch: build
	$(BIN) -watch=$(BLOG_DIR)

serve: build
	$(BIN) -serve -watch=$(BLOG_DIR)
