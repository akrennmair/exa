package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func newEditor(scr tcell.Screen) *editor {
	ed := &editor{
		scr: scr,
	}

	ed.ops = []keyMapping{
		{tcell.KeyCtrlSpace, ed.selectText, "start/stop selecting text"},
		{tcell.KeyCtrlA, ed.gotoBOL, "go to beginning of line"},
		{tcell.KeyCtrlB, ed.newBuffer, "create new buffer"},
		{tcell.KeyCtrlC, ed.copyText, "copy selected text to clipboard"},
		{tcell.KeyCtrlD, ed.closeBuffer, "close current buffer"},
		{tcell.KeyCtrlE, ed.gotoEOL, "go to end of line"},
		{tcell.KeyCtrlF, ed.find, "find text"},
		{tcell.KeyCtrlH, ed.showHelp, "show help"},
		{tcell.KeyCtrlK, ed.deleteToEOL, "delete text to end of line"},
		{tcell.KeyCtrlL, ed.redraw, "redraw screen"},
		{tcell.KeyCtrlN, ed.nextBuffer, "go to next file"},
		{tcell.KeyCtrlO, ed.openFile, "open file in new buffer"},
		{tcell.KeyCtrlP, ed.prevBuffer, "go to previous file"},
		{tcell.KeyCtrlQ, ed.quit, "quit"},
		{tcell.KeyCtrlR, ed.redo, "redo previously undone change"},
		{tcell.KeyCtrlS, ed.save, "save file"},
		{tcell.KeyCtrlU, ed.deleteFromBOL, "delete text from beginning of line"},
		{tcell.KeyCtrlV, ed.pasteText, "paste text from clipboard"},
		{tcell.KeyCtrlW, ed.saveAs, "save file as"},
		{tcell.KeyCtrlX, ed.cutText, "cut selected text to clipboard"},
		{tcell.KeyCtrlZ, ed.undo, "undo last change"},
		{tcell.KeyCR, ed.newLine, "insert new line"},
		{tcell.KeyUp, ed.keyUp, "go to previous line"},
		{tcell.KeyDown, ed.keyDown, "go to next line"},
		{tcell.KeyLeft, ed.keyLeft, "go to previous character"},
		{tcell.KeyRight, ed.keyRight, "go to next character"},
		{tcell.KeyPgDn, ed.pageDown, "go to next page"},
		{tcell.KeyPgUp, ed.pageUp, "go to previous page"},
		{tcell.KeyDEL, ed.keyBackspace, "delete character left from cursor"},
		{tcell.KeyDelete, ed.keyDel, "delete character right from cursor"},
	}

	return ed
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
	clipboard     [][]rune
	ops           []keyMapping
}

func (e *editor) inputLoop() {
	for {
		e.redrawScreen()

		if e.quitInputLoop {
			break
		}

		e.handleEvent()
	}
}

func (e *editor) handleEvent() {
	evt := e.scr.PollEvent()
	log.Printf("handleEvent: received event of type %T", evt)
	switch ev := evt.(type) {
	case *tcell.EventResize:
		width, height := ev.Size()
		log.Printf("handleEvent: resize event: %dx%d", width, height)
		return
	case *tcell.EventKey:
		log.Printf("handleEvent: key: %v rune = %d mod = %b", ev.Key(), ev.Rune(), ev.Modifiers())
		for _, op := range e.ops {
			if ev.Key() == op.Key {
				op.Func()
				return
			}
		}

		if ev.Key() == tcell.KeyRune || ev.Key() == tcell.KeyTAB {
			e.handleInput(ev.Rune())
		}
	}
}

func (e *editor) loadBufferFromFile(fn string) error {
	if _, err := os.Stat(fn); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			e.addNewBuffer()
			e.bufs[len(e.bufs)-1].fname = fn
			return nil
		}
	}

	f, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	buf := &buffer{
		fname:      fn,
		historyIdx: -1,
	}

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
		lines:      [][]rune{{}},
		historyIdx: -1,
	})
}

func (e *editor) handleInput(r rune) {
	log.Printf("handleInput: rune = %c", r)

	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset
	curLine := curBuf.lines[lineIdx]

	if curBuf.x >= len(curLine) {
		curLine = append(curLine, r)
	} else {
		curLine = append(curLine[:curBuf.x], append([]rune{r}, curLine[curBuf.x:]...)...)
	}
	curBuf.lines[lineIdx] = curLine

	curBuf.historyAddRune(r)

	curBuf.x++

	curBuf.modified = true
}

