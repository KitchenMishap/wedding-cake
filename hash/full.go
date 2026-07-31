package hash

import (
	"bytes"
	"io"
)

const MaxHashBytes = 64

type Full struct {
	bytes          [MaxHashBytes]byte
	hashBytesCount byte
}

func (hh *Full) Init(hashByteLength byte) {
	hh.hashBytesCount = hashByteLength
}

func (hh *Full) IsValid() bool {
	return hh.hashBytesCount > 0 && hh.hashBytesCount <= MaxHashBytes
}

func (hh *Full) Read(reader io.Reader) error {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	_, err := io.ReadFull(reader, hh.bytes[:hh.hashBytesCount])
	return err
}
func (hh *Full) Write(writer io.Writer) error {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	_, err := writer.Write(hh.bytes[:hh.hashBytesCount])
	return err
}
func (hh *Full) SetFromArray(array *[MaxHashBytes]byte) {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	hh.bytes = *array
}
func (hh *Full) GetToArray(array *[MaxHashBytes]byte) {
	if !hh.IsValid() {
		panic("HashHolder not valid")
	}
	*array = hh.bytes
}
func (hh *Full) Equal(other *Full) bool {
	if hh.hashBytesCount != other.hashBytesCount {
		panic("Cannot compare hashes of different sizes")
	}
	return bytes.Equal(hh.bytes[:hh.hashBytesCount], other.bytes[:other.hashBytesCount])
}
func (hh *Full) ExtractPrefixSuffix(resultPrefix *Prefix, resultSuffix *Suffix, splitNibbleIndex byte) {
	// Prefix
	resultPrefix.Init(hh.hashBytesCount, splitNibbleIndex)
	copy(resultPrefix.bytes[:resultPrefix.prefixBytesCount], hh.bytes[:resultPrefix.prefixBytesCount])

	// Suffix
	resultSuffix.Init(hh.hashBytesCount, splitNibbleIndex)
	copy(resultSuffix.bytes[resultSuffix.suffixBytesStart:hh.hashBytesCount],
		hh.bytes[resultSuffix.suffixBytesStart:hh.hashBytesCount])
}
func (hh Full) HashIsZeroes() bool {
	zeroes := [MaxHashBytes]byte{}
	return bytes.Equal(hh.bytes[:hh.hashBytesCount], zeroes[:hh.hashBytesCount])
}
