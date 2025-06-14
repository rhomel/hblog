.PHONY: all build run watch

all: run

build:
	go build -o bin/hblog cmd/main.go

run: build
	./bin/hblog

watch: build
	./bin/hblog -watch=./blog
