package inputtier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/types"
)

// CreateInputTierFiles Note that TierOffsets.txt will need updating after this call
func CreateInputTierFiles(cakeFolderPath string, offset types.PiOffset) error {
	folderName := fmt.Sprintf("InputTier_%d", offset)
	folderPath := filepath.Join(cakeFolderPath, folderName)
	err := os.MkdirAll(folderPath, 0755)
	if err != nil {
		return err
	}

	// Empty file
	filePath := filepath.Join(cakeFolderPath, folderName, "HashesOrder.bin")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	// Empty file in folder
	folderPath = filepath.Join(cakeFolderPath, folderName, "HashPrefix")
	err = os.MkdirAll(folderPath, 0755)
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
