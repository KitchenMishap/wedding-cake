package smalltree

import (
	"fmt"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"
)

func TestCodecNf(t *testing.T) {

	const prefixNibbles = 2

	stc := SmallTreeConfig{
		HashNibbleLength:        64,
		ReassuranceBytesCount:   2, // Two will occasionally give "random hash found"
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
	tf := DesignTreeFormat(st, &stc)

	lcf := NewLevelsCodecNfFactory(&stc)
	le := lcf.MakeLevelsEncoder()
	indexBytes, nodesBytes, rootNodeId, rootLevel := le.EncodeSubTree(st, tf)

	dlt := lcf.MakeDecoderLevelsTest(indexBytes, nodesBytes, prefixNibbles, rootNodeId, rootLevel)

	for i := range count {
		pi := dlt.Lookup(presentationArray[i].Hash)
		if pi == types.LocalPiNoMatch {
			t.Errorf("Hash not found")
		} else if pi != types.LocalPi(i) {
			t.Errorf("Hash mismatch")
		}
	}
	for range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		pi := dlt.Lookup(hash)
		if pi != types.LocalPiNoMatch {
			fmt.Printf("Random hash found\n")
		}
	}
}
