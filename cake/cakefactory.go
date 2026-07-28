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
	config          [5]smalltree.SmallTreeConfig
	hashBytesLength byte
}

func NewCakeFactory(folderPath string) *CakeFactory {
	result := CakeFactory{}
	result.folderPath = folderPath

	result.config[0].ReassuranceBytesCount = 2
	result.config[0].HashNibbleLength = 64 // For sha256
	result.config[0].NodeFormatSpecsPerLevel = 10
	result.config[0].NodeIdRWriter = smalltree.ID16[types.LocalNodeId]{}
	result.config[0].LocalPiRWriter = smalltree.ID16[types.LocalPi]{}
	result.config[0].PrefixIndexRWriter = smalltree.ID0[types.PrefixIndex]{}  // A read writer that writes no bytes!
	result.config[0].SuffixIndexRWriter = smalltree.ID24[types.SuffixIndex]{} // ToDo 24 not guaranteed enough

	result.config[1].ReassuranceBytesCount = 2
	result.config[1].HashNibbleLength = 64 // For sha256
	result.config[1].NodeFormatSpecsPerLevel = 10
	result.config[1].NodeIdRWriter = smalltree.ID16[types.LocalNodeId]{}
	result.config[1].LocalPiRWriter = smalltree.ID24[types.LocalPi]{}
	result.config[1].PrefixIndexRWriter = smalltree.ID8[types.PrefixIndex]{}
	result.config[1].SuffixIndexRWriter = smalltree.ID24[types.SuffixIndex]{}

	result.config[2].ReassuranceBytesCount = 2
	result.config[2].HashNibbleLength = 64 // For sha256
	result.config[2].NodeFormatSpecsPerLevel = 10
	result.config[2].NodeIdRWriter = smalltree.ID16[types.LocalNodeId]{}
	result.config[2].LocalPiRWriter = smalltree.ID24[types.LocalPi]{}
	result.config[2].PrefixIndexRWriter = smalltree.ID8[types.PrefixIndex]{}
	result.config[2].SuffixIndexRWriter = smalltree.ID24[types.SuffixIndex]{}

	result.config[3].ReassuranceBytesCount = 2
	result.config[3].HashNibbleLength = 64 // For sha256
	result.config[3].NodeFormatSpecsPerLevel = 10
	result.config[3].NodeIdRWriter = smalltree.ID16[types.LocalNodeId]{}
	result.config[3].LocalPiRWriter = smalltree.ID32[types.LocalPi]{}
	result.config[3].PrefixIndexRWriter = smalltree.ID16[types.PrefixIndex]{}
	result.config[3].SuffixIndexRWriter = smalltree.ID24[types.SuffixIndex]{}

	result.config[4].ReassuranceBytesCount = 2
	result.config[4].HashNibbleLength = 64 // For sha256
	result.config[4].NodeFormatSpecsPerLevel = 10
	result.config[4].NodeIdRWriter = smalltree.ID16[types.LocalNodeId]{}
	result.config[4].LocalPiRWriter = smalltree.ID32[types.LocalPi]{}
	result.config[4].PrefixIndexRWriter = smalltree.ID16[types.PrefixIndex]{}
	result.config[4].SuffixIndexRWriter = smalltree.ID24[types.SuffixIndex]{}

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

	// Initialize TierOffsets.txt
	ti := TiersInfo{}
	err = ti.ToDisk(cf.folderPath)
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
	result.config[0] = &cf.config[0]
	result.config[1] = &cf.config[1]
	result.config[2] = &cf.config[2]
	result.config[3] = &cf.config[3]
	result.config[4] = &cf.config[4]
	result.hashBytesLength = cf.hashBytesLength

	// Read the tier presentation index offsets from TierOffsets.txt
	err := result.tiersInfo.FromDisk(result.folderPath)
	if err != nil {
		return nil, err
	}

	result.openInputTier, err = inputtier.OpenInputTier(cf.folderPath, &cf.config[0], result.tiersInfo.inputOffset, cf.hashBytesLength)
	if err != nil {
		return nil, err
	}

	for t := range 5 {
		if result.tiersInfo.present[t] {
			result.openTiers[t], err = result.openTier(byte(t), result.tiersInfo.offset[t])
			if err != nil {
				return nil, err
			}
		} else {
			result.openTiers[t] = nil
		}
	}

	return &result, nil
}
