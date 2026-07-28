package smalltree

import (
	"encoding/binary"
	"fmt"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"
)

// LevelsEncoderNf the encoder (multiple levels)
type LevelsEncoderNf struct {
	config *SmallTreeConfig
}

// Check that implements
var _ LevelsEncoder = (*LevelsEncoderNf)(nil)

// EncodeSubTree The outer index represents the level, up to the last populated level.
func (lenf *LevelsEncoderNf) EncodeSubTree(tree *shallowtreebyte.ShallowTree, tf *TreeFormat) ([][]byte, [][]byte, types.LocalNodeId, byte) {
	resultIndexBytes, resultNodesBytes := lenf.allocateLevelBytes(tf)
	// Encode the indexBytes
	lenf.serializeIndexBytes(tf, resultIndexBytes)
	// Encode the nodesBytes
	rootNodeId, rootNodeLevel := lenf.serializeNodesBytes(tree, tf, resultNodesBytes)

	return resultIndexBytes, resultNodesBytes, rootNodeId, rootNodeLevel
}

// allocateLevelBytes() calculates and allocates the number of bytes for indexBytes and for nodeBytes for each level
// The outer index represents the level, up to the last populated level.
func (lenf *LevelsEncoderNf) allocateLevelBytes(tf *TreeFormat) ([][]byte, [][]byte) {
	nodeCountSize := lenf.config.NodeIdRWriter.StorageBytes()

	indexBytesCount := [129]uint64{}
	nodesBytesCount := [129]uint64{}

	lastPopulatedLevel := -1
	for level := byte(0); level <= 128; level++ {
		levelData := &tf.LevelSpecs[level]

		indexBytesCount[level] = 0
		nodesBytesCount[level] = 0

		// For each level, in indexBytes, the first 2 bytes represent a count of NodeSpecs ("groups") that follow
		indexBytesCount[level] += 2

		for groupIndex := range levelData.Groups {
			lastPopulatedLevel = int(level)
			group := &(levelData.Groups[groupIndex])
			// In the indexBytes, for this group (nodespec), N bytes specify the number of nodes,
			// and four bytes describe the formatSpec
			indexBytesCount[level] += uint64(nodeCountSize + 4)
			// In the nodesBytes, for this group (nodespec), the number of bytes has already been determined
			nodesBytesCount[level] += uint64(group.Bytes)
		}
	}
	resultIndexBytes := make([][]byte, lastPopulatedLevel+1)
	resultNodesBytes := make([][]byte, lastPopulatedLevel+1)
	for level := 0; level <= lastPopulatedLevel; level++ {
		// An indexBytesCount of exactly two must mean no nodespecs, and the two bytes must simply be zeroes.
		// So it's more efficient in that case to treat the indexByteCount as itself zero!
		if indexBytesCount[level] == 2 {
			resultIndexBytes[level] = nil
			resultNodesBytes[level] = nil
		} else {
			resultIndexBytes[level] = make([]byte, 0, indexBytesCount[level])
			resultNodesBytes[level] = make([]byte, 0, nodesBytesCount[level])
		}
	}
	return resultIndexBytes, resultNodesBytes
}

