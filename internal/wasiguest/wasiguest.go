// Package wasiguest embeds the compiled app guest (cmd/sharecrop-wasi-app-guest)
// so cmd/sharecrop can run production through the WASI host as a single
// artifact — the same WASM the browser demo runs, now the default for the
// server too.
//
// The committed app-guest.wasm is an EMPTY placeholder; `make wasi-app-guest`
// (run by `make build`) overwrites it with the real guest. An empty embed means
// the binary was not built for WASI, so serve falls back to the native mux;
// a non-empty guest is used by default and serve fails loudly if it cannot load.
package wasiguest

import _ "embed"

//go:embed app-guest.wasm
var Guest []byte

// Embedded reports whether a real guest (not the empty placeholder) is embedded.
func Embedded() bool { return len(Guest) > 0 }
