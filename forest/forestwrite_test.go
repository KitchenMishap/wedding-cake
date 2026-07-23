package forest

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
)

func TestSingleTreeForestWrite(t *testing.T) {

	testDir := filepath.Join("Temp_Testing")
	_ = os.RemoveAll(testDir) // Ignore error if it doesn't exist yet
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	const prefixNibbles = 0

	stc := smalltree.SmallTreeConfig{
		HashNibbleLength:        64,
		ReassuranceBytesCount:   2, // Two will occasionally give "random hash found"
		NodeFormatSpecsPerLevel: 10,
		NodeIdConfig:            smalltree.ID16[smalltree.LocalNodeIdType]{},
		HashIndexIdConfig:       smalltree.ID16[smalltree.HashIndexIdType]{},
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
	tf := smalltree.DesignTreeFormat(st, &stc)

	lcf := smalltree.NewLevelsCodecNfFactory(&stc)
	le := lcf.MakeLevelsEncoder()
	indexBytes, nodesBytes, rootNodeId, rootLevel := le.EncodeSubTree(st, tf)

	folderPath := "Temp_Testing"
	fw := NewForestWrite(folderPath)
	err = fw.StartWrite()
	if err != nil {
		t.Fatal(err)
	}
	err = fw.AppendTreeForPrefix(0, indexBytes, nodesBytes, rootNodeId, rootLevel)
	if err != nil {
		t.Fatal(err)
	}
	err = fw.EndWrite()
	if err != nil {
		t.Fatal(err)
	}
}

func helperRandomHash(hashLength int) []shallowtreebyte.NibbleVal {
	result := [128]shallowtreebyte.NibbleVal{}
	for i := range hashLength {
		result[i] = shallowtreebyte.NibbleVal(rand.Intn(16))
	}
	return result[:hashLength]
}
