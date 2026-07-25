package cake

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

type CakeFactory struct {
	folderPath      string
	config          smalltree.SmallTreeConfig
	hashBytesLength int
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
	filePath := filepath.Join(cf.folderPath, "InputTierOffset.txt")
	file, err := os.Open(filePath)
	if err == nil {
		_ = file.Close()
		return true
	}
	return false
}

func (cf CakeFactory) Create() error {
	err := inputtier.CreateInputTierFiles(cf.folderPath, 0)
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

	// InputTier is CURRENTLY configured with 16 bid LocalPi's
	var err error
	result.inputTier, err = inputtier.OpenInputTier(cf.folderPath, smalltree.ID16[types.LocalPi]{}, cf.hashBytesLength)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
