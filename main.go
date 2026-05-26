package main

import "github.com/xiaotiancaipro/nextunnel-client/cmd"

var version = "v0.1.3"

func main() {
	_ = cmd.New(version).Execute()
}
