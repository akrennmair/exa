package main

import (
	"log"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
)

func (e *editor) gotoBOL() {
	curBuf := e.bufs[e.bufIdx]
	curBuf.x = 0
	e.updateSelectedTextPos(curBuf)
	curBuf.historyFinishOp()
}

func (e *editor) gotoEOL() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset
	curLine := curBuf.lines[lineIdx]

	curBuf.x = len(curLine)

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) newLine() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset

	curLine, nextLine := curBuf.lines[lineIdx][:curBuf.x], curBuf.lines[lineIdx][curBuf.x:]
	log.Printf("newLine: %q -> %q, %q", string(curBuf.lines[lineIdx]), string(curLine), string(nextLine))
	curBuf.lines[lineIdx] = curLine
	curBuf.lines = append(curBuf.lines[:lineIdx+1], append([][]rune{nextLine}, curBuf.lines[lineIdx+1:]...)...)

	_, height := e.scr.Size()

	curBuf.incrY(height)

	curBuf.correctX()

	curBuf.historyAddLine()
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

		curBuf.decrY()

		curBuf.historyRemoveLine()
		return
	}

	r := curBuf.lines[lineIdx][curBuf.x-1]

	curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx][:curBuf.x-1], curBuf.lines[lineIdx][curBuf.x:]...)
	curBuf.x--
	curBuf.historyRemoveChar(r)
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
		curBuf.historyRemoveLine()
		return
	}

	r := curBuf.lines[lineIdx][curBuf.x]

	curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx][:curBuf.x], curBuf.lines[lineIdx][curBuf.x+1:]...)
	curBuf.historyRemoveChar(r)
}

func (e *editor) keyUp() {
	curBuf := e.bufs[e.bufIdx]

	if (curBuf.y + curBuf.offset) == 0 {
		return
	}

	curBuf.decrY()

	curBuf.correctX()

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) keyDown() {
	curBuf := e.bufs[e.bufIdx]

	if (curBuf.y + curBuf.offset) >= len(curBuf.lines)-1 {
		return
	}

	_, height := e.scr.Size()

	curBuf.incrY(height)

	curBuf.correctX()

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) keyLeft() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.x > 0 {
		curBuf.x--
	}

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) keyRight() {
	curBuf := e.bufs[e.bufIdx]

	if len(curBuf.curLine()) > curBuf.x {
		curBuf.x++
	}

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) quit() {
	for i := 0; i < len(e.bufs); i++ {
		e.bufIdx = i
		if e.bufs[e.bufIdx].modified {
			e.redrawScreen()
			switch e.query("Save file?", "ync") {
			case 'y':
				e.save()
			case 'n':
				// ignore file.
			case 'c':
				return // cancel quitting.
			}
		}
	}

	// all files checked whether user wants to save them -> quit
	e.quitInputLoop = true
}

func (e *editor) save() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.fname == "" {
		fname, ok := e.readString("Filename", nil)
		if !ok {
			return
		}

		curBuf.fname = fname
	}

	e.saveFile(curBuf)
}