func (lenf *LevelsEncoderNf) serializeIndexBytes(tf *TreeFormat, indexBytes [][]byte) {
	levels := len(indexBytes)
	nodeIdSize := lenf.config.NodeIdRWriter.StorageBytes()
	nodesCountSize := nodeIdSize
	hashIndexIdSize := lenf.config.LocalPiRWriter.StorageBytes()
	for levelNum := 0; levelNum < levels; levelNum++ {
		formatSpecGroups := &tf.LevelSpecs[levelNum].Groups
		levelIndexBytes := &indexBytes[levelNum]

		if len(*formatSpecGroups) == 0 && *levelIndexBytes != nil {
			panic("If there are no format specs then there should be no index bytes")
		}
		if *levelIndexBytes == nil && len(*formatSpecGroups) > 0 {
			panic("If there are no index bytes then there should be no format specs")
		}
		if *levelIndexBytes != nil {

			// In each level, we start with two bytes representing the count of NodeSpec's ("group"s) that follow
			var serializedGroupsCountBytes [2]byte
			binary.LittleEndian.PutUint16(serializedGroupsCountBytes[:], uint16(len(*formatSpecGroups)))
			*levelIndexBytes = append(*levelIndexBytes, serializedGroupsCountBytes[:]...)

			for groupIndex := range *formatSpecGroups {
				group := (*formatSpecGroups)[groupIndex]
				// Whilst we call this a "group", this has only come about by merging of individual
				// formatSpecs in StoreConfig.DesignTreeFormat(). The "group" is in fact governed
				// by a single FormatSpec, which we serialize here.
				const spareRoom = 8 // The most space we will ever need
				if nodesCountSize > spareRoom {
					panic("Not enough bytes")
				}
				serializedNodesCountBytes := [spareRoom]byte{} // The count of nodes expressed as "some" bytes
				lenf.config.NodeIdRWriter.WriteID(serializedNodesCountBytes[:nodesCountSize], types.LocalNodeId(group.NodesCount))
				*levelIndexBytes = append(*levelIndexBytes, serializedNodesCountBytes[:nodesCountSize]...)
				serializedNodeSpecBytes := [4]byte{} // The details of the FormatSpecs for these nodes
				switch group.Spec.Format {
				case NodeFormatFull:
					// Most significant bytes pair = zero, LS byte pair = number of bytes per node
					// Number of bytes per node is (1) pad + (1) hashByteIndex + (256 * N) node ids
					bytesPerNodeFull := 1 + 1 + 256*nodeIdSize
					binary.LittleEndian.PutUint32(serializedNodeSpecBytes[:], uint32(bytesPerNodeFull))
				case NodeFormatLeaf:
					// Most significant bytes pair = zero, LS byte pair = number of bytes per node
					// Number of bytes per node is (Reassurance bytes count) + (size of a hash index id)
					bytesPerNodeLeaf := uint32(int(lenf.config.ReassuranceBytesCount) + hashIndexIdSize)
					binary.LittleEndian.PutUint32(serializedNodeSpecBytes[:], bytesPerNodeLeaf)
				case NodeFormatMedium:
					// MS byte = zero, then slots byte, LS byte pair = number of bytes per node
					slotsFields := uint32(group.Spec.SlotsCapacity) << 16
					// Bytes per node = 1 (pad) + 1 (hash byte index) + 32 (slot flags) + N (node id) * slotsCapacity
					bytesPerNodeField := uint32(1 + 1 + 32 + nodeIdSize*int(group.Spec.SlotsCapacity))
					binary.LittleEndian.PutUint32(serializedNodeSpecBytes[:], slotsFields|bytesPerNodeField)
				case NodeFormatTiny:
					// MS byte slots capacity byte, then zero, LS byte pair = number of bytes per node
					slotsFields := uint32(group.Spec.SlotsCapacity) << 24
					// Bytes per node = 1 (hash byte index) + slots capacity * (1 (hash byte value) + N (node id))
					bytesPerNodeField := uint32(1 + int(group.Spec.SlotsCapacity)*(1+nodeIdSize))
					binary.LittleEndian.PutUint32(serializedNodeSpecBytes[:], slotsFields|bytesPerNodeField)
				}
				*levelIndexBytes = append(*levelIndexBytes, serializedNodeSpecBytes[:]...)
			} // for groupIndex

			// Check (because we can) that we have exactly reached capacity
			if len(*levelIndexBytes) != cap(*levelIndexBytes) {
				panic("Error in byte counting code")
			}
		} // if levelIndexBytes != nil
	} // for levelNum
}

