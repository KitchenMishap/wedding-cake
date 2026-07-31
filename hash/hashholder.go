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
func (hh HashHolder) ExtractPrefixSuffix(resultPrefix *PrefixHolder, resultSuffix *SuffixHolder, splitNibbleIndex byte) {

	// Prefix
	resultPrefix.Init(hh.hw.hashByteCount, splitNibbleIndex)
	copy(resultPrefix.bytes[:resultPrefix.prefixBytesCount], hh.hw.bytes[:resultPrefix.prefixBytesCount])

	// Suffix
	resultSuffix.Init(hh.hw.hashByteCount, splitNibbleIndex)
	copy(resultSuffix.bytes[resultSuffix.suffixBytesStart:hh.hw.hashByteCount],
		hh.hw.bytes[resultSuffix.suffixBytesStart:hh.hw.hashByteCount])
}
func (hh HashHolder) HashIsZeroes() bool {
	zeroes := [MaxHashBytes]byte{}
	return bytes.Equal(hh.hw.bytes[:hh.hw.hashByteCount], zeroes[:hh.hw.hashByteCount])
}
