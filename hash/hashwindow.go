package hash

const MaxHashBytes = 64

// HashWindow is the underlying mechanism for HashHolder, PrefixHolder, and SuffixHolder
type HashWindow struct {
	bytes         [MaxHashBytes]byte
	hashByteCount byte // Always ignore bytes[hashByteCount:MaxHashBytes]
}

func (hw *HashWindow) AsHashHolder(hashByteCount byte) HashHolder {
	hw.hashByteCount = hashByteCount
	return newHashHolder(hw)
}

func (hw *HashWindow) AsPrefixHolder(hashByteCount byte, splitNibbleIndex byte) PrefixHolder {
	hw.hashByteCount = hashByteCount
	return newPrefixHolder(hw, splitNibbleIndex)
}

func (hw *HashWindow) AsSuffixHolder(hashByteCount byte, splitNibbleIndex byte) SuffixHolder {
	hw.hashByteCount = hashByteCount
	return newSuffixHolder(hw, splitNibbleIndex)
}
