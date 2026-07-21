package shallowtreebyte

import (
	"testing"
)

func TestTreeShape(t *testing.T) {
	const count = 65535
	const prefixHashBytesCount = NibbleIndex(1)
	const lastByteOfPrefix = NibbleVal(42)
	const hashNibblesLength = 64 // 64 nibbles in sha256

	presentationArray := make([]HashPi, count)
	for i := range count {
		hash := helperRandomHash(int(hashNibblesLength))
		presentationArray[i].Hash = hash
		presentationArray[i].PresentationIndex = PiType(i)
	}
	st := GenerateShallowTree(presentationArray, prefixHashBytesCount, hashNibblesLength, reassuranceBytesCount, lastByteOfPrefix)
	ts := st.CountLevelShapes()
	ts.Print()
}
