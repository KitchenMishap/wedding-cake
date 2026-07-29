package tierbake

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

func BakeFrozenTierToDonutFolder(numberedFolderPath string, newDonutFolder string,
	sourceTierIndex byte,
	sourceConfig *smalltree.SmallTreeConfig, destConfig *smalltree.SmallTreeConfig,
	hashBytesCount byte, donutOffsets []types.PiOffset,
	newDonutOffset types.PiOffset) error {

	// Do a forward lookup of localPi 0 in Donut 0 as an initial check
	donut0FolderPath := filepath.Join(numberedFolderPath, "Donut0")
	hash, err := ForwardLookup(0, shallowtreebyte.NibbleIndex(sourceTierIndex), donut0FolderPath, sourceConfig)
	if err != nil {
		return err
	}

	sourceLocalPiWriter := sourceConfig.LocalPiRWriter
	sourcePrefixIndexRWriter := sourceConfig.PrefixIndexRWriter
	sourceSuffixIndexRWriter := sourceConfig.SuffixIndexRWriter
	destLocalPiWriter := destConfig.LocalPiRWriter
	destPrefixIndexRWriter := destConfig.PrefixIndexRWriter
	destSuffixIndexRWriter := destConfig.SuffixIndexRWriter

	sourceDonutsCount := len(donutOffsets)

	// Check for READONLY file
	fNameRo := filepath.Join(numberedFolderPath, "READONLY")
	_, err = os.Stat(fNameRo)
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
	sourcePrefixIndexCount := types.PrefixIndex(uint64(1) << (uint64(sourceTierIndex) * 4)) // 16 ^ sourceTierIndex
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

	for prefixIndex := types.PrefixIndex(0); prefixIndex < sourcePrefixIndexCount; prefixIndex++ {
		// Make the prefix from prefixIndex as a sequence of nibbles
		prefix := make([]shallowtreebyte.NibbleVal, sourcePrefixNibbles)
		// Work backwards to construct the prefix nibbles from the prefixIndex
		prefixIndexShift := prefixIndex
		for nibbleIndex := int(sourcePrefixNibbles) - 1; nibbleIndex >= 0; nibbleIndex-- {
			prefix[nibbleIndex] = shallowtreebyte.NibbleVal(prefixIndexShift & 0xF)
			prefixIndexShift >>= 4
		}
		prefixFilename, prefixFolder := FormatFilePathFilename(prefix, sourceFilenameDigits, sourceFoldersDigits)
		suffixFilename := prefixFilename + "HashSuffix.bin"

		// files to append to
		destAppendFiles := [16]*os.File{}
		destAppendCounts := [16]uint64{}
		newPrefix := make([]shallowtreebyte.NibbleVal, 0, sourcePrefixNibbles+1)
		newPrefix = append(newPrefix, prefix...)
		newPrefix = append(newPrefix, shallowtreebyte.NibbleVal(0)) // This element will be overwritten with newNibble
		for newNibble := shallowtreebyte.NibbleVal(0); newNibble < 16; newNibble++ {
			newPrefix[sourcePrefixNibbles] = newNibble
			newPrefixFilename, newPrefixFolder := FormatFilePathFilename(newPrefix, destFilenameDigits, destFoldersDigits)
			newPath := filepath.Join(newDonutFolder, "HashPrefix", newPrefixFolder)
			err = os.MkdirAll(newPath, 0755)
			if err != nil {
				return err
			}
			destAppendFiles[newNibble], err = os.Create(filepath.Join(newPath, newPrefixFilename+"HashSuffix.bin"))
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
				_, err = suffixFile.Read(suffixEntryBytes)
				if err != nil {
					readSuccess = false
				} else {
					localPi := sourceLocalPiWriter.ReadID(suffixEntryBytes[suffixLocalPiOffset:])
					if localPi == types.LocalPiNoMatch {
						panic("No match presentation index")
					}
					globalPi := localPi.ToGlobalPi(donutOffsets[forest])
					if globalPi == types.GlobalPresentationIndexNoMatch {
						panic("No match presentation index")
					}
					newLocalPi := globalPi.ToLocalPi(newDonutOffset)
					if newLocalPi == types.LocalPiNoMatch {
						panic("No match presentation index")
					}

					nextNibble := shallowtreebyte.NibbleVal(suffixEntryBytes[0] >> 4)
					destPrefix := make([]shallowtreebyte.NibbleVal, 0, sourcePrefixNibbles+1)
					destPrefix = append(destPrefix, prefix...)
					destPrefix = append(destPrefix, nextNibble)

					sourceSuffix := make([]shallowtreebyte.NibbleVal, suffixSizeNibbles)
					for i := range suffixSizeNibbles {
						if i&1 == 0 {
							sourceSuffix[i] = shallowtreebyte.NibbleVal((suffixEntryBytes[i/2] >> 4) & 0xF)
						} else {
							sourceSuffix[i] = shallowtreebyte.NibbleVal((suffixEntryBytes[i/2]) & 0xF)
						}
					}
					destSuffix := sourceSuffix[1:]

					// Bytes to write to destAppendFiles[nextNibble] (an <x>HashSuffix.bin file)
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
						_ = suffixFile.Close()
						return err
					}
					destAppendCounts[nextNibble]++

					// Append stuff to temp updates file for source donut
					// We write the triple (LocalPi, nextNibble, SuffixIndex)
					tempBytes := [8 + 1 + 8]byte{}
					binary.LittleEndian.PutUint64(tempBytes[:8], uint64(localPi))
					tempBytes[8] = byte(nextNibble)
					binary.LittleEndian.PutUint64(tempBytes[9:], destAppendCounts[nextNibble]-1)
					_, err = forestUpdatesFiles[forest].Write(tempBytes[:17])
					if err != nil {
						_ = suffixFile.Close()
						return err
					}
				}
			}
			err = suffixFile.Close()
			if err != nil {
				return err
			}
		} // for forest

		// We've been through all the donuts extracting the info we need pertaining to a particular prefix
		// To prevent overlapping disk bloat, we now delete all the files pertaining to that prefix from all the source
		// Donuts
		for forest := 0; forest < sourceDonutsCount; forest++ {
			err := os.Remove(suffixDonutFilepaths[forest])
			if err != nil {
				return err
			}
		}
	} // for prefixIndex

	// We've been through all the prefixes in all the source donuts.
	// Now we need to

	destFilename := filepath.Join(newDonutFolder, "HashesOrder.bin")
	destFile, err := os.Create(destFilename)
	if err != nil {
		return err
	}

	for forest := 0; forest < sourceDonutsCount; forest++ {
		// For each source donut (forest) we need to deal with the HashesOrder.bin file
		// We will need to take account of the updates stored in the TempUpdates file for each forest

		// Close and reopen the updates file
		err = forestUpdatesFiles[forest].Close()
		if err != nil {
			_ = destFile.Close()
			return err
		}
		updatesFile, err := os.Open(forestUpdatesFilenames[forest])
		if err != nil {
			_ = destFile.Close()
			return err
		}
		updatesStat, err := updatesFile.Stat()
		if err != nil {
			_ = destFile.Close()
			_ = updatesFile.Close()
			return err
		}
		size := updatesStat.Size()
		if size%17 != 0 {
			panic("Updates file size is not a multiple of 17")
		}
		entries := size / 17
		// Read it into a map
		type update struct {
			newSuffixIndex types.SuffixIndex
			nextNibble     byte
		}
		updateMap := make(map[types.LocalPi]update, entries)
		for range entries {
			entry := [17]byte{}
			_, err = updatesFile.Read(entry[:])
			if err != nil {
				_ = destFile.Close()
				_ = updatesFile.Close()
				return err
			}
			oldLocalPi := types.LocalPi(binary.LittleEndian.Uint64(entry[:8]))
			nextNibble := entry[8]
			newSuffixIndex := types.SuffixIndex(binary.LittleEndian.Uint64(entry[9:]))
			updateMap[oldLocalPi] = update{newSuffixIndex, nextNibble}
		}
		err = updatesFile.Close()
		if err != nil {
			_ = destFile.Close()
			return err
		}
		// And remove the file
		err = os.Remove(forestUpdatesFilenames[forest])
		if err != nil {
			_ = destFile.Close()
			return err
		}

		donutFolder := fmt.Sprintf("Donut%1X", forest)
		sourceFilename := filepath.Join(numberedFolderPath, donutFolder, "HashesOrder.bin")
		sourceFile, err := os.Open(sourceFilename)
		if err != nil {
			_ = destFile.Close()
			return err
		}

		// Each entry in the HashesOrder.bin file is a pair, prefixIndex followed by an index into suffix file.
		// for each tier, a different number of bytes are used to encode the prefixIndex
		sourcePrefixIndexBytesCount := sourcePrefixIndexRWriter.StorageBytes()
		sourceSuffixIndexBytesCount := sourceSuffixIndexRWriter.StorageBytes()
		sourceEntrySize := sourcePrefixIndexBytesCount + sourceSuffixIndexBytesCount

		destPrefixIndexBytesCount := destPrefixIndexRWriter.StorageBytes()
		destSuffixIndexBytesCount := destSuffixIndexRWriter.StorageBytes()
		destEntrySize := destPrefixIndexBytesCount + destSuffixIndexBytesCount

		const spareBytes = 8 + 8
		entry := [spareBytes]byte{}
		oldLocalPi := types.LocalPi(0)
		readSuccess := true
		for readSuccess {
			_, err := sourceFile.Read(entry[:sourceEntrySize])
			if err != nil {
				readSuccess = false
			} else {
				prefixIndex := sourcePrefixIndexRWriter.ReadID(entry[:])
				// old suffixIndex not used

				nextNibble := updateMap[oldLocalPi].nextNibble
				newPrefixIndex := prefixIndex<<4 | types.PrefixIndex(nextNibble)
				newSuffixIndex := updateMap[oldLocalPi].newSuffixIndex

				// Write these to the destination file
				newEntry := [spareBytes]byte{}
				destPrefixIndexRWriter.WriteID(newEntry[:], newPrefixIndex)
				destSuffixIndexRWriter.WriteID(newEntry[destPrefixIndexBytesCount:], newSuffixIndex)

				_, err = destFile.Write(newEntry[:destEntrySize])
				if err != nil {
					_ = destFile.Close()
					_ = sourceFile.Close()
					return err
				}

				oldLocalPi++
			}
		} // for readSuccess
		err = sourceFile.Close()
		if err != nil {
			_ = destFile.Close()
			return err
		}
	} // for forest
	err = destFile.Close()
	if err != nil {
		return err
	}

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

	// Do a forward lookup of localPi 0 in the new donut as an final check
	newHash, err := ForwardLookup(0, shallowtreebyte.NibbleIndex(sourceTierIndex+1), newDonutFolder, destConfig)
	if err != nil {
		return err
	}
	if slices.Equal(newHash, hash) {
		fmt.Printf("Baking hashes check match! :-)\n")
	} else {
		fmt.Printf("Mismatch after baking tier %d\n", sourceTierIndex+1)
		panic("Baking hashes check mismatch!")
	}

	return nil
}
