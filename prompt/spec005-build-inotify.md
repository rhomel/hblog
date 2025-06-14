Currently the Makefile has only one action `all`.

Split the Makefile into actions:
- build: this should build the go binary and place it into at ./bin/hblog
- run: this should run the built binary
- all: this should do the same thing as run
- typing just "make" should run the "all" target

---

The current hblog command must be invoked on every update.

Update the Go code to include a '-watch=directory-to-watch' option to run as a
blocking process where `directory-to-watch` is a directory to monitor for file
changes. When the process starts, it should log a message "watching
directory-to-watch..."

Use any Linux feature to receive file update events.

When a file has been updated, rerun the *entire* generation process. Only one
instance of the generator should be running at any point in time. When the
generator starts it should emit a "generating..." log message to stdout. When
the generator completes it should emit a "generated N files" message where N is
the total number of generated HTML files.

Use a debounce algorithm with a 0.5 second timer to avoid running the generator
too often.

Avoid using any deprecated methods like ioutil.ReadDir.

---

Update the make file to include a "watch" target. The "watch" target will:
- run the existing "build" target action
- start the binary in watch mode with `./blog` as the directory: ./bin/hblog -watch=./blog

