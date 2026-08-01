package cake

import (
	"fmt"
	"math/rand"
	"os"
	"testing"

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

	cf := NewCakeFactory(testDir)

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

	const count = 65535*257 + 1000

	hashes := make([][]byte, count)
	pis := make([]types.GlobalPi, count)

	for i := 0; i < count; i++ {
		hash := helperRandomHashByte(32)
		if i == 0 {
			hash[0] = 0x12 // Useful for debug
			hash[1] = 0x34
			hash[2] = 0x56
			hash[3] = 0x78
		}
		hashes[i] = hash
		pis[i] = types.GlobalPi(i)
		err = cake.AppendHash(types.GlobalPi(i), hash)
		if err != nil {
			t.Fatal(err)
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

	fmt.Printf("Checking %d hashes...\n", count)
	for i := 0; i < count; i++ {
		if i%10000 == 0 {
			fmt.Printf("\t%.1f%%\n", float64(i)/float64(count)*100)
		}
		hashExpected := hashes[i]
		piExpected := pis[i]

		piFound, err := cake.LookupHash(hashExpected)
		if err != nil {
			t.Fatal(err)
		}
		if piFound == types.GlobalPresentationIndexNoMatch {
			t.Fatal("Hash not found")
		}
		if piFound != piExpected {
			t.Fatal("Hash lookup mismatch")
		}
	}

	/*
		path, offset, donutOffsets, err := cake.FreezeTierForBaking(0)
		if err != nil {
			t.Fatal(err)
		}

		sourceLocalPiWriter := smalltree.ID16[types.LocalPi]{}
		destLocalPiWriter := smalltree.ID24[types.LocalPi]{}

		donutPath, _, err := cake.BakeFrozenTier(path, offset, donutOffsets, 0,
			sourceLocalPiWriter, destLocalPiWriter)
		fmt.Println(donutPath)
		if err != nil {
			t.Fatal(err)
		}*/

	err = cake.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCakeNoCheck(t *testing.T) {
	testDir := "Temp_Testing"
	_ = os.RemoveAll(testDir) // Ignore error if it doesn't exist yet
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	cf := NewCakeFactory(testDir)

	err = cf.Create()
	if err != nil {
		t.Fatal(err)
	}
	cake, err := cf.Open()
	if err != nil {
		t.Fatal(err)
	}

	const count = 65535*257 + 1000

	for i := 0; i < count; i++ {
		if i%65535 == 0 {
			fmt.Printf("%d\n", i/65535)
		}
		hash := helperRandomHashByte(32)
		pi, err := cake.LookupHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if pi != types.GlobalPresentationIndexNoMatch {
			t.Fatal("Hash should not be found yet")
		}
		if i == 1048559 {
			fmt.Println("i==1048559")
		}
		err = cake.AppendHash(types.GlobalPi(i), hash)
		if err != nil {
			t.Fatal(err)
		}
		pi, err = cake.LookupHash(hash)
		if err != nil {
			t.Fatal(err)
		}
		if pi == types.GlobalPresentationIndexNoMatch {
			t.Fatal("Hash should be found now")
		}
	}

	err = cake.Close()
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
