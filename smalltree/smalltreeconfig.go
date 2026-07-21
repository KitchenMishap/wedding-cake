package smalltree

import "github.com/kitchenmishap/wedding-cake/shallowtreebyte"

// SmallTreeConfig holds some parameters of the SmallTree being codec'ed
type SmallTreeConfig struct {
	HashNibbleLength        shallowtreebyte.NibbleIndex
	ReassuranceBytesCount   byte
	NodeFormatSpecsPerLevel byte
	NodeIdConfig            NByteIdConfig[LocalNodeIdType]
	HashIndexIdConfig       NByteIdConfig[HashIndexIdType]
}