func (e *editor) saveAs() {
	curBuf := e.bufs[e.bufIdx]

	fname, ok := e.readString("New filename", nil)
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

func (e *editor) nextBuffer() {
	e.bufIdx++
	if e.bufIdx >= len(e.bufs) {
		e.bufIdx = 0
	}
}

func (e *editor) prevBuffer() {
	e.bufIdx--
	if e.bufIdx < 0 {
		e.bufIdx = len(e.bufs) - 1
	}
}

func (e *editor) deleteToEOL() {
	curBuf := e.bufs[e.bufIdx]

	curBuf.lines[curBuf.curLineIdx()] = curBuf.curLine()[:curBuf.x]
}

func (e *editor) deleteFromBOL() {
	curBuf := e.bufs[e.bufIdx]

	curBuf.lines[curBuf.curLineIdx()] = curBuf.curLine()[curBuf.x:]
	curBuf.x = 0
}

func (e *editor) selectText() {
	curBuf := e.bufs[e.bufIdx]

	if !curBuf.selecting {
		curBuf.startX, curBuf.startY = curBuf.x, curBuf.curLineIdx()
		curBuf.endX, curBuf.endY = curBuf.startX, curBuf.startY
		log.Printf("Started selecting text from %d/%d", curBuf.startY, curBuf.startX)
	} else {
		log.Printf("Stopped selecting text at %d/%d", curBuf.endY, curBuf.endX)
	}

	curBuf.selecting = !curBuf.selecting
}

func (e *editor) copyText() {
	curBuf := e.bufs[e.bufIdx]
	curBuf.selecting = false

	lowerY, lowerX, higherY, higherX := curBuf.getSelection()

	copiedData := [][]rune{}

	for y := lowerY; y <= higherY; y++ {
		var firstX, lastX int
		if y == lowerY {
			firstX = lowerX
			lastX = len(curBuf.lines[y])
			if lowerY == higherY {
				lastX = higherX
			}
		} else if y == higherY {
			firstX = 0
			lastX = higherX
		} else {
			firstX = 0
			lastX = len(curBuf.lines[y])
		}
		copiedData = append(copiedData, curBuf.lines[y][firstX:lastX])
	}

	e.clipboard = copiedData
}

func (e *editor) cutText() {
	e.copyText()

	curBuf := e.bufs[e.bufIdx]
	lowerY, lowerX, higherY, higherX := curBuf.getSelection()

	newX := len(curBuf.lines[lowerY][:lowerX])

	replacementLine := append(curBuf.lines[lowerY][:lowerX], curBuf.lines[higherY][higherX:]...)

	curBuf.lines = append(curBuf.lines[:lowerY], append([][]rune{replacementLine}, curBuf.lines[higherY+1:]...)...)

	curBuf.startY, curBuf.startX, curBuf.endY, curBuf.endX = 0, 0, 0, 0

	for i := 0; i < higherY-lowerY; i++ {
		curBuf.decrY()
	}

	curBuf.x = newX

	curBuf.modified = true
}

func (e *editor) pasteText() {
	insertion := [][]rune{}
	insertion = append(insertion, e.clipboard...)

	for idx, line := range insertion {
		log.Printf("pasteText: clipboard %d = %s", idx, string(line))
	}

	curBuf := e.bufs[e.bufIdx]
	curY := curBuf.y + curBuf.offset

	lastLineX := len(insertion[len(insertion)-1])

	beforeInsertion, afterInsertion := curBuf.lines[curY][:curBuf.x], curBuf.lines[curY][curBuf.x:]

	insertion[0] = append(append([]rune{}, beforeInsertion...), insertion[0]...)
	insertion[len(insertion)-1] = append(insertion[len(insertion)-1], afterInsertion...)

	for idx, line := range insertion {
		log.Printf("pasteText: insertion %d = %s", idx, string(line))
	}

	curBuf.lines = append(curBuf.lines[:curY], append(insertion, curBuf.lines[curY+1:]...)...)

	_, height := e.scr.Size()

	for i := 0; i < len(insertion)-1; i++ {
		curBuf.incrY(height)
	}
	curBuf.x = lastLineX

	curBuf.modified = true
}

func (e *editor) pageDown() {
	_, height := e.scr.Size()

	curBuf := e.bufs[e.bufIdx]

	for i := 0; i < height-2; i++ {
		if curBuf.curLineIdx() == len(curBuf.lines)-1 {
			break
		}
		curBuf.incrY(height)
	}

	curBuf.correctX()
}

func (e *editor) pageUp() {
	_, height := e.scr.Size()

	curBuf := e.bufs[e.bufIdx]

	for i := 0; i < height-2; i++ {
		if curBuf.curLineIdx() == 0 {
			break
		}
		curBuf.decrY()
	}

	curBuf.correctX()
}

func (e *editor) undo() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.historyIdx < 0 {
		e.showError("Already at oldest change")
		return
	}

	op := curBuf.editHistory[curBuf.historyIdx]
	op.finished = true

	log.Printf("undo: op = %d y = %d x = %d", op.op, op.y, op.x)
	for idx, line := range op.text {
		log.Printf("undo: line %d: %s", idx, string(line))
	}

	op.undo(curBuf)
	curBuf.historyIdx--

	curBuf.correctY()
	curBuf.correctX()
}

