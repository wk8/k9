package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io/ioutil"
)

// TODO wkpo next
func main() {
	fname := "./k9-917742246"
	content, err := ioutil.ReadFile(fname)
	if err != nil {
		panic(err)
	}
	fmt.Println("File content:\n", content)

	reader, err := zlib.NewReader(bytes.NewReader(content))
	if err != nil {
		panic(err)
	}

	enflated, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("Enflated:\n", string(enflated))
}
