package smalltree

import (
	"github.com/kitchenmishap/wedding-cake/shallowtreebyte"
	"github.com/kitchenmishap/wedding-cake/types"
)

// DecoderLevelsTest should only be used for testing as it holds everything in memory
type DecoderLevelsTest struct {
	prefixNibbles byte
	levels        []LevelDecoder
	config        *SmallTreeConfig
	rootNodeId    types.LocalNodeId
	rootLevel     byte
}

func (dlt DecoderLevelsTest) Lookup(hash []shallowtreebyte.NibbleVal) types.LocalPi {
	level := dlt.rootLevel
	nodeId := dlt.rootNodeId

	decoder := dlt.levels[level]

	unusedNibbleFlags := shallowtreebyte.NewNibblesFlags(shallowtreebyte.NibbleIndex(dlt.prefixNibbles), dlt.config.HashNibbleLength)

	for {
		node := decoder.GetNode(nodeId)
		if node.IsLeafNode() {
			ln := node.GetLeafNode()
			// Check ressurance bytes
			reassuranceBytes := ln.GetReassuranceBytes()
			byteIndexToExamine := shallowtreebyte.ByteIndex(0)
			for reassuranceByteIndex := 0; reassuranceByteIndex < len(reassuranceBytes); reassuranceByteIndex++ {
				// Find the next unexamined byte index in the hash
				for !unusedNibbleFlags.FlagValByte(byteIndexToExamine) {
					byteIndexToExamine++
				}
				unusedNibbleFlags.ClearFlagOrPanicByte(byteIndexToExamine)
				nibble0 := hash[byteIndexToExamine*2+1]
				nibble1 := hash[byteIndexToExamine*2]
				byt := byte(nibble0 | nibble1<<4)
				if reassuranceBytes[reassuranceByteIndex] != byt {
					return types.LocalPiNoMatch
				}
			}
			return ln.GetHashId()
		} else {
			sn := node.GetSlotsNode()
			byteIndex := sn.GetHashByteToExamine()
			nibble1 := hash[byteIndex*2]   // MS
			nibble0 := hash[byteIndex*2+1] // LS
			byt := nibble0 | nibble1<<4
			unusedNibbleFlags.ClearFlagOrPanicByte(byteIndex)
			nodeId = sn.GetNextNode(SlotSelectorType(byt))
			if nodeId == types.LocalNodeIdNoMatch {
				return types.LocalPiNoMatch
			}
			level += 2
			decoder = dlt.levels[level]
		}
	}
}