func (e *editor) redo() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.historyIdx == len(curBuf.editHistory)-1 {
		e.showError("Already at newest change")
		return
	}

	curBuf.historyIdx++

	op := curBuf.editHistory[curBuf.historyIdx]
	log.Printf("redo: op = %d y = %d x = %d finished = %t", op.op, op.y, op.x, op.finished)
	for idx, line := range op.text {
		log.Printf("redo: line %d: %s", idx, string(line))
	}

	op.redo(curBuf)

	curBuf.correctX()
}

func (e *editor) showHelp() {
	e.scr.Clear()

	width, _ := e.scr.Size()

	e.clearLine(0, width, tcell.StyleDefault.Reverse(true))

	titleText := "Help - Press Any Key To Continue"

	x := 0
	for _, r := range titleText {
		e.scr.SetContent(x, 0, r, nil, tcell.StyleDefault.Reverse(true))
		x += runewidth.RuneWidth(r)
	}

	var helpElems []string

	keyWidth := 16
	for _, op := range e.ops {
		helpElem := tcell.KeyNames[op.Key] + " "
		helpElem += strings.Repeat(".", keyWidth-len(helpElem)) + " " + op.Desc
		helpElems = append(helpElems, helpElem)
	}

	widths := []int{0, width / 2}

	for i := 0; i < len(helpElems); i++ {
		x := widths[i%len(widths)]
		y := 1 + i/2
		for _, r := range helpElems[i] {
			e.scr.SetContent(x, y, r, nil, tcell.StyleDefault)
			x += runewidth.RuneWidth(r)
		}
	}

	e.scr.Show()

	for {
		evt := e.scr.PollEvent()
		// wait for next key event, discard it.
		if _, ok := evt.(*tcell.EventKey); ok {
			return
		}
	}
}

func (e *editor) openFile() {
	file, ok := e.readString("Filename", nil)
	if !ok {
		return
	}

	if err := e.loadBufferFromFile(file); err != nil {
		e.showError("Couldn't open file: %v", err)
		return
	}

	e.bufIdx = len(e.bufs) - 1
}

func (e *editor) newBuffer() {
	e.addNewBuffer()
	e.bufIdx = len(e.bufs) - 1
}

func (e *editor) closeBuffer() {
	if len(e.bufs) == 1 {
		e.showError("Can't close last remaining buffer")
		return
	}

	curBuf := e.bufs[e.bufIdx]
	if curBuf.modified {
		switch e.query("Save file?", "ync") {
		case 'y':
			e.save()
		case 'n':
			// nothing to do
		case 'c':
			return // cancel closing
		}
	}

	e.bufs = append(e.bufs[:e.bufIdx], e.bufs[e.bufIdx+1:]...)
	if e.bufIdx >= len(e.bufs) {
		e.bufIdx = len(e.bufs) - 1
	}
}

func (e *editor) find() {
	curBuf := e.bufs[e.bufIdx]

	_, height := e.scr.Size()

	findPhrase, ok := e.readString("Find", curBuf.findPhrase)
	if !ok {
		return
	}

	curBuf.findPhrase = []rune(findPhrase)

	y, x, found := curBuf.find([]rune(findPhrase))
	if !found {
		e.showError("Text not found")
		return
	}

	curBuf.x = x
	for y > curBuf.curLineIdx() {
		curBuf.incrY(height)
	}
	for y < curBuf.curLineIdx() {
		curBuf.decrY()
	}
}

func (e *editor) redraw() {
	e.scr.Sync()
}
