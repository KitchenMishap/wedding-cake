package hash

import (
	"bytes"
	"io"
)

type HashHolder struct {
	hw *HashWindow
}

func newHashHolder(hw *HashWindow) HashHolder {
	return HashHolder{hw: hw}
}

func (hh HashHolder) Read(reader io.Reader) error {
	_, err := io.ReadFull(reader, hh.hw.bytes[:hh.hw.hashByteCount])
	return err
}
func (hh HashHolder) Write(writer io.Writer) error {
	_, err := writer.Write(hh.hw.bytes[:hh.hw.hashByteCount])
	return err
}
func (hh HashHolder) SetFromArray(array *[MaxHashBytes]byte) {
	hh.hw.bytes = *array
}
func (hh HashHolder) GetToArray(array *[MaxHashBytes]byte) {
	*array = hh.hw.bytes
}
func (hh HashHolder) Equal(other HashHolder) bool {
	if hh.hw.hashByteCount != other.hw.hashByteCount {
		panic("Cannot compare hashes of different sizes")
	}
	return bytes.Equal(hh.hw.bytes[:hh.hw.hashByteCount], other.hw.bytes[:other.hw.hashByteCount])
}
func (hh HashHolder) ExtractPrefixSuffix(target2 *HashWindow,
	splitNibbleIndex byte) (PrefixHolder, SuffixHolder) {

	// Prefix
	prefix := newPrefixHolder(hh.hw.hashByteCount, splitNibbleIndex)
	copy(prefix.bytes[:prefix.prefixBytesCount], hh.hw.bytes[:prefix.prefixBytesCount])

	// Suffix
	target2.hashByteCount = hh.hw.hashByteCount
	suffix := newSuffixHolder(target2, splitNibbleIndex) // #EGGY as above
	copy(target2.bytes[suffix.suffixBytesStart:hh.hw.hashByteCount], hh.hw.bytes[suffix.suffixBytesStart:hh.hw.hashByteCount])

	return prefix, suffix
}
func (hh HashHolder) HashIsZeroes() bool {
	zeroes := [MaxHashBytes]byte{}
	return bytes.Equal(hh.hw.bytes[:hh.hw.hashByteCount], zeroes[:hh.hw.hashByteCount])
}
