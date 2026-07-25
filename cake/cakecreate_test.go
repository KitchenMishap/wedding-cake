package cake

import (
	"bytes"
	"math/rand"
	"os"
	"testing"

	"github.com/kitchenmishap/wedding-cake/inputtierbake"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

func TestCakeCreate(t *testing.T) {
	testDir := "Temp_Testing"
	_ = os.RemoveAll(testDir) // Ignore error if it doesn't exist yet
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	folderPath := "Temp_Testing"
	cf := NewCakeFactory(folderPath)

	if cf.Exists() {
		t.Fatal("Cake should not exist yet")
	}
	err = cf.Create()
	if err != nil {
		t.Fatal(err)
	}
	if !cf.Exists() {
		t.Fatal("Cake should exist now")
	}
	cake, err := cf.Open()
	if err != nil {
		t.Fatal(err)
	}
	err = cake.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCakeAppend(t *testing.T) {
	testDir := "Temp_Testing"
	_ = os.RemoveAll(testDir) // Ignore error if it doesn't exist yet
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	folderPath := "Temp_Testing"
	cf := NewCakeFactory(folderPath)

	if cf.Exists() {
		t.Fatal("Cake should not exist yet")
	}
	err = cf.Create()
	if err != nil {
		t.Fatal(err)
	}
	if !cf.Exists() {
		t.Fatal("Cake should exist now")
	}
	cake, err := cf.Open()
	if err != nil {
		t.Fatal(err)
	}

	hashes := make([][]byte, 9)
	pis := make([]types.GlobalPi, 9)

	j := 0
	for i := 0; i < 10; i++ {
		hash := helperRandomHashByte(32)
		if i != 5 { // Try a gap
			hashes[j] = hash
			pis[j] = types.GlobalPi(i)
			err = cake.AppendHash(types.GlobalPi(i), hash)
			if err != nil {
				t.Fatal(err)
			}
			j++
		}
	}

	err = cake.Close()
	if err != nil {
		t.Fatal(err)
	}

	cake, err = cf.Open()
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 9; i++ {
		hashExpected := hashes[i]
		piExpected := pis[i]

		piFound := cake.inputTier.LookupHash(hashExpected)
		if piFound != piExpected {
			t.Fatal("Hash lookup failed")
		}
		hashFound, ok := cake.inputTier.GetHashAtIndex(piExpected)
		if !ok {
			t.Fatal("GetHashAtIndex failed")
		}
		if !bytes.Equal(hashExpected, hashFound) {
			t.Fatal("GetHashAtIndex mismatch")
		}
	}

	err = cake.Close()
	if err != nil {
		t.Fatal(err)
	}

	tier, err := inputtierbake.FreezeInputForBaking(folderPath, smalltree.ID16[types.LocalPi]{}, 32)
	if err != nil {
		t.Fatal(err)
	}
	err = tier.Close()
	if err != nil {
		t.Fatal(err)
	}

}

func helperRandomHashByte(hashLength int) []byte {
	result := [64]byte{}
	for i := range hashLength {
		result[i] = byte(rand.Intn(256))
	}
	return result[:hashLength]
}
