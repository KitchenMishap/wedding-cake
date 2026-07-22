package smalltree

import (
	"fmt"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

func TestCodecNf(t *testing.T) {

	const prefixNibbles = 2

	stc := SmallTreeConfig{
		HashNibbleLength:        64,
		ReassuranceBytesCount:   2, // Two will occasionally give "random hash found"
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
	indexBytes, nodesBytes, rootNodeId, rootLevel := le.EncodeSubTree(st, tf)

	dlt := lcf.MakeDecoderLevelsTest(indexBytes, nodesBytes, prefixNibbles, rootNodeId, rootLevel)

	for i := range count {
		pi := dlt.Lookup(presentationArray[i].Hash)
		if pi == HashIndexIdNoMatch {
			t.Errorf("Hash not found")
		} else if pi != HashIndexIdType(i) {
			t.Errorf("Hash mismatch")
		}
	}
	for _ = range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		pi := dlt.Lookup(hash)
		if pi != HashIndexIdNoMatch {
			fmt.Printf("Random hash found\n")
		}
	}
}
