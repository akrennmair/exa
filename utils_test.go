package main

import (
	"testing"

	"github.com/stretchr/testify/require"
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
			require.Equal(t, tt.ExpectedIndex, idx, "haystack %q, needle %q", string(tt.Haystack), string(tt.Needle))
		})
	}
}

func TestRuneEqual(t *testing.T) {
	testData := map[string]struct {
		A           []rune
		B           []rune
		ExpectEqual bool
	}{
		"equal":              {[]rune("hello"), []rune("hello"), true},
		"not-equal":          {[]rune("hello"), []rune("world"), false},
		"not-equal-diff-len": {[]rune("hello"), []rune("good bye!"), false},
		"empty":              {[]rune{}, []rune{}, true},
		"nil":                {nil, nil, true},
	}

	for testName, tt := range testData {
		t.Run(testName, func(t *testing.T) {
			require.Equal(t, tt.ExpectEqual, runeEqual(tt.A, tt.B))
		})
	}
}

func TestRuneWidth(t *testing.T) {
	testData := map[string]struct {
		S             []rune
		ExpectedWidth int
	}{
		"simple":             {[]rune("abc"), 3},
		"empty":              {[]rune{}, 0},
		"simple-with-tab":    {[]rune("hello\tworld!"), 19},
		"chinese-characters": {[]rune("例子"), 4},
		"mixed-characters":   {[]rune("hello例world子!"), 15},
	}

	for testName, tt := range testData {
		t.Run(testName, func(t *testing.T) {
			require.Equal(t, tt.ExpectedWidth, runeWidth(tt.S))
		})
	}
}
