package hash

import (
	"io"
)

type SuffixHolder struct {
	bytes            [MaxHashBytes]byte
	hashBytesCount   byte // Specified
	splitNibbleIndex byte // Specified
	suffixBytesStart byte // Calculated
	suffixBytesCount byte // Calculated
}

// Slight whiff #EGGY needs to work even if hw.bytes have not yet been copied in!
func (sh *SuffixHolder) Init(hashBytesCount byte, splitNibbleIndex byte) {
	// Specify
	sh.hashBytesCount = hashBytesCount
	sh.splitNibbleIndex = splitNibbleIndex
	// Calculate
	totalNibblesCount := hashBytesCount * 2
	prefixNibblesCount := splitNibbleIndex
	suffixNibblesCount := totalNibblesCount - prefixNibblesCount
	sh.suffixBytesCount = suffixNibblesCount / 2
	if suffixNibblesCount&1 == 1 {
		sh.suffixBytesCount++
	}
	sh.suffixBytesStart = hashBytesCount - sh.suffixBytesCount
}

func (sh *SuffixHolder) IsValid() bool {
	return sh.hashBytesCount > 0 && sh.hashBytesCount <= MaxHashBytes
}

func (sh *SuffixHolder) Read(reader io.Reader) error {
	if !sh.IsValid() {
		panic("Invalid suffix holder")
	}
	_, err := io.ReadFull(reader, sh.bytes[sh.suffixBytesStart:sh.hashBytesCount])
	return err
}
func (sh *SuffixHolder) LastReadContainedSpareNibble() (bool, byte) {
	if !sh.IsValid() {
		panic("Invalid suffix holder")
	}
	// If we wrote an even number of nibbles, there is none
	if sh.splitNibbleIndex&1 == 0 {
		return false, 0
	}
	// The spare nibble is in the most significant nibble of the first byte read
	return true, (sh.bytes[sh.suffixBytesStart] & 0xF0) >> 4
}
func (sh *SuffixHolder) Write(writer io.Writer, spareNibble byte) error {
	if !sh.IsValid() {
		panic("Invalid suffix holder")
	}

	// If we are writing an odd number of nibbles, there is room for a spare nibble
	if sh.splitNibbleIndex&1 == 1 {
		if spareNibble > 0x0F {
			panic("Nibble too big")
		}
		// Don't worry, we are storing the spare nibble somewhere that "should not" be considered
		// "part of the suffix". This is in the most significant nibble of the first byte written,
		// therefore "one nibble before" the actual suffix.
		sh.bytes[sh.suffixBytesStart] &= 0x0F // Clear off any old "ignored" nibble in the top four bits
		sh.bytes[sh.suffixBytesStart] |= (spareNibble << 4)
	}
	_, err := writer.Write(sh.bytes[sh.suffixBytesStart:sh.hashBytesCount])
	return err
}

func (sh *SuffixHolder) RemoveFirstNibble() byte {
	if !sh.IsValid() {
		panic("Invalid suffix holder")
	}
	// "Easy" mode is when we originally have an even number pf nibbles
	if sh.splitNibbleIndex&1 == 0 {
		// It's in the most significant nibble of the first byte of the suffix
		result := sh.bytes[sh.suffixBytesStart] >> 4
		// We don't need to physically clear it, just specify that it is to be ignored
		sh.splitNibbleIndex++ // More prefix means less suffix
		return result         // And that's all
	} else {
		// It's in the least significant nibble of the first byte of the suffix
		result := sh.bytes[sh.suffixBytesStart] & 0x0F
		sh.splitNibbleIndex++ // More prefix, less suffix
		sh.suffixBytesCount-- // One less byte (we crossed a byte boundary)
		sh.suffixBytesStart++ // The suffix starts one byte later
		return result
	}
}
