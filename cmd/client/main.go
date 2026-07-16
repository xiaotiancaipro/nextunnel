package main

import "github.com/xiaotiancaipro/nextunnel/cmd/client/root"

var version = "v0.0.0"

func main() {
	_ = new(root.Root).New(version)
}
