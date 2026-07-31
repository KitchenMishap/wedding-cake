package hash

import (
	"io"
)

type PrefixHolder struct {
	hw               *HashWindow
	splitNibbleIndex byte // Specified
	prefixBytesCount byte // Calculated
}

// Slight whiff #EGGY needs to work even if hw.bytes have not yet been copied in!
func newPrefixHolder(hw *HashWindow, splitNibbleIndex byte) PrefixHolder {
	bytesCount := splitNibbleIndex / 2
	if splitNibbleIndex&1 == 1 {
		bytesCount++
	}
	return PrefixHolder{hw: hw, splitNibbleIndex: splitNibbleIndex, prefixBytesCount: bytesCount}
}

func (ph PrefixHolder) Read(reader io.Reader) error {
	_, err := io.ReadFull(reader, ph.hw.bytes[:ph.prefixBytesCount])
	return err
}

// LastReadContainedSpareNibble Please only call this immediately after a Read()
func (ph PrefixHolder) LastReadContainedSpareNibble() (bool, byte) {
	// If we wrote an even number of nibbles, there is none
	if ph.splitNibbleIndex&1 == 0 {
		return false, 0
	}
	return true, ph.hw.bytes[ph.prefixBytesCount-1] & 0x0F
}
func (ph PrefixHolder) Write(writer io.Writer, spareNibble byte) error {
	if spareNibble > 0x0F {
		panic("Nibble too large")
	}
	// If we are writing an odd number of nibbles, there is room for the spare nibble
	if ph.splitNibbleIndex&1 == 1 {
		// Don't worry about this, we are putting it somewhere that "should" be ignored (not officially part of the prefix,
		// but it is somewhere that we do the write from). It's the least significant nibble of the last byte we
		// write, so for the odd number of nibbles that we have here, "one nibble beyond" what should officially
		// be considered
		ph.hw.bytes[ph.prefixBytesCount-1] &= 0xF0 // Clear out any old "ignored" nibble from the bottom four bits
		ph.hw.bytes[ph.prefixBytesCount-1] |= spareNibble
	}
	_, err := writer.Write(ph.hw.bytes[:ph.prefixBytesCount])
	return err
}

// AppendSuffix Append the specified SuffixHolder to this PrefixHolder, using
// the supplied *HashWindow to construct the resultant HashHolder
func (ph PrefixHolder) AppendSuffix(target *HashWindow, suffix SuffixHolder) HashHolder {
	if suffix.splitNibbleIndex != ph.splitNibbleIndex || suffix.hw.hashByteCount != ph.hw.hashByteCount {
		panic("Prefix and suffix are not a matching pair")
	}
	if ph.splitNibbleIndex&1 == 0 {
		// Aah... Easy mode. Prefix and suffix are isolated byte slices. (Even number of nibbles each)
		copy(target.bytes[:ph.prefixBytesCount], ph.hw.bytes[:ph.prefixBytesCount]) // Prefix bytes
		copy(target.bytes[suffix.suffixBytesStart:suffix.hw.hashByteCount],         // Suffix bytes
			suffix.hw.bytes[suffix.suffixBytesStart:suffix.hw.hashByteCount])
	} else {
		// ooh... call the boss for this one
		// 1) There is one byte that we'll put aside the end of the prefix, and a byte from the start of the suffix.
		// This is because they overlap; they need to be "merged" into one destination byte
		skippedFromPrefixEnd := ph.hw.bytes[ph.prefixBytesCount-1]
		skippedFromSuffixStart := suffix.hw.bytes[ph.prefixBytesCount-1]                      // (yes they have the same index)
		skip := byte(1)                                                                       // Number of bytes we're skipping for the moment
		copy(target.bytes[:ph.prefixBytesCount-skip], ph.hw.bytes[:ph.prefixBytesCount-skip]) // Easy prefix bytes
		copy(target.bytes[suffix.suffixBytesStart+skip:suffix.hw.hashByteCount],              // Easy suffix bytes
			suffix.hw.bytes[suffix.suffixBytesStart+skip:suffix.hw.hashByteCount])
		// The final overlapping byte to merge in the middle isn't all that tricky
		mostSignificant := skippedFromPrefixEnd & 0xF0
		leastSignificant := skippedFromSuffixStart & 0x0F
		merged := mostSignificant | leastSignificant
		target.bytes[ph.prefixBytesCount-1] = merged
	}
	// Final odds and sods
	target.hashByteCount = ph.hw.hashByteCount
	hashHolder := HashHolder{hw: target}
	return hashHolder
}

