package smalltree

import (
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
)

// SmallTree encodes/decodes a subtree as a serialized indexBytes/nodesBytes pair of byte slices.
// A SmallTree is intended to be a self contained description of a subtree holding approximately 65536 hashes.
// indexBytes contains data on how to interpret nodesBytes.

// Initially, just to get something working, there are just two kinds of node encoded in nodesBytes:
// FormatSlots, and FormatLeaf.

func EncodeSmallTree(tree *shallowtreebyte.ShallowTree) ([]byte, []byte) {
	panic("not implemented")
}
