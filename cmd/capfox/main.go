package main

import (
	"github.com/haskel/capfox/internal/cli"
)

var (
	version = "0.2.0"
)

func main() {
	cli.SetVersion(version)
	cli.Execute()
}
