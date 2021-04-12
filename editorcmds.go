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
	log.Printf("gotoBOL: line %d set x = 0", curBuf.curLineIdx())
	curBuf.x = 0
	e.updateSelectedTextPos(curBuf)
	curBuf.historyFinishOp()
}

func (e *editor) gotoEOL() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.y + curBuf.offset
	curLine := curBuf.lines[lineIdx]

	curBuf.x = len(curLine)

	log.Printf("gotoEOL: line %d set x = %d", curBuf.curLineIdx(), curBuf.x)

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) newLine() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.curLineIdx()

	curLine, nextLine := curBuf.lines[lineIdx][:curBuf.x], curBuf.lines[lineIdx][curBuf.x:]
	log.Printf("newLine: line %d: %q -> %q, %q", lineIdx, string(curBuf.lines[lineIdx]), string(curLine), string(nextLine))
	curBuf.lines[lineIdx] = curLine
	curBuf.lines = append(curBuf.lines[:lineIdx+1], append([][]rune{nextLine}, curBuf.lines[lineIdx+1:]...)...)

	_, height := e.scr.Size()

	curBuf.historyAddLine()

	curBuf.incrY(height)
	curBuf.x = 0
}

func (e *editor) keyBackspace() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.curLineIdx()

	if curBuf.x == 0 {
		if lineIdx == 0 {
			log.Printf("keyBackspace: nothing to do as we're on the leftmost character on the first line")
			return
		}

		log.Printf("keyBackspace: joining line %d with previous line", lineIdx)

		curBuf.x = len(curBuf.lines[lineIdx-1])
		curBuf.lines[lineIdx-1] = append(curBuf.lines[lineIdx-1], curBuf.lines[lineIdx]...)
		curBuf.lines = append(curBuf.lines[:lineIdx], curBuf.lines[lineIdx+1:]...)

		curBuf.decrY()

		curBuf.historyRemoveLine()
		return
	}

	r := curBuf.lines[lineIdx][curBuf.x-1]

	log.Printf("keyBackspace: deleting character %c in line %d col %d", r, lineIdx, curBuf.x-1)

	curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx][:curBuf.x-1], curBuf.lines[lineIdx][curBuf.x:]...)
	curBuf.x--
	curBuf.historyRemoveChar(r)
}

func (e *editor) keyDel() {
	curBuf := e.bufs[e.bufIdx]
	lineIdx := curBuf.curLineIdx()

	if curBuf.x == len(curBuf.lines[lineIdx]) {
		if lineIdx == len(curBuf.lines)-1 {
			log.Printf("keyDel: nothing to do as we're on the last character on the last line")
			return
		}

		log.Printf("keyDel: joining line %d with next line", lineIdx)

		curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx], curBuf.lines[lineIdx+1]...)
		curBuf.lines = append(curBuf.lines[:lineIdx+1], curBuf.lines[lineIdx+2:]...)
		curBuf.historyRemoveLine()
		return
	}

	r := curBuf.lines[lineIdx][curBuf.x]

	log.Printf("keyDel: deleting character %c in line %d col %d", r, lineIdx, curBuf.x)

	curBuf.lines[lineIdx] = append(curBuf.lines[lineIdx][:curBuf.x], curBuf.lines[lineIdx][curBuf.x+1:]...)
	curBuf.historyRemoveChar(r)
}

func (e *editor) keyUp() {
	curBuf := e.bufs[e.bufIdx]

	if (curBuf.curLineIdx()) == 0 {
		log.Printf("keyUp: in first line already")
		return
	}

	curBuf.decrY()

	curBuf.correctX()

	log.Printf("keyUp: y = %d offset = %d x = %d", curBuf.y, curBuf.offset, curBuf.x)

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) keyDown() {
	curBuf := e.bufs[e.bufIdx]

	if (curBuf.y + curBuf.offset) >= len(curBuf.lines)-1 {
		log.Printf("keyDown: in last line already")
		return
	}

	_, height := e.scr.Size()

	curBuf.incrY(height)

	curBuf.correctX()

	log.Printf("keyDown: y = %d offset = %d x = %d", curBuf.y, curBuf.offset, curBuf.x)

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) keyLeft() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.x > 0 {
		curBuf.x--
	}

	log.Printf("keyLeft: line %d x = %d", curBuf.curLineIdx(), curBuf.x)

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) keyRight() {
	curBuf := e.bufs[e.bufIdx]

	if len(curBuf.curLine()) > curBuf.x {
		curBuf.x++
	}

	log.Printf("keyLeft: line %d x = %d", curBuf.curLineIdx(), curBuf.x)

	e.updateSelectedTextPos(curBuf)

	curBuf.historyFinishOp()
}

func (e *editor) quit() {
	log.Printf("quit: %d buffers to check", len(e.bufs))
	for i := 0; i < len(e.bufs); i++ {
		e.bufIdx = i
		if e.bufs[e.bufIdx].modified {
			log.Printf("quit: buffer %d (file %q) is modified", e.bufIdx, e.bufs[e.bufIdx].fname)
			e.redrawScreen()
			switch e.query("Save file?", "ync") {
			case 'y':
				e.save()
			case 'n':
				log.Printf("quit: ignoring buffer")
				// ignore file.
			case 'c':
				log.Printf("quit: cancelled quitting")
				return // cancel quitting.
			}
		}
	}

	// all files checked whether user wants to save them -> quit
	e.quitInputLoop = true
}

