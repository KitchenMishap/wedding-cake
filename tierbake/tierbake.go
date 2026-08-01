package tierbake

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/hash"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

func BakeFrozenTierToDonutFolder(numberedFolderPath string, newDonutFolder string,
	sourceTierIndex byte,
	sourceConfig *smalltree.SmallTreeConfig, destConfig *smalltree.SmallTreeConfig,
	hashBytesCount byte, donutOffsets []types.PiOffset,
	newDonutOffset types.PiOffset) error {

	sourcePrefixNibblesCount := sourceTierIndex
	destPrefixNibblesCount := sourceTierIndex + 1

	// Do a forward lookup of localPi 0 in Donut 0 as an initial check
	donut0FolderPath := filepath.Join(numberedFolderPath, "Donut0")
	theHash := hash.Full{}
	err := ForwardLookup(0, shallowtreebyte.NibbleIndex(sourceTierIndex), donut0FolderPath, sourceConfig, &theHash)
	if err != nil {
		return err
	}

	sourceLocalPiWriter := sourceConfig.LocalPiRWriter
	//	sourcePrefixIndexRWriter := sourceConfig.PrefixIndexRWriter
	sourceSuffixIndexRWriter := sourceConfig.SuffixIndexRWriter
	destLocalPiWriter := destConfig.LocalPiRWriter
	//	destPrefixIndexRWriter := destConfig.PrefixIndexRWriter
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
	//suffixEntrySize := int(suffixSizeBytes) + sourceLocalPiWriter.StorageBytes()
	//suffixEntryBytes := make([]byte, suffixEntrySize)
	suffixEntryLocalPiBytes := make([]byte, sourceLocalPiWriter.StorageBytes())
	//suffixLocalPiOffset := suffixSizeBytes
	suffixSourceObj := hash.Suffix{}

	forestUpdatesUnderlyingFiles := [16]*os.File{}
	forestUpdatesWriters := [16]*bufio.Writer{}
	forestUpdatesFilenames := [16]string{}
	for f := 0; f < sourceDonutsCount; f++ {
		fName := filepath.Join(newDonutFolder, fmt.Sprintf("TempUpdatesDonut%1X.bin", f))
		forestUpdatesUnderlyingFiles[f], err = os.Create(fName)
		if err != nil {
			return err
		}
		forestUpdatesWriters[f] = bufio.NewWriterSize(forestUpdatesUnderlyingFiles[f], 64*1024)
		forestUpdatesFilenames[f] = fName
	}

	for prefixIndex := types.PrefixIndex(0); prefixIndex < sourcePrefixIndexCount; prefixIndex++ {
		// Make the prefix from prefixIndex as a sequence of nibbles
		prefixObj := hash.Prefix{}
		prefixObj.Init(hashBytesCount, sourcePrefixNibbles)
		prefixObj.SetPrefixFromNumber(uint64(prefixIndex))
		// ToDo get rid of prefixNibbles
		prefixNibbles := make([]shallowtreebyte.NibbleVal, sourcePrefixNibbles)
		// Work backwards to construct the prefix nibbles from the prefixIndex
		prefixIndexShift := prefixIndex
		for nibbleIndex := int(sourcePrefixNibbles) - 1; nibbleIndex >= 0; nibbleIndex-- {
			prefixNibbles[nibbleIndex] = shallowtreebyte.NibbleVal(prefixIndexShift & 0xF)
			prefixIndexShift >>= 4
		}
		prefixFilename, prefixFolder := FormatFilePathFilename(prefixNibbles, sourceFilenameDigits, sourceFoldersDigits)
		suffixFilename := prefixFilename + "HashSuffix.bin"

		// files to append to, indexed by nextNibble
		destAppendUnderlyingFiles := [16]*os.File{}
		destAppendWriters := [16]*bufio.Writer{}
		destAppendCounts := [16]uint64{}
		// ToDo get rid of newPrefixNibbles
		newPrefixNibbles := make([]shallowtreebyte.NibbleVal, 0, sourcePrefixNibbles+1)
		newPrefixNibbles = append(newPrefixNibbles, prefixNibbles...)
		newPrefixNibbles = append(newPrefixNibbles, shallowtreebyte.NibbleVal(0)) // This element will be overwritten with newNibble
		for newNibble := shallowtreebyte.NibbleVal(0); newNibble < 16; newNibble++ {
			newPrefixNibbles[sourcePrefixNibbles] = newNibble
			newPrefixFilename, newPrefixFolder := FormatFilePathFilename(newPrefixNibbles, destFilenameDigits, destFoldersDigits)
			newPath := filepath.Join(newDonutFolder, "HashPrefix", newPrefixFolder)
			err = os.MkdirAll(newPath, 0755)
			if err != nil {
				return err
			}
			destAppendUnderlyingFiles[newNibble], err = os.Create(filepath.Join(newPath, newPrefixFilename+"HashSuffix.bin"))
			if err != nil {
				return err
			}
			destAppendWriters[newNibble] = bufio.NewWriterSize(destAppendUnderlyingFiles[newNibble], 64*1024)
		}

		suffixDonutFilepaths := [16]string{}
		for forest := 0; forest < sourceDonutsCount; forest++ {
			sourceForestPrefixPath := filepath.Join(numberedFolderPath, fmt.Sprintf("Donut%1X", forest), "HashPrefix")
			suffixFilePath := filepath.Join(sourceForestPrefixPath, prefixFolder, suffixFilename)
			suffixDonutFilepaths[forest] = suffixFilePath
			suffixUnderlyingFile, err := os.Open(suffixFilePath)
			if err != nil {
				return err
			}
			suffixReader := bufio.NewReaderSize(suffixUnderlyingFile, 64*1024)
			readSuccess := true
			for readSuccess {
				suffixSourceObj.Init(hashBytesCount, sourcePrefixNibbles)
				err1 := suffixSourceObj.Read(suffixReader)
				_, err2 := io.ReadFull(suffixReader, suffixEntryLocalPiBytes)
				if err1 != nil || err2 != nil {
					readSuccess = false
				} else {
					localPi := sourceLocalPiWriter.ReadID(suffixEntryLocalPiBytes[:])
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

					nextNibble := suffixSourceObj.RemoveFirstNibble() // suffixSourceObj now used as destSuffix
					// ToDo do away with destPrefix
					destPrefix := make([]shallowtreebyte.NibbleVal, 0, sourcePrefixNibbles+1)
					destPrefix = append(destPrefix, prefixNibbles...)
					destPrefix = append(destPrefix, shallowtreebyte.NibbleVal(nextNibble))

					// Write the suffix
					spareNibble := byte(0)
					err = suffixSourceObj.Write(destAppendWriters[nextNibble], spareNibble)
					if err != nil {
						_ = suffixUnderlyingFile.Close()
						return err
					}
					// Write the localPi
					const spareBytesLocalPi = 8
					destLocalPiBytes := [spareBytesLocalPi]byte{}
					destLocalPiWriter.WriteID(destLocalPiBytes[:], newLocalPi)
					_, err = destAppendWriters[nextNibble].Write(destLocalPiBytes[:destLocalPiWriter.StorageBytes()])
					if err != nil {
						_ = suffixUnderlyingFile.Close()
						return err
					}

					destAppendCounts[nextNibble]++

					// Append stuff to temp updates file for source donut
					// We write the triple (LocalPi, nextNibble, SuffixIndex)
					tempBytes := [8 + 1 + 8]byte{}
					binary.LittleEndian.PutUint64(tempBytes[:8], uint64(localPi))
					tempBytes[8] = byte(nextNibble)
					if destAppendCounts[nextNibble]-1 > 0xFFFFFF {
						panic("It's a bit big")
					}
					binary.LittleEndian.PutUint64(tempBytes[9:], destAppendCounts[nextNibble]-1)

					_, err = forestUpdatesWriters[forest].Write(tempBytes[:17])
					if err != nil {
						_ = suffixUnderlyingFile.Close()
						return err
					}
				}
			}
			err = suffixUnderlyingFile.Close()
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
		for nibble := 0; nibble < 16; nibble++ {
			err = destAppendWriters[nibble].Flush()
			if err != nil {
				return err
			}
			err = destAppendUnderlyingFiles[nibble].Close()
			if err != nil {
				return err
			}
		}
	} // for prefixIndex

	// We've been through all the prefixes in all the source donuts.

	destFilename := filepath.Join(newDonutFolder, "HashesOrder.bin")
	destUnderlyingFile, err := os.Create(destFilename)
	if err != nil {
		return err
	}
	destWriter := bufio.NewWriterSize(destUnderlyingFile, 64*1024)

	sourcePrefixObj := hash.Prefix{}
	sourcePrefixObj.Init(hashBytesCount, sourcePrefixNibblesCount)
	destPrefixObj := hash.Prefix{}
	destPrefixObj.Init(hashBytesCount, destPrefixNibblesCount)

	for forest := 0; forest < sourceDonutsCount; forest++ {
		// For each source donut (forest) we need to deal with the HashesOrder.bin file
		// We will need to take account of the updates stored in the TempUpdates file for each forest

		// Close and reopen the updates files
		err = forestUpdatesWriters[forest].Flush()
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}
		err = forestUpdatesUnderlyingFiles[forest].Close()
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}
		updatesUnderlyingFile, err := os.Open(forestUpdatesFilenames[forest])
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}
		updatesReader := bufio.NewReaderSize(updatesUnderlyingFile, 64*1024)
		updatesStat, err := updatesUnderlyingFile.Stat()
		if err != nil {
			_ = destUnderlyingFile.Close()
			_ = updatesUnderlyingFile.Close()
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
			_, err := io.ReadFull(updatesReader, entry[:])
			if err != nil {
				_ = destUnderlyingFile.Close()
				_ = updatesUnderlyingFile.Close()
				return err
			}
			oldLocalPi := types.LocalPi(binary.LittleEndian.Uint64(entry[:8]))
			nextNibble := entry[8]
			newSuffixIndex := types.SuffixIndex(binary.LittleEndian.Uint64(entry[9:]))
			if newSuffixIndex > types.SuffixIndex(0xFFFFFF) {
				panic("newSuffixIndex is rather big")
			}
			updateMap[oldLocalPi] = update{newSuffixIndex, nextNibble}
		}
		err = updatesUnderlyingFile.Close()
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}
		// And remove the file
		err = os.Remove(forestUpdatesFilenames[forest])
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}

		donutFolder := fmt.Sprintf("Donut%1X", forest)
		sourceFilename := filepath.Join(numberedFolderPath, donutFolder, "HashesOrder.bin")
		sourceUnderlyingFile, err := os.Open(sourceFilename)
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}
		sourceReader := bufio.NewReaderSize(sourceUnderlyingFile, 64*1024)

		// Each entry in the HashesOrder.bin file is a pair, prefixIndex followed by an index into suffix file.
		// for each tier, a different number of bytes are used to encode the prefixIndex
		//		sourcePrefixIndexBytesCount := sourcePrefixIndexRWriter.StorageBytes()
		sourceSuffixIndexBytesCount := sourceSuffixIndexRWriter.StorageBytes()
		//		sourceEntrySize := sourcePrefixIndexBytesCount + sourceSuffixIndexBytesCount

		//		destPrefixIndexBytesCount := destPrefixIndexRWriter.StorageBytes()
		destSuffixIndexBytesCount := destSuffixIndexRWriter.StorageBytes()
		//destEntrySize := destPrefixIndexBytesCount + destSuffixIndexBytesCount

		const spareBytes = 8
		suffixIndexBytes := [spareBytes]byte{}
		oldLocalPi := types.LocalPi(0)
		readSuccess := true
		for readSuccess {
			err1 := sourcePrefixObj.Read(sourceReader)
			_, err2 := io.ReadFull(sourceReader, suffixIndexBytes[:sourceSuffixIndexBytesCount])
			if err1 != nil || err2 != nil {
				readSuccess = false
			} else {
				prefixIndex := sourcePrefixObj.PrefixAsNumber()
				// old suffixIndex not used

				nextNibble := updateMap[oldLocalPi].nextNibble
				newPrefixIndex := prefixIndex<<4 | uint64(nextNibble)
				mapEntry, ok := updateMap[oldLocalPi]
				if !ok {
					panic("mapEntry not found")
				}
				newSuffixIndex := mapEntry.newSuffixIndex

				// Write these to the destination file
				// Prefix
				destPrefixObj.SetPrefixFromNumber(newPrefixIndex)
				spareNibble := byte(0)
				err = destPrefixObj.Write(destWriter, spareNibble)
				if err != nil {
					_ = destUnderlyingFile.Close()
					_ = sourceUnderlyingFile.Close()
					return err
				}
				// Suffix
				destSuffixIndexRWriter.WriteID(suffixIndexBytes[:], newSuffixIndex)
				_, err = destWriter.Write(suffixIndexBytes[:destSuffixIndexBytesCount])
				if err != nil {
					_ = destUnderlyingFile.Close()
					_ = sourceUnderlyingFile.Close()
					return err
				}

				oldLocalPi++
			}
		} // for readSuccess
		err = sourceUnderlyingFile.Close()
		if err != nil {
			_ = destUnderlyingFile.Close()
			return err
		}
	} // for forest
	err = destWriter.Flush()
	if err != nil {
		_ = destUnderlyingFile.Close()
		return err
	}
	err = destUnderlyingFile.Close()
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
	newHash := hash.Full{}
	err = ForwardLookup(0, shallowtreebyte.NibbleIndex(sourceTierIndex+1), newDonutFolder, destConfig, &newHash)
	if err != nil {
		return err
	}
	if theHash.Equal(&newHash) {
	} else {
		fmt.Printf("Mismatch after baking tier %d\n", sourceTierIndex+1)
		panic("Baking hashes check mismatch!")
	}

	return nil
}
