package cake

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/forest"
	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/inputtierbake"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

type Cake struct {
	folderPath      string
	config          *smalltree.SmallTreeConfig
	hashBytesLength byte

	inputTierPiWriter smalltree.NByteIdConfig[types.LocalPi]
	inputTier         *inputtier.InputTier
	inputTierOffset   types.PiOffset

	tierOffsets []types.PiOffset
	tiers       []*Tier
}

func (c *Cake) Close() error {
	err := c.inputTier.Close()
	if err != nil {
		return err
	}

	for t := range c.tiers {
		err = c.tiers[t].Close()
		if err != nil {
			return err
		}
		c.tiers[t] = nil
	}

	// Prevent this cake from being used
	c.inputTier = nil
	c.tierOffsets = nil

	return nil
}

func (c *Cake) AppendHash(gpi types.GlobalPi, hash []byte) error {
	err := c.inputTier.AppendHash(gpi, hash)
	if err != nil {
		return err
	}
	if c.inputTier.CountPresentationIndices() == 65535 {
		path, offset, err := c.FreezeInputTierForBaking()
		if err != nil {
			return err
		}
		donutPath, err := c.BakeFrozenInputTier(path, offset)
		if err != nil {
			return err
		}
		err = c.IceTheDonut(donutPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Cake) LookupHash(hash []byte) (types.GlobalPi, error) {
	res := c.inputTier.LookupHash(hash)
	if res != types.GlobalPresentationIndexNoMatch {
		return res, nil
	}
	nibbles := make([]shallowtreebyte.NibbleVal, c.hashBytesLength*2)
	for i := 0; i < int(c.hashBytesLength); i++ {
		nibbles[i*2] = shallowtreebyte.NibbleVal(hash[i] >> 4)     // MS
		nibbles[i*2+1] = shallowtreebyte.NibbleVal(hash[i] & 0x0F) // LS
	}
	for t := range c.tiers {
		res, err := c.tiers[t].LookupHash(nibbles)
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
	piCount := c.inputTier.CountPresentationIndices()
	newPiOffset := c.inputTierOffset + types.PiOffset(piCount)
	frozenOffset := c.inputTierOffset

	numberedFolderName := fmt.Sprintf("InputTier_%d", c.inputTierOffset)
	numberedFolderPath := filepath.Join(c.folderPath, numberedFolderName)

	err := inputtier.CreateInputTierFiles(c.folderPath, newPiOffset)
	if err != nil {
		return "", 0, err
	}

	c.inputTierOffset = newPiOffset
	offsets := make([]types.PiOffset, 1, len(c.tierOffsets)+1)
	offsets[0] = newPiOffset
	offsets = append(offsets, c.tierOffsets...)
	filePath := filepath.Join(c.folderPath, "TierOffsets.txt")
	err = c.writePiOffsetsFile(filePath, offsets)
	if err != nil {
		return "", 0, err
	}

	err = c.inputTier.Close()
	if err != nil {
		return "", 0, err
	}
	c.inputTier, err = inputtier.OpenInputTier(c.folderPath, c.inputTierPiWriter, newPiOffset, c.hashBytesLength)
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

func (c *Cake) BakeFrozenInputTier(numberedPath string, offset types.PiOffset) (string, error) {
	donutPath, err := c.newDonutFolder(0, offset)
	if err != nil {
		return "", err
	}
	err = inputtierbake.BakeFrozenInputTierToDonutFolder(numberedPath, donutPath)
	if err != nil {
		return "", err
	}
	err = os.RemoveAll(numberedPath)
	if err != nil {
		return "", err
	}
	return donutPath, nil
}

func (c *Cake) IceTheDonut(donutPath string) error {
	icingPath := filepath.Join(donutPath, "Icing")
	fw := forest.NewForestWrite(icingPath)
	factory := smalltree.NewLevelsCodecNfFactory(c.config)
	enc := factory.MakeLevelsEncoder()
	err := fw.StartWrite()
	if err != nil {
		return err
	}
	// ToDo more than one prefix (for tiers >0)
	prefixNibbles := shallowtreebyte.NibbleIndex(0)
	for prefixIndex := forest.PrefixIndexType(0); prefixIndex < 1; prefixIndex++ {
		suffixFilePath := filepath.Join(donutPath, "HashPrefix", "HashSuffix.bin")
		file, err := os.Open(suffixFilePath)
		defer func() { _ = file.Close() }()
		if err != nil {
			return err
		}
		contents, err := io.ReadAll(file)
		if err != nil {
			return err
		}
		byteCount := c.inputTierPiWriter.StorageBytes()
		entrySize := int(c.hashBytesLength) + byteCount
		entryCount := len(contents) / entrySize
		input := make([]shallowtreebyte.HashPi, entryCount)
		for i := 0; i < entryCount; i++ {
			start := i * entrySize
			nibbles := make([]shallowtreebyte.NibbleVal, c.hashBytesLength*2)
			for b := 0; b < int(c.hashBytesLength); b++ {
				nibbles[b*2] = shallowtreebyte.NibbleVal(contents[start+b] >> 4)     // MS
				nibbles[b*2+1] = shallowtreebyte.NibbleVal(contents[start+b] & 0x0F) // LS
			}
			input[i] = shallowtreebyte.HashPi{
				Hash:              nibbles,
				PresentationIndex: c.inputTierPiWriter.ReadID(contents[start+int(c.hashBytesLength) : start+int(c.hashBytesLength)+byteCount]),
			}
		}
		st := shallowtreebyte.GenerateShallowTree(input, prefixNibbles, shallowtreebyte.NibbleIndex(c.hashBytesLength*2), shallowtreebyte.ByteIndex(c.config.ReassuranceBytesCount), 0)
		tf := smalltree.DesignTreeFormat(st, c.config)
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
	return nil
}

func (c *Cake) openTier(tierIndex byte, piOffset types.PiOffset,
	localPiWriter smalltree.NByteIdConfig[types.LocalPi]) (*Tier, error) {
	tierFolderPath := filepath.Join(c.folderPath, fmt.Sprintf("Tier%d_%d", tierIndex, piOffset))
	donutOffsetsFilePath := filepath.Join(tierFolderPath, "DonutOffsets.txt")
	donutOffsets, err := c.readPiOffsetsFile(donutOffsetsFilePath)
	if err != nil {
		return nil, err
	}

	tier := Tier{}
	tier.folderPath = tierFolderPath
	tier.localPiWriter = localPiWriter

	for d := range donutOffsets {
		donut, err := c.openDonut(tierIndex, tierFolderPath, d)
		if err != nil {
			return nil, err
		}
		tier.donuts = append(tier.donuts, donut)
		tier.offsets = append(tier.offsets, donutOffsets[d])
	}
	return &tier, nil
}

func (c *Cake) openDonut(tierIndex byte, tierFolderPath string, donutIndex int) (*forest.ForestRead, error) {
	donutIcingPath := filepath.Join(tierFolderPath, fmt.Sprintf("Donut%X", donutIndex), "Icing")
	fr := forest.NewForestRead(donutIcingPath, tierIndex, c.config)
	err := fr.Open()
	if err != nil {
		return nil, err
	}
	return fr, nil
}

func (c *Cake) newDonutFolder(tierIndex int, piOffset types.PiOffset) (string, error) {
	if len(c.tierOffsets) < tierIndex+1 {
		// New tier folder
		tierFolder := fmt.Sprintf("Tier%d_%d", tierIndex, piOffset)
		tierFolderPath := filepath.Join(c.folderPath, tierFolder)
		err := os.MkdirAll(tierFolderPath, 0755)
		if err != nil {
			return "", err
		}
		c.tierOffsets = append(c.tierOffsets, piOffset)
		filePath := filepath.Join(c.folderPath, "TierOffsets.txt")
		err = c.writePiOffsetsFile(filePath, c.tierOffsets)
		if err != nil {
			return "", err
		}
		// New DonutOffsets.txt file
		donutOffsetsFilePath := filepath.Join(tierFolderPath, "DonutOffsets.txt")
		donutOffsets := make([]types.PiOffset, 0)
		err = c.writePiOffsetsFile(donutOffsetsFilePath, donutOffsets)
		if err != nil {
			return "", err
		}
	}
	// Read DonutOffsets.txt file
	tierFolder := fmt.Sprintf("Tier%d_%d", tierIndex, c.tierOffsets[tierIndex])
	tierFolderPath := filepath.Join(c.folderPath, tierFolder)
	donutOffsetsFilepath := filepath.Join(tierFolderPath, "DonutOffsets.txt")
	donutOffsets, err := c.readPiOffsetsFile(donutOffsetsFilepath)
	if err != nil {
		return "", err
	}

	donutCount := len(donutOffsets)
	if donutCount >= 16 {
		return "", fmt.Errorf("tier %d already has 16 donuts", tierIndex)
	}

	folder := fmt.Sprintf("Donut%X", donutCount)
	path := filepath.Join(tierFolderPath, folder)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		return "", err
	}
	donutOffsets = append(donutOffsets, piOffset)
	err = c.writePiOffsetsFile(donutOffsetsFilepath, donutOffsets)
	if err != nil {
		return "", err
	}

	return path, nil
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
