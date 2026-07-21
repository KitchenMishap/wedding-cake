package smalltree

import (
	"fmt"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

func TestCodecNf(t *testing.T) {

	const prefixNibbles = 1

	stc := SmallTreeConfig{
		HashNibbleLength:        64,
		ReassuranceBytesCount:   2,
		NodeFormatSpecsPerLevel: 10,
		NodeIdConfig:            ID16[LocalNodeIdType]{},
		HashIndexIdConfig:       ID16[HashIndexIdType]{},
	}

	count := 65535
	fmt.Printf("Tree of %d hashes...\n", count)
	presentationArray := make([]shallowtreebyte.HashPi, count)
	for i := range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		presentationArray[i].Hash = hash
		presentationArray[i].PresentationIndex = shallowtreebyte.PiType(i)
	}
	st := shallowtreebyte.GenerateShallowTree(presentationArray, prefixNibbles, stc.HashNibbleLength, shallowtreebyte.NibbleIndex(stc.ReassuranceBytesCount*2), 0)
	tf := DesignTreeFormat(st, &stc)

	lcf := NewLevelsCodecNfFactory(&stc)
	le := lcf.MakeLevelsEncoder()
	le.EncodeSubTree(st, tf)
}
