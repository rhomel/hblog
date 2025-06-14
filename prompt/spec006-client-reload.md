Update the Go code to include a simple HTTP server mode with the -serve=address
flag. If just the `-serve` flag is specified with no argument, the default
value 'localhost:8888' should be used. This mode should start an http server
that serves the static files in ./public directory as its server root.

The http server should also implement necessary endpoints to support
server-sent-events prefixed with /_sse

When the server serves a static HTML file to the client, it should dynamically
inject a small javascript payload into the HTML head that includes code to
automatically reload the current page any time the server requests a reload.

If the command is started with the -watch flag, the watch process should
trigger a server-client reload request after it has finished generating.

---

Update the Makefile to include a new target `serve`. This command should call
the existing `build` command and then start the command with the `-serve` flag
and enable watching on the ./blog directory. Ensure the commonly used values
like the `./blog` directory are only specified once in the Makefile.

