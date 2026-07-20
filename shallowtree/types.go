package shallowtree

// shallowtree handles cryptographic hashes of various lengths as slices of nibbles (4 bits)
type NibbleVal byte   // Values 0 to 15
type NibbleIndex byte // For 512 bit hash this would be between 0 and 127

// Presentation indices ("PI") are the order in which the hashes are presented to the shallowtree
// In contrast to previous implementations, zero is a valid presentation index
type PiType uint64

const PiNoMatch PiType = 0xFFFFFFFFFFFFFFFF
