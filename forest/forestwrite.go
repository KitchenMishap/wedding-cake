package forest

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kitchenmishap/wedding-cake/types"
)

// A forest is a folder on disk representing a hash lookup store for a contiguous sequence of hashes
// between two specified presentation indices.
// Lookup proceeds by first examining a fixed prefix number of nibbles of the hash to determine a "prefix index".
// The prefix index is used to index into a logical "smalltree" that examines a non-sequential set of the remaining
// bytes to arrive at a leaf presentation index (based on the order of hashes originally presented).
// Each logical smalltree's are however stored multiplexed across multiple pairs of "level" files.
// "Level" refers to the number of (non-sequential) nibbles of the hash that have been examined to arrive
// at a node in the smalltree.
// Each smalltree's nodes are spread across multiple levels, and a level holds some nodes from multiple smalltree's.
// A smalltree's nodes at a particular level are characterized by an indexBytes []byte and a nodesBytes []byte.
// At each level LL is a LevelLLlengths.bin file.
// Indexed by the prefix index, Level<LL>Lengths.bin holds the lengths of indexBytes and nodeBytes for the particular
// smalltree (represented by prefix index) at a particular level LL.
// Level<LL>Lengths.bin is therefore used as an index into the other file for the level, Level<LL>Nodes.bin.
// For each prefix index, starting at 0, Level<LL>Nodes.bin holds the actual variable length indexBytes and nodesBytes.

type PrefixIndexType uint64

type ForestWrite struct {
	folderPath        string
	levelLengthsFiles [129]*os.File
	levelNodesFiles   [129]*os.File
	treeRootsFile     *os.File
}

func NewForestWrite(folderPath string) *ForestWrite {
	result := ForestWrite{}
	result.folderPath = folderPath
	return &result
}

func (fw *ForestWrite) StartWrite() error {
	err := os.MkdirAll(fw.folderPath, 0755)
	if err != nil {
		return err
	}
	fPath := filepath.Join(fw.folderPath, "TreesRoots.bin")
	fw.treeRootsFile, err = os.OpenFile(fPath, os.O_WRONLY|os.O_CREATE, 0755)
	return nil
}

func (fw *ForestWrite) EndWrite() error {
	for level := byte(0); level <= byte(128); level++ {
		if fw.levelLengthsFiles[level] != nil {
			err := fw.levelLengthsFiles[level].Close()
			if err != nil {
				return err
			}
			fw.levelLengthsFiles[level] = nil
		}
		if fw.levelNodesFiles[level] != nil {
			err := fw.levelNodesFiles[level].Close()
			if err != nil {
				return err
			}
			fw.levelNodesFiles[level] = nil
		}
	}
	err := fw.treeRootsFile.Close()
	if err != nil {
		return err
	}
	return nil
}

func (fw *ForestWrite) AppendTreeForPrefix(prefixIndex PrefixIndexType, indexBytes [][]byte, nodesBytes [][]byte,
	rootNode types.LocalNodeId, rootLevel byte) error {
	levels := len(indexBytes)
	levelsNodes := len(nodesBytes)
	if levelsNodes > levels {
		levels = levelsNodes
	}
	for level := 0; level < levels; level++ {

		// If bytes for level are non-empty
		if len(indexBytes[level]) > 0 || len(nodesBytes[level]) > 0 {

			// Create new Lengths file for level if not done already
			if fw.levelLengthsFiles[level] == nil {
				fName := fmt.Sprintf("Level%02XLengths.bin", level)
				fPath := filepath.Join(fw.folderPath, fName)
				var err error
				fw.levelLengthsFiles[level], err = os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY, 0755)
				if err != nil {
					return err
				}

				// If prefixIndex is non-zero, we'll have to write some zero lengths to the file to "catch up"
				zeroes := make([]byte, prefixIndex*2*4) // For each prefixIndex missed, two uint32's
				_, err = fw.levelLengthsFiles[level].Write(zeroes)
				if err != nil {
					return err
				}
			}

			// Create new Nodes file for level if not done already
			if fw.levelNodesFiles[level] == nil {
				fName := fmt.Sprintf("Level%02XNodes.bin", level)
				fPath := filepath.Join(fw.folderPath, fName)
				var err error
				fw.levelNodesFiles[level], err = os.OpenFile(fPath, os.O_CREATE|os.O_WRONLY, 0755)
				if err != nil {
					return err
				}
			}

			// Write indexBytes length
			indexLen := [4]byte{}
			binary.LittleEndian.PutUint32(indexLen[:], uint32(len(indexBytes[level])))
			_, err := fw.levelLengthsFiles[level].Write(indexLen[:])
			if err != nil {
				return err
			}

			// Write nodesBytes length
			nodesLen := [4]byte{}
			binary.LittleEndian.PutUint32(nodesLen[:], uint32(len(nodesBytes[level])))
			_, err = fw.levelLengthsFiles[level].Write(nodesLen[:])
			if err != nil {
				return err
			}

			// Write index bytes
			_, err = fw.levelNodesFiles[level].Write(indexBytes[level])
			if err != nil {
				return err
			}

			// Write nodes bytes
			_, err = fw.levelNodesFiles[level].Write(nodesBytes[level])
			if err != nil {
				return err
			}
		} // if len > 0
	} // for level
	treeRootNodeId := [2]byte{}
	treeRootLevel := [1]byte{}
	binary.LittleEndian.PutUint16(treeRootNodeId[:], uint16(rootNode))
	treeRootLevel[0] = rootLevel
	_, err := fw.treeRootsFile.Write(treeRootLevel[:])
	if err != nil {
		return err
	}
	_, err = fw.treeRootsFile.Write(treeRootNodeId[:])
	if err != nil {
		return err
	}
	return nil
}
