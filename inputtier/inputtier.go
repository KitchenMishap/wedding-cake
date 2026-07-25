package inputtier

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kitchenmishap/wedding-cake/smalltree"
	"github.com/kitchenmishap/wedding-cake/types"
)

// InputTier is where new hash/global presentation indices are initially injected.
// A check to avoid presentation of duplicate hashes is presumed to have already passed.
// Such a check results in gaps in the presentation indices, which is allowed although they cannot be presented
// out of sequence. A note is made of the gaps.

// InputTier has a representation on disk.

type InputTier struct {
	numberedFolderPath string

	hashesOrderFile *os.File // Remains open for append whilst InputTier is open
	hashSuffixFile  *os.File // Remains open for append whilst InputTier is open

	localPiWriter   smalltree.NByteIdConfig[types.LocalPi]
	hashBytesLength int

	piOffset           types.PiOffset
	hashBytesToLocalPi map[string]types.LocalPi
	// Indexed by types.LocalPi. Gaps in presentation indices are represented by the empty string.
	hashBytes []string
}

func OpenInputTier(cakeFolderPath string, localPiWriter smalltree.NByteIdConfig[types.LocalPi],
	hashBytesLength int) (*InputTier, error) {
	result := InputTier{}
	result.localPiWriter = localPiWriter
	result.hashBytesLength = hashBytesLength

	// First we read the offset from a parent file. This will help us determine the InputTier_xxx foldername
	fName := filepath.Join(cakeFolderPath, "InputTierOffset.txt")
	file, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	offset, err := strconv.Atoi(string(contents))
	if err != nil {
		return nil, err
	}
	result.piOffset = types.PiOffset(offset)
	folder := fmt.Sprintf("InputTier_%d", result.piOffset)
	folderName := filepath.Join(cakeFolderPath, folder)
	result.numberedFolderPath = folderName

	// The following call operates without a fully populated InputTier
	err = result.readHashes()
	if err != nil {
		return nil, err
	}
	fName = filepath.Join(result.numberedFolderPath, "HashesOrder.bin")
	result.hashesOrderFile, err = os.OpenFile(fName, os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	fName = filepath.Join(result.numberedFolderPath, "HashPrefix", "HashSuffix.bin")
	result.hashSuffixFile, err = os.OpenFile(fName, os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (it *InputTier) Close() error {
	err := it.hashesOrderFile.Close()
	if err != nil {
		return err
	}
	err = it.hashSuffixFile.Close()
	if err != nil {
		return err
	}

	// Prevent use of closed InputTier
	it.hashesOrderFile = nil
	it.hashSuffixFile = nil
	it.hashesOrderFile = nil
	it.hashBytes = nil

	return nil
}

// AppendHash gpi's must be presented in order, but gaps are permitted.
// These gaps are to accommodate gpi's of hashes that turned out to be duplicates (detected in an external mechanism).
func (it *InputTier) AppendHash(gpi types.GlobalPi, hash []byte) error {
	piByteCount := uint64(it.localPiWriter.StorageBytes())

	localPi := gpi.ToLocalPi(it.piOffset)

	localPiExpected := len(it.hashBytes)
	gap := uint64(localPi) - uint64(localPiExpected)

	// Append to the hashesOrderFile.
	// Presentation index gaps are represented by types.LocalPiNoMatch (all ones)
	const spareBytes = 8 // Enough room for the biggest localPiWriter output
	bytesToWrite := make([]byte, gap*piByteCount+spareBytes)
	offset := uint64(0)
	for i := uint64(0); i < gap; i++ {
		it.localPiWriter.WriteAllOnes(bytesToWrite[offset:])
		offset += piByteCount
	}
	it.localPiWriter.WriteID(bytesToWrite[offset:], localPi)
	localPiOffset := offset
	offset += piByteCount
	_, err := it.hashesOrderFile.Write(bytesToWrite[:offset])
	if err != nil {
		return err
	}

	// Append to the hashSuffixFile (hash and localPi)
	entry := append(hash, bytesToWrite[localPiOffset:localPiOffset+piByteCount]...)
	_, err = it.hashSuffixFile.Write(entry)
	if err != nil {
		return err
	}

	// Put into the local InputTier vars
	it.hashBytesToLocalPi[string(hash)] = localPi
	for i := uint64(0); i < gap; i++ {
		it.hashBytes = append(it.hashBytes, "")
	}
	it.hashBytes = append(it.hashBytes, string(hash))

	return nil
}

func (it *InputTier) LookupHash(hash []byte) types.GlobalPi {
	localPi, ok := it.hashBytesToLocalPi[string(hash)]
	if !ok {
		return types.GlobalPresentationIndexNoMatch
	}
	return localPi.ToGlobalPi(it.piOffset)
}

func (it *InputTier) GetHashAtIndex(gpi types.GlobalPi) ([]byte, bool) {
	if uint64(gpi) < uint64(it.piOffset) {
		return nil, false
	}
	if uint64(gpi) >= uint64(it.piOffset)+uint64(len(it.hashBytes)) {
		return nil, false
	}
	localPi := gpi.ToLocalPi(it.piOffset)
	s := it.hashBytes[localPi]
	if s == "" {
		return nil, false
	}
	return []byte(it.hashBytes[localPi]), true
}

func (it *InputTier) readHashes() error {
	byteSize := it.localPiWriter.StorageBytes()

	orderFName := filepath.Join(it.numberedFolderPath, "HashesOrder.bin")
	orderFile, err := os.Open(orderFName)
	defer func() { _ = orderFile.Close() }()
	if err != nil {
		return err
	}
	orderBytes, err := io.ReadAll(orderFile)
	if err != nil {
		return err
	}
	numPis := len(orderBytes) / byteSize

	suffixFName := filepath.Join(it.numberedFolderPath, "HashPrefix", "HashSuffix.bin")
	suffixFile, err := os.Open(suffixFName)
	defer func() { _ = suffixFile.Close() }()
	if err != nil {
		return err
	}
	suffixBytes, err := io.ReadAll(suffixFile)
	if err != nil {
		return err
	}

	it.hashBytesToLocalPi = make(map[string]types.LocalPi, numPis) // numPis is more than enough
	it.hashBytes = make([]string, numPis)

	// Within numPis, there are gaps represented by "all ones" in the HashesOrde.bin file
	// For such items, an entry is NOT present in the HashSuffix.bin file
	offsetSuffix := 0
	for i := 0; i < numPis; i++ {
		offsetOrder := i * byteSize
		localPi1 := it.localPiWriter.ReadID(orderBytes[offsetOrder : offsetOrder+byteSize])
		if localPi1 != types.LocalPiNoMatch {
			hash := suffixBytes[offsetSuffix : offsetSuffix+it.hashBytesLength]
			localPi2 := it.localPiWriter.ReadID(suffixBytes[offsetSuffix+it.hashBytesLength : offsetSuffix+it.hashBytesLength+byteSize])
			if localPi1 != localPi2 {
				return errors.New("LocalPi mismatch in InputTier files")
			}
			it.hashBytes[i] = string(hash)
			it.hashBytesToLocalPi[string(hash)] = localPi1
			offsetSuffix += it.hashBytesLength + byteSize
		} else {
			it.hashBytes[i] = ""
		}
	}

	return nil
}
