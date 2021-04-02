package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
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
		{tcell.KeyCtrlL, ed.redraw, "redraw screen"},
		{tcell.KeyCtrlS, ed.save, "save file"},
		{tcell.KeyCtrlW, ed.saveAs, "save file as"},
		{tcell.KeyCR, ed.newLine, "insert new line"},
		{tcell.KeyUp, ed.keyUp, "go to previous line"},
		{tcell.KeyDown, ed.keyDown, "go to next line"},
		{tcell.KeyLeft, ed.keyLeft, "go to previous character"},
		{tcell.KeyRight, ed.keyRight, "go to next character"},
		{tcell.KeyDEL, ed.keyBackspace, "delete character left from cursor"},
		{tcell.KeyDelete, ed.keyDel, "delete character right from cursor"},
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
			log.Printf("event key: %v rune = %d mod = %b", e.Key(), e.Rune(), e.Modifiers())
			matched := false
			for _, op := range ops {
				if e.Key() == op.Key {
					op.Func()
					matched = true
					break
				}
			}
			if matched {
				continue
			}

			if e.Key() == tcell.KeyRune {
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
	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	buf := &buffer{fname: fn}

	for scanner.Scan() {
		buf.lines = append(buf.lines, []rune(scanner.Text()))
	}

	if len(buf.lines) == 0 {
		buf.lines = [][]rune{{}}
	}

	e.bufs = append(e.bufs, buf)

	return nil
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

	curBuf.modified = true
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

func (e *editor) newLine() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset

	curLine, nextLine := curBuf.lines[lineIdx][:curBuf.x], curBuf.lines[lineIdx][curBuf.x:]
	log.Printf("newLine: %q -> %q, %q", string(curBuf.lines[lineIdx]), string(curLine), string(nextLine))
	curBuf.lines[lineIdx] = curLine
	curBuf.lines = append(curBuf.lines[:lineIdx+1], append([][]rune{nextLine}, curBuf.lines[lineIdx+1:]...)...)

	e.keyDown()
	curBuf.x = 0
}

func (e *editor) keyBackspace() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset

	if curBuf.x == 0 {
		if lineIdx == 0 {
			// nothing that can be done as we're on the leftmost character on the first line.
			return
		}

		curBuf.x = len(curBuf.lines[lineIdx-1])
		curBuf.lines[lineIdx-1] = append(curBuf.lines[lineIdx-1], curBuf.lines[lineIdx]...)
		curBuf.lines = append(curBuf.lines[:lineIdx], curBuf.lines[lineIdx+1:]...)

		if curBuf.offset > 0 {
			curBuf.offset--
		} else {
			curBuf.y--
		}
		return
	}

	curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx][:curBuf.x-1], curBuf.lines[lineIdx][curBuf.x:]...)
	curBuf.x--
}

func (e *editor) keyDel() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset

	if curBuf.x == len(curBuf.lines[lineIdx]) {
		if lineIdx == len(curBuf.lines)-1 {
			// nothing that can be done, as we're on the last character on the last line
			return
		}

		curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx], curBuf.lines[lineIdx+1]...)
		curBuf.lines = append(curBuf.lines[:lineIdx+1], curBuf.lines[lineIdx+2:]...)
		return
	}

	curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx][:curBuf.x], curBuf.lines[lineIdx][curBuf.x+1:]...)
}

func (e *editor) keyUp() {
	curBuf := e.bufs[e.bufIdx]

	if (curBuf.y + curBuf.offset) == 0 {
		return
	}

	if curBuf.offset > 0 {
		curBuf.offset--
	} else {
		curBuf.y--
	}

	if l := len(curBuf.lines[curBuf.y+curBuf.offset]); curBuf.x > l {
		curBuf.x = l
	}
}

func (e *editor) keyDown() {
	curBuf := e.bufs[e.bufIdx]

	if (curBuf.y + curBuf.offset) >= len(curBuf.lines)-1 {
		return
	}

	_, height := e.scr.Size()

	if curBuf.y >= height-2 {
		curBuf.offset++
	} else {
		curBuf.y++
	}

	if l := len(curBuf.lines[curBuf.y+curBuf.offset]); curBuf.x > l {
		curBuf.x = l
	}
}

func (e *editor) keyLeft() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.x > 0 {
		curBuf.x--
	}
}

func (e *editor) keyRight() {
	curBuf := e.bufs[e.bufIdx]

	if len(curBuf.lines[curBuf.y+curBuf.offset]) > curBuf.x {
		curBuf.x++
	}
}

func (e *editor) quit() {
	// TODO: check whether files need saving or something.
	e.quitInputLoop = true
}

func (e *editor) save() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.fname == "" {
		fname, ok := e.readString("Filename")
		if !ok {
			return
		}

		curBuf.fname = fname
	}

	e.saveFile(curBuf)
}

