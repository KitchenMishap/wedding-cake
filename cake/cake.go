package cake

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/kitchenmishap/wedding-cake/forest"
	"github.com/kitchenmishap/wedding-cake/hash"
	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/tierbake"
	"github.com/kitchenmishap/wedding-cake/types"
)

type Cake struct {
	folderPath      string
	config          [5]*smalltree.SmallTreeConfig
	hashBytesLength byte

	openInputTier *inputtier.InputTier

	tiersInfo TiersInfo
	openTiers [5]*Tier
}

func (c *Cake) Close() error {
	if c.openInputTier != nil {
		err := c.openInputTier.Close()
		if err != nil {
			return err
		}
		c.openInputTier = nil
	}

	for t := range c.openTiers {
		if c.openTiers[t] != nil {
			err := c.openTiers[t].Close()
			if err != nil {
				return err
			}
			c.openTiers[t] = nil
		}
	}

	return nil
}

func (c *Cake) AppendHash(gpi types.GlobalPi, hash []byte) error {
	err := c.openInputTier.AppendHash(gpi, hash)
	if err != nil {
		return err
	}
	if c.openInputTier.CountPresentationIndices() == 65535 {
		path, offset, err := c.FreezeInputTierForBaking()
		if err != nil {
			return err
		}
		donutPath, donutsCount, err := c.BakeFrozenInputTier(path, offset)
		if err != nil {
			return err
		}
		err = c.IceTheDonut(donutPath, 0)
		if err != nil {
			return err
		}

		// We have baked a new donut, possibly into a new tier
		// We may need to open the new tier
		if donutsCount == 1 {
			// New tier 0
			c.tiersInfo.offset[0] = offset
			c.tiersInfo.present[0] = true
			err = c.tiersInfo.ToDisk(c.folderPath)
			if err != nil {
				return err
			}
			tier, err := c.openTier(0, offset)
			if err != nil {
				return err
			} else {
				c.openTiers[0] = tier
			}
		} else {
			// New donut in tier 0
			if c.openTiers[0] == nil {
				panic("tier 0 should be non-nil if we've just baked a donut into it")
			} else {
				tierPath := c.openTiers[0].folderPath
				donut, err := c.openDonut(0, tierPath, donutsCount-1)
				if err != nil {
					return err
				}
				c.openTiers[0].donuts = append(c.openTiers[0].donuts, donut)
				if len(c.openTiers[0].donuts) > 16 {
					panic("Too many donuts")
				}
			}
		} // if donutCounts==1 else

		// We might have to do further bakes
		tierWeAreBaking := byte(0)
		for donutsCount == 16 {
			path, offset, donutOffsets, err := c.FreezeTierForBaking(tierWeAreBaking) // ToDo look into this re openTiers
			if err != nil {
				return err
			}

			var donutPath string
			donutPath, donutsCount, err = c.BakeFrozenTier(path, offset, donutOffsets, tierWeAreBaking) // ToDo is this folder-only?
			if err != nil {
				return err
			}
			err = c.IceTheDonut(donutPath, tierWeAreBaking+1)
			if err != nil {
				return err
			}

			// ONCE AGAIN...
			// We have baked a new donut, possibly into a new tier
			// We may need to open the new tier
			if donutsCount == 1 {
				// New tier
				c.tiersInfo.offset[tierWeAreBaking+1] = offset
				c.tiersInfo.present[tierWeAreBaking+1] = true
				err = c.tiersInfo.ToDisk(c.folderPath)
				if err != nil {
					return err
				}
				tier, err := c.openTier(tierWeAreBaking+1, offset)
				if err != nil {
					return err
				} else {
					c.openTiers[tierWeAreBaking+1] = tier
				}
			} else {
				// New donut in tier
				if c.openTiers[tierWeAreBaking+1] == nil {
					panic("tier should be non-nil if we've just baked a donut into it")
				} else {
					tierPath := c.openTiers[tierWeAreBaking+1].folderPath
					donut, err := c.openDonut(tierWeAreBaking+1, tierPath, donutsCount-1)
					if err != nil {
						return err
					}
					c.openTiers[tierWeAreBaking+1].donuts = append(c.openTiers[tierWeAreBaking+1].donuts, donut)
					if len(c.openTiers[tierWeAreBaking+1].donuts) > 16 {
						panic("Too many donuts")
					}
				}
			} // if donutsCount==1 else
			tierWeAreBaking++
		} // for donutCounts==16
	}

	return nil
}

