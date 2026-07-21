package smalltree

import "github.com/kitchenmishap/wedding-cake/shallowtreebyte"

// A codec pair for a "NodeFormats" (Nf) kind of node encoding

// LevelsEncoderNf the encoder (multiple levels)
type LevelsEncoderNf struct {
	hashIdConfig NByteIdConfig[HashIndexIdType]
}

// Check that implements
var _ LevelsEncoder = (*LevelsEncoderNf)(nil)

// LevelDecoderNf the decoder (single level)
type LevelDecoderNf struct {
	hashIdConfig NByteIdConfig[HashIndexIdType]
}

// Check that implements
var _ LevelDecoder = (*LevelDecoderNf)(nil)

func (le *LevelsEncoderNf) EncodeSubTree(tree *shallowtreebyte.ShallowTree, tf *TreeFormat) ([][]byte, [][]byte) {
	panic("Not implemented")
}

func (ldn *LevelDecoderNf) GetNode(id LocalNodeId) Node {
	panic("Not implemented")
}

type LevelsCodecNfFactory struct {
	hashIdConfig NByteIdConfig[HashIndexIdType]
}

// Check that implements
var _ LevelsCodecFactory = (*LevelsCodecNfFactory)(nil)

func NewLevelsCodecFactoryNf(hashIdConfig NByteIdConfig[HashIndexIdType]) LevelsCodecFactory {
	return &LevelsCodecNfFactory{
		hashIdConfig: hashIdConfig,
	}
}

func (LevelsCodecNfFactory *LevelsCodecNfFactory) MakeLevelsEncoder() LevelsEncoder {
	panic("Not implemented")
}
func (LevelsCodecNfFactory *LevelsCodecNfFactory) MakeLevelDecoder(indexBytes []byte, nodesBytes []byte) LevelDecoder {
	panic("Not implemented")
}

// The various concrete types that the nodes returned by the level decoder expose
type NodeNf struct {
	leafNode  *LeafNodeNf
	slotsNode *SlotsNodeNf
}

// Check that implements
var _ Node = (*NodeNf)(nil)

func (nnf *NodeNf) IsLeafNode() bool {
	return nnf.leafNode != nil
}
func (nnf *NodeNf) GetLeafNode() LeafNode {
	return nnf.leafNode
}
func (nnf *NodeNf) GetSlotsNode() SlotsNode {
	return nnf.slotsNode
}

type SlotsNodeNf struct {
	slots [256]LocalNodeId
}

// Check that implements
var _ SlotsNode = (*SlotsNodeNf)(nil)

func (snnf *SlotsNodeNf) GetSlotNode(valSeen SlotSelectorType) LocalNodeId {
	return snnf.slots[valSeen]
}

type LeafNodeNf struct {
	hashId HashIndexIdType
}

// Check that implements
var _ LeafNode = (*LeafNodeNf)(nil)

func (lnnf *LeafNodeNf) GetHashId() HashIndexIdType {
	return lnnf.hashId
}
