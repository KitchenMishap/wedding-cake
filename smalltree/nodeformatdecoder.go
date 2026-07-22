package smalltree

import (
	"encoding/binary"
	"math/bits"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

// LevelDecoderNf the decoder (single level)
type LevelDecoderNf struct {
	config     *SmallTreeConfig
	nodesBytes []byte
	indexBytes []byte
}

// Check that implements
var _ LevelDecoder = (*LevelDecoderNf)(nil)

func (ldn *LevelDecoderNf) ConfigureWithIndexBytes(indexBytes []byte) {
	ldn.indexBytes = indexBytes
}

func (ldn *LevelDecoderNf) GetNode(id LocalNodeIdType) Node {
	node := tempNode{}
	success := ldn.extractNode(id, &node)
	if !success {
		panic("extractNode failed")
	}

	resultNode := NodeNf{}
	resultNode.formatSpecBytes = node.formatSpecBytes
	resultNode.nodeBytes = node.nodeBytes
	isLeaf, reassuranceBytes, hashIndexId := ldn.detailsIfLeaf(&node)
	if isLeaf {
		ln := LeafNodeNf{}
		ln.reassuranceBytes = reassuranceBytes
		ln.hashIndexId = hashIndexId
		resultNode.leafNode = &ln
		resultNode.slotsNode = nil
	} else {
		resultNode.leafNode = nil
		hashByteToExamine, mediumSlots, tinySlots := ldn.hashByteIndexToExamine(&node)
		sn := SlotsNodeNf{}
		sn.hashByteIndexToExamine = shallowtreebyte.ByteIndex(hashByteToExamine)
		sn.mediumSlots = mediumSlots
		sn.tinySlots = tinySlots
		sn.nodeBytes = node.nodeBytes
		sn.nodeIdConfig = ldn.config.NodeIdConfig
		resultNode.slotsNode = &sn
	}
	return &resultNode
}

// tempNode: For processing (eg lookup) a node is temporarily internally represented by a tempNode
type tempNode struct {
	formatSpecBytes uint32
	nodeBytes       []byte
}

// detailsIfLeaf returns (true, reassuranceBytes, presentationIndex) if is a leaf
// returns (false, nil, 0) otherwise
func (ldn *LevelDecoderNf) detailsIfLeaf(tn *tempNode) (bool, []byte, HashIndexIdType) {

	// A leaf node is identified as two MSB zero bytes followed by an appropriate two LSB bytes bytesCount
	hashIndexIdSize := ldn.config.HashIndexIdConfig.StorageBytes()
	reassuranceBytesCount := ldn.config.ReassuranceBytesCount
	if tn.formatSpecBytes != uint32(reassuranceBytesCount)+uint32(hashIndexIdSize) {
		return false, nil, 0
	}
	// tn.nodeBytes interpreted as FormatLeaf
	reassuranceBytes := tn.nodeBytes[0:reassuranceBytesCount]
	hashIndexId := ldn.config.HashIndexIdConfig.ReadID(tn.nodeBytes[reassuranceBytesCount : int(reassuranceBytesCount)+hashIndexIdSize])
	return true, reassuranceBytes, hashIndexId
}

// hashByteIndexToExamine() should only be called if detailsIfLeaf() has already returned false
// It returns (index, 0, 0) for a FormatFull,
// or (index, slots, 0) for a FormatMedium,
// or (index, 0, slots) for a FormatTiny
func (ldn *LevelDecoderNf) hashByteIndexToExamine(tn *tempNode) (byte, byte, byte) {
	// Is it a FormatFull?
	nodeIdSize := ldn.config.NodeIdConfig.StorageBytes()
	if tn.formatSpecBytes == uint32(1+1+256*nodeIdSize) {
		if tn.nodeBytes[0] != 0xAA {
			panic("Expected an 0xAA byte for FormatFull padding")
		}
		return tn.nodeBytes[1], 0, 0
	}
	// Is it a FormatMedium?
	mediumSlots := byte((tn.formatSpecBytes & 0x00FF0000) >> 16)
	if mediumSlots > 0 {
		return tn.nodeBytes[1], mediumSlots, 0
	}
	// Is it a FormatTiny?
	tinySlots := byte((tn.formatSpecBytes & 0xFF000000) >> 24)
	if tinySlots > 0 {
		// No padding in this case, because FormatTiny is often a small odd number of bytes, and we want
		// them tightly packed
		return tn.nodeBytes[0], 0, tinySlots
	}
	panic("Unrecognised format spec")
}

// ExtractNode extracts to an existing chunkNode to avoid a busy heap
// Returns true for success
func (ldn *LevelDecoderNf) extractNode(nodeId LocalNodeIdType, target *tempNode) bool {
	currentSpecNodeId := LocalNodeIdType(0)
	currentSpecNodeByteOffset := BytesCountType(0)
	byteCount := BytesCountType(0)
	// Read the number of format specs from start of indexBytes
	formatSpecCount := binary.LittleEndian.Uint16(ldn.indexBytes[byteCount : byteCount+2])
	byteCount += 2

	// The results of the following loop
	found := false
	var formatSpecBytes uint32
	var nodeByteOffset BytesCountType
	var nodeByteSize uint16

	nodeCountSize := ldn.config.NodeIdConfig.StorageBytes()
	for fs := uint16(0); fs < formatSpecCount; fs++ {
		formatSpecNodeCount := ldn.config.NodeIdConfig.ReadID(ldn.indexBytes[byteCount : byteCount+BytesCountType(nodeCountSize)])
		byteCount += BytesCountType(nodeCountSize)
		formatSpecBytes = binary.LittleEndian.Uint32(ldn.indexBytes[byteCount : byteCount+4])
		byteCount += 4
		nodeByteSize = uint16(formatSpecBytes & 0xFFFF) // Held in the bottom 16 bits
		if currentSpecNodeId+formatSpecNodeCount > nodeId {
			// Found it in the current spec
			found = true
			nodeByteOffset = currentSpecNodeByteOffset + BytesCountType(nodeId-currentSpecNodeId)*BytesCountType(nodeByteSize)
			break
		}
		currentSpecNodeId += formatSpecNodeCount
		currentSpecNodeByteOffset += BytesCountType(formatSpecNodeCount) * BytesCountType(nodeByteSize)
	}
	if !found {
		return false
	}
	target.formatSpecBytes = formatSpecBytes
	target.nodeBytes = ldn.nodesBytes[nodeByteOffset : nodeByteOffset+BytesCountType(nodeByteSize)]
	return true
}

// The various concrete types that the nodes returned by the level decoder expose
type NodeNf struct {
	formatSpecBytes uint32
	nodeBytes       []byte
	leafNode        *LeafNodeNf // nil if not a leaf node
	slotsNode       *SlotsNodeNf
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
	hashByteIndexToExamine shallowtreebyte.ByteIndex
	mediumSlots            byte
	tinySlots              byte
	nodeBytes              []byte
	nodeIdConfig           NByteIdConfig[LocalNodeIdType]
}

// Check that implements
var _ SlotsNode = (*SlotsNodeNf)(nil)

func (snnf *SlotsNodeNf) GetHashByteToExamine() shallowtreebyte.ByteIndex {
	return snnf.hashByteIndexToExamine
}

func (snnf *SlotsNodeNf) GetNextNode(valSeen SlotSelectorType) LocalNodeIdType {
	nodeIdSize := snnf.nodeIdConfig.StorageBytes()
	// Is it a FormatFull?
	if snnf.mediumSlots == 0 && snnf.tinySlots == 0 {
		// (Could check formatSpecBytes == 1 + 1 + 256 * nodeIdSize)
		byteIndex := 1 + 1 + int(valSeen)*nodeIdSize
		return snnf.nodeIdConfig.ReadID(snnf.nodeBytes[byteIndex : byteIndex+nodeIdSize])
	}
	// Is it a FormatMedium?
	if snnf.mediumSlots > 0 {
		// There are 256 bits within snnf.nodeBytes which tell us (if each is a '1') which slots are represented
		// in the NodeIdType's which follow.
		// The first question is, for the value of valSeen, is the corresponding bit a zero?
		// 1. Isolate the target bit's coordinates inside the 256-bit space
		byteNumber := valSeen >> 3
		bitNumberWithinByte := valSeen & 0x07 // Keep this for the byte presence check

		// The 32-byte bitmask flags slice starts at offset 2 of snnf.nodeBytes
		flagsOffset := 2
		flagsByte := snnf.nodeBytes[flagsOffset+int(byteNumber)]
		if flagsByte&(1<<bitNumberWithinByte) == 0 {
			return LocalNodeIdNoMatch
		}

		// 2. Identify which of the 4 uint64 buckets our target bit belongs to
		targetBlock := valSeen >> 6 // 0 to 3 (Which uint64)
		// Calculate the exact bit shift position inside the 64-bit integer (0 to 63)
		bitNumberWithinBlock := valSeen & 0x3F

		// Read the four 64-bit blocks out of the node bytes stream
		u0 := binary.LittleEndian.Uint64(snnf.nodeBytes[flagsOffset+0 : flagsOffset+8])
		u1 := binary.LittleEndian.Uint64(snnf.nodeBytes[flagsOffset+8 : flagsOffset+16])
		u2 := binary.LittleEndian.Uint64(snnf.nodeBytes[flagsOffset+16 : flagsOffset+24])
		u3 := binary.LittleEndian.Uint64(snnf.nodeBytes[flagsOffset+24 : flagsOffset+32])

		var onesBefore int

		switch targetBlock {
		case 0:
			mask := (uint64(1) << bitNumberWithinBlock) - 1
			onesBefore = bits.OnesCount64(u0 & mask)
		case 1:
			mask := (uint64(1) << bitNumberWithinBlock) - 1
			onesBefore = bits.OnesCount64(u0) + bits.OnesCount64(u1&mask)
		case 2:
			mask := (uint64(1) << bitNumberWithinBlock) - 1
			onesBefore = bits.OnesCount64(u0) + bits.OnesCount64(u1) + bits.OnesCount64(u2&mask)
		case 3:
			mask := (uint64(1) << bitNumberWithinBlock) - 1
			onesBefore = bits.OnesCount64(u0) + bits.OnesCount64(u1) + bits.OnesCount64(u2) + bits.OnesCount64(u3&mask)
		}
		// 3. Compute physical NodeIdType payload layout offset
		// The NodeIdType data payloads start directly after our 32-byte bitmask (offset 34).
		uint16PayloadStart := flagsOffset + 32
		nodeIdByteOffset := uint16PayloadStart + (onesBefore * nodeIdSize)

		return snnf.nodeIdConfig.ReadID(snnf.nodeBytes[nodeIdByteOffset : nodeIdByteOffset+nodeIdSize])
	}
	// Is it a FormatTiny?
	if snnf.tinySlots > 0 {
		// FormatTiny is a hashByteIndex (byte) followed by
		// a series of {byteValue (byte), (NodeIdType)} pairs
		offset := 1
		for slot := 0; slot < int(snnf.tinySlots); slot++ {
			byteValue := snnf.nodeBytes[offset]
			offset++
			if byte(valSeen) == byteValue {
				return snnf.nodeIdConfig.ReadID(snnf.nodeBytes[offset : offset+nodeIdSize])
			}
			offset += nodeIdSize
		}
		return LocalNodeIdNoMatch
	}
	panic("Unrecognized format")
}

type LeafNodeNf struct {
	hashIndexId      HashIndexIdType
	reassuranceBytes []byte
}

// Check that implements
var _ LeafNode = (*LeafNodeNf)(nil)

func (lnnf *LeafNodeNf) GetHashId() HashIndexIdType {
	return lnnf.hashIndexId
}
func (lnnf *LeafNodeNf) GetReassuranceBytes() []byte {
	return lnnf.reassuranceBytes
}
