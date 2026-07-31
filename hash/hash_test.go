package hash

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func TestPrefixSuffixDisk(tes *testing.T) {
	hashByteLengths := []byte{20, 32, 64}
	prefixNibbleLengths := []byte{0, 1, 2, 3, 4, 8, 15, 16}

	err := os.MkdirAll("Temp_Testing", 0755)
	if err != nil {
		tes.Errorf("Failed to create directory: %v", err)
	}

	for _, hashByteLength := range hashByteLengths {
		for _, prefixNibbleLength := range prefixNibbleLengths {

			hashArray := helperRandomHash()
			hash := Full{}
			hash.Init(hashByteLength)
			hash.SetFromArray(hashArray)

			hashFile, err := os.Create("Temp_Testing/Hash")
			if err != nil {
				tes.Errorf("Failed to create file: %v", err)
			}
			prefixFile, err := os.Create("Temp_Testing/Prefix")
			if err != nil {
				tes.Errorf("Failed to create file: %v", err)
			}
			suffixFile, err := os.Create("Temp_Testing/Suffix")
			if err != nil {
				tes.Errorf("Failed to create file: %v", err)
			}

			err = hash.Write(hashFile)
			if err != nil {
				tes.Errorf("Failed to write hash to file: %v", err)
			}

			// Split into prefix and suffix
			prefix := Prefix{}
			suffix := Suffix{}
			hash.ExtractPrefixSuffix(&prefix, &suffix, prefixNibbleLength)

			// Write and read back
			spareNibble := byte(0xA)
			err = prefix.Write(prefixFile, spareNibble)
			if err != nil {
				tes.Errorf("Failed to write prefix to file: %v", err)
			}
			err = suffix.Write(suffixFile, spareNibble)
			if err != nil {
				tes.Errorf("Failed to write suffix to file: %v", err)
			}
			err = prefixFile.Close()
			if err != nil {
				tes.Errorf("Failed to close prefix file: %v", err)
			}
			err = suffixFile.Close()
			if err != nil {
				tes.Errorf("Failed to close suffix file: %v", err)
			}
			prefixFile2, err := os.Open("Temp_Testing/Prefix")
			if err != nil {
				tes.Errorf("Failed to re-open prefix file: %v", err)
			}
			suffixFile2, err := os.Open("Temp_Testing/Suffix")
			if err != nil {
				tes.Errorf("Failed to re-open suffix file: %v", err)
			}
			prefix2 := Prefix{}
			prefix2.Init(hashByteLength, prefixNibbleLength)
			suffix2 := Suffix{}
			suffix2.Init(hashByteLength, prefixNibbleLength)
			err = prefix2.Read(prefixFile2)
			if err != nil {
				tes.Errorf("Failed to read prefix from file: %v", err)
			}
			exists, spare := prefix2.LastReadContainedSpareNibble()
			if exists && spare != spareNibble {
				tes.Fatal("Spare nibble mismatch")
			}
			err = suffix2.Read(suffixFile2)
			if err != nil {
				tes.Errorf("Failed to read suffix from file: %v", err)
			}
			exists, spare = suffix2.LastReadContainedSpareNibble()
			if exists && spare != spareNibble {
				tes.Fatal("Spare nibble mismatch")
			}
			hash2 := Full{}
			prefix2.AppendSuffix(&hash2, &suffix2)

			if !hash.Equal(&hash2) {
				tes.Errorf("Hash does not match: %d bytes hash, %d nibble prefix (%d byte prefix)", hashByteLength, prefixNibbleLength, prefixNibbleLength/2)
				fmt.Println(hash.bytes)
				fmt.Println(hash2.bytes)
			} else {
				fmt.Printf("Success: %d bytes hash, %d nibble prefix (%d byte prefix)\n", hashByteLength, prefixNibbleLength, prefixNibbleLength/2)
			}
		}
	}
}

func TestAppendPrefixSuffix(tes *testing.T) {
	hashByteLengths := []byte{20, 32, 64}
	prefixNibbleLengths := []byte{0, 1, 2, 3, 4, 8, 15, 16}

	err := os.MkdirAll("Temp_Testing", 0755)
	if err != nil {
		tes.Errorf("Failed to create directory: %v", err)
	}

	for _, hashByteLength := range hashByteLengths {
		for _, prefixNibbleLength := range prefixNibbleLengths {

			hashArray := helperRandomHash()
			hash := Full{}
			hash.Init(hashByteLength)
			hash.SetFromArray(hashArray)

			// Split into prefix and suffix
			prefix := Prefix{}
			suffix := Suffix{}
			hash.ExtractPrefixSuffix(&prefix, &suffix, prefixNibbleLength)

			hash2 := Full{}
			prefix.AppendSuffix(&hash2, &suffix)

			if !hash.Equal(&hash2) {
				tes.Errorf("Hash does not match: %d bytes hash, %d nibble prefix (%d byte prefix)", hashByteLength, prefixNibbleLength, prefixNibbleLength/2)
				fmt.Println(hash.bytes)
				fmt.Println(hash2.bytes)
			} else {
				fmt.Printf("Success: %d bytes hash, %d nibble prefix (%d byte prefix)\n", hashByteLength, prefixNibbleLength, prefixNibbleLength/2)
			}
		}
	}
}

func TestPrefixAsNumber(tes *testing.T) {
	hashByteLengths := []byte{20, 32, 64}
	prefixNibbleLengths := []byte{0, 1, 2, 3, 4, 8, 15}

	err := os.MkdirAll("Temp_Testing", 0755)
	if err != nil {
		tes.Errorf("Failed to create directory: %v", err)
	}

	for _, hashByteLength := range hashByteLengths {
		for _, prefixNibbleLength := range prefixNibbleLengths {

			hashArray := helperRandomHash()
			hash := Full{}
			hash.Init(hashByteLength)
			hash.SetFromArray(hashArray)

			// Split into prefix and suffix
			prefix := Prefix{}
			suffix := Suffix{}
			hash.ExtractPrefixSuffix(&prefix, &suffix, prefixNibbleLength)

			// Store and recall prefix as number
			number := prefix.PrefixAsNumber()
			prefix2 := Prefix{}
			prefix2.Init(hashByteLength, prefixNibbleLength)
			prefix2.SetPrefixFromNumber(number)
			hash2 := Full{}
			prefix2.AppendSuffix(&hash2, &suffix)

			if !hash.Equal(&hash2) {
				tes.Errorf("Hash does not match: %d bytes hash, %d nibble prefix (%d byte prefix)", hashByteLength, prefixNibbleLength, prefixNibbleLength/2)
				fmt.Println(hash.bytes)
				fmt.Println(hash2.bytes)
			} else {
				fmt.Printf("Success: %d bytes hash, %d nibble prefix (%d byte prefix)\n", hashByteLength, prefixNibbleLength, prefixNibbleLength/2)
			}
		}
	}
}

func helperRandomHash() *[MaxHashBytes]byte {
	result := [MaxHashBytes]byte{}
	for i := range MaxHashBytes {
		result[i] = byte(rand.Intn(256))
	}
	return &result
}
