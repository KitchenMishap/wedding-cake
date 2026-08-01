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
	NodeIdRWriter           NByteIdConfig[types.LocalNodeId]
	LocalPiRWriter          NByteIdConfig[types.LocalPi]
	//	PrefixIndexRWriter      NByteIdConfig[types.PrefixIndex]
	SuffixIndexRWriter NByteIdConfig[types.SuffixIndex]
}
