package common

import (
	"encoding/binary"
	"io"
)

var (
	binaryOrder = binary.LittleEndian
)

type StructInfo struct {
	ID      StructureID `json:"StructInfoID"`
	Version uint8       `json:"StructInfoVersion"`
}

func (s StructInfo) StructInfo() StructInfo {
	return s
}

type StructureID [8]byte

func (s StructureID) String() string {
	return string(s[:])
}

type Structure interface {
	io.ReaderFrom
	io.WriterTo
	TotalSize() uint64
	//PrettyString(depth uint, withHeader bool, opts ...pretty.Option) string
}

type Element interface {
	Structure
	ReadDataFrom(r io.Reader) (int64, error)
	GetStructInfo() StructInfo
	SetStructInfo(StructInfo)
}

type ElementsContainer interface {
	Structure
	GetFieldByStructID(structID string) interface{}
}

type Manifest interface {
	Structure
}