func (c *Cake) LookupHash(hash []byte) (types.GlobalPi, error) {
	res := c.openInputTier.LookupHash(hash)
	if res != types.GlobalPresentationIndexNoMatch {
		return res, nil
	}
	nibbles := make([]shallowtreebyte.NibbleVal, c.hashBytesLength*2)
	for i := 0; i < int(c.hashBytesLength); i++ {
		nibbles[i*2] = shallowtreebyte.NibbleVal(hash[i] >> 4)     // MS
		nibbles[i*2+1] = shallowtreebyte.NibbleVal(hash[i] & 0x0F) // LS
	}
	for t := range c.openTiers {
		if c.openTiers[t] == nil {
			continue
		}
		res, err := c.openTiers[t].LookupHash(nibbles)
		if err != nil {
			return 0, err
		}
		if res != types.GlobalPresentationIndexNoMatch {
			return res, nil
		}
	}
	return types.GlobalPresentationIndexNoMatch, nil
}

func (c *Cake) FreezeInputTierForBaking() (string, types.PiOffset, error) {
	piCount := c.openInputTier.CountPresentationIndices()
	newPiOffset := c.tiersInfo.inputOffset + types.PiOffset(piCount)
	frozenOffset := c.tiersInfo.inputOffset

	numberedFolderName := fmt.Sprintf("InputTier_%d", c.tiersInfo.inputOffset)
	numberedFolderPath := filepath.Join(c.folderPath, numberedFolderName)

	err := inputtier.CreateInputTierFiles(c.folderPath, newPiOffset)
	if err != nil {
		return "", 0, err
	}

	c.tiersInfo.inputOffset = newPiOffset
	err = c.tiersInfo.ToDisk(c.folderPath)
	if err != nil {
		return "", 0, err
	}

	err = c.openInputTier.Close()
	if err != nil {
		return "", 0, err
	}
	c.openInputTier, err = inputtier.OpenInputTier(c.folderPath, c.config[0], newPiOffset, c.hashBytesLength)
	if err != nil {
		return "", 0, err
	}

	// Create a READONLY file to prevent anyone from opening for write
	fNameRo := filepath.Join(numberedFolderPath, "READONLY")
	file, err := os.Create(fNameRo)
	if err != nil {
		return "", 0, err
	}
	err = file.Close()
	if err != nil {
		return "", 0, err
	}

	return numberedFolderPath, frozenOffset, nil
}

func (c *Cake) FreezeTierForBaking(tierIndex byte) (string, types.PiOffset,
	[]types.PiOffset, error) {

	//piCount := c.tiers[tierIndex].CountPresentationIndices()
	//newPiOffset := c.inputTierOffset + types.PiOffset(piCount)
	frozenOffset := c.tiersInfo.offset[tierIndex]

	donutOffsets := c.openTiers[tierIndex].donutOffsets
	if len(donutOffsets) > 16 {
		panic("donutOffsets should not have more than 16 elements")
	}

	numberedFolderName := fmt.Sprintf("Tier%d_%d", tierIndex, frozenOffset)
	numberedFolderPath := filepath.Join(c.folderPath, numberedFolderName)

	//err := inputtier.CreateInputTierFiles(c.folderPath, newPiOffset)
	//if err != nil {
	//	return "", 0, err
	//}

	//c.inputTierOffset = newPiOffset
	//offsets := make([]types.PiOffset, 1, len(c.tierOffsets)+1)
	//offsets[0] = newPiOffset
	//offsets = append(offsets, c.tierOffsets...)
	//filePath := filepath.Join(c.folderPath, "TierOffsets.txt")
	//err = c.writePiOffsetsFile(filePath, offsets)
	//if err != nil {
	//	return "", 0, err
	//}

	//c.inputTier, err = inputtier.OpenInputTier(c.folderPath, c.inputTierPiWriter, newPiOffset, c.hashBytesLength)
	//if err != nil {
	//	return "", 0, err
	//}

	// Create a READONLY file to prevent anyone from opening for write
	fNameRo := filepath.Join(numberedFolderPath, "READONLY")
	file, err := os.Create(fNameRo)
	if err != nil {
		return "", 0, nil, err
	}
	err = file.Close()
	if err != nil {
		return "", 0, nil, err
	}

	c.tiersInfo.present[tierIndex] = false
	err = c.tiersInfo.ToDisk(c.folderPath)
	if err != nil {
		return "", 0, nil, err
	}
	err = c.openTiers[tierIndex].Close()
	if err != nil {
		return "", 0, nil, err
	}
	c.openTiers[tierIndex] = nil

	return numberedFolderPath, frozenOffset, donutOffsets, nil
}

