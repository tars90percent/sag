package main

import (
	"os"
	"testing"
)

func TestMainHelp(_ *testing.T) {
	orig := os.Args
	defer func() { os.Args = orig }()
	os.Args = []string{"sag", "--help"}
	main()
}
