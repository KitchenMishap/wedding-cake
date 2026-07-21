package shallowtreebyte

// NibblesFlags acts rather like an array of 128 bools
// There can be up to 128 nibbles in a hash (for a 512 bit hash)
// If you want a flag representing each nibble, this would be too many for a uint64
type NibblesFlags struct {
	Flags0to63   uint64
	Flags64to127 uint64
}

func NewNibblesFlags(prefixNibblesCount NibbleIndex, nibblesCount NibbleIndex) NibblesFlags {
	if prefixNibblesCount >= nibblesCount {
		panic("prefixNibblesCount >= nibblesCount not supported")
	}
	result := NibblesFlags{}
	// First set all LSB nibbleCount flags to 1
	if nibblesCount >= 64 {
		result.Flags0to63 = ^uint64(0)
	} else {
		result.Flags0to63 = ^uint64(0) >> (64 - nibblesCount)
	}
	// Then the MSB flags to 1
	if nibblesCount <= 64 {
		result.Flags64to127 = uint64(0)
	} else {
		result.Flags64to127 = ^uint64(0) >> (64 - (nibblesCount - 64))
	}
	// Then clear the LSB prefixNibblesCount
	if prefixNibblesCount >= 64 {
		panic("prefixNibblesCount >= 64 not supported")
	}
	prefixFlags := ^uint64(0) >> (64 - prefixNibblesCount)
	inverse := (^uint64(0)) ^ prefixFlags
	result.Flags0to63 &= inverse
	return result
}

func (flags *NibblesFlags) FlagVal(nibbleIndex NibbleIndex) bool {
	if nibbleIndex < 64 {
		return flags.Flags0to63&(1<<nibbleIndex) != 0
	} else {
		return flags.Flags64to127&(1<<(nibbleIndex-64)) != 0
	}
}

func (flags *NibblesFlags) FlagValByte(byteIndex ByteIndex) bool {
	nibbleIndex0 := NibbleIndex(byteIndex * 2)
	nibbleIndex1 := NibbleIndex(byteIndex*2 + 1)
	return flags.FlagVal(nibbleIndex0) && flags.FlagVal(nibbleIndex1) // byte flagged if both nibbles flagged
}

func (flags *NibblesFlags) ClearFlagOrPanic(nibbleIndex NibbleIndex) {
	if nibbleIndex < 64 {
		mask := uint64(1) << nibbleIndex
		if flags.Flags0to63&mask == 0 {
			panic("NibblesFlag already cleared")
		}
		flags.Flags0to63 ^= mask
	} else {
		mask := uint64(1) << (nibbleIndex - 64)
		if flags.Flags64to127&mask == 0 {
			panic("NibblesFlag already cleared")
		}
		flags.Flags64to127 ^= mask
	}
}

func (flags *NibblesFlags) ClearFlagOrPanicByte(byteIndex ByteIndex) {
	nibbleIndex0 := NibbleIndex(byteIndex * 2)
	nibbleIndex1 := NibbleIndex(byteIndex*2 + 1)
	flags.ClearFlagOrPanic(nibbleIndex0)
	flags.ClearFlagOrPanic(nibbleIndex1)
}

func (flags *NibblesFlags) IsEmpty() bool {
	return flags.Flags0to63 == 0 && flags.Flags64to127 == 0
}

func (flags *NibblesFlags) Copy() *NibblesFlags {
	result := NibblesFlags{}
	result.Flags0to63 = flags.Flags0to63
	result.Flags64to127 = flags.Flags64to127
	return &result
}
