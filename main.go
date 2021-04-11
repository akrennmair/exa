package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
)

const tabWidth = 8

func main() {
	log.SetOutput(io.Discard)

	logFile := flag.String("log", "", "if not empty, debug log output is written to this file")

	flag.Parse()

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("Failed to open log file %s for writing: %v\n", *logFile, err)
			os.Exit(1)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	scr, err := tcell.NewScreen()
	if err != nil {
		fmt.Printf("Couldn't create new screen: %v\n", err)
		os.Exit(1)
	}

	ed := newEditor(scr)

	for _, arg := range flag.Args() {
		if err := ed.loadBufferFromFile(arg); err != nil {
			fmt.Printf("Failed to load file %s: %v\n", arg, err)
			os.Exit(1)
		}
	}

	if len(flag.Args()) == 0 {
		ed.addNewBuffer()
	}

	if err := scr.Init(); err != nil {
		fmt.Printf("Couldn't init screen: %v\n", err)
		os.Exit(1)
	}
	defer scr.Fini()

	ed.inputLoop()
}
