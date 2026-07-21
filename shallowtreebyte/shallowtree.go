package shallowtreebyte

import (
	"math"
	"slices"
	"sort"
)

// shallowtreebyte.ShallowTree is an adaptation of shallowtree.ShallowTree which forks nodes on bytes not nibbles.

// ShallowTree is a mechanism whereby a nibble index of the hash (after the first n nibbles) is selected
// to achieve the widest possible "spread" across sub-nodes at a node (fork) in the tree.
// shallow is a tree graph in which vertices are called Nodes and arcs are called (populated) Slots.
// (A leaf is a Node with no Slots)
// SingleTree does not tolerate duplicate hashes; these are presumed filtered out by prior code.

// ShallowTree can be configured to ignore the first n nibbles; this is useful where those are used
// for preselection of a particular subtree.
// * It assumes that the first n bytes of the hash have been handled elsewhere
//    * More specifically, when generating the tree, it assumes that those bytes of the supplied hashes pertain to this tree
//    * And on lookup, it assumes that those bytes of the hash being looked up have already been found to match
// * n is a configuration parameter

// ShallowTree does not support forward lookup (index to hash.)
// ShallowTree does not support duplicated hashes

type ShallowTree struct {
	NibblesLength                  NibbleIndex // eg 64 for SHA-256 hashes, 40 for RIPEMD-160
	PrefixNibblesN                 NibbleIndex
	ReassuranceNibblesCount        NibbleIndex
	NibbleValueExaminedPriorToTree NibbleVal

	HashCount int
	// The root is a slot, not a node.
	RootSlot Slot // The root slot has "Depth" 0
}

// Node represents a subtree, after a particular set of hash nibbles
// (PrefixNibblesN nibbles, in the case of the root node) have been examined, and particular values found in them.
// It specifies which nibble of the hash to examine next, and what to do in the case of
// each of the possible 16 values found (16 slots).
// It represents a "fork" in the tree... one slot POINTS to a node, and from a node up to 16 Slots POINT to further Nodes
type Node struct {
	Level                  NibbleIndex // The Node at "Level" n is the pointed to by the Slot at "Depth" n
	NibbleValueLeadingHere NibbleVal   // The examined nibble value that led to this node
	LeafNode               *LeafNode   // One of these two
	SlotsNode              *SlotsNode  // pointers will be nil
}

type LeafNode struct {
	ReassuranceHashNibbles []NibbleVal // Additional nibbles from the hash to give statistical reassurance
	PresentationIndex      PiType      // The index that was initially presented with the hash corresponding to this leaf
	Hash                   []NibbleVal // The entire hash for this leaf
}

type SlotsNode struct {
	HashByteIndex ByteIndex // Which byte of the hash to examine to choose between the slots
	Slots         [256]Slot // What to do when each of 256 possible nibble values are found at HashByteIndex
}

// Slot represents how the tree progresses, when the next specified nibble of the hash has been examined.
// It represents an "arc" (line) of the tree graph, sitting between two Nodes (vertices)
type Slot struct {
	NextNode *Node // If nil, arc does not exist
}

// When created with ShallowTreeSlot{}, they start out as IsEmpty()
func (sts Slot) IsEmpty() bool {
	return sts.NextNode == nil
}

// HashPi is used in the parameter to GenerateShallowTree()
type HashPi struct {
	Hash              []NibbleVal // len(Hash) must equal ShallowTree.NibbleLength
	PresentationIndex PiType
}

