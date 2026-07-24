package shallowtreebyte

import (
	"testing"

	"github.com/kitchenmishap/wedding-cake/types"
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
		presentationArray[i].PresentationIndex = types.LocalPi(i)
	}
	st := GenerateShallowTree(presentationArray, prefixHashBytesCount, hashNibblesLength, reassuranceBytesCount, lastByteOfPrefix)
	ts := st.CountLevelShapes()
	ts.Print()
}