// Returns the root node id and level
func (lenf *LevelsEncoderNf) serializeNodesBytes(tree *shallowtreebyte.ShallowTree,
	tf *TreeFormat, nodesBytes [][]byte) (types.LocalNodeId, byte) {

	levels := len(nodesBytes)

	// 1. Group nodes by level just like before
	nodesByLevel := make([][]*shallowtreebyte.Node, levels)
	tree.VisitAllNodes(func(node *shallowtreebyte.Node) {
		nodesByLevel[node.Level] = append(nodesByLevel[node.Level], node)
	})

	// 2. We only need ONE map for the "level below us" at any given time
	var nextLevelIdMap map[*shallowtreebyte.Node]types.LocalNodeId

	// 3. Process bottom-up
	lastProcessedNodeId := types.LocalNodeIdNoMatch
	lastProcessedNodeLevel := levels - 1
	for levelNum := levels - 1; levelNum >= 0; levelNum-- {
		//fmt.Printf("Processing level %d\n", levelNum)
		currentLevelNodes := nodesByLevel[levelNum]
		if len(currentLevelNodes) == 0 { // Because a level represents a number of NIBBLES of hash examined,
			continue // but shallowtreebyte examines whole BYTES, we often have empty levels.
		} // Furthermore, the nextLevelIdMap has to "skip" the empty level.
		levelNodesBytes := &nodesBytes[levelNum]
		// Create a fresh map for the current level allocations
		currentLevelIdMap := make(map[*shallowtreebyte.Node]types.LocalNodeId, len(currentLevelNodes))

		// Pass A: Allocate IDs and populate our current level map
		for _, node := range currentLevelNodes {
			activeSlots := node.ActiveSlotsCount()
			nodeID, _ := tf.AllocateIdAndSpecForNode(byte(node.Level), activeSlots)
			lastProcessedNodeId = nodeID
			lastProcessedNodeLevel = levelNum
			currentLevelIdMap[node] = nodeID
		}

		// Pass B: Serialize this level's nodes.
		// When a node looks up a child, it queries nextLevelIdMap in O(1) time!
		for groupIdx, group := range tf.LevelSpecs[levelNum].Groups {
			spec := &group.Spec

			// Only serialize nodes at this level that belong to the current format group
			for _, node := range currentLevelNodes {
				nodeGroup := tf.LevelSpecs[levelNum].SlotCountToGroup[node.ActiveSlotsCount()]
				if int(nodeGroup) != groupIdx {
					continue // Skip until we hit this group's turn
				}

				// Pass the map belonging to levelNum + 1 down to the serializer
				switch spec.Format {
				case NodeFormatLeaf:
					// fmt.Println("Serializing FormatLeaf node")
					lenf.serializeLeafNode(node.LeafNode, levelNodesBytes)
				case NodeFormatFull:
					// fmt.Println("Serializing FormatFull node")
					lenf.serializeFullNode(node.SlotsNode, nextLevelIdMap, levelNodesBytes)
				case NodeFormatMedium:
					// fmt.Println("Serializing FormatMedium node")
					lenf.serializeMediumNode(node.SlotsNode, spec, nextLevelIdMap, levelNodesBytes)
				case NodeFormatTiny:
					// fmt.Println("Serializing FormatTiny node")
					lenf.serializeTinyNode(node.SlotsNode, spec, nextLevelIdMap, levelNodesBytes)
				}
			}
		}
		// Promote the current map to be the "nextLevel" map for the tier above us,
		// allowing the old nextLevelIdMap to be immediately garbage collected!
		nextLevelIdMap = currentLevelIdMap
		// Just because we can, check that level nodes bytes are full to capacity
		if len(*levelNodesBytes) != cap(*levelNodesBytes) {
			fmt.Printf("Level %d: len(*levelNodeBytes) = %d, cap(*levelNodeBytes) = %d\n", levelNum, len(*levelNodesBytes), cap(*levelNodesBytes))
			panic("Expected nodes bytes to be full to capacity")
		}
	}
	// Because we work from bottom up through levels, the last node processed is the root node
	rootNodeId := lastProcessedNodeId
	rootNodeLevel := lastProcessedNodeLevel
	return rootNodeId, byte(rootNodeLevel)
}

func (lenf *LevelsEncoderNf) serializeLeafNode(leafNode *shallowtreebyte.LeafNode, bytes *[]byte) {
	// A leaf node is the reassurance bytes followed by the hash index id

	// In ShallowTree, it is clever enough to give fewer reassurance bytes than configured, in cases where
	// there are not enough bytes left to examine in the hash. But our serialized leaf node has a fixed
	// capacity for these, so we need to pad them.
	reassuranceBytes := leafNode.ReassuranceHashBytes
	reassurancePadding := lenf.config.ReassuranceBytesCount - byte(len(reassuranceBytes))
	*bytes = append(*bytes, reassuranceBytes...)
	if reassurancePadding > 0 {
		for pad := byte(0); pad < reassurancePadding; pad++ {
			*bytes = append(*bytes, 0)
		}
	}
	pi := leafNode.PresentationIndex
	hashIndexIdSize := lenf.config.LocalPiRWriter.StorageBytes()
	const spareRoom = 8
	var hashIndexIdBytes [spareRoom]byte
	lenf.config.LocalPiRWriter.WriteID(hashIndexIdBytes[:hashIndexIdSize], pi)
	*bytes = append(*bytes, hashIndexIdBytes[:hashIndexIdSize]...)
}

func (lenf *LevelsEncoderNf) HashNibblesToHashBytes(nibbles []shallowtreebyte.NibbleVal) []byte {
	if len(nibbles)&1 != 0 {
		panic("Expected reassurance hash nibbles to be even")
	}
	hashBytes := make([]byte, len(nibbles)/2)
	for i := 0; i < len(nibbles); i += 2 {
		hashBytes[i/2] = byte(nibbles[i]<<4 | nibbles[i+1])
	}
	return hashBytes
}

func (lenf *LevelsEncoderNf) serializeFullNode(slotsNode *shallowtreebyte.SlotsNode,
	nextLevelIdMap map[*shallowtreebyte.Node]types.LocalNodeId, bytes *[]byte) {
	// A full node is one byte padding (0), one byte hash byte index, and 256 N-byte nodeId slots.
	// (a nodeId of 0 is used to indicate an empty slot)
	// A full node is therefore fixed size (for a particular nodeIdsize configuration) and can be done in one append
	nodeIdSize := lenf.config.NodeIdRWriter.StorageBytes()
	fullNodeSize := 1 + 1 + 256*nodeIdSize
	const spareRoom = 1 + 1 + 256*8
	var nodeBytes [spareRoom]byte
	nodeBytes[0] = 0xAA // Padding
	nodeBytes[1] = byte(slotsNode.HashByteIndex)
	p := 2
	for s := 0; s < 256; s++ {
		if slotsNode.Slots[s].IsEmpty() {
			lenf.config.NodeIdRWriter.WriteAllOnes(nodeBytes[p : p+nodeIdSize])
		} else {
			nodeId, ok := nextLevelIdMap[slotsNode.Slots[s].NextNode]
			if !ok {
				panic("Node pointer not found in map")
			}
			lenf.config.NodeIdRWriter.WriteID(nodeBytes[p:p+nodeIdSize], nodeId)
		}
		p += nodeIdSize
	}
	if p != fullNodeSize {
		panic("Error in byte counting code")
	}
	*bytes = append(*bytes, nodeBytes[:fullNodeSize]...)
}