// GenerateShallowTree generates a tree from the supplied hashes and presentationIndices
func GenerateShallowTree(input []HashPi, prefixNibblesN NibbleIndex, nibblesLength NibbleIndex,
	reassuranceNibbles NibbleIndex, prevNibbleValueExamined NibbleVal) *ShallowTree {
	if prefixNibblesN > nibblesLength {
		panic("PrefixNibblesN cannot exceed nibbleLength")
	}
	if nibblesLength < 2 || nibblesLength > 128 {
		panic("Only 2 to 128 nibble hashes are supported")
	}
	if nibblesLength&1 == 1 {
		panic("Only even nibble lengths are supported")
	}
	if reassuranceNibbles > nibblesLength {
		panic("Reassurance nibbles must be less than or equal to nibble length")
	}
	for i := range input {
		if input[i].Hash == nil {
			panic("Malformed input: ShallowTreeHash contains a nil Hash slice")
		}
		if len(input[i].Hash) != int(nibblesLength) {
			panic("Malformed input: HashPi slice length does not match specified nibbleLength")
		}
	}
	result := ShallowTree{}
	result.NibblesLength = nibblesLength
	result.PrefixNibblesN = prefixNibblesN
	result.ReassuranceNibblesCount = reassuranceNibbles
	result.HashCount = len(input)
	result.NibbleValueExaminedPriorToTree = prevNibbleValueExamined
	if len(input) == 0 {
		// No hashes, empty tree (no nodes)
		// The root slot is already IsEmpty()
		return &result
	}
	// Important special case for a lone hash, because recurseGenerateNode() assumes at least two hashes.
	if len(input) == 1 {
		// On creation of ShallowTree{] above, result.RootSlot is currently IsEmpty().
		// We need to point it to single leaf node
		leafNode := LeafNode{}
		leafNode.PresentationIndex = input[0].PresentationIndex
		leafNode.ReassuranceHashNibbles = make([]NibbleVal, reassuranceNibbles)
		copy(leafNode.ReassuranceHashNibbles, input[0].Hash[:reassuranceNibbles])
		leafNode.Hash = make([]NibbleVal, nibblesLength)
		copy(leafNode.Hash, input[0].Hash)
		node := Node{}
		node.Level = prefixNibblesN
		node.NibbleValueLeadingHere = prevNibbleValueExamined
		node.LeafNode = &leafNode
		node.SlotsNode = nil
		result.RootSlot.NextNode = &node
		return &result
	}
	// Because we will be mutating it (sorting it), we take a copy of the input so as not to surprise the caller
	inputCopy := make([]HashPi, len(input))
	copy(inputCopy, input)

	// Create root node and recursively its children
	unusedNibbles := NewNibblesFlags(prefixNibblesN, nibblesLength)

	// We start recursing at hash nibble index PrefixBytesN
	rootNode := result.recurseGenerateNode(inputCopy, unusedNibbles, prefixNibblesN, prevNibbleValueExamined)
	result.RootSlot.NextNode = rootNode
	return &result
}

// LookupHash uses ShallowTree to lookup one presentationIndex if it exists.
// If the tree contains no matches for the hash, PiNoMatch is returned.
func (st *ShallowTree) LookupHash(hash []NibbleVal) PiType {
	if len(hash) != int(st.NibblesLength) {
		panic("Wrong hash length")
	}
	// A tree that has no nodes (it contains no hashes), will always fail without even looking at the hash
	if st.RootSlot.NextNode == nil {
		return PiNoMatch
	}
	node := st.RootSlot.NextNode

	// Keep track (by way of a mask) of which bytes of the mask have been examined
	unusedNibbles := NewNibblesFlags(st.PrefixNibblesN, st.NibblesLength)

	const dummy = -1
	mostRecentNibbleIndexExamined := dummy
	for {
		leafNode := node.LeafNode
		if leafNode != nil {
			// We've reached a leaf node, a potential match

			// Firstly, to get to a leaf the last examined byte must match what the node says.
			if mostRecentNibbleIndexExamined == dummy {
				// No bytes were examined in this tree!
				// Nothing to base a panic on
			} else {
				if hash[mostRecentNibbleIndexExamined] != node.NibbleValueLeadingHere {
					panic("ShallowTree lost track of examined bytes")
				}
			}
			// Check whole hash (not just the reassurance bytes)
			if !slices.Equal(leafNode.Hash, hash) {
				return PiNoMatch
			}
			return leafNode.PresentationIndex // A match
		}
		// Not a leaf node. It's a slots node. Examine the slots...
		byteIndexToExamine := node.SlotsNode.HashByteIndex
		mostRecentNibbleIndexExamined = int(byteIndexToExamine*2 + 1) // Lets say we examined the LS nibble
		// Let's check we're not being asked to examine one of the prefix bytes
		if NibbleIndex(byteIndexToExamine*2) < st.PrefixNibblesN {
			panic("Nibble index is part of the prefix")
		}
		// Mark byte index as examined
		unusedNibbles.ClearFlagOrPanicByte(byteIndexToExamine)

		// Reconstruct the byte in the hash from the nibbles in the hash
		// Most significant nibble is at [byteIndexTExamine*2]
		examinedNibble1 := hash[byteIndexToExamine*2]
		// Least significant nibble is at [byteIndexToExamine*2+1]
		examinedNibble0 := hash[byteIndexToExamine*2+1]
		examinedByteValue := ByteVal(examinedNibble0 | examinedNibble1<<4)

		if node.SlotsNode.Slots[examinedByteValue].IsEmpty() {
			return PiNoMatch
		}
		node = node.SlotsNode.Slots[examinedByteValue].NextNode
	}
}

// We implement a visitor pattern to enable you to visit every node in the tree

// ShallowTreeNodeVisitor is the signature for your custom processing functions
type NodeVisitor func(node *Node)

type traversalFrame struct {
	node *Node
}

