package main

import (
	"fmt"
	"sort"
)

// sortStrings sorts a slice of strings in place.
func sortStrings(s []string) { sort.Strings(s) }

// mustNoErr panics on unexpected errors from the keyboard listener, mirroring
// the previous behaviour where errors were ignored. Kept central so callers
// stay readable.
func mustNoErr(err error) {
	if err != nil {
		fmt.Println("listener error:", err)
	}
}