func (e *editor) saveFile(curBuf *buffer) {
	f, err := os.OpenFile(curBuf.fname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		e.showError("Failed to write file: %v", err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range curBuf.lines {
		w.WriteString(string(line) + "\n") // TODO: handle error better, maybe write to temporary file, then move?
	}
	if err := w.Flush(); err != nil {
		e.showError("Failed to write file content: %v", err)
		return
	}

	curBuf.modified = false
}

func (e *editor) saveAs() {
	curBuf := e.bufs[e.bufIdx]

	fname, ok := e.readString("New filename")
	if !ok {
		return
	}

	_, err := os.Stat(fname)
	if err == nil {
		switch e.query("Are you sure you want to overwrite file?", "yn") {
		case 'y':
			// continue as normal
		case 'n':
			return
		}
	}

	curBuf.fname = fname

	e.saveFile(curBuf)
}

func (e *editor) showError(s string, args ...interface{}) {
	str := fmt.Sprintf(s, args...)

	width, height := e.scr.Size()

	e.clearLine(height-1, width, tcell.StyleDefault)

	e.scr.SetCell(0, height-1, tcell.StyleDefault, []rune(str)...)
}

func (e *editor) readString(prompt string) (input string, ok bool) {
	var (
		inputRunes []rune
		cursorPos  = 0
	)

	defer func() {
		width, height := e.scr.Size()
		e.clearLine(height-1, width, tcell.StyleDefault)
	}()

	prompt += ": "

	for {
		width, height := e.scr.Size()

		e.clearLine(height-1, width, tcell.StyleDefault)

		promptStyle := tcell.StyleDefault.Bold(true)

		x := 0
		for _, r := range prompt {
			e.scr.SetContent(x, height-1, r, nil, promptStyle)
			x += runewidth.RuneWidth(r)
		}

		i := 0
		inputx := x

		cursorPosFound := false

		for _, r := range inputRunes {
			log.Printf("readString: i = %d cursorPos = %d r = %c", i, cursorPos, r)
			if i == cursorPos {
				log.Printf("readString: showing cursor at x = %d y = %d", inputx, height-1)
				e.scr.ShowCursor(inputx, height-1)
				cursorPosFound = true
			}

			e.scr.SetContent(inputx, height-1, r, nil, tcell.StyleDefault)

			i++
			inputx += runewidth.RuneWidth(r)
		}

		if !cursorPosFound {
			e.scr.ShowCursor(inputx, height-1)
		}

		e.scr.Show()

		evt := e.scr.PollEvent()
		switch ev := evt.(type) {
		case *tcell.EventResize:
			e.redrawScreen()
			continue
		case *tcell.EventKey:
			log.Printf("readString: %v rune = %d mod = %b", ev.Key(), ev.Rune(), ev.Modifiers())

			switch ev.Key() {
			case tcell.KeyCR:
				return string(inputRunes), true
			case tcell.KeyESC, tcell.KeyCtrlG:
				return "", false
			case tcell.KeyLeft:
				if cursorPos > 0 {
					cursorPos--
				}
			case tcell.KeyRight:
				if cursorPos < len(inputRunes) {
					cursorPos++
				}
			case tcell.KeyDEL: // backspace
				if cursorPos > 0 {
					inputRunes = append(inputRunes[:cursorPos-1], inputRunes[cursorPos:]...)
				}
				cursorPos--
			case tcell.KeyDelete: // DEL
				if cursorPos < len(inputRunes) {
					inputRunes = append(inputRunes[:cursorPos], inputRunes[cursorPos+1:]...)
				}
			case tcell.KeyCtrlU:
				inputRunes = inputRunes[cursorPos:]
				cursorPos = 0
			case tcell.KeyCtrlK:
				inputRunes = inputRunes[:cursorPos]
			case tcell.KeyCtrlA:
				cursorPos = 0
			case tcell.KeyCtrlE:
				cursorPos = len(inputRunes)
			case tcell.KeyRune:
				log.Printf("readString: rune input: %[1]c (%[1]d)", ev.Rune())
				inputRunes = append(inputRunes[:cursorPos], append([]rune{ev.Rune()}, inputRunes[cursorPos:]...)...)
				cursorPos++
			}
		}
	}
}

func (e *editor) query(prompt string, validAnswers string) rune {
	defer func() {
		width, height := e.scr.Size()
		e.clearLine(height-1, width, tcell.StyleDefault)
	}()

	prompt += " [" + validAnswers + "]"

	for {
		width, height := e.scr.Size()

		e.clearLine(height-1, width, tcell.StyleDefault)

		promptStyle := tcell.StyleDefault.Bold(true)

		x := 0
		for _, r := range prompt {
			e.scr.SetContent(x, height-1, r, nil, promptStyle)
			x += runewidth.RuneWidth(r)
		}

		e.scr.ShowCursor(x, height-1)

		e.scr.Show()

		evt := e.scr.PollEvent()
		switch ev := evt.(type) {
		case *tcell.EventResize:
			e.redrawScreen()
			continue
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyRune {
				for _, r := range validAnswers {
					if r == ev.Rune() {
						return r
					}
				}
			}
		}
	}
}

func (e *editor) redrawScreen() {
	width, height := e.scr.Size()

	curBuf := e.bufs[e.bufIdx]

	for i := curBuf.offset; i < curBuf.offset+height-2; i++ {
		e.drawLine(curBuf, i-curBuf.offset, i, width, i == curBuf.offset+curBuf.y)
	}

	x := strwidth(curBuf.lines[curBuf.y+curBuf.offset][:curBuf.x])

	e.scr.ShowCursor(x, curBuf.y)

	e.drawStatus(height-2, width)

	e.scr.Show()

	e.clearLine(height-1, width, tcell.StyleDefault)

	for i := 0; i < len(curBuf.lines); i++ {
		log.Printf("redrawScreen: line %d = %q\n", i, string(curBuf.lines[i]))
	}
}

func strwidth(s []rune) (w int) {
	for _, r := range s {
		if r == '\t' {
			w += tabWidth
		} else {
			w += runewidth.RuneWidth(r)
		}
	}
	return w
}

func (e *editor) redraw() {
	e.scr.Sync()
}

func (e *editor) drawLine(buf *buffer, y int, lineIdx int, width int, curLine bool) {
	if len(buf.lines) <= lineIdx {
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

	e.scr.SetCell(0, y, statusStyle, []rune(status)...)
}

type buffer struct {
	fname    string
	lines    [][]rune
	x        int
	y        int
	offset   int
	modified bool
}
