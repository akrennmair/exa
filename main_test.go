package main

import (
	"io"
	"log"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/require"
)

func TestEditor(t *testing.T) {
	log.SetOutput(io.Discard)

	scr := tcell.NewSimulationScreen("utf-8")

	ed := newEditor(scr)

	require.NoError(t, scr.Init())

	ed.addNewBuffer()

	require.False(t, ed.bufs[ed.bufIdx].modified)

	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0)))
	ed.handleEvent()
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0)))
	ed.handleEvent()
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0)))
	ed.handleEvent()
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'c', 0)))
	ed.handleEvent()
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'd', 0)))
	ed.handleEvent()
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'e', 0)))
	ed.handleEvent()

	require.True(t, ed.bufs[ed.bufIdx].modified)
	require.Equal(t, ed.bufs[ed.bufIdx].lines, [][]rune{{'a', 'b'}, {'c', 'd', 'e'}})

	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyCtrlZ, 0, 0)))
	ed.handleEvent()

	require.Equal(t, ed.bufs[ed.bufIdx].lines, [][]rune{{}})

	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyCtrlR, 0, 0)))
	ed.handleEvent()

	require.Equal(t, ed.bufs[ed.bufIdx].lines, [][]rune{{'a', 'b'}, {'c', 'd', 'e'}})

	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyCtrlQ, 0, 0)))
	require.NoError(t, scr.PostEvent(tcell.NewEventKey(tcell.KeyRune, 'n', 0)))

	require.False(t, ed.quitInputLoop)

	ed.handleEvent()

	require.True(t, ed.quitInputLoop)
}
