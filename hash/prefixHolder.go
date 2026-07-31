package hash

import (
	"io"
)

type PrefixHolder struct {
	bytes            [8]byte
	hashBytesCount   byte // Specified. The bytes in the source hash
	splitNibbleIndex byte // Specified
	prefixBytesCount byte // Calculated
}

// Slight whiff #EGGY needs to work even if hw.bytes have not yet been copied in!
func (ph *PrefixHolder) Init(hashBytesCount byte, splitNibbleIndex byte) {
	// Specify
	ph.hashBytesCount = hashBytesCount
	ph.splitNibbleIndex = splitNibbleIndex

	// Calculate
	if splitNibbleIndex > 16 {
		panic("Prefixes support up to 16 nibbles")
	}
	bytesCount := splitNibbleIndex / 2
	if splitNibbleIndex&1 == 1 {
		bytesCount++
	}
	ph.prefixBytesCount = bytesCount
}

func (ph *PrefixHolder) IsValid() bool {
	return ph.hashBytesCount > 0 && ph.hashBytesCount <= MaxHashBytes
}

func (ph *PrefixHolder) Read(reader io.Reader) error {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	_, err := io.ReadFull(reader, ph.bytes[:ph.prefixBytesCount])
	return err
}

// LastReadContainedSpareNibble Please only call this immediately after a Read()
func (ph *PrefixHolder) LastReadContainedSpareNibble() (bool, byte) {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	// If we wrote an even number of nibbles, there is none
	if ph.splitNibbleIndex&1 == 0 {
		return false, 0
	}
	return true, ph.bytes[ph.prefixBytesCount-1] & 0x0F
}
func (ph *PrefixHolder) Write(writer io.Writer, spareNibble byte) error {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	// If we are writing an odd number of nibbles, there is room for the spare nibble
	if ph.splitNibbleIndex&1 == 1 {
		if spareNibble > 0x0F {
			panic("Nibble too large")
		}
		// Don't worry about this, we are putting it somewhere that "should" be ignored (not officially part of the prefix,
		// but it is somewhere that we do the write from). It's the least significant nibble of the last byte we
		// write, so for the odd number of nibbles that we have here, "one nibble beyond" what should officially
		// be considered
		ph.bytes[ph.prefixBytesCount-1] &= 0xF0 // Clear out any old "ignored" nibble from the bottom four bits
		ph.bytes[ph.prefixBytesCount-1] |= spareNibble
	}
	_, err := writer.Write(ph.bytes[:ph.prefixBytesCount])
	return err
}

// AppendSuffix Append the specified SuffixHolder to this PrefixHolder, using
// the supplied *HashWindow to construct the resultant HashHolder
func (ph *PrefixHolder) AppendSuffix(result *HashHolder, suffix *SuffixHolder) {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	if suffix.splitNibbleIndex != ph.splitNibbleIndex || suffix.hashBytesCount != ph.hashBytesCount {
		panic("Prefix and suffix are not a matching pair")
	}
	result.Init(ph.hashBytesCount)
	if ph.splitNibbleIndex&1 == 0 {
		// Aah... Easy mode. Prefix and suffix are isolated byte slices. (Even number of nibbles each)
		copy(result.bytes[:ph.prefixBytesCount], ph.bytes[:ph.prefixBytesCount]) // Prefix bytes
		copy(result.bytes[suffix.suffixBytesStart:suffix.hashBytesCount],        // Suffix bytes
			suffix.bytes[suffix.suffixBytesStart:suffix.hashBytesCount])
	} else {
		// ooh... call the boss for this one
		// 1) There is one byte that we'll put aside the end of the prefix, and a byte from the start of the suffix.
		// This is because they overlap; they need to be "merged" into one destination byte
		skippedFromPrefixEnd := ph.bytes[ph.prefixBytesCount-1]
		skippedFromSuffixStart := suffix.bytes[ph.prefixBytesCount-1]                      // (yes they have the same index)
		skip := byte(1)                                                                    // Number of bytes we're skipping for the moment
		copy(result.bytes[:ph.prefixBytesCount-skip], ph.bytes[:ph.prefixBytesCount-skip]) // Easy prefix bytes
		copy(result.bytes[suffix.suffixBytesStart+skip:suffix.hashBytesCount],             // Easy suffix bytes
			suffix.bytes[suffix.suffixBytesStart+skip:suffix.hashBytesCount])
		// The final overlapping byte to merge in the middle isn't all that tricky
		mostSignificant := skippedFromPrefixEnd & 0xF0
		leastSignificant := skippedFromSuffixStart & 0x0F
		merged := mostSignificant | leastSignificant
		result.bytes[ph.prefixBytesCount-1] = merged
	}
}

func (ph *PrefixHolder) AppendNibble(nibble byte) {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	if nibble > 0x0F {
		panic("Nibble too big")
	}
	if ph.splitNibbleIndex&1 == 1 {
		// In this situation an originally ODD prefix nibble count is the easy case
		byteIndex := ph.prefixBytesCount - 1 // Simply the last prefix byte
		ph.bytes[byteIndex] &= 0xF0          // Clear out any old "ignored" nibble in the bottom four bits
		ph.bytes[byteIndex] |= nibble        // Or in the new nibble
		ph.splitNibbleIndex++
		return // That is all
	} else {
		byteIndex := ph.prefixBytesCount
		ph.bytes[byteIndex] &= 0x0F          // Clear out any old "ignored" nibble in the top four bits
		ph.bytes[byteIndex] |= (nibble << 4) // Or on the new nibble
		ph.splitNibbleIndex++
		// That is not all
		ph.prefixBytesCount++ // Ok so it was easy
		return
	}
}

func (ph *PrefixHolder) CountSupportedHashPrefixValues() uint64 {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	return uint64(1) << (4 * uint64(ph.splitNibbleIndex))
}

func (ph *PrefixHolder) PrefixAsNumber() uint64 {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
	// Special legal edge case... no nibbles
	if ph.splitNibbleIndex == 0 {
		return 0
	}
	result := uint64(0)
	lastNibbleExists := ph.splitNibbleIndex&1 == 1
	// Work FORWARDS through bytes, shifting up result
	for index := range int(ph.prefixBytesCount) {
		byteVal := ph.bytes[index]
		if lastNibbleExists && index == int(ph.prefixBytesCount-1) {
			// Only shift up by 4 bits
			// Last nibble is in top four bits
			result = (result << 4) | uint64(byteVal>>4)
		} else {
			result = (result << 8) | uint64(byteVal)
		}
	}
	return result
}

func (ph *PrefixHolder) SetPrefixFromNumber(number uint64) {
	if !ph.IsValid() {
		panic("Invalid prefix holder")
	}
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
		ph.bytes[byteIndex] &= 0x0F // Clear out any "ignored" nibble that's already there
		ph.bytes[byteIndex] |= (nibble << 4)
		number >>= 4
		byteIndex--
	}
	// Work backwards a byte at a time
	for index := byteIndex; index >= 0; index-- {
		byteVal := byte(number & 0xFF)
		number >>= 8
		ph.bytes[index] = byteVal
	}
	return
}
