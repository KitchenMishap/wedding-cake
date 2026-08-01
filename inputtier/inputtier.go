package inputtier

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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

	hashesOrderFile   *os.File // Remains open for append whilst InputTier is open
	hashSuffixFile    *os.File // Remains open for append whilst InputTier is open
	hashesOrderWriter *bufio.Writer
	hashSuffixWriter  *bufio.Writer

	config          *smalltree.SmallTreeConfig
	hashBytesLength byte

	piOffset           types.PiOffset
	hashBytesToLocalPi map[string]types.LocalPi
	// Indexed by types.LocalPi. Gaps in presentation indices are represented by the empty string.
	hashBytes []string
}

func OpenInputTier(cakeFolderPath string, tierConfig *smalltree.SmallTreeConfig,
	piOffset types.PiOffset, hashBytesLength byte) (*InputTier, error) {
	result := InputTier{}
	result.config = tierConfig
	result.hashBytesLength = hashBytesLength

	result.piOffset = piOffset
	folder := fmt.Sprintf("InputTier_%d", result.piOffset)
	folderName := filepath.Join(cakeFolderPath, folder)
	result.numberedFolderPath = folderName

	// Check for READONLY file
	fNameRo := filepath.Join(result.numberedFolderPath, "READONLY")
	_, err := os.Stat(fNameRo)
	if err == nil {
		panic("Input tier folder is read only")
	}

	// The following call operates without a fully populated InputTier
	err = result.readHashes()
	if err != nil {
		return nil, err
	}
	fName := filepath.Join(result.numberedFolderPath, "HashesOrder.bin")
	result.hashesOrderFile, err = os.OpenFile(fName, os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	result.hashesOrderWriter = bufio.NewWriterSize(result.hashesOrderFile, 64*1024)

	fName = filepath.Join(result.numberedFolderPath, "HashPrefix", "HashSuffix.bin")
	result.hashSuffixFile, err = os.OpenFile(fName, os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	result.hashSuffixWriter = bufio.NewWriterSize(result.hashSuffixFile, 64*1024)
	return &result, nil
}

func (it *InputTier) Close() error {
	err := it.hashesOrderWriter.Flush()
	if err != nil {
		return err
	}
	err = it.hashesOrderFile.Close()
	if err != nil {
		return err
	}
	err = it.hashSuffixWriter.Flush()
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
	piByteCount := uint64(it.config.LocalPiRWriter.StorageBytes())
	suffixIndexByteCount := uint64(it.config.SuffixIndexRWriter.StorageBytes())

	localPi := gpi.ToLocalPi(it.piOffset)

	localPiExpected := len(it.hashBytes)
	gap := uint64(localPi) - uint64(localPiExpected)

	// Append to the hashesOrderFile.
	// Presentation index gaps are represented by types.LocalPiNoMatch (all ones)

	// Usually we would write prefixIndex, suffixIndex pairs (suffix indices for the input tier are equal to localPi's)
	// However in the case of the input tier, prefix indices are length zero!
	//if it.config.PrefixIndexRWriter.StorageBytes() != 0 {
	//	panic("Input tier expects zero length prefix index configuration")
	//}

	bytesToWriteLength := (gap + 1) * suffixIndexByteCount
	bytesToWrite := make([]byte, bytesToWriteLength)
	offset := uint64(0)
	for i := uint64(0); i < gap; i++ {
		it.config.SuffixIndexRWriter.WriteAllOnes(bytesToWrite[offset:])
		offset += suffixIndexByteCount
	}
	// For the input tier, suffix indices are equal to LocalPi values.
	// But they might not be encoded with the same number of bytes!
	it.config.SuffixIndexRWriter.WriteID(bytesToWrite[offset:], types.SuffixIndex(localPi))
	offset += suffixIndexByteCount
	_, err := it.hashesOrderWriter.Write(bytesToWrite[:bytesToWriteLength])
	if err != nil {
		return err
	}

	// Append to the hashSuffixFile (hash and localPi)
	const spareBytes = 64 + 8
	bytesToWriteSuffix := [spareBytes]byte{}
	copy(bytesToWriteSuffix[:len(hash)], hash)
	it.config.LocalPiRWriter.WriteID(bytesToWriteSuffix[len(hash):], localPi)
	_, err = it.hashSuffixWriter.Write(bytesToWriteSuffix[0 : uint64(len(hash))+piByteCount])
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

func (it *InputTier) CountPresentationIndices() uint64 {
	return uint64(len(it.hashBytes)) // Includes the "gaps" that don't have hashes
}

func (it *InputTier) readHashes() error {
	byteSizeSuffix := it.config.SuffixIndexRWriter.StorageBytes()
	byteSizeLocalPi := it.config.LocalPiRWriter.StorageBytes()

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
	if len(orderBytes)%byteSizeSuffix != 0 {
		panic("Wrong size file")
	}
	numPis := len(orderBytes) / byteSizeSuffix

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
		offsetOrder := i * byteSizeSuffix
		localPi1 := it.config.SuffixIndexRWriter.ReadID(orderBytes[offsetOrder : offsetOrder+byteSizeSuffix])
		if uint64(localPi1) != uint64(types.LocalPiNoMatch) {
			hash := suffixBytes[offsetSuffix : offsetSuffix+int(it.hashBytesLength)]
			localPi2 := it.config.LocalPiRWriter.ReadID(suffixBytes[offsetSuffix+int(it.hashBytesLength) : offsetSuffix+int(it.hashBytesLength)+byteSizeLocalPi])
			if uint64(localPi1) != uint64(localPi2) {
				return errors.New("LocalPi mismatch in InputTier files")
			}
			it.hashBytes[i] = string(hash)
			it.hashBytesToLocalPi[string(hash)] = localPi2
			offsetSuffix += int(it.hashBytesLength) + byteSizeLocalPi
		} else {
			it.hashBytes[i] = ""
		}
	}

	return nil
}
