package main

import "os"

var version = "v0.0.0"

func main() {
	if err := New(version).Execute(); err != nil {
		os.Exit(1)
	}
}
