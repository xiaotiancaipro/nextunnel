package main

import "github.com/xiaotiancaipro/nextunnel-client/cmd"

var version = "v1.0.0"

func main() {
	_ = cmd.New(version).Execute()
}
