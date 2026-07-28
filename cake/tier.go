package cake

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/kitchenmishap/wedding-cake/forest"
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

func (t *Tier) LookupHash(hash []shallowtreebyte.NibbleVal) (types.GlobalPi, error) {
	for d := range t.donuts {
		res := t.donuts[d].Lookup(hash)
		if res != types.LocalPiNoMatch {
			verified, err := t.VerifyHash(hash, res, d)
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

func (t *Tier) VerifyHash(hash []shallowtreebyte.NibbleVal, candidatePi types.LocalPi, donutIndex int) (bool, error) {
	tierIndex := t.tierIndex
	prefixNibblesCount := tierIndex
	prefix := hash[:prefixNibblesCount]

	// First look in HashesOrder.bin for an index into the other file
	hashOrderPath := filepath.Join(t.folderPath, fmt.Sprintf("Donut%X", donutIndex), "HashesOrder.bin")
	file1, err := os.Open(hashOrderPath)
	defer func() { _ = file1.Close() }()
	if err != nil {
		return false, err
	}

	// Each entry in HashesOrder.bin is a pair of numbers: prefixIndex, suffixIndex
	// The number of bytes to encode each varies by tier
	prefixIndexBytesCount := t.config.PrefixIndexRWriter.StorageBytes()
	suffixIndexBytesCount := t.config.SuffixIndexRWriter.StorageBytes()
	entrySizeOrder := prefixIndexBytesCount + suffixIndexBytesCount
	_, err = file1.Seek(int64(candidatePi)*int64(entrySizeOrder), 0)
	if err != nil {
		return false, err
	}
	const spareBytes = 8 + 8
	bytesSpare := [spareBytes]byte{}
	_, err = file1.Read(bytesSpare[:entrySizeOrder])
	if err != nil {
		return false, err
	}
	prefixIndex := t.config.PrefixIndexRWriter.ReadID(bytesSpare[:])
	suffixIndex := t.config.SuffixIndexRWriter.ReadID(bytesSpare[prefixIndexBytesCount:])

	// Turn prefixIndex into a slice of nibbles
	prefixProposed := [128]shallowtreebyte.NibbleVal{}
	for i := 0; i < int(prefixNibblesCount); i++ {
		// Do it from the LS nibble end first
		nibbleIndex := int(prefixNibblesCount) - 1 - i
		nibble := prefixIndex & 0x0F
		prefixProposed[nibbleIndex] = shallowtreebyte.NibbleVal(nibble)
		prefixIndex >>= 4
	}

	// We can take an "easy out" if the prefix doesn't match (saves a file read)
	if !slices.Equal(prefixProposed[:prefixNibblesCount], prefix) {
		return false, nil
	}

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
	suffixNibbleCount := len(hash) - int(prefixNibblesCount)
	suffixHashBytesCount := suffixNibbleCount / 2
	if suffixNibbleCount&1 != 0 {
		suffixHashBytesCount++
	}
	entrySizeSuffix := suffixHashBytesCount + byteCount
	_, err = file2.Seek(int64(suffixIndex)*int64(entrySizeSuffix), 0)
	if err != nil {
		return false, err
	}
	const spareBytes2 = 64 + 8
	bytes := [spareBytes2]byte{}
	_, err = file2.Read(bytes[:entrySizeSuffix])
	if err != nil {
		return false, err
	}

	// bytes should now be equivalent to hash
	for suffixNibbleIndex := 0; suffixNibbleIndex < suffixNibbleCount; suffixNibbleIndex++ {
		byteVal := bytes[suffixNibbleIndex/2]
		var nibble byte
		if suffixNibbleIndex&1 == 0 {
			// Most significant nibble
			nibble = byteVal >> 4
		} else {
			// Least significant nibble
			nibble = byteVal & 0x0F
		}
		if shallowtreebyte.NibbleVal(nibble) != hash[int(prefixNibblesCount)+suffixNibbleIndex] {
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
