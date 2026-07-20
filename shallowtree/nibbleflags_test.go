package shallowtree

import "testing"

func TestNibblesFlags(t *testing.T) {
	flags := NewNibblesFlags(3, 8)
	if flags.Flags0to63 != 0xF8 {
		t.Fatal("flags.Flags0to63 != 0xF8")
	}
}
