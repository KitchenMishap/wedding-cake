package cake

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/forest"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

type Tier struct {
	folderPath    string
	localPiWriter smalltree.NByteIdConfig[types.LocalPi]
	donuts        []*forest.ForestRead
	offsets       []types.PiOffset
}

func (t *Tier) LookupHash(hash []shallowtreebyte.NibbleVal) (types.GlobalPi, error) {
	for d := range t.donuts {
		res := t.donuts[d].Lookup(hash)
		if res != types.LocalPiNoMatch {
			verified, err := t.VerifyHash(hash, res, d)
			if err != nil {
				return 0, err
			}
			if verified {
				return res.ToGlobalPi(t.offsets[d]), nil
			}
		}
	}
	return types.GlobalPresentationIndexNoMatch, nil
}

func (t *Tier) VerifyHash(hash []shallowtreebyte.NibbleVal, candidatePi types.LocalPi, donutIndex int) (bool, error) {
	// Todo: Implement this for tiers other than zero

	// First look in HashesOrder.bin for an index into the other file
	hashOrderPath := filepath.Join(t.folderPath, fmt.Sprintf("Donut%X", donutIndex), "HashesOrder.bin")
	file, err := os.Open(hashOrderPath)
	defer func() { _ = file.Close() }()
	if err != nil {
		return false, err
	}
	byteCount := t.localPiWriter.StorageBytes()
	_, err = file.Seek(int64(candidatePi)*int64(byteCount), 0)
	if err != nil {
		return false, err
	}
	bytes := make([]byte, byteCount)
	_, err = file.Read(bytes)
	if err != nil {
		return false, err
	}
	index := t.localPiWriter.ReadID(bytes)

	// Now look in Suffix file
	suffixPath := filepath.Join(t.folderPath, fmt.Sprintf("Donut%X", donutIndex), "HashPrefix", "HashSuffix.bin")
	file, err = os.Open(suffixPath)
	defer func() { _ = file.Close() }()
	if err != nil {
		return false, err
	}
	fieldSize := len(hash)/2 + byteCount
	_, err = file.Seek(int64(index)*int64(fieldSize), 0)
	if err != nil {
		return false, err
	}
	bytes = make([]byte, len(hash)/2)
	_, err = file.Read(bytes)
	if err != nil {
		return false, err
	}

	// bytes should now be equivalent to hash
	for b := 0; b < len(hash)/2; b++ {
		nibble0 := shallowtreebyte.NibbleVal(bytes[b] & 0x0F) // LSB
		nibble1 := shallowtreebyte.NibbleVal(bytes[b] >> 4)   // MSB
		if hash[b*2] != nibble1 || hash[b*2+1] != nibble0 {
			return false, nil
		}
	}
	return true, nil
}

func (t *Tier) Close() error {
	for d := range t.donuts {
		err := t.donuts[d].Close()
		if err != nil {
			return err
		}
		t.donuts[d] = nil
	}
	return nil
}
