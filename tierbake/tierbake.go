package tierbake

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

func BakeFrozenTierToDonutFolder(numberedFolderPath string, newDonutFolder string,
	sourceTierIndex byte,
	sourceLocalPiWriter smalltree.NByteIdConfig[types.LocalPi], destLocalPiWriter smalltree.NByteIdConfig[types.LocalPi],
	hashBytesCount byte, donutOffsets []types.PiOffset,
	newDonutOffset types.PiOffset) error {

	sourceDonutsCount := len(donutOffsets)

	// Check for READONLY file
	fNameRo := filepath.Join(numberedFolderPath, "READONLY")
	_, err := os.Stat(fNameRo)
	if err != nil {
		panic("Input tier folder should have READONLY flag file")
	}

	folderName := filepath.Join(newDonutFolder, "HashPrefix")
	err = os.MkdirAll(folderName, 0755)
	if err != nil {
		return err
	}

	// Mark the folder as being baked with a BAKING flag file
	bakingFlagFileName := filepath.Join(newDonutFolder, "BAKING")
	file, err := os.Create(bakingFlagFileName)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	const digitsPerFolder = 2
	sourceFilenameDigits, sourceFoldersDigits := CalculatePrefixPattern(sourceTierIndex, digitsPerFolder)
	destFilenameDigits, destFoldersDigits := CalculatePrefixPattern(sourceTierIndex+1, digitsPerFolder)
	sourcePrefixIndexCount := uint64(1) << (uint64(sourceTierIndex) * 4) // 16 ^ sourceTierIndex
	sourcePrefixNibbles := sourceTierIndex

	suffixSizeNibbles := hashBytesCount*2 - sourcePrefixNibbles
	suffixSizeBytes := suffixSizeNibbles / 2
	if suffixSizeNibbles&1 == 1 {
		suffixSizeBytes++
	}
	suffixEntrySize := int(suffixSizeBytes) + sourceLocalPiWriter.StorageBytes()
	suffixEntryBytes := make([]byte, suffixEntrySize)
	suffixLocalPiOffset := suffixSizeBytes

	forestUpdatesFiles := [16]*os.File{}
	forestUpdatesFilenames := [16]string{}
	for f := 0; f < sourceDonutsCount; f++ {
		fName := filepath.Join(newDonutFolder, fmt.Sprintf("TempUpdatesDonut%1X.bin", f))
		forestUpdatesFiles[f], err = os.Create(fName)
		if err != nil {
			return err
		}
		forestUpdatesFilenames[f] = fName
	}

	for prefixIndex := uint64(0); prefixIndex < sourcePrefixIndexCount; prefixIndex++ {
		// Make the prefix from prefixIndex as a sequence of nibbles
		prefix := make([]shallowtreebyte.NibbleVal, sourcePrefixNibbles)
		for nibbleIndex := byte(0); nibbleIndex < sourcePrefixNibbles; nibbleIndex++ {
			prefix[nibbleIndex] = shallowtreebyte.NibbleVal((prefixIndex >> (sourcePrefixNibbles - nibbleIndex - 1) * 4) & 0xF)
		}
		prefixFilename, prefixFolder := formatFilePathFilename(prefix, sourceFilenameDigits, sourceFoldersDigits)
		suffixFilename := prefixFilename + "HashSuffix.bin"

		// files to append to
		destAppendFiles := [16]*os.File{}
		destAppendCounts := [16]uint64{}
		newPrefix := make([]shallowtreebyte.NibbleVal, sourcePrefixNibbles+1)
		copy(newPrefix[1:], prefix)
		for newNibble := shallowtreebyte.NibbleVal(0); newNibble < 16; newNibble++ {
			newPrefix[0] = newNibble
			newPrefixFilename, newPrefixFolder := formatFilePathFilename(newPrefix, destFilenameDigits, destFoldersDigits)
			newPath := filepath.Join(newDonutFolder, "HashPrefix", newPrefixFolder)
			err = os.MkdirAll(newPath, 0755)
			if err != nil {
				return err
			}
			destAppendFiles[newNibble], err = os.OpenFile(filepath.Join(newPath, newPrefixFilename+"HashSuffix.bin"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return err
			}
		}

		suffixDonutFilepaths := [16]string{}
		for forest := 0; forest < sourceDonutsCount; forest++ {
			sourceForestPrefixPath := filepath.Join(numberedFolderPath, fmt.Sprintf("Donut%1X", forest), "HashPrefix")
			suffixFilePath := filepath.Join(sourceForestPrefixPath, prefixFolder, suffixFilename)
			suffixDonutFilepaths[forest] = suffixFilePath
			suffixFile, err := os.Open(suffixFilePath)
			if err != nil {
				return err
			}
			readSuccess := true
			for readSuccess {
				_, err = file.Read(suffixEntryBytes)
				if err != nil {
					readSuccess = false
				} else {
					localPi := sourceLocalPiWriter.ReadID(suffixEntryBytes[suffixLocalPiOffset:])
					globalPi := localPi.ToGlobalPi(donutOffsets[forest])
					newLocalPi := globalPi.ToLocalPi(newDonutOffset)

					nextNibble := shallowtreebyte.NibbleVal(suffixEntryBytes[0] >> 4)
					destPrefix := make([]shallowtreebyte.NibbleVal, 1, suffixSizeNibbles+1)
					destPrefix[0] = nextNibble
					destPrefix = append(destPrefix, prefix...)

					sourceSuffix := make([]shallowtreebyte.NibbleVal, suffixSizeNibbles)
					for i := range suffixSizeNibbles {
						if i&1 == 0 {
							sourceSuffix[i] = shallowtreebyte.NibbleVal((suffixEntryBytes[i/2] >> 4) & 0xF)
						} else {
							sourceSuffix[i] = shallowtreebyte.NibbleVal((suffixEntryBytes[i/2]) & 0xF)
						}
					}
					destSuffix := sourceSuffix[1:]

					// Bytes to write to destAppendFiles[nextNibble]
					const spareBytes = 64 + 8
					suffixAppendBytes := [spareBytes]byte{}
					lastByte := byte(0)
					for suffixNibble := range shallowtreebyte.NibbleVal(suffixSizeNibbles - 1) {
						if suffixNibble&1 == 0 {
							suffixAppendBytes[suffixNibble/2] = byte(destSuffix[suffixNibble] << 4) // MS
						} else {
							suffixAppendBytes[suffixNibble/2] |= byte(destSuffix[suffixNibble] & 0x0F) // LS
						}
						lastByte = byte(suffixNibble) / 2
					}
					destLocalPiWriter.WriteID(suffixAppendBytes[lastByte+1:], newLocalPi)
					writeLength := lastByte + 1 + byte(destLocalPiWriter.StorageBytes())
					_, err = destAppendFiles[nextNibble].Write(suffixAppendBytes[:writeLength])
					if err != nil {
						return err
					}
					destAppendCounts[nextNibble]++

					// Append stuff to temp updates file for source donut
					// We write the triple (LocalPi, nextNibble, SuffixIndex)
					tempBytes := [8 + 1 + 8]byte{}
					binary.LittleEndian.PutUint64(tempBytes[:8], uint64(localPi))
					tempBytes[8] = byte(nextNibble)
					binary.LittleEndian.PutUint64(tempBytes[9:], destAppendCounts[nextNibble])
					_, err = forestUpdatesFiles[forest].Write(suffixAppendBytes[:17])
					if err != nil {
						return err
					}

				}
			}
			err = suffixFile.Close()
			if err != nil {
				return err
			}
		}
		// We've been through all the donuts extracting the info we need pertaining to a particular prefix
		// To prevent overlapping disk bloat, we now delete all the files pertaining to that prefix from all the source
		// Donuts
		for forest := 0; forest < sourceDonutsCount; forest++ {
			err := os.Remove(suffixDonutFilepaths[forest])
			if err != nil {
				return err
			}
		}
	}
	// We've been through all the prefixes in all the source donuts.
	// Now we need to
	for forest := 0; forest < sourceDonutsCount; forest++ {
		// For each source donut (forest) we need to deal with the HashesOrder.bin file
		// We will need to take account of the updates stored in the TempUpdates file for each forest

		// Close and reopen the updates file
		err = forestUpdatesFiles[forest].Close()
		if err != nil {
			return err
		}
		updatesFile, err := os.Open(forestUpdatesFilenames[forest])
		if err != nil {
			return err
		}
		updatesStat, err := updatesFile.Stat()
		if err != nil {
			return err
		}
		size := updatesStat.Size()
		if size%17 != 0 {
			panic("Updates file size is not a multiple of 17")
		}
		entries := size / 17
		// Read it into a map
		type update struct {
			newSuffixIndex uint64
			nextNibble     byte
		}
		updateMap := make(map[types.LocalPi]update, entries)
		for range entries {
			entry := [17]byte{}
			_, err = updatesFile.Read(entry[:])
			if err != nil {
				return err
			}
			oldLocalPi := types.LocalPi(binary.BigEndian.Uint64(entry[:8]))
			nextNibble := entry[8]
			newSuffixIndex := binary.BigEndian.Uint64(entry[9:])
			updateMap[oldLocalPi] = update{newSuffixIndex, nextNibble}
		}
		err = updatesFile.Close()
		if err != nil {
			return err
		}
		// And remove the file
		err = os.Remove(forestUpdatesFilenames[forest])
		if err != nil {
			return err
		}

		donutFolder := fmt.Sprintf("Donut%1X", forest)
		sourceFilename := filepath.Join(numberedFolderPath, donutFolder, "HashesOrder.bin")
		sourceFile, err := os.Open(sourceFilename)
		if err != nil {
			return err
		}

		destFilename := filepath.Join(newDonutFolder, "HashesOrder.bin")
		destFile, err := os.Create(destFilename)
		if err != nil {
			return err
		}

		// Each entry in the HashesOrder.bin file is a pair, prefixIndex followed by an index into suffix file.
		// for each tier, a different number of bytes are used to encode the prefixIndex
		sourcePrefixIndexBytesCount := bytesToEncodePrefixIndex(sourceTierIndex)
		sourceSuffixIndexBytesCount := bytesToEncodeSuffixIndex(sourceTierIndex)
		sourceEntrySize := sourcePrefixIndexBytesCount + sourceSuffixIndexBytesCount

		destPrefixIndexBytesCount := bytesToEncodePrefixIndex(sourceTierIndex + 1)
		destSuffixIndexBytesCount := bytesToEncodeSuffixIndex(sourceTierIndex + 1)
		destEntrySize := destPrefixIndexBytesCount + destSuffixIndexBytesCount

		const spareBytes = 8 + 8
		entry := [spareBytes]byte{}
		readSuccess := true
		oldLocalPi := types.LocalPi(0)
		for readSuccess {
			_, err := sourceFile.Read(entry[:sourceEntrySize])
			if err != nil {
				readSuccess = false
			} else {
				eightBytes := [8]byte{}
				copy(eightBytes[:], entry[0:sourcePrefixIndexBytesCount])
				prefixIndex := binary.LittleEndian.Uint64(eightBytes[:])
				eightBytes = [8]byte{}
				copy(eightBytes[:], entry[sourcePrefixIndexBytesCount:sourcePrefixIndexBytesCount+sourceSuffixIndexBytesCount])
				_ = binary.LittleEndian.Uint64(eightBytes[:]) // Old suffixIndex not used

				newPrefixIndex := prefixIndex<<4 | uint64(updateMap[oldLocalPi].nextNibble)
				newSuffixIndex := updateMap[oldLocalPi].newSuffixIndex

				// Write these to the destination file
				eightBytes = [8]byte{}
				binary.LittleEndian.PutUint64(eightBytes[:], newPrefixIndex)

				newEntry := [spareBytes]byte{}
				copy(newEntry[0:destPrefixIndexBytesCount], eightBytes[:destPrefixIndexBytesCount])

				eightBytes = [8]byte{}
				binary.LittleEndian.PutUint64(eightBytes[:], newSuffixIndex)
				copy(newEntry[destPrefixIndexBytesCount:destPrefixIndexBytesCount+destSuffixIndexBytesCount], eightBytes[:destSuffixIndexBytesCount])

				_, err = destFile.Write(newEntry[:destEntrySize])
				if err != nil {
					return err
				}

				oldLocalPi++
			}
		} // for readSuccess
		err = destFile.Close()
		if err != nil {
			return err
		}
		err = sourceFile.Close()
		if err != nil {
			return err
		}
	} // for forest

	// It is now baked. Remove the flag file
	err = os.Remove(bakingFlagFileName)
	if err != nil {
		return err
	}

	// Remove the whole input tier folder
	err = os.RemoveAll(numberedFolderPath)
	if err != nil {
		return err
	}

	return nil
}

func bytesToEncodePrefixIndex(tierIndex byte) byte {
	// tierIndex = 0: Zero bytes (!)
	// tierIndex = 1: One byte (one nibble prefix)
	// tierIndex = 2: One byte (two nibbles prefix)
	// tierIndex = 3: Two bytes (three nibbles prefix)
	nibblesPrefix := tierIndex
	return (nibblesPrefix + 1) / 2
}

func bytesToEncodeSuffixIndex(tierIndex byte) byte {
	return 3 // Todo: This is approximate! Not guaranteed in all cases
}
