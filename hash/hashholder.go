package hash

import (
	"bytes"
	"io"
)

type HashHolder struct {
	bytes          [MaxHashBytes]byte
	hashBytesCount byte
}

func (hh *HashHolder) Init(hashByteLength byte) {
	hh.hashBytesCount = hashByteLength
}

func (hh *HashHolder) IsValid() bool {
	return hh.hashBytesCount > 0 && hh.hashBytesCount <= MaxHashBytes
}

func (hh *HashHolder) Read(reader io.Reader) error {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	_, err := io.ReadFull(reader, hh.bytes[:hh.hashBytesCount])
	return err
}
func (hh *HashHolder) Write(writer io.Writer) error {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	_, err := writer.Write(hh.bytes[:hh.hashBytesCount])
	return err
}
func (hh *HashHolder) SetFromArray(array *[MaxHashBytes]byte) {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	hh.bytes = *array
}
func (hh *HashHolder) GetToArray(array *[MaxHashBytes]byte) {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	*array = hh.bytes
}
func (hh *HashHolder) Equal(other *HashHolder) bool {
	if hh.hashBytesCount != other.hashBytesCount {
		panic("Cannot compare hashes of different sizes")
	}
	return bytes.Equal(hh.bytes[:hh.hashBytesCount], other.bytes[:other.hashBytesCount])
}
func (hh *HashHolder) ExtractPrefixSuffix(resultPrefix *PrefixHolder, resultSuffix *SuffixHolder, splitNibbleIndex byte) {
	// Prefix
	resultPrefix.Init(hh.hashBytesCount, splitNibbleIndex)
	copy(resultPrefix.bytes[:resultPrefix.prefixBytesCount], hh.bytes[:resultPrefix.prefixBytesCount])

	// Suffix
	resultSuffix.Init(hh.hashBytesCount, splitNibbleIndex)
	copy(resultSuffix.bytes[resultSuffix.suffixBytesStart:hh.hashBytesCount],
		hh.bytes[resultSuffix.suffixBytesStart:hh.hashBytesCount])
}
func (hh HashHolder) HashIsZeroes() bool {
	zeroes := [MaxHashBytes]byte{}
	return bytes.Equal(hh.bytes[:hh.hashBytesCount], zeroes[:hh.hashBytesCount])
}
