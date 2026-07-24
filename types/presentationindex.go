package types

// GlobalPi A gloabl presentation index refers to the order in which a hash index is presented to the cake
type GlobalPi uint64

const GlobalPresentationIndexNoMatch = ^GlobalPi(0)

// PiOffset An offset to add to a local presentation index to get the global presentation index
type PiOffset GlobalPi

// LocalPi A local presentation index is a GlobalPi - a PiOffset.
// It is used to allow storage in a smaller number of bytes.
type LocalPi uint64

const LocalPiNoMatch = ^LocalPi(0)

// Functions for converting between the above
func (gpi GlobalPi) ToLocalPi(offset PiOffset) LocalPi {
	if gpi == GlobalPresentationIndexNoMatch {
		return LocalPiNoMatch
	}
	return LocalPi(uint64(gpi) - uint64(offset))
}
func (gpi GlobalPi) ToOffsetPi() PiOffset {
	if gpi == GlobalPresentationIndexNoMatch {
		panic("No match cannot be converted to an offset")
	}
	return PiOffset(gpi)
}
func (lpi LocalPi) ToGlobalPi(offset PiOffset) GlobalPi {
	if lpi == LocalPiNoMatch {
		return GlobalPresentationIndexNoMatch
	}
	return GlobalPi(uint64(lpi) + uint64(offset))
}