func (lenf *LevelsEncoderNf) serializeMediumNode(slotsNode *shallowtreebyte.SlotsNode, spec *NodeFormatSpec,
	nextLevelIdMap map[*shallowtreebyte.Node]types.LocalNodeId, bytes *[]byte) {

	// Total length matching our index bytes estimation:
	// 1 (pad) + 1 (index) + 32 (bitmask flags) + N * SlotsCapacity
	nodeIdSize := lenf.config.NodeIdRWriter.StorageBytes()
	totalBytesCount := 1 + 1 + 32 + (nodeIdSize * int(spec.SlotsCapacity))
	nodeBytes := make([]byte, totalBytesCount)

	nodeBytes[0] = 0                             // 1 byte padding
	nodeBytes[1] = byte(slotsNode.HashByteIndex) // 1 byte index

	flagsOffset := 2
	payloadOffset := flagsOffset + 32

	// 1. Build out the 256-bit flag bitmask and collect active target nodes sequentially
	activeChildren := make([]*shallowtreebyte.Node, 0, 256)

	for s := 0; s < 256; s++ {
		if !slotsNode.Slots[s].IsEmpty() {
			// Find byte bucket (0-31) and target bit location (0-7)
			byteNum := s >> 3
			bitNum := s & 0x07

			// Set the flag matching our bit layout query
			nodeBytes[flagsOffset+byteNum] |= 1 << bitNum

			// Collect the target child in strict iteration order
			activeChildren = append(activeChildren, slotsNode.Slots[s].NextNode)
		}
	}

	// 2. Write the 16-bit nodeIDs for active slots into the payload track
	for _, childNode := range activeChildren {
		nodeId, ok := nextLevelIdMap[childNode]
		if !ok {
			panic("Node pointer not found in map")
		}

		lenf.config.NodeIdRWriter.WriteID(nodeBytes[payloadOffset:payloadOffset+nodeIdSize], nodeId)
		payloadOffset += nodeIdSize
	}

	// 3. Right-pad trailing payload space with 0x0000
	// (Unpopulated capacity 'words' remain zero-initialized as bytes automatically from make)

	*bytes = append(*bytes, nodeBytes...)
}

func (lenf *LevelsEncoderNf) serializeTinyNode(slotsNode *shallowtreebyte.SlotsNode, spec *NodeFormatSpec,
	nextLevelIdMap map[*shallowtreebyte.Node]types.LocalNodeId, bytes *[]byte) {
	// FormatTiny consists of one byte hash byte index (no padding this time) followed
	// by a sequence of {one byte hash byte value, and N-bytes nodeId} with empty slots allowed (nodeId=0).
	// Crucially, the length of the sequence is NOT NECESSARILY equal to the number of non-empty slots.
	nodeIdSize := lenf.config.NodeIdRWriter.StorageBytes()
	nodeBytesCount := 1 + (1+nodeIdSize)*int(spec.SlotsCapacity)
	const spareRoom = 1 + (1+8)*5
	if nodeBytesCount > spareRoom {
		panic("Not enough room for tiny node")
	}
	nodeBytes := [spareRoom]byte{}
	nodeBytes[0] = byte(slotsNode.HashByteIndex)
	// Find the non-empty slots (which will always fit into the capacity, by prior arrangement)
	p := 1
	for sInt := 0; sInt < 256; sInt++ {
		if slotsNode.Slots[sInt].IsEmpty() {
			// If empty, it simply is not stored as part of the sequence!
		} else {
			nodeBytes[p] = byte(sInt)
			nodeId, ok := nextLevelIdMap[slotsNode.Slots[sInt].NextNode]
			if !ok {
				panic("Node pointer not found in map")
			}
			lenf.config.NodeIdRWriter.WriteID(nodeBytes[p+1:p+1+nodeIdSize], nodeId)
			p += 1 + nodeIdSize
		}
	}
	// If there is remaining capacity, we leave these as zero bytes (the zero bytes for nodeId imply
	// an empty slot)
	*bytes = append(*bytes, nodeBytes[:nodeBytesCount]...)
}
