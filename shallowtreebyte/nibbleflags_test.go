package shallowtreebyte

import "testing"

func TestNibblesFlags(t *testing.T) {
	flags := NewNibblesFlags(3, 8)
	if flags.Flags0to63 != 0xF8 {
		t.Fatal("flags.Flags0to63 != 0xF8")
	}
	unused := flags.FlagValByte(2)
	if unused == false {
		panic("Expected true")
	}
	flags.ClearFlagOrPanicByte(2)
	unused = flags.FlagValByte(2)
	if unused == true {
		panic("Expected false")
	}
}
