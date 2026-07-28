package smalltree

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"
)

func TestDesignTreeFormat(t *testing.T) {

	const prefixNibbles = 1

	stc := SmallTreeConfig{
		HashNibbleLength:        64,
		ReassuranceBytesCount:   2,
		NodeFormatSpecsPerLevel: 10,
		NodeIdRWriter:           ID16[types.LocalNodeId]{},
		LocalPiRWriter:          ID16[types.LocalPi]{},
	}

	count := 65535
	fmt.Printf("Tree of %d hashes...\n", count)
	presentationArray := make([]shallowtreebyte.HashPi, count)
	for i := range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		presentationArray[i].Hash = hash
		presentationArray[i].PresentationIndex = types.LocalPi(i)
	}
	st := shallowtreebyte.GenerateShallowTree(presentationArray, prefixNibbles, stc.HashNibbleLength, shallowtreebyte.ByteIndex(stc.ReassuranceBytesCount), 0)
	_ = DesignTreeFormat(st, &stc)
}

func helperRandomHash(hashLength int) []shallowtreebyte.NibbleVal {
	result := [128]shallowtreebyte.NibbleVal{}
	for i := range hashLength {
		result[i] = shallowtreebyte.NibbleVal(rand.Intn(16))
	}
	return result[:hashLength]
}
