package cake

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/kitchenmishap/wedding-cake/forest"
	"github.com/kitchenmishap/wedding-cake/hash"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/tierbake"
	"github.com/kitchenmishap/wedding-cake/types"
)

type Tier struct {
	folderPath string
	tierIndex  byte

	config *smalltree.SmallTreeConfig

	donuts       []*forest.ForestRead
	donutOffsets []types.PiOffset
}

func (t *Tier) LookupHash(theHash []shallowtreebyte.NibbleVal) (types.GlobalPi, error) {
	for d := range t.donuts {
		res := t.donuts[d].Lookup(theHash)
		if res != types.LocalPiNoMatch {
			verified, err := t.VerifyHash(theHash, res, d)
			if err != nil {
				return 0, err
			}
			if verified {
				return res.ToGlobalPi(t.donutOffsets[d]), nil
			} else {
				fmt.Printf("Verification failed (non fatal)\n")
			}
		}
	}
	return types.GlobalPresentationIndexNoMatch, nil
}

func (t *Tier) VerifyHash(theHash []shallowtreebyte.NibbleVal, candidatePi types.LocalPi, donutIndex int) (bool, error) {
	tierIndex := t.tierIndex
	prefixNibblesCount := tierIndex
	prefix := theHash[:prefixNibblesCount]

	// First look in HashesOrder.bin for an index into the other file
	hashOrderPath := filepath.Join(t.folderPath, fmt.Sprintf("Donut%X", donutIndex), "HashesOrder.bin")
	file1, err := os.Open(hashOrderPath)
	defer func() { _ = file1.Close() }()
	if err != nil {
		return false, err
	}

	prefixBytesCount := prefixNibblesCount / 2
	if prefixNibblesCount&1 == 1 {
		prefixBytesCount++
	}

	// Each entry in HashesOrder.bin is a pair of numbers: prefixIndex, suffixIndex
	// The number of bytes to encode each varies by tier
	prefixIndexBytesCount := int(prefixBytesCount)
	suffixIndexBytesCount := t.config.SuffixIndexRWriter.StorageBytes()
	entrySizeOrder := prefixIndexBytesCount + suffixIndexBytesCount
	_, err = file1.Seek(int64(candidatePi)*int64(entrySizeOrder), 0)
	if err != nil {
		return false, err
	}

	// First we read the prefixIndex
	prefixObj := hash.Prefix{}
	prefixObj.Init(byte(len(theHash)/2), prefixNibblesCount)
	err = prefixObj.Read(file1)
	if err != nil {
		return false, err
	}
	prefixIndex := prefixObj.PrefixAsNumber()

	// Then the suffixIndex
	const spareBytes = 8
	bytesSpare := [spareBytes]byte{}
	_, err = io.ReadFull(file1, bytesSpare[:suffixIndexBytesCount])
	if err != nil {
		return false, err
	}
	suffixIndex := t.config.SuffixIndexRWriter.ReadID(bytesSpare[:])

	// Turn prefixIndex into a slice of nibbles
	prefixProposed := [128]shallowtreebyte.NibbleVal{}
	prefixIndexShift := prefixIndex
	for i := 0; i < int(prefixNibblesCount); i++ {
		// Do it from the LS nibble end first
		nibbleIndex := int(prefixNibblesCount) - 1 - i
		nibble := prefixIndexShift & 0x0F
		prefixProposed[nibbleIndex] = shallowtreebyte.NibbleVal(nibble)
		prefixIndexShift >>= 4
	}

	// We can take an "easy out" if the prefix doesn't match (saves a file read)
	if !slices.Equal(prefixProposed[:prefixNibblesCount], prefix) {
		return false, nil
	}

	prefixObj.Init(byte(len(theHash)/2), prefixNibblesCount)
	prefixObj.SetPrefixFromNumber(uint64(prefixIndex))

	// Now look in Suffix file
	fileDigits, foldersDigits := tierbake.CalculatePrefixPattern(prefixNibblesCount, 2)
	filenamePrefix, folder := tierbake.FormatFilePathFilename(prefixProposed[:prefixNibblesCount], fileDigits, foldersDigits)

	suffixPath := filepath.Join(t.folderPath, fmt.Sprintf("Donut%X", donutIndex), "HashPrefix", folder, filenamePrefix+"HashSuffix.bin")
	file2, err := os.Open(suffixPath)
	defer func() { _ = file2.Close() }()
	if err != nil {
		return false, err
	}
	byteCount := t.config.LocalPiRWriter.StorageBytes()
	suffixNibbleCount := len(theHash) - int(prefixNibblesCount)
	suffixHashBytesCount := suffixNibbleCount / 2
	if suffixNibbleCount&1 != 0 {
		suffixHashBytesCount++
	}
	entrySizeSuffix := suffixHashBytesCount + byteCount
	_, err = file2.Seek(int64(suffixIndex)*int64(entrySizeSuffix), 0)
	if err != nil {
		return false, err
	}

	// Read suffix
	suffixObj := hash.Suffix{}
	suffixObj.Init(byte(len(theHash)/2), prefixNibblesCount)
	err = suffixObj.Read(file2)
	if err != nil {
		return false, err
	}

	// (We won't bother to read the localPi for now)

	hashObj := hash.Full{}
	prefixObj.AppendSuffix(&hashObj, &suffixObj)

	array := [hash.MaxHashBytes]byte{}
	hashObj.GetToArray(&array)

	// start of array should now be equivalent to hash
	for hashNibbleIndex := 0; hashNibbleIndex < len(theHash); hashNibbleIndex++ {
		byteVal := array[hashNibbleIndex/2]
		var nibble byte
		if hashNibbleIndex&1 == 0 {
			// Most significant nibble
			nibble = byteVal >> 4
		} else {
			// Least significant nibble
			nibble = byteVal & 0x0F
		}
		if shallowtreebyte.NibbleVal(nibble) != theHash[hashNibbleIndex] {
			return false, nil
		}
	}
	return true, nil
}

func (t *Tier) Close() error {
	for d := range t.donuts {
		err := t.donuts[d].Close()
		if err != nil {
			return err
		}
		t.donuts[d] = nil
	}
	return nil
}
