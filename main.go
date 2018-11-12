package main

import (
	"github.com/jrmarkle/protoc-gen-go-template/generator"
	"io/ioutil"
	"os"
)

type osFileSystem struct{}

func (osFileSystem) ReadFile(filename string) ([]byte, error) { return ioutil.ReadFile(filename) }

func main() {
	err := generator.NewGenerator(os.Stdin, os.Stdout, osFileSystem{}).Run()
	if err != nil {
		os.Stderr.WriteString("Failed: " + err.Error() + "\n")
		os.Exit(1)
	}
}
