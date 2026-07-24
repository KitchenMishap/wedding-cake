package forest

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/edsrzf/mmap-go"
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"

	"github.com/kitchenmishap/wedding-cake/smalltree"
)

type ForestRead struct {
	folderPath string
	tierIndex  byte
	config     *smalltree.SmallTreeConfig

	rootNodeIdForPrefix []types.LocalNodeId
	rootLevelForPrefix  []byte

	levels          []*ForestLevel // Some may be nil
	decoderForLevel []*smalltree.LevelDecoderNf
}

func NewForestRead(folderPath string, tierIndex byte, config *smalltree.SmallTreeConfig) *ForestRead {
	result := ForestRead{}
	result.folderPath = folderPath
	result.tierIndex = tierIndex
	result.config = config
	return &result
}

type ForestLevel struct {
	levelMmap         mmap.MMap
	decodersForPrefix []smalltree.LevelDecoder // This holds slices indexBytes and nodesBytes referring into mmap
}

func (fr *ForestRead) Open() error {
	// Read from TreesRoots.bin

	fPath := filepath.Join(fr.folderPath, "TreesRoots.bin")
	rootsFile, err := os.Open(fPath)
	if err != nil {
		return err
	}
	defer func() { _ = rootsFile.Close() }()

	// 1 for tierIndex==0, 16 for tierIndex==1, 256 etc...
	prefixIndexCount := 1 << (4 * fr.tierIndex) // 16 ^ tierIndex
	expectedRootsBytes := 3 * prefixIndexCount
	rootsBytes, err := io.ReadAll(rootsFile)
	if err != nil {
		return err
	}
	if len(rootsBytes) != expectedRootsBytes {
		return errors.New("TreesRoots.bin wrong length for tier")
	}

	offset := 0
	fr.rootNodeIdForPrefix = make([]types.LocalNodeId, prefixIndexCount)
	fr.rootLevelForPrefix = make([]byte, prefixIndexCount)
	for prefixIndex := 0; prefixIndex < prefixIndexCount; prefixIndex++ {
		fr.rootLevelForPrefix[prefixIndex] = rootsBytes[offset]
		fr.rootNodeIdForPrefix[prefixIndex] = types.LocalNodeId(binary.LittleEndian.Uint16(rootsBytes[offset+1 : offset+3]))
		offset += 3
	}

	factory := smalltree.NewLevelsCodecNfFactory(fr.config)

	// Read from various Level<LL>Lengths.bin and Level<LL>Nodes.bin files
	// Work backwards so we know the right length
	foundLevel := false
	for level := 128; level >= 0; level-- {
		fPathLengths := filepath.Join(fr.folderPath, fmt.Sprintf("Level%02XLengths.bin", level))
		fLengths, err := os.Open(fPathLengths)
		if err == nil {
			defer func() { _ = fLengths.Close() }()
			if !foundLevel {
				fr.levels = make([]*ForestLevel, level+1)
				foundLevel = true
			}
			fr.levels[level] = &ForestLevel{}
			// mmap the corresponding Nodes file
			fPathNodes := filepath.Join(fr.folderPath, fmt.Sprintf("Level%02XNodes.bin", level))
			fNodes, err := os.Open(fPathNodes)
			if err != nil {
				return err
			}
			defer func() { _ = fNodes.Close() }()
			fr.levels[level].levelMmap, err = mmap.Map(fNodes, mmap.RDONLY, 0)
			if err != nil {
				return err
			}

			// Read the lengths file
			lengthsBytes, err := io.ReadAll(fLengths)
			if err != nil {
				return err
			}
			if len(lengthsBytes) != prefixIndexCount*2*4 {
				return errors.New("Level<LL>Lengths.bin wrong length for tier")
			}
			offsetMmap := uint64(0)
			fr.levels[level].decodersForPrefix = make([]smalltree.LevelDecoder, prefixIndexCount)
			for prefixIndex := 0; prefixIndex < prefixIndexCount; prefixIndex++ {
				lengthIndex := uint64(binary.LittleEndian.Uint32(lengthsBytes[prefixIndex*8 : prefixIndex*8+4]))
				lengthNodes := uint64(binary.LittleEndian.Uint32(lengthsBytes[prefixIndex*8+4 : prefixIndex*8+8]))

				// Refer the slices for indexBytes and nodesBytes into the mmap'd nodes file
				indexBytes := fr.levels[level].levelMmap[offsetMmap : offsetMmap+lengthIndex]
				offsetMmap += lengthIndex
				nodesBytes := fr.levels[level].levelMmap[offsetMmap : offsetMmap+lengthNodes]
				offsetMmap += lengthNodes

				// Make a decoder that refers to indexBytes and nodesBytes in mmap
				fr.levels[level].decodersForPrefix[prefixIndex] = factory.MakeLevelDecoder(indexBytes, nodesBytes)
			}
		}
	}
	return nil
}

