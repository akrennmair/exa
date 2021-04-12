package main

import (
	"log"
)

type buffer struct {
	fname    string
	lines    [][]rune
	x        int
	y        int
	offset   int
	modified bool

	// fields to track selected text:
	selecting bool
	startX    int
	startY    int
	endX      int
	endY      int

	// stuff to track edit history for undo/redo:
	editHistory []*editOp
	historyIdx  int

	findLastLine int
	findPhrase   []rune
}

func (buf *buffer) getSelection() (lowerY, lowerX, higherY, higherX int) {
	lowerY, lowerX, higherY, higherX = buf.startY, buf.startX, buf.endY, buf.endX

	if lowerY > higherY {
		lowerY, higherY = higherY, lowerY
		lowerX, higherX = higherX, lowerX
	} else if lowerY == higherY && lowerX > higherX {
		lowerX, higherX = higherX, lowerX
	}

	return
}

func (buf *buffer) isWithinSelectedText(y, x int) bool {
	lowerY, lowerX, higherY, higherX := buf.getSelection()

	if lowerY == higherY && lowerX == higherX {
		return false
	}

	if y > lowerY && y < higherY {
		return true
	}
	if y == lowerY && x >= lowerX && (y < higherY || (y == higherY && x < higherX)) {
		return true
	}
	if y == higherY && x < higherX && (y > lowerY || (y == lowerY && x >= lowerX)) {
		return true
	}
	return false
}

func (buf *buffer) incrY(height int) {
	if buf.y < height-3 {
		buf.y++
	} else {
		buf.offset++
	}
}

func (buf *buffer) decrY() {
	if buf.offset > 0 {
		buf.offset--
	} else {
		buf.y--
	}
}

func (buf *buffer) correctX() {
	if l := len(buf.lines[buf.curLineIdx()]); l < buf.x {
		buf.x = l
	}
}

func (buf *buffer) correctY() {
	for buf.curLineIdx() >= len(buf.lines) {
		buf.decrY()
	}
}

func (buf *buffer) curLineIdx() int {
	return buf.y + buf.offset
}

func (buf *buffer) curLine() []rune {
	return buf.lines[buf.curLineIdx()]
}

func (buf *buffer) getOrCreateLatestOp(code opcode) *editOp {
	op := &editOp{
		op:       code,
		text:     [][]rune{{}},
		y:        buf.curLineIdx(),
		x:        buf.x,
		finished: false,
	}

	if len(buf.editHistory) == 0 {
		log.Printf("getOrCreateLatestOp: no edit history, creating new op")
		buf.editHistory = append(buf.editHistory, op)
		buf.historyIdx++
		return op
	}

	if op := buf.editHistory[buf.historyIdx]; op.op == code && !op.finished {
		log.Printf("getOrCreateLatestOp: returning op at index %d", buf.historyIdx)
		return op
	}

	buf.editHistory = append(buf.editHistory[:buf.historyIdx+1], op)
	buf.historyIdx++
	log.Printf("getOrCreateLatestOp: added op at index %d", buf.historyIdx)

	return op
}

func (buf *buffer) historyAddRune(r rune) {
	op := buf.getOrCreateLatestOp(opInsertText)

	op.text[len(op.text)-1] = append(op.text[len(op.text)-1], r)

	log.Printf("historyAddRune: %d lines", len(op.text))
	for idx, line := range op.text {
		log.Printf("historyAddRune: line %d: %s", idx, string(line))
	}
}

func (buf *buffer) historyFinishOp() {
	if len(buf.editHistory) == 0 {
		return
	}
	buf.editHistory[buf.historyIdx].finished = true
}

func (buf *buffer) historyAddLine() {
	op := buf.getOrCreateLatestOp(opInsertText)

	op.text = append(op.text, []rune{})
	log.Printf("historyAddLine: %d lines", len(op.text))
	for idx, line := range op.text {
		log.Printf("historyAddLine: line %d: %s", idx, string(line))
	}
}

func (buf *buffer) historyRemoveLine() {
	op := buf.getOrCreateLatestOp(opRemoveText)

	op.text = append([][]rune{{}}, op.text...)
	op.y = buf.curLineIdx()
	op.x = buf.x
}

