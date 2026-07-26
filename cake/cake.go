package cake

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/inputtierbake"
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
}

func (c *Cake) Close() error {
	err := c.inputTier.Close()
	if err != nil {
		return err
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
	return nil
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

func (c *Cake) BakeFrozenInputTier(numberedPath string, offset types.PiOffset) error {
	donutPath, err := c.newDonutFolder(0, offset)
	if err != nil {
		return err
	}
	err = inputtierbake.BakeFrozenInputTierToDonutFolder(numberedPath, donutPath)
	if err != nil {
		return err
	}
	err = os.RemoveAll(numberedPath)
	if err != nil {
		return err
	}
	return nil
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
