package shallowtreebyte

import (
	"fmt"
	"math"
)

// A shallowtreebyte.ShallowTree has zero or more levels, with level n being pointed to by the root slot,
// where n is ShallowTree.PrefixNibblesN.
// A ShallowTree containing no hashes has no nodes, and so has no levels.
// The activeSlotsCount of a node is the number of its slots for which IsEmpty() is false.
// The "shape" of a level in a tree holds the following:
// 		* the histogram of activeSlotCounts of all the tree's nodes at that level
// The level shapes of a tree can be used to optimize the storage format of a tree (indexBytes).

// An empty ShallowTree has a single empty slot.
// A ShallowTree containing only one hash, has a single root slot pointing to a single leaf node.

type NodeCountType uint16

type LevelShape struct {
	// Leaf nodes appear at index 0 in this histogram (as they have no subnodes
	ActiveSlotCountHistogram [257]NodeCountType // Nodes at each level can have between 0 and 256 subnodes
}

// As up to 64 bytes per hash are supported (128 nibbles), and because leaf nodes are nodes,
// a tree can have a maximum of 129 levels (0 to 128).
// A level n describes what happens after n nibbles of the hash have been examined (possibly non-sequentially).
// Level 0 is the root, where a choice is made based on the first selected nibble of the hash.
// Level 127 is where a choice is made based on the last available nibble of a 64 byte hash.
// Level 128 may then contain leaf nodes, hence the maximum 129 levels for a 64 byte hash.

type TreeShape struct {
	LevelShapes           [129]LevelShape
	GreatestNodesPerLevel NodeCountType
}

func (st *ShallowTree) CountLevelShapes() *TreeShape {
	result := TreeShape{} // Already a valid (empty) TreeShape
	result.GreatestNodesPerLevel = 0

	st.VisitAllNodes(func(node *Node) {
		if node == nil {
			panic("Didn't expect nil node")
		}
		// If this node is a slots node, count this node's activeSlotsCount
		activeSlotsCount := 0
		if node.LeafNode == nil {
			for hashByteVal := 0; hashByteVal < 256; hashByteVal++ {
				if !node.SlotsNode.Slots[hashByteVal].IsEmpty() {
					activeSlotsCount++
				}
			}
		}
		// We want to count even when activeSlotsCount is zero (ie a leaf node)
		result.LevelShapes[node.Level].ActiveSlotCountHistogram[activeSlotsCount]++
	})
	for level := byte(0); level <= 128; level++ {
		nodesPerLevel := NodeCountType(0)
		// nodeCount at this level is the sum of the histogram
		for slotsCount := 0; slotsCount <= 256; slotsCount++ {
			nodesPerLevel += result.LevelShapes[level].ActiveSlotCountHistogram[slotsCount]
		}
		if nodesPerLevel > result.GreatestNodesPerLevel {
			result.GreatestNodesPerLevel = nodesPerLevel
		}
	}
	return &result
}

func (ts *TreeShape) Print() {
	fmt.Printf("Out of the non-zero counts:\n")
	for level := 0; level < 64; level++ {
		totalNodesAtThisLevel := NodeCountType(0)
		totalActiveSlotsSeen := NodeCountType(0)
		maxActiveSlotsSeen := 0
		minActiveSlotsSeen := math.MaxInt

		// Scan through the histogram buckets (0 to 256 active slots possible)
		for activeSlotsCount := 0; activeSlotsCount <= 256; activeSlotsCount++ {
			nodeCount := ts.LevelShapes[level].ActiveSlotCountHistogram[activeSlotsCount]

			if nodeCount > 0 {
				totalNodesAtThisLevel += nodeCount
				totalActiveSlotsSeen += NodeCountType(activeSlotsCount) * nodeCount

				if activeSlotsCount > maxActiveSlotsSeen {
					maxActiveSlotsSeen = activeSlotsCount
				}
				if activeSlotsCount < minActiveSlotsSeen {
					minActiveSlotsSeen = activeSlotsCount
				}
			}
		}

		if totalNodesAtThisLevel > 0 {
			averageActiveSlots := float64(totalActiveSlotsSeen) / float64(totalNodesAtThisLevel)
			fmt.Printf("Level %d: %d total nodes. Active slots per node metrics: (min: %d, av: %.1f, max: %d)\n",
				level, totalNodesAtThisLevel, minActiveSlotsSeen, averageActiveSlots, maxActiveSlotsSeen)
		}
	}
}
