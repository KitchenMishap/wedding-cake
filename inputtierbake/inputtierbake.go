package inputtierbake

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kitchenmishap/wedding-cake/inputtier"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

// FreezeInputForBaking renames the InputTier folder and creates a new one.
// InputTier must be closed before calling this.
// Returns a new opened InputTier
func FreezeInputForBaking(cakeFolderPath string,
	localPiWriter smalltree.NByteIdConfig[types.LocalPi],
	hashBytesLength int) (*inputtier.InputTier, error) {

	originalPi, err := readOffsetFromFile(cakeFolderPath)
	if err != nil {
		return nil, err
	}
	numberedFolderName := fmt.Sprintf("InputTier_%d", originalPi)
	numberedFolderPath := filepath.Join(cakeFolderPath, numberedFolderName)
	piCount, err := countPisInFile(numberedFolderPath, localPiWriter)
	if err != nil {
		return nil, err
	}
	newPi := originalPi + types.PiOffset(piCount)

	err = inputtier.CreateInputTierFiles(cakeFolderPath, newPi)
	if err != nil {
		return nil, err
	}

	tier, err := inputtier.OpenInputTier(cakeFolderPath, localPiWriter, hashBytesLength)
	if err != nil {
		return nil, err
	}
	return tier, nil
}

func readOffsetFromFile(cakeFolderPath string) (types.PiOffset, error) {
	// Read the previous offset from the file
	offsetFName := filepath.Join(cakeFolderPath, "InputTierOffset.txt")
	offsetFile, err := os.Open(offsetFName)
	defer func() { _ = offsetFile.Close() }()
	if err != nil {
		return 0, err
	}
	offsetBytes, err := io.ReadAll(offsetFile)
	if err != nil {
		return 0, err
	}
	offset, err := strconv.Atoi(string(offsetBytes))
	if err != nil {
		return 0, err
	}
	return types.PiOffset(offset), nil
}

func countPisInFile(numberedFolderPath string, localPiWriter smalltree.NByteIdConfig[types.LocalPi]) (uint64, error) {
	// Count the pi's represented in the original input tier
	orderFName := filepath.Join(numberedFolderPath, "HashesOrder.bin")
	offsetFile, err := os.Open(orderFName)
	defer func() { _ = offsetFile.Close() }()
	if err != nil {
		return 0, err
	}
	info, err := offsetFile.Stat()
	if err != nil {
		return 0, err
	}
	return uint64(info.Size() / int64(localPiWriter.StorageBytes())), nil
}