func (c *Cake) BakeFrozenTier(numberedPath string, offset types.PiOffset,
	donutOffsets []types.PiOffset, tierIndex byte) (string, byte, error) {

	donutPath, donutsCount, err := c.newDonutFolder(tierIndex+1, offset)
	if err != nil {
		return "", 0, err
	}
	tabs := ""
	if tierIndex > 1 {
		tabs = "\t\t"
	}
	fmt.Printf("%sBaking to %s...\n", tabs, donutPath)

	// Note the following has "no clue" about the tier objects it would be working with.
	// This is as it should be - the tier is frozen so this function should work
	// purely from files on disk!
	err = tierbake.BakeFrozenTierToDonutFolder(numberedPath, donutPath, tierIndex,
		c.config[tierIndex], c.config[tierIndex+1],
		c.hashBytesLength, donutOffsets, offset)
	if err != nil {
		return "", 0, err
	}
	err = os.RemoveAll(numberedPath)
	if err != nil {
		return "", 0, err
	}

	return donutPath, donutsCount, nil
}

func (c *Cake) BakeFrozenInputTier(numberedPath string, offset types.PiOffset) (string, byte, error) {
	donutPath, donutsCount, err := c.newDonutFolder(0, offset)
	if err != nil {
		return "", 0, err
	}

	// Initial check by forward lookup pf localPi 0
	theHash := hash.Full{}
	err = tierbake.ForwardLookup(0, 0, numberedPath, c.config[0], &theHash)
	if err != nil {
		return "", 0, err
	}

	// This function works only on disk files, no references to tier objects.
	err = tierbake.BakeFrozenInputTierToDonutFolder(numberedPath, donutPath)
	if err != nil {
		return "", 0, err
	}
	err = os.RemoveAll(numberedPath)
	if err != nil {
		return "", 0, err
	}

	// Check for same hash in baked donut
	newHash := hash.Full{}
	err = tierbake.ForwardLookup(0, 0, donutPath, c.config[0], &newHash)
	if err != nil {
		return "", 0, err
	}
	if theHash.Equal(&newHash) {
	} else {
		panic("hash mismatch after bake to tier 0")
	}

	return donutPath, donutsCount, nil
}

