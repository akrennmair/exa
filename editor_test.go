package main

import (
	"io"
	"log"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/require"
)

func playKeys(t *testing.T, ed *editor, keys ...*tcell.EventKey) {
	for _, key := range keys {
		require.NoError(t, ed.scr.PostEvent(key))
		ed.handleEvent()
	}
}

func TestEditor1(t *testing.T) {
	log.SetOutput(io.Discard)

	scr := tcell.NewSimulationScreen("utf-8")

	ed := newEditor(scr)

	require.NoError(t, scr.Init())

	ed.addNewBuffer()

	require.False(t, ed.bufs[ed.bufIdx].modified)

	playKeys(
		t, ed,
		tcell.NewEventKey(tcell.KeyRune, 'a', 0),
		tcell.NewEventKey(tcell.KeyRune, 'b', 0),
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		tcell.NewEventKey(tcell.KeyRune, 'c', 0),
		tcell.NewEventKey(tcell.KeyRune, 'd', 0),
		tcell.NewEventKey(tcell.KeyRune, 'e', 0),
	)

	require.True(t, ed.bufs[ed.bufIdx].modified)
	require.Equal(t, [][]rune{{'a', 'b'}, {'c', 'd', 'e'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))

	require.Equal(t, ed.bufs[ed.bufIdx].lines, [][]rune{{}})

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlR, 0, 0))

	require.Equal(t, [][]rune{{'a', 'b'}, {'c', 'd', 'e'}}, ed.bufs[ed.bufIdx].lines)

	ed.bufs[ed.bufIdx].y = 1 // TODO: fix cursor placement after Ctrl-R and remove these two fixes.
	ed.bufs[ed.bufIdx].x = 3

	require.Equal(t, 1, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 3, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyBackspace2, 0, 0), tcell.NewEventKey(tcell.KeyBackspace2, 0, 0))

	require.Equal(t, [][]rune{{'a', 'b'}, {'c'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0))

	require.Equal(t, [][]rune{{'a', 'b'}, {'c', 'd', 'e'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlR, 0, 0))

	require.Equal(t, [][]rune{{'a', 'b'}, {'c'}}, ed.bufs[ed.bufIdx].lines)

	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)))
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'n', 0)))

	require.False(t, ed.quitInputLoop)

	ed.handleEvent()

	require.True(t, ed.quitInputLoop)
}

func TestEditor2(t *testing.T) {
	log.SetOutput(io.Discard)

	scr := tcell.NewSimulationScreen("utf-8")

	ed := newEditor(scr)

	require.NoError(t, scr.Init())

	ed.addNewBuffer()

	require.False(t, ed.bufs[ed.bufIdx].modified)

	playKeys(
		t, ed,
		tcell.NewEventKey(tcell.KeyRune, 'q', 0),
		tcell.NewEventKey(tcell.KeyRune, 'w', 0),
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		tcell.NewEventKey(tcell.KeyRune, 'e', 0),
		tcell.NewEventKey(tcell.KeyRune, 'r', 0),
		tcell.NewEventKey(tcell.KeyRune, 't', 0),
		tcell.NewEventKey(tcell.KeyEnter, 0, 0),
		tcell.NewEventKey(tcell.KeyRune, 'z', 0),
		tcell.NewEventKey(tcell.KeyRune, 'u', 0),
		tcell.NewEventKey(tcell.KeyRune, 'i', 0),
		tcell.NewEventKey(tcell.KeyRune, 'o', 0),
	)

	require.True(t, ed.bufs[ed.bufIdx].modified)
	require.Equal(t, [][]rune{{'q', 'w'}, {'e', 'r', 't'}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	require.Equal(t, 2, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 4, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlA, 0, 0))
	require.Equal(t, 0, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlE, 0, 0))
	require.Equal(t, 4, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyUp, 0, 0))
	require.Equal(t, 1, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 3, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlA, 0, 0))
	require.Equal(t, 0, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyDEL, 0, 0))
	require.Equal(t, 0, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 2, ed.bufs[ed.bufIdx].x)

	require.Equal(t, [][]rune{{'q', 'w', 'e', 'r', 't'}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyEnter, 0, 0))

	require.Equal(t, [][]rune{{'q', 'w'}, {'e', 'r', 't'}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	require.Equal(t, 1, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 0, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyDelete, 0, 0))

	require.Equal(t, [][]rune{{'q', 'w'}, {'r', 't'}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyDown, 0, 0))
	require.Equal(t, 2, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 0, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyDown, 0, 0))
	require.Equal(t, 2, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 0, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyUp, 0, 0), tcell.NewEventKey(tcell.KeyUp, 0, 0), tcell.NewEventKey(tcell.KeyCtrlE, 0, 0))
	require.Equal(t, 0, ed.bufs[ed.bufIdx].curLineIdx())
	require.Equal(t, 2, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyDelete, 0, 0))

	require.Equal(t, [][]rune{{'q', 'w', 'r', 't'}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	require.Equal(t, 1, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed,
		tcell.NewEventKey(tcell.KeyRight, 0, 0),
		tcell.NewEventKey(tcell.KeyRight, 0, 0),
		tcell.NewEventKey(tcell.KeyRight, 0, 0),
	)
	require.Equal(t, 4, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyLeft, 0, 0), tcell.NewEventKey(tcell.KeyLeft, 0, 0))
	require.Equal(t, 2, ed.bufs[ed.bufIdx].x)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlU, 0, 0))

	require.Equal(t, [][]rune{{'r', 't'}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	playKeys(t, ed, tcell.NewEventKey(tcell.KeyCtrlK, 0, 0))

	require.Equal(t, [][]rune{{}, {'z', 'u', 'i', 'o'}}, ed.bufs[ed.bufIdx].lines)

	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)))
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'n', 0)))

	require.False(t, ed.quitInputLoop)

	ed.handleEvent()

	require.True(t, ed.quitInputLoop)
}
