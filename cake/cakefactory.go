package cake

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

type CakeFactory struct {
	folderPath string
	config     smalltree.SmallTreeConfig
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
	return &result
}

func (cf *CakeFactory) Exists() bool {
	filePath := filepath.Join(cf.folderPath, "Offset.txt")
	file, err := os.Open(filePath)
	if err == nil {
		_ = file.Close()
		return true
	}
	return false
}

func (cf CakeFactory) Create() error {
	filePath := filepath.Join(cf.folderPath, "Offset.txt")
	file, err := os.Create(filePath)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		return err
	}
	// Single ascii 0
	zero := [1]byte{'0'}
	_, err = file.Write(zero[:])
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// Empty file
	filePath = filepath.Join(cf.folderPath, "HashesOrder.bin")
	file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// Empty file in folder
	folderPath := filepath.Join(cf.folderPath, "HashPrefix")
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return err
	}
	filePath = filepath.Join(folderPath, "HashSuffix.bin")
	file, err = os.Create(filePath)
	if err != nil {
		return err
	}
	err = file.Close()
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

	return &result, nil
}
