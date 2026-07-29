package tierbake

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

func ForwardLookup(pi types.LocalPi, prefixNibblesCount shallowtreebyte.NibbleIndex,
	donutPath string, donutConfig *smalltree.SmallTreeConfig) ([]shallowtreebyte.NibbleVal, error) {

	hashesOrderFname := filepath.Join(donutPath, "HashesOrder.bin")
	hashesOrderFile, err := os.Open(hashesOrderFname)
	defer func() { _ = hashesOrderFile.Close() }()
	if err != nil {
		return nil, err
	}

	prefixSize := donutConfig.PrefixIndexRWriter.StorageBytes()
	suffixSize := donutConfig.SuffixIndexRWriter.StorageBytes()
	entrySize := int64(prefixSize + suffixSize)

	info, err := hashesOrderFile.Stat()
	if err != nil {
		return nil, err
	}
	overflow := info.Size() % entrySize
	if overflow != 0 {
		panic("Wrong size for HashesOrder.bin")
	}

	entry := make([]byte, entrySize)
	_, err = hashesOrderFile.Seek(int64(pi)*entrySize, 0)
	if err != nil {
		return nil, err
	}
	_, err = hashesOrderFile.Read(entry)
	if err != nil {
		return nil, err
	}

	prefixIndex := donutConfig.PrefixIndexRWriter.ReadID(entry[:])
	suffixIndex := donutConfig.SuffixIndexRWriter.ReadID(entry[prefixSize:])

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
		return nil, err
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
		return nil, err
	}
	suffixEntry := make([]byte, suffixEntrySize)
	_, err = suffixFile.Read(suffixEntry)
	if err != nil {
		return nil, err
	}

	piResult := donutConfig.LocalPiRWriter.ReadID(suffixEntry[suffixByteSize:])

	suffix := make([]shallowtreebyte.NibbleVal, suffixNibblesSize)
	for nibbleIndex := shallowtreebyte.NibbleIndex(0); nibbleIndex < suffixNibblesSize; nibbleIndex++ {
		byteVal := suffixEntry[nibbleIndex/2]
		nibbleVal := shallowtreebyte.NibbleVal(byteVal & 0x0F) // LS nibble
		if nibbleIndex&1 == 0 {
			nibbleVal = shallowtreebyte.NibbleVal(byteVal >> 4) // MS nibble
		}
		suffix[nibbleIndex] = nibbleVal
	}

	hash := append(prefix, suffix...)

	if piResult == pi {
		fmt.Printf("Pi match in xxxHashSuffix.bin file!\n")
	} else {
		panic("Pi should match in xxxHashSuffix.bin file")
	}

	return hash, nil
}
