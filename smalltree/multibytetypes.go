package smalltree

import "encoding/binary"

// This file defines the way that ids can be stored in various numbers of bytes

// IdTypeConstraint defines the underlying unsigned integers we support parameterizing on
type IdTypeConstraint interface {
	~uint16 | ~uint32 | ~uint64
}

// NByteIdConfig handles the dynamic bytes sizing for any underlying IdTypeConstraint
type NByteIdConfig[T IdTypeConstraint] interface {
	StorageBytes() int
	WriteID(b []byte, id T)
	ReadID(b []byte) T
}

// ID16 implements 16-bit IDs (2 bytes)
type ID16[T IdTypeConstraint] struct{}

func (ID16[T]) StorageBytes() int { return 2 }
func (ID16[T]) WriteID(b []byte, id T) {
	binary.LittleEndian.PutUint16(b[:2], uint16(id))
	if id-T(uint16(id)) != 0 {
		panic("ID16.WriteID: id is too large")
	}
}
func (ID16[T]) ReadID(b []byte) T { return T(binary.LittleEndian.Uint16(b[:2])) }

// ID24 implements 24-bit IDs (3 bytes)
type ID24[T IdTypeConstraint] struct{}

func (ID24[T]) StorageBytes() int { return 3 }
func (ID24[T]) WriteID(b []byte, id T) {
	val := uint32(id)
	b[0] = byte(val)
	b[1] = byte(val >> 8)
	b[2] = byte(val >> 16)
	if byte(id>>24) != 0 {
		panic("ID24.WriteID: id is too large")
	}
	if byte(id>>32) != 0 {
		panic("ID24.WriteID: id is too large")
	}
	if byte(id>>40) != 0 {
		panic("ID24.WriteID: id is too large")
	}
	if byte(id>>48) != 0 {
		panic("ID24.WriteID: id is too large")
	}
	if byte(id>>56) != 0 {
		panic("ID24.WriteID: id is too large")
	}
}
func (ID24[T]) ReadID(b []byte) T {
	return T(uint32(b[0]) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16))
}

// ID32 implements 32-bit IDs (4 bytes)
type ID32[T IdTypeConstraint] struct{}

func (ID32[T]) StorageBytes() int { return 4 }
func (ID32[T]) WriteID(b []byte, id T) {
	binary.LittleEndian.PutUint32(b[:4], uint32(id))
	if id-T(uint32(id)) != 0 {
		panic("ID16.WriteID: id is too large")
	}
}
func (ID32[T]) ReadID(b []byte) T { return T(binary.LittleEndian.Uint32(b[:4])) }

// ID40 implements 40-bit node IDs (5 bytes, allows access to over a trillion nodes - we won't need more!)
type ID40[T IdTypeConstraint] struct{}

func (ID40[T]) StorageBytes() int { return 5 }
func (ID40[T]) WriteID(b []byte, id T) {
	b[0] = byte(id)
	b[1] = byte(id >> 8)
	b[2] = byte(id >> 16)
	b[3] = byte(id >> 24)
	b[4] = byte(id >> 32)
	if byte(id>>40) != 0 {
		panic("ID24.WriteID: id is too large")
	}
	if byte(id>>48) != 0 {
		panic("ID24.WriteID: id is too large")
	}
	if byte(id>>56) != 0 {
		panic("ID24.WriteID: id is too large")
	}
}
func (ID40[T]) ReadID(b []byte) T {
	return T(b[0]) | (T(b[1]) << 8) | (T(b[2]) << 16) |
		T(b[3])<<24 | T(b[4])<<32
}