func (buf *buffer) historyRemoveChar(r rune) {
	op := buf.getOrCreateLatestOp(opRemoveText)

	op.text[0] = append([]rune{r}, op.text[0]...)
	op.y = buf.curLineIdx()
	op.x = buf.x
}

func (buf *buffer) find(phrase []rune) (y, x int, found bool) {
	if !runeEqual(buf.findPhrase, phrase) {
		buf.findLastLine = buf.curLineIdx()
		if buf.findLastLine == 0 {
			buf.findLastLine = len(buf.lines) - 1
		} else {
			buf.findLastLine--
		}
	}

	for y := buf.findLastLine + 1; y < len(buf.lines); y++ {
		if x := runeIndex(buf.lines[y], phrase); x >= 0 {
			buf.findLastLine = y
			return y, x, true
		}
	}
	for idx := 0; idx <= buf.findLastLine; idx++ {
		if x := runeIndex(buf.lines[y], phrase); x >= 0 {
			buf.findLastLine = y
			return y, x, true
		}
	}

	return 0, 0, false
}

type editOp struct {
	op       opcode
	text     [][]rune
	y        int
	x        int
	finished bool
}

type opcode int

const (
	opInsertText opcode = iota
	opRemoveText
)

func (op *editOp) undo(buf *buffer) {
	log.Printf("editOp.undo: op = %d y = %d x = %d", op.op, op.y, op.x)
	for idx, line := range op.text {
		log.Printf("editOp.undo: line %d: %s", idx, string(line))
	}
	for idx, line := range buf.lines {
		log.Printf("editOp.undo: buf line %d: %s", idx, string(line))
	}
	switch op.op {
	case opInsertText:
		op.removeText(buf)
		for idx, line := range buf.lines {
			log.Printf("editOp.undo: after buf line %d: %s", idx, string(line))
		}
	case opRemoveText:
		op.insertText(buf)
	}
}

func (op *editOp) redo(buf *buffer) {
	switch op.op {
	case opInsertText:
		op.insertText(buf)
	case opRemoveText:
		op.removeText(buf)
	}
}

func (op *editOp) removeText(buf *buffer) {
	for idx, line := range buf.lines {
		log.Printf("removeText: before: line %d: %q", idx, string(line))
	}
	log.Printf("removeText: op: y = %d x = %d len(op.text) = %d", op.y, op.x, len(op.text))
	for idx, line := range op.text {
		log.Printf("removeText: op buf line %d: %q", idx, string(line))
	}
	if len(op.text) == 1 {
		buf.lines[op.y] = append(buf.lines[op.y][:op.x], buf.lines[op.y][op.x+len(op.text[0]):]...)
	} else {
		log.Printf("removeText: before part: %d %d %q", op.y, op.x, string(buf.lines[op.y][:op.x]))
		log.Printf("removeText: after part: %d %d %q", op.y+len(op.text)-1, len(op.text[len(op.text)-1]), string(buf.lines[op.y+len(op.text)-1][len(op.text[len(op.text)-1:])]))
		buf.lines[op.y] = append(buf.lines[op.y][:op.x], buf.lines[op.y+len(op.text)-1][len(op.text[len(op.text)-1]):]...)
		log.Printf("removeText: new line %d: %q", op.y, string(buf.lines[op.y]))
		buf.lines = append(buf.lines[:op.y+1], buf.lines[op.y+len(op.text):]...)
	}
	for idx, line := range buf.lines {
		log.Printf("removeText: after: line %d: %q", idx, string(line))
	}
}

func (op *editOp) insertText(buf *buffer) {
	if len(op.text) == 1 {
		buf.lines[op.y] = append(buf.lines[op.y][:op.x], append(op.text[0], buf.lines[op.y][op.x:]...)...)
	} else {
		insertion := [][]rune{}
		insertion = append(insertion, op.text...)

		beforeInsertion, afterInsertion := buf.lines[op.y][:op.x], buf.lines[op.y][op.x:]
		insertion[0] = append(append([]rune{}, beforeInsertion...), insertion[0]...)
		insertion[len(insertion)-1] = append(insertion[len(insertion)-1], afterInsertion...)

		buf.lines = append(buf.lines[:op.y], append(insertion, buf.lines[op.y+1:]...)...)
	}
}
