package main

import sh "github.com/james-orcales/james-orcales/golang_snacks/sh/default"

func main() {
	shell := sh.Init_Default_Shell()
	sh.Shell_Spawn(
		shell,
		"go",
		"test",
		"./cli",
		"./itlog",
		"./myers",
		"./sh",
		"-count=1",
	)
}