func (fr *ForestRead) Close() error {
	for level := range fr.levels {
		if fr.levels[level] != nil {
			err := fr.levels[level].levelMmap.Unmap()
			if err != nil {
				return err
			}
		}
	}
	fr.rootLevelForPrefix = nil
	fr.rootNodeIdForPrefix = nil
	fr.levels = nil
	return nil
}

func (fr *ForestRead) Lookup(hash []shallowtreebyte.NibbleVal) types.LocalPi {
	// Work out the prefix index from the first nibbles of the hash
	prefixNibblesCount := shallowtreebyte.NibbleIndex(fr.tierIndex)
	prefixIndex := PrefixIndexType(0)
	for nibbleIndex := shallowtreebyte.NibbleIndex(0); nibbleIndex < prefixNibblesCount; nibbleIndex++ {
		nibbleVal := hash[nibbleIndex]
		prefixIndex |= PrefixIndexType(nibbleVal) << (4 * PrefixIndexType(nibbleIndex))
	}
	level := fr.rootLevelForPrefix[prefixIndex]
	nodeId := fr.rootNodeIdForPrefix[prefixIndex]
	decoder := fr.levels[level].decodersForPrefix[prefixIndex]

	unusedNibbleFlags := shallowtreebyte.NewNibblesFlags(prefixNibblesCount, fr.config.HashNibbleLength)

	for {
		node := decoder.GetNode(nodeId)
		if node.IsLeafNode() {
			ln := node.GetLeafNode()
			// Check reassurance nibbles
			reassuranceBytes := ln.GetReassuranceBytes()
			byteIndexToExamine := shallowtreebyte.ByteIndex(0)
			for reassuranceByteIndex := 0; reassuranceByteIndex < len(reassuranceBytes); reassuranceByteIndex++ {
				// Find the next unexamined byte index in the hash
				for !unusedNibbleFlags.FlagValByte(byteIndexToExamine) {
					byteIndexToExamine++
				}
				unusedNibbleFlags.ClearFlagOrPanicByte(byteIndexToExamine)
				nibble0 := hash[byteIndexToExamine*2+1]
				nibble1 := hash[byteIndexToExamine*2]
				byt := byte(nibble0 | nibble1<<4)
				if reassuranceBytes[reassuranceByteIndex] != byt {
					return types.LocalPiNoMatch
				}
			}
			return ln.GetHashId()
		} else {
			sn := node.GetSlotsNode()
			byteIndex := sn.GetHashByteToExamine()
			nibble1 := hash[byteIndex*2]   // MS
			nibble0 := hash[byteIndex*2+1] // LS
			byt := nibble0 | nibble1<<4
			unusedNibbleFlags.ClearFlagOrPanicByte(byteIndex)
			nodeId = sn.GetNextNode(smalltree.SlotSelectorType(byt))
			if nodeId == types.LocalNodeIdNoMatch {
				return types.LocalPiNoMatch
			}
			level += 2
			decoder = fr.levels[level].decodersForPrefix[prefixIndex]
		}
	}
}