func (ph PrefixHolder) AppendNibble(nibble byte) {
	if nibble > 0x0F {
		panic("Nibble too big")
	}
	if ph.splitNibbleIndex&1 == 1 {
		// In this situation an originally ODD prefix nibble count is the easy case
		byteIndex := ph.prefixBytesCount - 1 // Simply the last prefix byte
		ph.hw.bytes[byteIndex] &= 0xF0       // Clear out any old "ignored" nibble in the bottom four bits
		ph.hw.bytes[byteIndex] |= nibble     // Or in the new nibble
		ph.splitNibbleIndex++
		return // That is all
	} else {
		byteIndex := ph.prefixBytesCount
		ph.hw.bytes[byteIndex] &= 0x0F          // Clear out any old "ignored" nibble in the top four bits
		ph.hw.bytes[byteIndex] |= (nibble << 4) // Or on the new nibble
		ph.splitNibbleIndex++
		// That is not all
		ph.prefixBytesCount++ // Ok so it was easy
		return
	}
}

func (ph PrefixHolder) CountSupportedHashPrefixValues() uint64 {
	return uint64(1) << (4 * uint64(ph.splitNibbleIndex))
}

func (ph PrefixHolder) PrefixAsNumber() uint64 {
	// Special legal edge case... no nibbles
	if ph.splitNibbleIndex == 0 {
		return 0
	}
	result := uint64(0)
	lastNibbleExists := ph.splitNibbleIndex&1 == 1
	// Work FORWARDS through bytes, shifting up result
	for index := range int(ph.prefixBytesCount) {
		byteVal := ph.hw.bytes[index]
		if lastNibbleExists && index == int(ph.prefixBytesCount-1) {
			// Only shift up by 4 bits
			// Last nibble is in top four bits
			result = (result << 4) | uint64(byteVal>>4)
		} else {
			result = (result << 8) | uint64(byteVal)
		}
	}
	return result

	/* Gemini - note that test doesn't call these fns!
	if ph.splitNibbleIndex == 0 {
		return 0
	}

	result := uint64(0)

	// 1. Process full bytes from left to right (MSB first)
	wholeBytes := int(ph.splitNibbleIndex / 2)
	for index := 0; index < wholeBytes; index++ {
		result = (result << 8) | uint64(ph.hw.bytes[index])
	}

	// 2. If there's an odd nibble at the end, append it as the 4 least significant bits
	if ph.splitNibbleIndex&1 == 1 {
		lastByte := ph.hw.bytes[ph.prefixBytesCount-1]
		lastNibble := (lastByte >> 4) & 0x0F
		result = (result << 4) | uint64(lastNibble)
	}

	return result*/
}

func (ph PrefixHolder) SetPrefixFromNumber(number uint64) {
	if number >= ph.CountSupportedHashPrefixValues() {
		panic("hash prefix number too large")
	}

	// Special edge case... no nibbles
	if ph.splitNibbleIndex == 0 {
		// Don't write any bytes. Everything else should already be in place
		return
	}

	// If there is an odd number of nibbles, first set the end nibble, which we are treating as least significant
	byteIndex := int(ph.prefixBytesCount - 1)
	if ph.splitNibbleIndex&1 == 1 {
		nibble := byte(number & 0x0F)
		// Nibble goes in upper 4 bits of last byte
		ph.hw.bytes[byteIndex] &= 0x0F // Clear out any "ignored" nibble that's already there
		ph.hw.bytes[byteIndex] |= (nibble << 4)
		number >>= 4
		byteIndex--
	}
	// Work backwards a byte at a time
	for index := byteIndex; index >= 0; index-- {
		byteVal := byte(number & 0xFF)
		number >>= 8
		ph.hw.bytes[index] = byteVal
	}
	return
}
