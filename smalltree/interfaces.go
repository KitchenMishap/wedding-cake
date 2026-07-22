package smalltree

import "github.com/kitchenmishap/wedding-cake/shallowtreebyte"

// A codec is a way of encoding/decoding nodes to/from nodesBytes []byte
// A codec is initialized from indexBytes []byte

const LocalNodeIdNoMatch LocalNodeIdType = ^LocalNodeIdType(0)

type SlotSelectorType byte

type LevelsCodecFactory interface {
	// The codec is in two parts.
	// (1) LevelsEncoder encodes a given tree into an indexBytes/nodesBytes pair for each level of the tree
	MakeLevelsEncoder() LevelsEncoder
	// (2) LevelDecoder provides a node-based interface to a single level of the tree
	MakeLevelDecoder(indexBytes []byte, nodesBytes []byte) LevelDecoder
}

type LevelsEncoder interface {
	EncodeSubTree(*shallowtreebyte.ShallowTree, *TreeFormat) ([][]byte, [][]byte, LocalNodeIdType, byte)
}

type LevelDecoder interface {
	GetNode(id LocalNodeIdType) Node
}

type Node interface {
	IsLeafNode() bool
	GetLeafNode() LeafNode
	GetSlotsNode() SlotsNode
}

type LeafNode interface {
	GetHashId() HashIndexIdType
	GetReassuranceBytes() []byte
}

type SlotsNode interface {
	GetNextNode(valSeen SlotSelectorType) LocalNodeIdType
	GetHashByteToExamine() shallowtreebyte.ByteIndex
}
