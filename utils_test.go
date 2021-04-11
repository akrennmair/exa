package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuneIndex(t *testing.T) {
	testData := map[string]struct {
		Haystack      []rune
		Needle        []rune
		ExpectedIndex int
	}{
		"simple": {
			[]rune("hello"),
			[]rune("ll"),
			2,
		},
		"not-found": {
			[]rune("world"),
			[]rune("x"),
			-1,
		},
		"not-found-2": {
			[]rune("world"),
			[]rune("a really long phrase"),
			-1,
		},
		"chinese-characters": {
			[]rune("例子"),
			[]rune("子"),
			1,
		},
		"match-at-beginning": {
			[]rune("hello"),
			[]rune("he"),
			0,
		},
		"full-match-at-end": {
			[]rune("hello"),
			[]rune("lo"),
			3,
		},
		"partial-match-at-end": {
			[]rune("hello"),
			[]rune("lonely"),
			-1,
		},
	}

	for testName, tt := range testData {
		t.Run(testName, func(t *testing.T) {
			idx := runeIndex(tt.Haystack, tt.Needle)
			assert.Equal(t, tt.ExpectedIndex, idx, "haystack %q, needle %q", string(tt.Haystack), string(tt.Needle))
		})
	}
}
