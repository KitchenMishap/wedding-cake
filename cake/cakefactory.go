package cake

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

type CakeFactory struct {
	folderPath      string
	config          smalltree.SmallTreeConfig
	hashBytesLength byte
}

func NewCakeFactory(folderPath string) *CakeFactory {
	result := CakeFactory{}
	result.folderPath = folderPath
	// ToDo these will probably need per-tier configuration
	result.config.ReassuranceBytesCount = 2
	result.config.HashNibbleLength = 64 // For sha256
	result.config.NodeFormatSpecsPerLevel = 10
	result.config.NodeIdConfig = smalltree.ID16[types.LocalNodeId]{}
	result.config.HashIndexIdConfig = smalltree.ID24[types.LocalPi]{}
	result.hashBytesLength = 32
	return &result
}

func (cf *CakeFactory) Exists() bool {
	filePath := filepath.Join(cf.folderPath, "TierOffsets.txt")
	file, err := os.Open(filePath)
	if err == nil {
		_ = file.Close()
		return true
	}
	return false
}

func (cf *CakeFactory) Create() error {
	piOffset := types.PiOffset(0)
	err := inputtier.CreateInputTierFiles(cf.folderPath, piOffset)
	if err != nil {
		return err
	}

	// Initialize TierOffsets.txt with a zero for the new lone InputTier
	filePath := filepath.Join(cf.folderPath, "TierOffsets.txt")
	file, err := os.Create(filePath)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		return err
	}
	// ascii decimal offset
	_, err = fmt.Fprintf(file, "%d", piOffset)
	if err != nil {
		return err
	}

	return nil
}

func (cf *CakeFactory) Open() (*Cake, error) {
	if !cf.Exists() {
		return nil, errors.New("cake does not exist")
	}
	result := Cake{}
	result.folderPath = cf.folderPath
	result.config = &cf.config
	result.hashBytesLength = cf.hashBytesLength

	// Read the tier presentation index offsets from TierOffsets.txt
	fName := filepath.Join(cf.folderPath, "TierOffsets.txt")
	offsets, err := result.readPiOffsetsFile(fName)
	if err != nil {
		return nil, err
	}
	if len(offsets) == 0 {
		return nil, fmt.Errorf("%s parsing error: expected at least 1 offset", fName)
	}
	result.inputTierOffset = offsets[0]
	result.tierOffsets = offsets[1:]

	// InputTier is CURRENTLY configured with 16 bid LocalPi's
	result.inputTierPiWriter = smalltree.ID16[types.LocalPi]{}

	result.inputTier, err = inputtier.OpenInputTier(cf.folderPath, result.inputTierPiWriter, result.inputTierOffset, cf.hashBytesLength)
	if err != nil {
		return nil, err
	}

	for t := range result.tierOffsets {
		tier, err := result.openTier(byte(t), result.tierOffsets[t], result.inputTierPiWriter)
		if err != nil {
			return nil, err
		}
		result.tiers = append(result.tiers, tier)
	}

	return &result, nil
}
