//go:build !js || !wasm

package main

import "fmt"

func main() {
	fmt.Println("sharecrop-wasm must be built with GOOS=js GOARCH=wasm")
}
