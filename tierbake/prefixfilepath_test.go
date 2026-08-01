package tierbake

import (
	"bytes"
	"testing"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

func TestCalculatePrefixPattern(t *testing.T) {
	const digitsPerFolder = byte(2)
	file, folders := CalculatePrefixPattern(0, digitsPerFolder)
	if file != 0 {
		t.Fatal("Expected file digits to be 0")
	}
	if !bytes.Equal(folders, []byte{0}) {
		t.Fatal("Expected folder digits to be a single 0")
	}

	file, folders = CalculatePrefixPattern(1, digitsPerFolder)
	if file != 1 {
		t.Fatal("Expected file digits to be 1")
	}
	if !bytes.Equal(folders, []byte{0}) {
		t.Fatal("Expected folder digits to be a single 0")
	}

	file, folders = CalculatePrefixPattern(2, digitsPerFolder)
	if file != 2 {
		t.Fatal("Expected file digits to be 2")
	}
	if !bytes.Equal(folders, []byte{0}) {
		t.Fatal("Expected folder digits to be a single 0")
	}

	file, folders = CalculatePrefixPattern(3, digitsPerFolder)
	if file != 2 {
		t.Fatal("Expected file digits to be 2")
	}
	if !bytes.Equal(folders, []byte{1}) {
		t.Fatal("Expected folder digits to be a single 1")
	}

	file, folders = CalculatePrefixPattern(4, digitsPerFolder)
	if file != 2 {
		t.Fatal("Expected file digits to be 2")
	}
	if !bytes.Equal(folders, []byte{2}) {
		t.Fatal("Expected folder digits to be a single 2")
	}

	file, folders = CalculatePrefixPattern(5, digitsPerFolder)
	if file != 2 {
		t.Fatal("Expected file digits to be 2")
	}
	if !bytes.Equal(folders, []byte{1, 2}) {
		t.Fatal("Expected folder digits to be (1,2)")
	}
}

func TestFormatFilePathFilename(t *testing.T) {
	const digitsPerFolder = byte(2)
	nibbleCount := byte(3)
	filenameDigits, foldersDigits := CalculatePrefixPattern(nibbleCount, digitsPerFolder)
	filename, folderName := FormatFilePathFilename([]shallowtreebyte.NibbleVal{0, 1, 2}, filenameDigits, foldersDigits)
	if filename != "12" {
		t.Fatal("Expected filename to be 12")
	}
	if folderName != "0/" {
		t.Fatal("Expected folderName to be 0/")
	}

	nibbleCount = byte(5)
	filenameDigits, foldersDigits = CalculatePrefixPattern(nibbleCount, digitsPerFolder)
	filename, folderName = FormatFilePathFilename([]shallowtreebyte.NibbleVal{1, 2, 3, 4, 5}, filenameDigits, foldersDigits)
	if filename != "45" {
		t.Fatal("Expected filename to be 45")
	}
	if folderName != "1/23/" {
		t.Fatal("Expected folderName to be 1/23/")
	}

	nibbleCount = byte(0)
	filenameDigits, foldersDigits = CalculatePrefixPattern(nibbleCount, digitsPerFolder)
	filename, folderName = FormatFilePathFilename([]shallowtreebyte.NibbleVal{}, filenameDigits, foldersDigits)
	if filename != "" {
		t.Fatal("Expected filename to be empty")
	}
	if folderName != "/" {
		t.Fatal("Expected folderName to be /")
	}

	nibbleCount = byte(1)
	filenameDigits, foldersDigits = CalculatePrefixPattern(nibbleCount, digitsPerFolder)
	filename, folderName = FormatFilePathFilename([]shallowtreebyte.NibbleVal{5}, filenameDigits, foldersDigits)
	if filename != "5" {
		t.Fatal("Expected filename to be 5")
	}
	if folderName != "/" {
		t.Fatal("Expected folderName to be /")
	}

}
