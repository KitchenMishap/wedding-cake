package cake

import (
	"os"
	"testing"
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
