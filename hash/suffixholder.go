package hash

import "io"

type SuffixHolder struct {
	hw               *HashWindow
	splitNibbleIndex byte // Specified
	suffixBytesStart byte // Calculated
	suffixBytesCount byte // Calculated
}

// Slight whiff #EGGY needs to work even if hw.bytes have not yet been copied in!
func newSuffixHolder(hw *HashWindow, splitNibbleIndex byte) SuffixHolder {
	totalNibblesCount := hw.hashByteCount * 2
	prefixNibblesCount := splitNibbleIndex
	suffixNibblesCount := totalNibblesCount - prefixNibblesCount
	suffixBytesCount := suffixNibblesCount / 2
	if suffixNibblesCount&1 == 1 {
		suffixBytesCount++
	}
	suffixBytesStart := hw.hashByteCount - suffixBytesCount
	return SuffixHolder{hw: hw, splitNibbleIndex: splitNibbleIndex,
		suffixBytesStart: suffixBytesStart, suffixBytesCount: suffixBytesCount}
}

func (sh SuffixHolder) Read(reader io.Reader) error {
	_, err := io.ReadFull(reader, sh.hw.bytes[sh.suffixBytesStart:sh.hw.hashByteCount])
	return err
}
func (sh SuffixHolder) LastReadContainedSpareNibble() (bool, byte) {
	// If we wrote an even number of nibbles, there is none
	if sh.splitNibbleIndex&1 == 0 {
		return false, 0
	}
	// The spare nibble is in the most significant nibble of the first byte read
	return true, (sh.hw.bytes[sh.suffixBytesStart] & 0xF0) >> 4
}
func (sh SuffixHolder) Write(writer io.Writer, spareNibble byte) error {
	// If we are writing an odd number of nibbles, there is room for a spare nibble
	if sh.splitNibbleIndex&1 == 1 {
		if spareNibble > 0x0F {
			panic("Nibble too big")
		}
		// Don't worry, we are storing the spare nibble somewhere that "should not" be considered
		// "part of the suffix". This is in the most significant nibble of the first byte written,
		// therefore "one nibble before" the actual suffix.
		sh.hw.bytes[sh.suffixBytesStart] &= 0x0F // Clear off any old "ignored" nibble in the top four bits
		sh.hw.bytes[sh.suffixBytesStart] |= (spareNibble << 4)
	}
	_, err := writer.Write(sh.hw.bytes[sh.suffixBytesStart:sh.hw.hashByteCount])
	return err
}

func (sh SuffixHolder) RemoveFirstNibble() byte {
	// "Easy" mode is when we originally have an even number pf nibbles
	if sh.splitNibbleIndex&1 == 0 {
		// It's in the most significant nibble of the first byte of the suffix
		result := sh.hw.bytes[sh.suffixBytesStart] >> 4
		// We don't need to physically clear it, just specify that it is to be ignored
		sh.splitNibbleIndex++ // More prefix means less suffix
		return result         // And that's all
	} else {
		// It's in the least significant nibble of the first byte of the suffix
		result := sh.hw.bytes[sh.suffixBytesStart] & 0x0F
		sh.splitNibbleIndex++ // More prefix, less suffix
		sh.suffixBytesCount-- // One less byte (we crossed a byte boundary)
		sh.suffixBytesStart++ // The suffix starts one byte later
		return result
	}
}