func (e *editor) save() {
	curBuf := e.bufs[e.bufIdx]

	log.Printf("save: saving buffer %d (%q)", e.bufIdx, curBuf.fname)

	if curBuf.fname == "" {
		fname, ok := e.readString("Filename", nil)
		if !ok {
			log.Printf("save: cancelled entering filename")
			return
		}

		curBuf.fname = fname
	}

	e.saveFile(curBuf)
}

func (e *editor) saveAs() {
	curBuf := e.bufs[e.bufIdx]

	log.Printf("saveAs: saving buffer %d under new name", e.bufIdx)

	fname, ok := e.readString("New filename", nil)
	if !ok {
		log.Printf("saveAs: cancelled entering filename")
		return
	}

	_, err := os.Stat(fname)
	if err == nil {
		switch e.query("Are you sure you want to overwrite file?", "yn") {
		case 'y':
			// continue as normal
		case 'n':
			log.Printf("saveAs: cancelled overwriting existing file")
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
	log.Printf("nextBuffer: new bufIdx = %d", e.bufIdx)
}

func (e *editor) prevBuffer() {
	e.bufIdx--
	if e.bufIdx < 0 {
		e.bufIdx = len(e.bufs) - 1
	}
	log.Printf("prevBuffer: new bufIdx = %d", e.bufIdx)
}

func (e *editor) deleteToEOL() {
	curBuf := e.bufs[e.bufIdx]

	log.Printf("deleteToEOL: line %d x = %d", curBuf.curLineIdx(), curBuf.x)

	curBuf.lines[curBuf.curLineIdx()] = curBuf.curLine()[:curBuf.x]
}

func (e *editor) deleteFromBOL() {
	curBuf := e.bufs[e.bufIdx]

	log.Printf("deleteFromBOL: line %d x = %d", curBuf.curLineIdx(), curBuf.x)

	curBuf.lines[curBuf.curLineIdx()] = curBuf.curLine()[curBuf.x:]
	curBuf.x = 0
}

func (e *editor) selectText() {
	curBuf := e.bufs[e.bufIdx]

	if !curBuf.selecting {
		curBuf.startX, curBuf.startY = curBuf.x, curBuf.curLineIdx()
		curBuf.endX, curBuf.endY = curBuf.startX, curBuf.startY
		log.Printf("selectText: starting from %d/%d", curBuf.startY, curBuf.startX)
	} else {
		log.Printf("selectText: stopped at %d/%d", curBuf.endY, curBuf.endX)
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

	log.Printf("copyText: copied data to clipboard")
	for idx, line := range e.clipboard {
		log.Printf("copyText: clipboard line %d: %s", idx, string(line))
	}
}

func (e *editor) cutText() {
	log.Printf("cutText: calling copyText first")
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

	log.Printf("cutText: removed selected text")
}

func (e *editor) pasteText() {
	insertion := [][]rune{}
	insertion = append(insertion, e.clipboard...)

	log.Printf("pasteText: inserting data from clipboard")
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

	log.Printf("pageDown: new line %d x = %d", curBuf.curLineIdx(), curBuf.x)
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

	log.Printf("pageUp: new line %d x = %d", curBuf.curLineIdx(), curBuf.x)

}

func (e *editor) undo() {
	curBuf := e.bufs[e.bufIdx]

	if curBuf.historyIdx < 0 {
		log.Printf("undo: nothing to undo")
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
		log.Printf("redo: nothing to redo")
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
	log.Printf("showHelp: showing help")
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
		log.Printf("openFile: entering filename cancelled")
		return
	}

	if err := e.loadBufferFromFile(file); err != nil {
		log.Printf("openFile: loading file %q failed: %v", file, err)
		e.showError("Couldn't open file: %v", err)
		return
	}

	e.bufIdx = len(e.bufs) - 1

	log.Printf("openFile: loaded file %q to buffer %d", file, e.bufIdx)
}

func (e *editor) newBuffer() {
	e.addNewBuffer()
	e.bufIdx = len(e.bufs) - 1
	log.Printf("newBuffer: added empty buffer")
}

func (e *editor) closeBuffer() {
	if len(e.bufs) == 1 {
		log.Printf("closeBuffer: can't close last remaining buffer")
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
			log.Printf("closeBuffer: cancelled closing buffer")
			return // cancel closing
		}
	}

	log.Printf("closeBuffer: closed buffer at index %d", e.bufIdx)

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
		log.Printf("find: entering search phrase cancelled")
		return
	}

	log.Printf("find: searching for phrase %q", findPhrase)

	curBuf.findPhrase = []rune(findPhrase)

	y, x, found := curBuf.find([]rune(findPhrase))
	if !found {
		log.Printf("find: phrase %q not found", findPhrase)
		e.showError("Text not found")
		return
	}

	log.Printf("find: found phrase %q at line %d col %d", findPhrase, y, x)

	curBuf.x = x
	for y > curBuf.curLineIdx() {
		curBuf.incrY(height)
	}
	for y < curBuf.curLineIdx() {
		curBuf.decrY()
	}
}

func (e *editor) redraw() {
	log.Printf("redraw: syncing whole screen")
	e.scr.Sync()
}
