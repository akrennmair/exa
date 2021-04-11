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

func TestGetSelection(t *testing.T) {
	testData := map[string]struct {
		buf             *buffer
		expectedLowerY  int
		expectedLowerX  int
		expectedHigherY int
		expectedHigherX int
	}{
		"simple": {
			buf:             &buffer{startY: 0, startX: 23, endY: 3, endX: 42},
			expectedLowerY:  0,
			expectedLowerX:  23,
			expectedHigherY: 3,
			expectedHigherX: 42,
		},
		"reversed": {
			buf:             &buffer{startY: 3, startX: 23, endY: 0, endX: 42},
			expectedLowerY:  0,
			expectedLowerX:  42,
			expectedHigherY: 3,
			expectedHigherX: 23,
		},
		"same-line-reversed": {
			buf:             &buffer{startY: 0, startX: 15, endY: 0, endX: 3},
			expectedLowerY:  0,
			expectedLowerX:  3,
			expectedHigherY: 0,
			expectedHigherX: 15,
		},
	}

	for testName, tt := range testData {
		t.Run(testName, func(t *testing.T) {
			lowerY, lowerX, higherY, higherX := tt.buf.getSelection()
			assert.Equal(t, tt.expectedLowerY, lowerY)
			assert.Equal(t, tt.expectedLowerX, lowerX)
			assert.Equal(t, tt.expectedHigherY, higherY)
			assert.Equal(t, tt.expectedHigherX, higherX)
		})
	}
}

func TestIsWithinSelectedText(t *testing.T) {
	testData := map[string]struct {
		buf            *buffer
		y              int
		x              int
		expectedResult bool
	}{
		"simple": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              2,
			x:              3,
			expectedResult: true,
		},
		"simple-first-line": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              1,
			x:              10,
			expectedResult: true,
		},
		"simple-last-line": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              3,
			x:              7,
			expectedResult: true,
		},
		"outside-first-line": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              1,
			x:              0,
			expectedResult: false,
		},
		"outside-last-line": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              3,
			x:              20,
			expectedResult: false,
		},
		"outside-before-first-line": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              0,
			x:              20,
			expectedResult: false,
		},
		"outside-after-last-line": {
			buf:            &buffer{startY: 1, startX: 5, endY: 3, endX: 10},
			y:              5,
			x:              3,
			expectedResult: false,
		},
		"empty-selection": {
			buf:            &buffer{startY: 1, startX: 5, endY: 1, endX: 5},
			y:              1,
			x:              5,
			expectedResult: false,
		},
	}

	for testName, tt := range testData {
		t.Run(testName, func(t *testing.T) {
			result := tt.buf.isWithinSelectedText(tt.y, tt.x)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
