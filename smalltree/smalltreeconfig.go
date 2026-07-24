package smalltree

import (
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"
)

// SmallTreeConfig holds some parameters of the SmallTree being codec'ed
type SmallTreeConfig struct {
	HashNibbleLength        shallowtreebyte.NibbleIndex
	ReassuranceBytesCount   byte
	NodeFormatSpecsPerLevel byte
	NodeIdConfig            NByteIdConfig[types.LocalNodeId]
	HashIndexIdConfig       NByteIdConfig[types.LocalPi]
}