func (c *Cake) IceTheDonut(donutPath string, tierIndex byte) error {
	startTime := time.Now()
	if tierIndex > 0 {
		fmt.Printf("Icing a donut to %s...\n", donutPath)
	}

	icingPath := filepath.Join(donutPath, "Icing")
	fw := forest.NewForestWrite(icingPath)
	factory := smalltree.NewLevelsCodecNfFactory(c.config[tierIndex])
	enc := factory.MakeLevelsEncoder()
	err := fw.StartWrite()
	if err != nil {
		return err
	}
	prefixNibbles := shallowtreebyte.NibbleIndex(tierIndex)
	suffixNibbles := shallowtreebyte.NibbleIndex(c.hashBytesLength*2) - prefixNibbles
	// The hash suffix in the file is a whole number of bytes (possibly one nibble padding)
	suffixBytes := suffixNibbles / 2
	if suffixNibbles&1 == 1 {
		suffixBytes++
	}
	const spareBytes = 8
	piBytes := [spareBytes]byte{}
	piByteCount := c.config[tierIndex].LocalPiRWriter.StorageBytes()

	prefixObj := hash.Prefix{}
	prefixObj.Init(c.hashBytesLength, byte(prefixNibbles))
	suffixObj := hash.Suffix{}
	suffixObj.Init(c.hashBytesLength, byte(prefixNibbles)) // Yes prefixNibbles!
	hashObj := hash.Full{}

	hashArray := [hash.MaxHashBytes]byte{}

	prefixIndexCount := 1 << (prefixNibbles * 4) // 16 ^ prefixNibbles
	filenameDigits, foldersDigits := tierbake.CalculatePrefixPattern(byte(prefixNibbles), 2)
	for prefixIndex := forest.PrefixIndexType(0); prefixIndex < forest.PrefixIndexType(prefixIndexCount); prefixIndex++ {
		prefixObj.SetPrefixFromNumber(uint64(prefixIndex))
		prefixNibblesSlice := make([]shallowtreebyte.NibbleVal, prefixNibbles)
		for nibbleIndex := shallowtreebyte.NibbleIndex(0); nibbleIndex < prefixNibbles; nibbleIndex++ {
			nibblesToShiftRightBy := prefixNibbles - nibbleIndex - 1
			prefixNibblesSlice[nibbleIndex] = shallowtreebyte.NibbleVal(prefixIndex >> (4 * nibblesToShiftRightBy) & 0x0F)
		}
		filename, folder := tierbake.FormatFilePathFilename(prefixNibblesSlice, filenameDigits, foldersDigits)
		suffixFilePath := filepath.Join(donutPath, "HashPrefix", folder, filename+"HashSuffix.bin")
		file, err := os.Open(suffixFilePath)
		if err != nil {
			return err
		}
		stat, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return err
		}
		suffixReader := bufio.NewReaderSize(file, 64*1024)
		//		contents, err := io.ReadAll(file)
		//		if err != nil {
		//			_ = file.Close()
		//			return err
		//		}
		//		err = file.Close()
		//		if err != nil {
		//			return err
		//		}
		byteCount := c.config[tierIndex].LocalPiRWriter.StorageBytes()
		entrySize := int(suffixBytes) + byteCount
		size := stat.Size()
		if size%int64(entrySize) != 0 {
			panic("Wrong size suffix file")
		}
		entryCount := size / int64(entrySize)
		input := make([]shallowtreebyte.HashPi, entryCount)
		for i := int64(0); i < entryCount; i++ {
			//start := i * entrySize

			// Here nibbles is ALL the nibbles in the hash.
			// Some come from the prefix (from filename), some come from the suffix (from file contents)

			// Read the suffix
			err = suffixObj.Read(suffixReader)
			if err != nil {
				return err
			}

			// Make a hash out of the prefix and the suffix
			prefixObj.AppendSuffix(&hashObj, &suffixObj)
			hashObj.GetToArray(&hashArray)

			// Compile into nibblesArray
			nibblesArray := make([]shallowtreebyte.NibbleVal, c.hashBytesLength*2)
			for j := byte(0); j < c.hashBytesLength; j++ {
				nibblesArray[j*2] = shallowtreebyte.NibbleVal(hashArray[j] >> 4)
				nibblesArray[j*2+1] = shallowtreebyte.NibbleVal(hashArray[j] & 0x0F)
			}

			/*
				nibbles := make([]shallowtreebyte.NibbleVal, c.hashBytesLength*2)
				copy(nibbles[0:prefixNibbles], prefixNibblesSlice)
				suffixByteIndex := 0
				suffixNibbleIndex := 0
				for nibbleIndex := prefixNibbles; nibbleIndex < shallowtreebyte.NibbleIndex(c.hashBytesLength*2); nibbleIndex++ {
					byteVal := contents[start+suffixByteIndex]
					if suffixNibbleIndex&1 == 0 {
						// Most significant nibble of pair
						nibbles[nibbleIndex] = shallowtreebyte.NibbleVal(byteVal >> 4)
					} else {
						nibbles[nibbleIndex] = shallowtreebyte.NibbleVal(byteVal & 0x0F)
						suffixByteIndex++
					}
					suffixNibbleIndex++
				}*/

			// Read the presentation index
			_, err = io.ReadFull(suffixReader, piBytes[:piByteCount])
			if err != nil {
				return err
			}

			input[i] = shallowtreebyte.HashPi{
				Hash:              nibblesArray,
				PresentationIndex: c.config[tierIndex].LocalPiRWriter.ReadID(piBytes[:]),
			}
			if input[i].PresentationIndex == types.LocalPiNoMatch {
				panic("No match presentation index")
			}
		}
		err = file.Close()
		if err != nil {
			return err
		}
		st := shallowtreebyte.GenerateShallowTree(input, prefixNibbles, shallowtreebyte.NibbleIndex(c.hashBytesLength*2), shallowtreebyte.ByteIndex(c.config[tierIndex].ReassuranceBytesCount), 0)
		tf := smalltree.DesignTreeFormat(st, c.config[tierIndex])
		indexBytes, nodesBytes, rootNodeId, rootLevel := enc.EncodeSubTree(st, tf)
		err = fw.AppendTreeForPrefix(prefixIndex, indexBytes, nodesBytes, rootNodeId, rootLevel)
		if err != nil {
			return err
		}
	}
	err = fw.EndWrite()
	if err != nil {
		return err
	}

	if tierIndex > 0 {
		fmt.Printf("\t...%.1f minutes\n", time.Since(startTime).Minutes())
	}

	return nil
}

