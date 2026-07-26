package tierbake

import (
	"io"
	"os"
	"path/filepath"
)

func BakeFrozenInputTierToDonutFolder(numberedFolderPath string, tier0DonutFolder string) error {
	// Check for READONLY file
	fNameRo := filepath.Join(numberedFolderPath, "READONLY")
	_, err := os.Stat(fNameRo)
	if err != nil {
		panic("Input tier folder should have READONLY flag file")
	}

	// Mark the folder as being baked with a BAKING flag file
	bakingFlagFileName := filepath.Join(tier0DonutFolder, "BAKING")
	file, err := os.Create(bakingFlagFileName)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	folderName := filepath.Join(tier0DonutFolder, "HashPrefix")
	err = os.MkdirAll(folderName, 0755)
	if err != nil {
		return err
	}

	// The new donut (folder) has the same offset as the input tier at the time of baking.
	// Therefore the IDs in te HashesOrder.bin and HashSuffix.bin remain the same.
	// The files just need to be copied
	sourceFile := filepath.Join(numberedFolderPath, "HashesOrder.bin")
	destFile := filepath.Join(tier0DonutFolder, "HashesOrder.bin")
	err = copyFileStream(sourceFile, destFile)
	if err != nil {
		return err
	}

	sourceFile = filepath.Join(numberedFolderPath, "HashPrefix", "HashSuffix.bin")
	destFile = filepath.Join(tier0DonutFolder, "HashPrefix", "HashSuffix.bin")
	err = copyFileStream(sourceFile, destFile)
	if err != nil {
		return err
	}

	// It is now baked. Remove the flag file
	err = os.Remove(bakingFlagFileName)
	if err != nil {
		return err
	}

	return nil
}

func copyFileStream(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	// Streams chunks from source to destination without loading everything into memory
	_, err = io.Copy(out, in)
	return err
}
