package hash

const MaxHashBytes = 64

// HashWindow is the underlying mechanism for HashHolder, PrefixHolder, and SuffixHolder
type HashWindow struct {
	bytes         [MaxHashBytes]byte
	hashByteCount byte // Always ignore bytes[hashByteCount:MaxHashBytes]
}
