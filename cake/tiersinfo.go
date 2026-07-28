package cake

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/types"
)

// The "zero value" of a TiersInfo struct represents a cake with a single input tier with an offset of zero.
type TiersInfo struct {
	inputOffset types.PiOffset
	offset      [5]types.PiOffset
	present     [5]bool
}

func (ti *TiersInfo) ToDisk(folderPath string) error {
	fName := filepath.Join(folderPath, "TierOffsets.txt")
	file, err := os.Create(fName)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = fmt.Fprintf(file, "%d\n", ti.inputOffset)
	if err != nil {
		return fmt.Errorf("failed to write inputOffset: %w", err)
	}
	for i := 0; i < 5; i++ {
		if ti.present[i] {
			_, err = fmt.Fprintf(file, "%d\n", ti.offset[i])
			if err != nil {
				return fmt.Errorf("failed to write tier offset %d: %w", i, err)
			}
		} else {
			_, err = fmt.Fprintf(file, "%d\n", -1)
			if err != nil {
				return fmt.Errorf("failed to write tier offset %d: %w", i, err)
			}
		}
	}
	return nil
}

func (ti *TiersInfo) FromDisk(folderPath string) error {
	fName := filepath.Join(folderPath, "TierOffsets.txt")
	file, err := os.Open(fName)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close }()

	// Read the initial inputOffset
	if _, err := fmt.Fscan(file, &ti.inputOffset); err != nil {
		return fmt.Errorf("failed to read inputOffset: %w", err)
	}

	// Read each of the 5 tier offsets
	for i := 0; i < 5; i++ {
		var val int64
		if _, err := fmt.Fscan(file, &val); err != nil {
			return fmt.Errorf("failed to read tier offset %d: %w", i, err)
		}

		if val == -1 {
			ti.present[i] = false
			ti.offset[i] = 0 // Reset to zero value when not present
		} else {
			ti.present[i] = true
			ti.offset[i] = types.PiOffset(val)
		}
	}

	return nil
}
