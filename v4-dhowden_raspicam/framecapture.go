package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dhowden/raspicam"
)

func main() {
	f, err := os.Create(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "create file: %v", err)
		return
	}
	defer f.Close()

	s := raspicam.NewStill()
	errCh := make(chan error)
	go func() {
		for x := range errCh {
			fmt.Fprintf(os.Stderr, "%v\n", x)
		}
	}()
	log.Println("Capturing image...")
	raspicam.Capture(s, f, errCh)
}