func (e *editor) saveFile(curBuf *buffer) {
	tmpName := filepath.Join(filepath.Dir(curBuf.fname), fmt.Sprintf(".tmp%x", time.Now().UnixNano()))

	log.Printf("saveFile: saving buffer of %d lines to %s (temporary file: %s)", len(curBuf.lines), curBuf.fname, tmpName)

	f, err := os.OpenFile(tmpName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("saveFile: opening temporary file %s failed: %v", tmpName, err)
		e.showError("Failed to open temporary file: %v", err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range curBuf.lines {
		if _, err := w.WriteString(string(line) + "\n"); err != nil {
			log.Printf("saveFile: writing to temporary file failed: %v", err)
			e.showError("Failed to write to temporary file: %v", err)
			return
		}
	}
	if err := w.Flush(); err != nil {
		log.Printf("saveFile: Flush failed: %v", err)
		e.showError("Failed to write file content: %v", err)
		return
	}

	if err := os.Rename(tmpName, curBuf.fname); err != nil {
		log.Printf("saveFile: Rename failed: %v", err)
		e.showError("Failed to replace %s with temporary file %s: %v", curBuf.fname, tmpName, err)
		return
	}

	curBuf.modified = false
}

func (e *editor) updateSelectedTextPos(buf *buffer) {
	if buf.selecting {
		buf.endX, buf.endY = buf.x, buf.offset+buf.y
		log.Printf("updateSelectedTextPos: new selection end point at %d/%d", buf.endY, buf.endX)
	}
}

func (e *editor) showError(s string, args ...interface{}) {
	str := fmt.Sprintf(s, args...)

	width, height := e.scr.Size()

	e.clearLine(height-1, width, tcell.StyleDefault)

	x := 0
	for _, r := range str {
		e.scr.SetContent(x, height-1, r, nil, tcell.StyleDefault)
		x += runewidth.RuneWidth(r)
	}
}

func (e *editor) readString(prompt string, inputRunes []rune) (input string, ok bool) {
	log.Printf("readString: prompt %q, initial text %q", prompt, string(inputRunes))

	cursorPos := len(inputRunes)

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
				s := string(inputRunes)
				log.Printf("readString: returning %q", s)
				return s, true
			case tcell.KeyESC, tcell.KeyCtrlG:
				log.Printf("readString: cancelled input")
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
	log.Printf("query: prompt %q valid answers: %q", prompt, validAnswers)
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
						log.Printf("query: returning %c", r)
						return r
					}
				}
			}
		}
	}
}

func (e *editor) redrawScreen() {
	width, height := e.scr.Size()

	log.Printf("redrawScreen: %dx%d", width, height)

	curBuf := e.bufs[e.bufIdx]

	for i := curBuf.offset; i < curBuf.offset+height-2; i++ {
		e.drawLine(curBuf, i-curBuf.offset, i, width, i == curBuf.curLineIdx())
	}

	x := runeWidth(curBuf.curLine()[:curBuf.x])

	e.scr.ShowCursor(x, curBuf.y)

	e.drawStatus(height-2, width)

	e.scr.Show()

	e.clearLine(height-1, width, tcell.StyleDefault)

	for i := 0; i < len(curBuf.lines); i++ {
		log.Printf("redrawScreen: line %d = %q\n", i, string(curBuf.lines[i]))
	}
}

func (e *editor) drawLine(buf *buffer, y int, lineIdx int, width int, curLine bool) {
	if len(buf.lines) <= lineIdx {
		e.scr.SetContent(0, y, '~', nil, tcell.StyleDefault.Bold(true))
		for i := 1; i < width; i++ {
			e.scr.SetContent(i, y, ' ', nil, tcell.StyleDefault)
		}
		return
	}

	line := buf.lines[lineIdx]

	style := tcell.StyleDefault.Underline(curLine)

	e.clearLine(y, width, style)

	x := 0
	for idx, r := range line {
		if x >= width {
			r = '$'
		} else if r == '\t' {
			r = ' '
			x += tabWidth - 1
		}
		charStyle := style
		if buf.isWithinSelectedText(lineIdx, idx) {
			charStyle = charStyle.Background(tcell.ColorYellow).Foreground(tcell.ColorBlack)
		}
		e.scr.SetContent(x, y, r, nil, charStyle)
		x += runewidth.RuneWidth(r)
	}
}

func (e *editor) clearLine(y int, width int, style tcell.Style) {
	for x := 0; x < width; x++ {
		e.scr.SetContent(x, y, ' ', nil, style)
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

	status += fmt.Sprintf("(%d of %d) [%d|%d-%d] - Press Ctrl-H for Help", e.bufIdx+1, len(e.bufs), curBuf.curLineIdx(), curBuf.x, runeWidth(curBuf.curLine()[:curBuf.x]))

	statusStyle := tcell.StyleDefault.Reverse(true)

	e.clearLine(y, width, statusStyle)

	x := 0
	for _, r := range status {
		e.scr.SetContent(x, y, r, nil, statusStyle)
		x += runewidth.RuneWidth(r)
	}
}