func (c *Cake) openTier(tierIndex byte, piOffset types.PiOffset) (*Tier, error) {
	tierFolderPath := filepath.Join(c.folderPath, fmt.Sprintf("Tier%d_%d", tierIndex, piOffset))
	donutOffsetsFilePath := filepath.Join(tierFolderPath, "DonutOffsets.txt")
	donutOffsets, err := c.readPiOffsetsFile(donutOffsetsFilePath)
	if err != nil {
		return nil, err
	}

	tier := Tier{}
	tier.folderPath = tierFolderPath
	tier.config = c.config[tierIndex]
	tier.tierIndex = tierIndex

	for d := range donutOffsets {
		donut, err := c.openDonut(tierIndex, tierFolderPath, byte(d))
		if err != nil {
			return nil, err
		}
		tier.donuts = append(tier.donuts, donut)
		tier.donutOffsets = append(tier.donutOffsets, donutOffsets[d])
		if len(tier.donutOffsets) > 16 {
			panic("Too many donut offsets")
		}
	}
	return &tier, nil
}

func (c *Cake) openDonut(tierIndex byte, tierFolderPath string, donutIndex byte) (*forest.ForestRead, error) {
	donutIcingPath := filepath.Join(tierFolderPath, fmt.Sprintf("Donut%X", donutIndex), "Icing")
	fr := forest.NewForestRead(donutIcingPath, tierIndex, c.config[tierIndex])
	err := fr.Open()
	if err != nil {
		return nil, err
	}
	return fr, nil
}

func (c *Cake) newDonutFolder(tierIndex byte, piOffset types.PiOffset) (string, byte, error) {
	if c.openTiers[tierIndex] == nil {
		// Tier doesn't exist (might have earlier been baked into a bigger ter))
		// New tier folder
		tierFolder := fmt.Sprintf("Tier%d_%d", tierIndex, piOffset)
		tierFolderPath := filepath.Join(c.folderPath, tierFolder)
		err := os.MkdirAll(tierFolderPath, 0755)
		if err != nil {
			return "", 0, err
		}
		// Update the tier offset record
		c.tiersInfo.offset[tierIndex] = piOffset
		c.tiersInfo.present[tierIndex] = true
		err = c.tiersInfo.ToDisk(c.folderPath)
		if err != nil {
			return "", 0, err
		}
		// New empty DonutOffsets.txt file
		donutOffsetsFilePath := filepath.Join(tierFolderPath, "DonutOffsets.txt")
		donutOffsets := make([]types.PiOffset, 0)
		err = c.writePiOffsetsFile(donutOffsetsFilePath, donutOffsets)
		if err != nil {
			return "", 0, err
		}
		tier, err := c.openTier(tierIndex, piOffset)
		c.openTiers[tierIndex] = tier
	}
	// Read DonutOffsets.txt file
	tierFolder := fmt.Sprintf("Tier%d_%d", tierIndex, c.tiersInfo.offset[tierIndex])
	tierFolderPath := filepath.Join(c.folderPath, tierFolder)
	donutOffsetsFilepath := filepath.Join(tierFolderPath, "DonutOffsets.txt")
	donutOffsets, err := c.readPiOffsetsFile(donutOffsetsFilepath)
	if err != nil {
		return "", 0, err
	}

	preExistingDonutCount := len(donutOffsets)
	if preExistingDonutCount >= 16 {
		return "", 0, fmt.Errorf("tier %d already has 16 donuts", tierIndex)
	}

	folder := fmt.Sprintf("Donut%X", preExistingDonutCount)
	path := filepath.Join(tierFolderPath, folder)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return "", 0, err
	}
	donutOffsets = append(donutOffsets, piOffset)
	if len(donutOffsets) > 16 {
		panic("Too many donut offsets (limit 16)")
	}
	c.openTiers[tierIndex].donutOffsets = donutOffsets
	err = c.writePiOffsetsFile(donutOffsetsFilepath, donutOffsets)
	if err != nil {
		return "", 0, err
	}

	return path, byte(preExistingDonutCount + 1), nil
}

func (c *Cake) writePiOffsetsFile(filepath string, offsets []types.PiOffset) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	for i := range offsets {
		_, err = fmt.Fprintf(file, " %d\n", offsets[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Cake) readPiOffsetsFile(filepath string) ([]types.PiOffset, error) {
	file, err := os.Open(filepath)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		return nil, err
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(contents)
	var numbers []types.PiOffset
	for {
		var n int
		// Fscan reads the next ASCII number and parses it into n
		_, err := fmt.Fscan(buf, &n)
		if err != nil {
			if err == io.EOF {
				break // Reached the end of the slice cleanly
			}
			return nil, fmt.Errorf("%s parsing error: %w", filepath, err)
		}
		numbers = append(numbers, types.PiOffset(n))
	}
	return numbers, nil
}
