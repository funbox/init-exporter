package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                           Copyright (c) 2006-2023 FUNBOX                           //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	_ "embed"

	CLI "github.com/funbox/init-exporter/cli"
)

// ////////////////////////////////////////////////////////////////////////////////// //

//go:embed go.mod
var gomod []byte

// gitrev is short hash of the latest git commit
var gitrev string

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	CLI.Run(gitrev, gomod)
}