// VisitAllNodes performs a complete iterative stack traversal of the tree,
// invoking the supplied visitor function on every leaf node or slots node.
func (st *ShallowTree) VisitAllNodes(visitor NodeVisitor) {
	// 1. Return if there are no nodes
	if st.RootSlot.IsEmpty() {
		return
	}

	// 2. Initialize our iterative LIFO stack with the first branch node frame
	stack := []traversalFrame{{
		node: st.RootSlot.NextNode,
	}}

	// 3. Process the explicit stack loop
	for len(stack) > 0 {
		// Pop the top frame off the heap slice
		currentFrame := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Whether its a leaf node or a slots node, visit it (user supplied callback)
		visitor(currentFrame.node)

		// If it's a slot node, go through its slots
		if currentFrame.node.LeafNode == nil {
			// Iterate across the full fixed 256-slot routing block
			for byteValInt := 0; byteValInt < 256; byteValInt++ {
				byteVal := ByteVal(byteValInt)
				slot := &currentFrame.node.SlotsNode.Slots[byteVal]

				if !slot.IsEmpty() {
					// push the node that the slot points to (its frame) onto the stack to explore later
					nextFrame := traversalFrame{
						node: slot.NextNode,
					}
					stack = append(stack, nextFrame)
				}
			}
		}
	}
}

func (st *ShallowTree) CountNodes() int {
	count := 0
	st.VisitAllNodes(func(node *Node) {
		count++
	})
	return count
}

// LeavesVisitor a similar pattern for visiting all leaves
type LeavesVisitor func(node *LeafNode)

// VisitAllLeaves performs a complete iterative stack traversal of the tree,
// invoking the supplied visitor function on every leaf node (NOT slots nodes)
func (st *ShallowTree) VisitAllLeaves(visitor LeavesVisitor) {
	st.VisitAllNodes(func(node *Node) {
		if node.SlotsNode == nil {
			visitor(node.LeafNode)
		}
	})
}

func (st *ShallowTree) CountLeaves() int {
	count := 0
	st.VisitAllLeaves(func(leaf *LeafNode) {
		count++
	})
	return count
}

// Helper to quickly find the active slots count for a node
func (stn *Node) activeSlotsCount() int {
	if stn.SlotsNode == nil {
		return 0
	}
	count := 0
	for i := 0; i < 16; i++ {
		if !stn.SlotsNode.Slots[i].IsEmpty() {
			count++
		}
	}
	return count
}

