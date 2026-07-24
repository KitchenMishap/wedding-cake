package types

// LocalNodeId For referring to a node within a particular level of a particular smalltree
type LocalNodeId uint16

const LocalNodeIdNoMatch LocalNodeId = ^LocalNodeId(0)
