package forest

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
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
		NodeIdRWriter:           smalltree.ID16[types.LocalNodeId]{},
		LocalPiRWriter:          smalltree.ID16[types.LocalPi]{},
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

	fr := NewForestRead("Temp_Testing", 0, &stc)
	err = fr.Open()
	if err != nil {
		t.Fatal(err)
	}

	for i := range count {
		pi := fr.Lookup(presentationArray[i].Hash)
		if pi == types.LocalPiNoMatch {
			t.Errorf("Hash not found")
		} else if pi != types.LocalPi(i) {
			t.Errorf("Hash mismatch")
		}
	}
	for range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		pi := fr.Lookup(hash)
		if pi != types.LocalPiNoMatch {
			fmt.Printf("Random hash found\n")
		}
	}

	err = fr.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultiTreeForestWrite(t *testing.T) {

	testDir := "Temp_Testing"
	_ = os.RemoveAll(testDir) // Ignore error if it doesn't exist yet
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	folderPath := "Temp_Testing"
	fw := NewForestWrite(folderPath)

	const prefixNibbles = 1

	stc := smalltree.SmallTreeConfig{
		HashNibbleLength:        64,
		ReassuranceBytesCount:   2, // Two will occasionally give "random hash found"
		NodeFormatSpecsPerLevel: 10,
		NodeIdRWriter:           smalltree.ID16[types.LocalNodeId]{},
		LocalPiRWriter:          smalltree.ID24[types.LocalPi]{},
	}

	count := 65535 * 16
	fmt.Printf("Forest of %d hashes...\n", count)
	presentationArray := make([]shallowtreebyte.HashPi, count)
	filteredArray := make([][]shallowtreebyte.HashPi, 16)
	for i := 0; i < 16; i++ {
		filteredArray[i] = make([]shallowtreebyte.HashPi, 0)
	}
	for i := range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		presentationArray[i].Hash = hash
		presentationArray[i].PresentationIndex = types.LocalPi(i)

		firstNibble := hash[0]
		filteredArray[firstNibble] = append(filteredArray[firstNibble], presentationArray[i])
	}

	lcf := smalltree.NewLevelsCodecNfFactory(&stc)
	le := lcf.MakeLevelsEncoder()

	err = fw.StartWrite()
	if err != nil {
		t.Fatal(err)
	}
	for nibble := 0; nibble < 16; nibble++ {
		st := shallowtreebyte.GenerateShallowTree(filteredArray[nibble], prefixNibbles, stc.HashNibbleLength, shallowtreebyte.ByteIndex(stc.ReassuranceBytesCount), 0)
		tf := smalltree.DesignTreeFormat(st, &stc)

		indexBytes, nodesBytes, rootNodeId, rootLevel := le.EncodeSubTree(st, tf)

		err = fw.AppendTreeForPrefix(PrefixIndexType(nibble), indexBytes, nodesBytes, rootNodeId, rootLevel)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = fw.EndWrite()
	if err != nil {
		t.Fatal(err)
	}

	fr := NewForestRead("Temp_Testing", prefixNibbles, &stc)
	err = fr.Open()
	if err != nil {
		t.Fatal(err)
	}

	for i := range count {
		pi := fr.Lookup(presentationArray[i].Hash)
		if pi == types.LocalPiNoMatch {
			t.Errorf("Hash not found")
		} else if pi != types.LocalPi(i) {
			t.Errorf("Hash mismatch")
		}
	}
	for range count {
		hash := helperRandomHash(int(stc.HashNibbleLength))
		pi := fr.Lookup(hash)
		if pi != types.LocalPiNoMatch {
			fmt.Printf("Random hash found\n")
		}
	}

	err = fr.Close()
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
