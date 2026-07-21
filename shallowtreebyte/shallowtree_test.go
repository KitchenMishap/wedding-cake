package shallowtreebyte

import (
	"fmt"
	"math/rand"
	"slices"
	"testing"
)

const reassuranceBytesCount = 2

func TestEmptyTree(t *testing.T) {
	for prefixNibblesN := NibbleIndex(0); prefixNibblesN <= 4; prefixNibblesN++ {
		for hashLength := NibbleIndex(8); hashLength <= NibbleIndex(128); hashLength += 8 {
			st := GenerateShallowTree(make([]HashPi, 0), prefixNibblesN, hashLength, reassuranceBytesCount, 0)
			if st.LookupHash(helperRandomHash(int(hashLength))) != PiNoMatch {
				t.Error("LookupHash should return PiNoMatch, when looking up in an empty tree")
			}
		}
	}
}

func TestSingleHashPresent(t *testing.T) {
	for prefixNibblesN := NibbleIndex(0); prefixNibblesN <= 4; prefixNibblesN++ {
		for hashLength := NibbleIndex(8); hashLength <= NibbleIndex(128); hashLength += 8 {
			presentationArray := make([]HashPi, 1)
			hash := helperRandomHash(int(hashLength))
			presentationArray[0].Hash = hash
			presentationArray[0].PresentationIndex = 0
			st := GenerateShallowTree(presentationArray, prefixNibblesN, hashLength, reassuranceBytesCount, 0)
			presentationIndex := st.LookupHash(hash)
			if presentationIndex != 0 {
				t.Error("Expected presentationIndex 0")
			}
		}
	}
}

func TestSingleHashAbsent(t *testing.T) {
	for prefixNibblesN := NibbleIndex(0); prefixNibblesN <= 4; prefixNibblesN++ {
		for hashLength := NibbleIndex(8); hashLength <= NibbleIndex(128); hashLength += 8 {
			presentationArray := make([]HashPi, 1)
			hash := helperRandomHash(int(hashLength))
			presentationArray[0].Hash = hash
			presentationArray[0].PresentationIndex = 0
			st := GenerateShallowTree(presentationArray, prefixNibblesN, hashLength, reassuranceBytesCount, 0)
			hash = helperRandomHash(int(hashLength))
			presentationIndex := st.LookupHash(hash)
			if presentationIndex != PiNoMatch {
				t.Error("Expected no match")
			}
		}
	}
}

func Test65535Hashes(t *testing.T) {
	const count = 65535
	const prefixHashBytesCount = NibbleIndex(1)
	const lastByteOfPrefix = NibbleVal(42)

	firstHashLength := NibbleIndex(16) // So that we're "very unlikely" to get a duplicate
	for hashLength := firstHashLength; hashLength <= 128; hashLength += 8 {
		fmt.Printf("Hash size %d...\n", hashLength)
		presentationArray := make([]HashPi, count)
		for i := range count {
			hash := helperRandomHash(int(hashLength))
			presentationArray[i].Hash = hash
			presentationArray[i].PresentationIndex = PiType(i)
		}
		st := GenerateShallowTree(presentationArray, prefixHashBytesCount, hashLength, reassuranceBytesCount, lastByteOfPrefix)
		for i := range count {
			hash := presentationArray[i].Hash
			presentationIndex := st.LookupHash(hash)
			if presentationIndex == PiNoMatch {
				t.Error("Lookup failed, returned SingleTreeNoMatch")
			}
			if !slices.Equal(presentationArray[presentationIndex].Hash, hash) {
				t.Error("Lookup failed, returned index of wrong hash")
			}
		}
		randomHash := helperRandomHash(int(hashLength))
		presentationIndex := st.LookupHash(randomHash)
		if presentationIndex != PiNoMatch {
			fmt.Printf("Hash size %d: Random hash returned a match. Surprising? Yes as we now check the full hash!",
				hashLength)
		}
	}
}

func helperRandomHash(hashLength int) []NibbleVal {
	result := [128]NibbleVal{}
	for i := range hashLength {
		result[i] = NibbleVal(rand.Intn(16))
	}
	return result[:hashLength]
}
