// Package main binds host capabilities to the markdown_to_pdf command policy.
package main

import (
	"io"
	"os"
	"os/exec"

	markdown_to_pdf "local/james-orcales/markdown_to_pdf/internal"
)

func main() {
	os.Exit(markdown_to_pdf.Main(&markdown_to_pdf.Main_Input{
		Arguments:    os.Args,
		Output:       os.Stdout,
		Error_Output: os.Stderr,
		Open_File: func(path string) (file io.ReadCloser, err error) {
			return os.Open(path)
		},
		Write_File: func(path string, document []byte) (err error) {
			return os.WriteFile(path, document, 0o666)
		},
		Path_Exists: func(path string) (exists bool) {
			_, stat_err := os.Stat(path)
			return stat_err == nil
		},
		Temporary_Directory: os.TempDir(),
		Open_Path: func(path string) (err error) {
			return exec.Command("open", path).Run()
		},
	}))
}
