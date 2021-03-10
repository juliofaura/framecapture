package main

import (
	"fmt"
	"github.com/loranbriggs/go-camera"
)


func main() {

	c := camera.New(".")
	s, err := c.Capture()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(s)
}
