package inputtier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/types"
)

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

	// Now we have created the folder successfully, write the offset to a parent index file
	filePath = filepath.Join(cakeFolderPath, "InputTierOffset.txt")
	file, err = os.Create(filePath)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		return err
	}
	// ascii decimal offset
	_, err = fmt.Fprintf(file, "%d", offset)
	if err != nil {
		return err
	}

	return nil
}
