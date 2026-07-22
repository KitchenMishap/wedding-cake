package smalltree

import (
	"fmt"
	"math"
	"sort"

	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

// nodeformat.go is concerned with choosing, optimizing, and specifying a variety of variously
// sized representations of nodes.
// Different node formats are better for representing nodes with different numbers of non-empty slots.

// NodeFormatChoice is a type for holding the choice of node format
type NodeFormatChoice byte

type LocalNodeCountType uint16
type BytesCountType uint64

// NodeFormatSlotCapacity is a type for configuring the maximum number of slots a format spec can hold
type NodeFormatSlotCapacity uint16 // Needs to represent up to 256

const (
	NodeFormatTiny NodeFormatChoice = iota
	NodeFormatMedium
	NodeFormatFull
	NodeFormatLeaf
)

// NodeFormatSpec is a way of describing a specific way of representing a node
type NodeFormatSpec struct {
	Format        NodeFormatChoice
	SlotsCapacity NodeFormatSlotCapacity
}

func (nfs *NodeFormatSpec) ByteSize(stc *SmallTreeConfig) int {
	idSize := stc.NodeIdConfig.StorageBytes()
	hashIdSize := stc.HashIndexIdConfig.StorageBytes()

	switch nfs.Format {
	case NodeFormatTiny:
		// FormatTiny: 1 byte (hash byte index) + Slots * (1 byte key + Node ID size)
		return 1 + int(nfs.SlotsCapacity)*(1+idSize)
	case NodeFormatMedium:
		// FormatMedium: 1 byte pad + 1 byte index + 32 bytes bitmask + (Slots * Node ID size)
		return 1 + 1 + 32 + (int(nfs.SlotsCapacity) * idSize)
	case NodeFormatFull:
		// FormatFull: 1 byte pad + 1 byte index + (256 * Node ID size)
		return 1 + 1 + (256 * idSize)
	case NodeFormatLeaf:
		// FormatLeaf: Reassurance bytes payload + Hash Id size
		return int(stc.ReassuranceBytesCount) + hashIdSize
	default:
		panic("unknown node format")
	}
}

type NodeFormatGroup struct {
	StartSlotsCount int                // The first slots count value that this group applies to
	EndSlotsCount   int                // One past the last slots count value that this group applies to
	NodesCount      LocalNodeCountType // The number of nodes that fall within this group, for the tree considered
	Spec            NodeFormatSpec     // The node format spec that is currently proposed for this group
	Bytes           BytesCountType     // Number of bytes used for all the nodes in this group
}

func (nfg *NodeFormatGroup) groupByteSize(stc *SmallTreeConfig) uint64 {
	return uint64(nfg.NodesCount) * uint64(nfg.Spec.ByteSize(stc))
}

func ProposeNodeFormatForSlotsCount(activeSlots int) NodeFormatSpec {
	result := NodeFormatSpec{}
	result.SlotsCapacity = NodeFormatSlotCapacity(activeSlots)
	if activeSlots == 0 {
		// With no active slots, we have a leaf node
		result.Format = NodeFormatLeaf
		return result
	}
	if activeSlots == 1 {
		panic("There should be no nodes with one active slot") // As it should already be a leaf
	}
	if activeSlots <= 5 {
		result.Format = NodeFormatTiny
		return result
	}
	if activeSlots >= 245 {
		result.Format = NodeFormatFull
		result.SlotsCapacity = 256
		return result
	}
	result.Format = NodeFormatMedium
	return result
}

func ProposeNodeFormatGroupsForLevelShape(shape shallowtreebyte.LevelShape, stc *SmallTreeConfig) []NodeFormatGroup {
	result := make([]NodeFormatGroup, 0, 257)
	for slotsCount := 0; slotsCount <= 256; slotsCount++ {
		if shape.ActiveSlotCountHistogram[slotsCount] > 0 {
			// At least one node needs exactly slotsCount slots
			group := NodeFormatGroup{}
			group.StartSlotsCount = slotsCount
			group.EndSlotsCount = slotsCount + 1
			group.NodesCount = LocalNodeCountType(shape.ActiveSlotCountHistogram[slotsCount])
			group.Spec = ProposeNodeFormatForSlotsCount(slotsCount)
			bytes := group.groupByteSize(stc) // We store this to avoid repeatedly recalculating
			group.Bytes = BytesCountType(bytes)
			result = append(result, group)
		}
	}
	return result
}

func AllowMergeGroups(left *NodeFormatGroup, right *NodeFormatGroup) bool {
	return right.Spec.Format == left.Spec.Format
}

func ProposeMergeGroups(left *NodeFormatGroup, right *NodeFormatGroup, stc *SmallTreeConfig) NodeFormatGroup {
	if right.StartSlotsCount < left.EndSlotsCount {
		panic("Illegal group merge: overlapping")
	}
	if right.Spec.Format != left.Spec.Format {
		panic("Illegal group merge: different formats")
	}
	result := NodeFormatGroup{}
	result.StartSlotsCount = left.StartSlotsCount
	result.EndSlotsCount = right.EndSlotsCount
	result.NodesCount = left.NodesCount + right.NodesCount
	result.Spec.Format = left.Spec.Format
	result.Spec.SlotsCapacity = right.Spec.SlotsCapacity
	bytes := result.groupByteSize(stc)
	result.Bytes = BytesCountType(bytes)
	return result
}

func RefineNodeFormatGroups(groups []NodeFormatGroup, stc *SmallTreeConfig) ([]NodeFormatGroup, bool) {
	// Try each neighbouring pair of groups
	// We are looking for the lowest cost merge (counted in bytes)
	lowestCost := uint64(math.MaxUint64)
	bestProposalLeft := -1
	bestProposedMerge := NodeFormatGroup{}
	for left := 0; left < len(groups)-1; left++ {
		right := left + 1
		if AllowMergeGroups(&groups[left], &groups[right]) {
			leftBytes := groups[left].Bytes
			rightBytes := groups[right].Bytes
			proposedGroup := ProposeMergeGroups(&groups[left], &groups[right], stc)
			mergedBytes := proposedGroup.Bytes
			proposedCost := mergedBytes - (leftBytes + rightBytes)
			if uint64(proposedCost) < lowestCost {
				lowestCost = uint64(proposedCost)
				bestProposalLeft = left
				bestProposedMerge = proposedGroup
			}
		}
	}
	if bestProposalLeft == -1 {
		return groups, false
	}
	// Replace (left, right) with (merged)
	result := make([]NodeFormatGroup, 0, len(groups)-1)
	result = append(result, groups[:bestProposalLeft]...)
	result = append(result, bestProposedMerge)
	result = append(result, groups[bestProposalLeft+2:]...)

	return result, true
}

type LevelFormat struct {
	// These are sorted for efficiency, most popular first (but with FormatTiny's at the end which use odd byte numbers)
	Groups []NodeFormatGroup
	// These have the same index as Groups
	NodeIdAllocations []NodeIdAllocation
	// These are indexed by active slot count, and hold indices into the above
	SlotCountToGroup [257]byte
}
type TreeFormat struct {
	// These are indexed by level, with root at index 0
	LevelSpecs [129]LevelFormat
}
type NodeIdAllocation struct {
	NextAvailableNodeId LocalNodeIdType
	AvailableNodeIds    LocalNodeIdType
	OriginalNodeIds     LocalNodeIdType
}

func (tns *TreeFormat) InitializeNodeIdAllocations() {
	levels := len(tns.LevelSpecs)
	for level := 0; level < levels; level++ {
		// Node Id's are now PER LEVEL; you need a level AND a node id to identify a node
		nodeId := LocalNodeIdType(0) // 0 is no longer a "special meaning" id

		// Note that duplicates are no longer tolerated

		groups := len(tns.LevelSpecs[level].Groups)
		tns.LevelSpecs[level].NodeIdAllocations = make([]NodeIdAllocation, groups)
		for group := 0; group < groups; group++ {
			// These will later allocate us NodeIDs from each group in each level
			nodes := tns.LevelSpecs[level].Groups[group].NodesCount
			tns.LevelSpecs[level].NodeIdAllocations[group] = NodeIdAllocation{
				NextAvailableNodeId: nodeId,
				AvailableNodeIds:    LocalNodeIdType(nodes),
				OriginalNodeIds:     LocalNodeIdType(nodes),
			}
			//fmt.Printf("Allocating %d node ids for level %d group %d\n", nodes, level, group)
			// These will later tell us, for a given active slot count, which
			// group should allocate us NodeIDs
			start := tns.LevelSpecs[level].Groups[group].StartSlotsCount
			end := tns.LevelSpecs[level].Groups[group].EndSlotsCount
			for activeSlotsCount := start; activeSlotsCount < end; activeSlotsCount++ {
				tns.LevelSpecs[level].SlotCountToGroup[activeSlotsCount] = byte(group)
			}
			nodeId += LocalNodeIdType(nodes)
		}
	}
}

func (tns *TreeFormat) AllocateIdAndSpecForNode(level byte, activeSlotsCount int) (LocalNodeIdType, NodeFormatSpec) {
	group := tns.LevelSpecs[level].SlotCountToGroup[activeSlotsCount]

	// Read directly from the underlying slice index
	alloc := tns.LevelSpecs[level].NodeIdAllocations[group]
	if alloc.AvailableNodeIds == 0 {
		fmt.Printf("Panicking. Level=%d, Group=%d, StartSlotsCount=%d, EndSlotsCount=%d\n",
			level, group, tns.LevelSpecs[level].Groups[group].StartSlotsCount, tns.LevelSpecs[level].Groups[group].EndSlotsCount)
		fmt.Printf("activeSlotsCount=%d, OriginalNodeIds=%d\n", activeSlotsCount, alloc.OriginalNodeIds)
		panic("Too many nodes")
	}

	nodeID := alloc.NextAvailableNodeId

	// Update the live data tracking fields directly back inside the slice holder
	tns.LevelSpecs[level].NodeIdAllocations[group].NextAvailableNodeId++
	tns.LevelSpecs[level].NodeIdAllocations[group].AvailableNodeIds--

	nodeSpec := tns.LevelSpecs[level].Groups[group].Spec
	return nodeID, nodeSpec
}

func ChooseNodeFormatSpecsForTreeShape(treeShape *shallowtreebyte.TreeShape, stc *SmallTreeConfig) *TreeFormat {
	// 4 are required to allow for a single formatTiny, a formatMedium, a formatFull. and a formatLeaf
	// BUT 4 is not sensible as 4 could be achieved without this expensive procedure!
	// A sensible number is perhaps 8 or more
	if stc.NodeFormatSpecsPerLevel < 5 {
		panic("nodeSpecFormatSpecsPerLevel must be at least 5")
	}
	result := TreeFormat{}
	for level := byte(0); level <= 128; level++ {
		// Initially we propose a separate NodeFormatSpec for each count of slots represented in the level
		nfgs := ProposeNodeFormatGroupsForLevelShape(treeShape.LevelShapes[level], stc)
		// originalCount := len(nfgs)
		originalCost := BytesCountType(0)
		for _, nfg := range nfgs {
			originalCost += nfg.Bytes
		}
		// Then we repeatedly reduce by merging pairs until we reach nodeFormatSpecsPerLevel
		for {
			if len(nfgs) <= int(stc.NodeFormatSpecsPerLevel) {
				break
			}
			proposed, reduced := RefineNodeFormatGroups(nfgs, stc)
			if !reduced {
				panic("Could not reduce node format specs! nodeFormatSpecsPerLevel too low?")
			}
			nfgs = proposed
		}
		cost := BytesCountType(0)
		for _, nfg := range nfgs {
			cost += nfg.Bytes
		}
		if cost > 0 {
			//fmt.Printf("Level %d optimization: Reduced nfgs -> %d, bytes %d -> %d\n",
			//	level, len(nfgs), originalCost, cost)
			//fmt.Printf("ActiveSlotCountHistogram[0]: %d\n", treeShape.LevelShapes[level].ActiveSlotCountHistogram[0])
		}
		// Sort descending by NodesCount (so popular nodeFormatSpecs appear first)
		// Now with FormatTiny forced to the end of the sort (they have odd numbers of bytes, so we don't want them
		// to result in the others being on odd byte boundaries)
		// Sort primarily to push FormatTiny to the end, and secondarily by NodesCount descending
		sort.SliceStable(nfgs, func(i, j int) bool {
			iIsTiny := nfgs[i].Spec.Format == NodeFormatTiny
			jIsTiny := nfgs[j].Spec.Format == NodeFormatTiny

			// Rule 1 (Primary): If one is Tiny and the other isn't, non-Tiny comes first
			if iIsTiny != jIsTiny {
				return jIsTiny // Returns true if j is Tiny (meaning i is non-Tiny, so i comes first)
			}

			// Rule 2 (Secondary): If they are of the same category, bigger NodesCount comes first
			return nfgs[i].NodesCount > nfgs[j].NodesCount
		})
		result.LevelSpecs[level].Groups = nfgs
	}
	return &result
}

func DesignTreeFormat(tree *shallowtreebyte.ShallowTree, stc *SmallTreeConfig) *TreeFormat {
	treeShape := tree.CountLevelShapes()
	if treeShape.GreatestNodesPerLevel > 65535 {
		panic("Too many nodes at a single level")
	}
	result := ChooseNodeFormatSpecsForTreeShape(treeShape, stc)
	result.InitializeNodeIdAllocations()
	return result
}
