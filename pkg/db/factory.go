package db

import (
	"encoding/binary"
)

type DB interface {
	Open() error
	Close() error
	SaveObject(object PBBytesTransfer) error
	GetObjectByID(id []byte, object PBBytesTransfer) error
	LoadObjectsByPrefix(prefix []byte, objects PBBytesAppender) error
}

type PBBytesTransfer interface {
	Key() []byte
	TransObjectToProtoBytes() ([]byte, error)
	TransProtoBytesToObject([]byte) error
}

type PBBytesAppender interface {
	AppendPBBytesObject(bs []byte) error
}

func Int64ToBytes(n int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(n))
	return b
}

func BytesToInt64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

//func GetStorageEngine(name string) DB {
//	switch name {
//	case "badger":
//		return &badger.BadgerDB{}
//	case "boltdb":
//		return &boltdb.BoltDB{}
//	default:
//		return &boltdb.BoltDB{}
//	}
//	return nil
//}
