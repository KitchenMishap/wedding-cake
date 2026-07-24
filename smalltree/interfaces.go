package smalltree

import (
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"
)

// A codec is a way of encoding/decoding nodes to/from nodesBytes []byte
// A codec is initialized from indexBytes []byte

type SlotSelectorType byte

type LevelsCodecFactory interface {
	// The codec is in two parts.
	// (1) LevelsEncoder encodes a given tree into an indexBytes/nodesBytes pair for each level of the tree
	MakeLevelsEncoder() LevelsEncoder
	// (2) LevelDecoder provides a node-based interface to a single level of the tree
	MakeLevelDecoder(indexBytes []byte, nodesBytes []byte) LevelDecoder
}

type LevelsEncoder interface {
	EncodeSubTree(*shallowtreebyte.ShallowTree, *TreeFormat) ([][]byte, [][]byte, types.LocalNodeId, byte)
}

type LevelDecoder interface {
	GetNode(id types.LocalNodeId) Node
}

type Node interface {
	IsLeafNode() bool
	GetLeafNode() LeafNode
	GetSlotsNode() SlotsNode
}

type LeafNode interface {
	GetHashId() types.LocalPi
	GetReassuranceBytes() []byte
}

type SlotsNode interface {
	GetNextNode(valSeen SlotSelectorType) types.LocalNodeId
	GetHashByteToExamine() shallowtreebyte.ByteIndex
}
