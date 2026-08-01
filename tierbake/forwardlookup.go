package tierbake

import (
	"io"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/hash"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

func ForwardLookup(pi types.LocalPi, prefixNibblesCount shallowtreebyte.NibbleIndex,
	donutPath string, donutConfig *smalltree.SmallTreeConfig,
	result *hash.Full) error {

	hashesOrderFname := filepath.Join(donutPath, "HashesOrder.bin")
	hashesOrderFile, err := os.Open(hashesOrderFname)
	defer func() { _ = hashesOrderFile.Close() }()
	if err != nil {
		return err
	}

	prefixBytesCount := prefixNibblesCount / 2
	if prefixNibblesCount&1 == 1 {
		prefixBytesCount++
	}
	prefixSize := int(prefixBytesCount)
	suffixSize := donutConfig.SuffixIndexRWriter.StorageBytes()
	entrySize := int64(prefixSize + suffixSize)

	info, err := hashesOrderFile.Stat()
	if err != nil {
		return err
	}
	overflow := info.Size() % entrySize
	if overflow != 0 {
		panic("Wrong size for HashesOrder.bin")
	}

	prefixObj := hash.Prefix{}
	prefixObj.Init(byte(donutConfig.HashNibbleLength/2), byte(prefixNibblesCount))

	_, err = hashesOrderFile.Seek(int64(pi)*entrySize, 0)
	if err != nil {
		return err
	}
	// First we read the prefix index
	err = prefixObj.Read(hashesOrderFile)
	if err != nil {
		return err
	}
	prefixIndex := prefixObj.PrefixAsNumber()
	// Then we read the suffix index
	const spareBytes = 8
	suffixIndexBytes := [spareBytes]byte{}
	suffixIndexBytesCount := donutConfig.SuffixIndexRWriter.StorageBytes()
	_, err = hashesOrderFile.Read(suffixIndexBytes[:suffixIndexBytesCount])
	if err != nil {
		return err
	}
	suffixIndex := donutConfig.SuffixIndexRWriter.ReadID(suffixIndexBytes[:])

	prefix := make([]shallowtreebyte.NibbleVal, prefixNibblesCount)
	// Work from LS nibble to MS nibble
	prefixIndexCopy := prefixIndex
	for nibbleIndex := int(prefixNibblesCount) - 1; nibbleIndex >= 0; nibbleIndex-- {
		prefix[nibbleIndex] = shallowtreebyte.NibbleVal(prefixIndexCopy & 0xF)
		prefixIndexCopy >>= 4
	}

	suffixFilenameDigits, suffixFoldersDigits := CalculatePrefixPattern(byte(prefixNibblesCount), 2)
	suffixFilenamePrefix, suffixFoldersPath := FormatFilePathFilename(prefix, suffixFilenameDigits, suffixFoldersDigits)
	suffixFolderPath := filepath.Join(donutPath, "HashPrefix", suffixFoldersPath)
	suffixFilePath := filepath.Join(suffixFolderPath, suffixFilenamePrefix+"HashSuffix.bin")

	suffixFile, err := os.Open(suffixFilePath)
	if err != nil {
		return err
	}
	defer func() { _ = suffixFile.Close() }()

	suffixNibblesSize := donutConfig.HashNibbleLength - prefixNibblesCount
	suffixByteSize := suffixNibblesSize / 2
	if suffixNibblesSize&1 != 0 {
		suffixByteSize++
	}
	localPiByteSize := donutConfig.LocalPiRWriter.StorageBytes()
	suffixEntrySize := int64(suffixByteSize) + int64(localPiByteSize)

	suffixInfo, err := suffixFile.Stat()
	overflow = suffixInfo.Size() % suffixEntrySize
	if overflow != 0 {
		panic("Wrong size xxxHashSuffix.bin file")
	}

	_, err = suffixFile.Seek(int64(suffixIndex)*suffixEntrySize, io.SeekStart)
	if err != nil {
		return err
	}

	// Read suffix
	suffixObj := hash.Suffix{}
	suffixObj.Init(byte(donutConfig.HashNibbleLength/2), byte(prefixNibblesCount))
	err = suffixObj.Read(suffixFile)
	if err != nil {
		return err
	}

	// Read localPi
	someBytes := [spareBytes]byte{}
	_, err = suffixFile.Read(someBytes[:donutConfig.LocalPiRWriter.StorageBytes()])
	if err != nil {
		return err
	}
	piResult := donutConfig.LocalPiRWriter.ReadID(someBytes[:])

	prefixObjB := hash.Prefix{}
	prefixObjB.Init(byte(donutConfig.HashNibbleLength/2), byte(prefixNibblesCount))
	prefixObjB.SetPrefixFromNumber(uint64(prefixIndex))
	prefixObjB.AppendSuffix(result, &suffixObj)

	if piResult == pi {
	} else {
		panic("Pi should match in xxxHashSuffix.bin file")
	}

	return nil
}
