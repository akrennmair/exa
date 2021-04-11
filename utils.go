package main

import "github.com/mattn/go-runewidth"

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

func runeEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}

	for idx := range a {
		if a[idx] != b[idx] {
			return false
		}
	}

	return true
}

func runeIndex(haystack, needle []rune) (idx int) {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		mismatch := false
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				mismatch = true
				break
			}
		}
		if !mismatch {
			return i
		}
	}
	return -1
}
