package smalltree

// LevelDecoderNf the decoder (single level)
type LevelDecoderNf struct {
	config *SmallTreeConfig
}

// Check that implements
var _ LevelDecoder = (*LevelDecoderNf)(nil)

func (ldn *LevelDecoderNf) GetNode(id LocalNodeId) Node {
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
