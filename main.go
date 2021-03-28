package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

const tabWidth = 8

func main() {
	flag.Parse()

	ed := &editor{}

	for _, arg := range flag.Args() {
		if err := ed.loadBufferFromFile(arg); err != nil {
			fmt.Printf("Failed to load file %s: %v\n", arg, err)
			os.Exit(1)
		}
	}

	if len(flag.Args()) == 0 {
		ed.addNewBuffer()
	}

	scr, err := tcell.NewScreen()
	if err != nil {
		fmt.Printf("Couldn't create new screen: %v\n", err)
		os.Exit(1)
	}

	if err := scr.Init(); err != nil {
		fmt.Printf("Couldn't init screen: %v\n", err)
		os.Exit(1)
	}
	defer scr.Fini()

	ed.scr = scr

	ops := []keyMapping{
		{tcell.KeyCtrlA, ed.gotoBOL, "go to beginning of line"},
		{tcell.KeyCtrlE, ed.gotoEOL, "go to end of line"},
		{tcell.KeyCtrlQ, ed.quit, "quit"},
	}

	for {
		ed.redrawScreen()

		if ed.quitInputLoop {
			break
		}

		evt := scr.PollEvent()
		switch e := evt.(type) {
		case *tcell.EventResize:
			continue
		case *tcell.EventKey:
			for _, op := range ops {
				if e.Key() == op.Key {
					op.Func()
					continue
				}
			}
			if e.Modifiers() == 0 {
				ed.handleInput(e.Rune())
			}
		}
	}

}

type keyMapping struct {
	Key  tcell.Key
	Func func()
	Desc string
}

type editor struct {
	bufs          []*buffer
	scr           tcell.Screen
	bufIdx        int
	quitInputLoop bool
}

func (e *editor) loadBufferFromFile(fn string) error {
	return errors.New("not implemented")
}

func (e *editor) addNewBuffer() {
	e.bufs = append(e.bufs, &buffer{
		lines: [][]rune{{}},
	})
}

func (e *editor) handleInput(r rune) {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset
	curLine := curBuf.lines[lineIdx]

	if curBuf.x >= len(curLine) {
		curLine = append(curLine, r)
	} else {
		curLine = append(curLine[:curBuf.x], append([]rune{r}, curLine[curBuf.x:]...)...)
	}
	curBuf.lines[lineIdx] = curLine
	curBuf.x++
}

func (e *editor) gotoBOL() {
	e.bufs[e.bufIdx].x = 0
}

func (e *editor) gotoEOL() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset
	curLine := curBuf.lines[lineIdx]

	curBuf.x = len(curLine)
}

func (e *editor) quit() {
	// TODO: check whether files need saving or something.
	e.quitInputLoop = true
}

func (e *editor) redrawScreen() {
	width, height := e.scr.Size()

	curBuf := e.bufs[e.bufIdx]

	for i := curBuf.offset; i < curBuf.offset+height-1; i++ {
		fmt.Fprintf(os.Stderr, "redraw: i = %d\n", i)
		e.drawLine(curBuf, i-curBuf.offset, i, width, i == curBuf.offset+curBuf.y)
	}

	e.scr.ShowCursor(curBuf.x, curBuf.y)

	e.drawStatus(height-1, width)

	e.scr.Show()
}

func (e *editor) drawLine(buf *buffer, y int, lineIdx int, width int, curLine bool) {
	if len(buf.lines) <= lineIdx {
		fmt.Fprintf(os.Stderr, "drawLine: drawing past file y = %d, lineIdx = %d\n", y, lineIdx)
		e.scr.SetCell(0, y, tcell.StyleDefault.Bold(true), '~')
		for i := 1; i < width; i++ {
			e.scr.SetCell(i, y, tcell.StyleDefault, ' ')
		}
		return
	}

	line := buf.lines[lineIdx]

	style := tcell.StyleDefault.Underline(curLine)

	e.clearLine(y, width, style)

	x := 0
	for _, r := range line {
		if x >= width {
			r = '$'
		} else if r == '\t' {
			r = ' '
			x += tabWidth - 1
		}
		e.scr.SetCell(x, y, style, r)
		x += runewidth.RuneWidth(r)
	}
}

func (e *editor) clearLine(y int, width int, style tcell.Style) {
	for x := 0; x < width; x++ {
		e.scr.SetCell(x, y, style, ' ')
	}
}

func (e *editor) drawStatus(y int, width int) {
	curBuf := e.bufs[e.bufIdx]

	status := ""
	if curBuf.modified {
		status += "* "
	} else {
		status += "- "
	}

	if curBuf.fname == "" {
		status += "<no file> "
	} else {
		status += curBuf.fname + " "
	}

	status += fmt.Sprintf("[%d|%d]", curBuf.y, curBuf.x)

	statusStyle := tcell.StyleDefault.Reverse(true)

	e.clearLine(y, width, statusStyle)

	x := 0
	for _, r := range status {
		e.scr.SetCell(x, y, statusStyle, r)
		x += runewidth.RuneWidth(r)
	}
}

type buffer struct {
	fname    string
	lines    [][]rune
	x        int
	y        int
	offset   int
	modified bool
}