// recurseGenerateNode() is a recursive call to populate a SingleTree based on a slice of SingleTreeHash.
// The SingleTreeHash will be modified (sorted), so send in a copy if this is not tolerated.
// It returns a pointer to a new node. Duplicate hashes are not tolerated.
func (st *ShallowTree) recurseGenerateNode(inputCopy []HashPi,
	unusedNibbles NibblesFlags, level NibbleIndex,
	prevNibbleExamined NibbleVal) *Node {
	if len(inputCopy) < 2 {
		panic("recurseGenerateNode() should only be called with multiple hashes")
	}
	node := Node{}
	node.Level = level
	node.NibbleValueLeadingHere = prevNibbleExamined
	slotsNode := SlotsNode{}
	node.SlotsNode = &slotsNode
	node.LeafNode = nil // It won't be a leaf node because we know we have multiple hashes

	// Try partitioning by each of the (up to 64) nibble-pairs in the hashes. Just the ones we haven't used
	const dummy = -1
	byteIndexInt := dummy
	maxEntropyFound := float64(0)
	maxEntropyIndex := byteIndexInt
	for byteIndex := ByteIndex(0); byteIndex < ByteIndex(st.NibblesLength/2); byteIndex++ {
		unused := unusedNibbles.FlagValByte(byteIndex)
		if unused {
			entropy := partitioningEntropy(inputCopy, byteIndex)
			if entropy > maxEntropyFound {
				maxEntropyIndex = int(byteIndex)
				maxEntropyFound = entropy
			}
		}
	}

	if maxEntropyFound == 0.0 {
		// We know there were multiple hashes input to this function.
		// An entropy of 0 indicates these hashes are all duplicates.
		// Duplicates are not tolerated! They should have first been removed by a higher authority
		panic("Duplicate hashes are not tolerated")
	}

	// Use the best one
	bi := maxEntropyIndex
	if bi*2 < int(st.PrefixNibblesN) {
		panic("Byte index is part of the prefix")
	}
	unusedNibbles.ClearFlagOrPanicByte(ByteIndex(bi))
	node.SlotsNode.HashByteIndex = ByteIndex(bi)

	// Now we'll need to sort by that byte, so we can pass subsets of the hash list to each child.
	// Use a stable sort to prevent Go from randomly scrambling the presentation order of duplicate hashes.
	// This is a little tricky for a nibbles array...
	sort.SliceStable(inputCopy, func(i int, j int) bool {
		byteValI := inputCopy[i].Hash[bi*2]<<4 | inputCopy[i].Hash[bi*2+1]
		byteValJ := inputCopy[j].Hash[bi*2]<<4 | inputCopy[j].Hash[bi*2+1]
		return byteValI < byteValJ
	})

	// We have decided to split this node into 256 (fork the tree) based on the value found in the hashes
	// at byte index bi. Consider each value we might find at bi, and what to do.
	index := 0
	for byteValInt := 0; byteValInt < 256; byteValInt++ {
		byteVal := ByteVal(byteValInt)
		nibbleVal0 := NibbleVal(byteVal & 0x0F)
		nibbleVal1 := NibbleVal(byteVal >> 4)
		startIndex := index // index into the list of hashes
		// Look for as many "nibbleVal's at bi" in a row that we can find in the list of hashes
		// The most significant nibble is at [bi*2] and the least significant nibble is at [bi*2+1]
		for index < len(inputCopy) && inputCopy[index].Hash[bi*2] == nibbleVal1 && inputCopy[index].Hash[bi*2+1] == nibbleVal0 {
			index++
		}
		if index == startIndex {
			// Didn't find any; empty slot (the bytes examined up to this point in the tree lead to no hash entries)
			// (and the slot was already created empty; nothing to do)
		} else if index == startIndex+1 {
			// Found exactly one hash at startIndex, so we need a leaf node, and don't recurse
			leafNode := LeafNode{}
			leafNode.PresentationIndex = inputCopy[startIndex].PresentationIndex
			leafNode.Hash = make([]NibbleVal, st.NibblesLength)
			copy(leafNode.Hash, inputCopy[startIndex].Hash)

			// The reassurance hash bytes are (sequentially) a maximum of st.ReassuranceBytesCount
			// bytes, out of the hash bytes that haven't been examined yet
			leafNode.ReassuranceHashNibbles = make([]NibbleVal, 0, st.ReassuranceNibblesCount)
			localUnusedFlags := unusedNibbles.Copy()
			ind := NibbleIndex(0)
			for b := NibbleIndex(0); b < st.ReassuranceNibblesCount; b++ {
				// Abort if all nibbles have been examined
				if localUnusedFlags.IsEmpty() {
					break
				}
				// Find the next hash byte that has not yet been examined
				for localUnusedFlags.FlagVal(ind) == false {
					ind++
				}
				// Mark it as examined in our local copy
				localUnusedFlags.ClearFlagOrPanic(ind)
				// Record the byte value
				nibbleValue := inputCopy[startIndex].Hash[ind]
				leafNode.ReassuranceHashNibbles = append(leafNode.ReassuranceHashNibbles, nibbleValue)
			}

			newNode := Node{}
			newNode.Level = level + 1
			newNode.NibbleValueLeadingHere = nibbleVal0
			newNode.LeafNode = &leafNode
			newNode.SlotsNode = nil
			node.SlotsNode.Slots[byteVal].NextNode = &newNode
		} else if index > startIndex+1 {
			// Found more than one, we need a fully fledged slots node child
			// We add TWO to the level, as w have examined two nibbles (one byte)
			childNode := st.recurseGenerateNode(inputCopy[startIndex:index], unusedNibbles, level+2, nibbleVal0)
			node.NibbleValueLeadingHere = prevNibbleExamined
			node.SlotsNode.Slots[byteVal].NextNode = childNode
		} else {
			panic("Error in code logic")
		}
	} // for nibbleVal
	return &node
}

// Partitioning entropy if we were to partition by a particular byte index of the hash
// Shannon Entropy calculation: Maximizes value for an even distribution
func partitioningEntropy(input []HashPi, hashByteIndex ByteIndex) float64 {
	var byteValCounts [256]int
	for i := range input {
		// Most significant nibble is at [hashByteIndex*2]
		// Least significant nibble is at [hashByteIndex*2+1]
		nibbleVal1 := input[i].Hash[hashByteIndex*2]
		nibbleVal0 := input[i].Hash[hashByteIndex*2+1]
		byteVal := ByteVal(nibbleVal1<<4 | nibbleVal0)
		byteValCounts[byteVal]++
	}
	// Loop over the partitions, calculating probabilities and entropy
	total := float64(len(input))
	entropySum := float64(0)
	for byteValInt := 0; byteValInt < 256; byteValInt++ {
		byteVal := ByteVal(byteValInt)
		count := byteValCounts[byteVal]
		if count > 0 { // Avoid log of 0
			prob := float64(count) / total
			entropySum -= prob * math.Log2(prob)
		}
	}
	return entropySum
}
