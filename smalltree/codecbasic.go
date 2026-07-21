package smalltree

import "encoding/binary"

// A codec for a basic kind of nibble-based node encoding

type CodecBasic struct {
	numSlotsNodes LocalNodeId
	nodesBytes    []byte
	hashIdConfig  NByteIdConfig[HashIndexIdType]
}

// Check that implements
var _ Codec = (*CodecBasic)(nil)

const slotsNodeSize = 16 * 2 // 16 slots of 2 bytes each

func (cb *CodecBasic) GetNode(id LocalNodeId) Node {
	nb := NodeBasic{}
	if id < cb.numSlotsNodes {
		// It's a slots node
		nodeBytesOffset := int(id) * slotsNodeSize
		snb := SlotsNodeBasic{}
		for i := 0; i < 16; i++ {
			snb.slots[i] = LocalNodeId(binary.LittleEndian.Uint16(cb.nodesBytes[nodeBytesOffset+i*2 : nodeBytesOffset+i*2+2]))
		}
		nb.slotsNode = &snb
		nb.leafNode = nil
	} else {
		leafNodeSize := cb.hashIdConfig.StorageBytes()
		nodeBytesOffset := int(cb.numSlotsNodes*slotsNodeSize) + int(id-cb.numSlotsNodes)*leafNodeSize
		lnb := LeafNodeBasic{}
		lnb.hashId = cb.hashIdConfig.ReadID(cb.nodesBytes[nodeBytesOffset : nodeBytesOffset+leafNodeSize])
		nb.leafNode = &lnb
		nb.slotsNode = nil
	}
	return &nb
}

type CodecBasicMaker struct {
	hashIdConfig NByteIdConfig[HashIndexIdType]
}

// Check that implements
var _ CodecMaker = (*CodecBasicMaker)(nil)

func NewCodecBasicMaker(hashIdConfig NByteIdConfig[HashIndexIdType]) *CodecBasicMaker {
	return &CodecBasicMaker{
		hashIdConfig: hashIdConfig,
	}
}

// MakeCodec Interpret indexBytes as appropriate to a CodecBasic
func (cbm *CodecBasicMaker) MakeCodec(indexBytes []byte, nodesBytes []byte) Codec {
	result := CodecBasic{}
	if len(indexBytes) != 2 {
		panic("Wrong number of indexBytes for CodecBasic")
	}
	result.numSlotsNodes = LocalNodeId(binary.LittleEndian.Uint16(indexBytes))
	result.nodesBytes = nodesBytes
	result.hashIdConfig = cbm.hashIdConfig
	return &result
}

type NodeBasic struct {
	leafNode  *LeafNodeBasic
	slotsNode *SlotsNodeBasic
}

// Check that implements
var _ Node = (*NodeBasic)(nil)

func (nb *NodeBasic) IsLeafNode() bool {
	return nb.leafNode != nil
}
func (nb *NodeBasic) GetLeafNode() LeafNode {
	return nb.leafNode
}
func (nb *NodeBasic) GetSlotsNode() SlotsNode {
	return nb.slotsNode
}

type SlotsNodeBasic struct {
	slots [16]LocalNodeId
}

// Check that implements
var _ SlotsNode = (*SlotsNodeBasic)(nil)

func (snb *SlotsNodeBasic) GetSlotNode(valSeen SlotSelectorType) LocalNodeId {
	return snb.slots[valSeen]
}

type LeafNodeBasic struct {
	hashId HashIndexIdType
}

// Check that implements
var _ LeafNode = (*LeafNodeBasic)(nil)

func (lnb *LeafNodeBasic) GetHashId() HashIndexIdType {
	return lnb.hashId
}
