package smalltree

// A codec is a way of encoding/decoding nodes to/from nodesBytes []byte
// A codec is initialized from indexBytes []byte

type LocalNodeId uint16

const LocalNodeIdNoMatch LocalNodeId = ^LocalNodeId(0)

type SlotSelectorType byte

type CodecMaker interface {
	MakeCodec(indexBytes []byte, nodesBytes []byte) Codec
}

type Codec interface {
	GetNode(id LocalNodeId) Node
}

type Node interface {
	IsLeafNode() bool
	GetLeafNode() LeafNode
	GetSlotsNode() SlotsNode
}

type LeafNode interface {
	GetHashId() HashIndexIdType
}

type SlotsNode interface {
	GetSlotNode(valSeen SlotSelectorType) LocalNodeId
}
