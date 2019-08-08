# go-life

Conway's Game of Life in Go, cross-platform
Uses [Gio](https://git.sr.ht/~eliasnaur/gio) by [@EliasNaur](https://github.com/eliasnaur).

## To build

For the desktop app, Go 1.12 is fine.

To build the web version (which you don't have to do if you only want to run it), you'll need 1.13 / go tip.  (Explaining "go tip" is beyond the scope of this readme, sorry.)

Aagin: the www/ directory in the repo is ready to run, no need to (re)build it.  **So if you just want to try it**, you don't need "go tip".

### Desktop

```bash
% go run life.go
```

### Browser:

```bash
% go get github.com/shurcooL/goexec
% goexec 'http.ListenAndServe(":8080", http.FileServer(http.Dir("www")))'
```

Browse to http://localhost:8080.

## Hotkeys

These work in the desktop app and in the browser.

* q to quit (in the browser this just halts execution)
* `-` (minus key) to reduce the scale (zoom out)
* `+` to increase the scale (zoom in) (`=` also works)
* `<` or `,` to slow down the generations
* `>` or `.` to speed them up

You can't enter or save patterns yet.

