package tierbake

import (
	"fmt"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

func CalculatePrefixPattern(prefixNibblesDigits byte, digitsPerFolder byte) (byte, []byte) {
	remainingDigits := prefixNibblesDigits

	// As a first step, we allocate up to digitsPerFolder digits to the filename
	resultFilenameDigits := digitsPerFolder
	if remainingDigits < digitsPerFolder {
		resultFilenameDigits = remainingDigits
	}
	remainingDigits -= resultFilenameDigits

	// Secondly, we add a few digits to each folder level until none ar left
	resultFolderDigits := []byte{0} // To start with, one folder level is up for negotation
	for remainingDigits > 0 {
		nextLevelFolderDigits := digitsPerFolder
		if remainingDigits < digitsPerFolder {
			nextLevelFolderDigits = remainingDigits
		}
		resultFolderDigits[0] = nextLevelFolderDigits
		remainingDigits -= nextLevelFolderDigits

		if remainingDigits > 0 {
			// Add a new folder level at the start for negotiation
			resultFolderDigits = append([]byte{0}, resultFolderDigits...)
		}
	}

	return resultFilenameDigits, resultFolderDigits
}

func formatNibble(nibble shallowtreebyte.NibbleVal) string {
	return fmt.Sprintf("%01X", nibble)
}

func formatFilePathFilename(prefixNibbles []shallowtreebyte.NibbleVal, filenameDigits byte, folderNameDigits []byte) (string, string) {
	length := len(prefixNibbles)
	if length == 0 {
		return "", "/"
	}
	filename := ""
	firstFilenamePrefixNibbleIndex := length - int(filenameDigits)
	for digit := range int(filenameDigits) {
		filename += formatNibble(prefixNibbles[firstFilenamePrefixNibbleIndex+digit])
	}
	folderName := ""
	digit := 0
	for folderNum := range len(folderNameDigits) {
		for range folderNameDigits[folderNum] {
			folderName += formatNibble(prefixNibbles[digit])
			digit++
		}
		folderName += "/"
	}
	return filename, folderName
}
