package badgerdb

import (
	"github.com/lionstory/tsslib-grpc/pkg/db"
	"github.com/dgraph-io/badger/v3"
)

type BadgerDB struct {
	Options *badger.Options
	db      *badger.DB
}

func (b *BadgerDB) Open() error {
	if b.Options != nil {
		if db, err := badger.Open(*b.Options); err != nil {
			return err
		} else {
			b.db = db
		}
	} else {
		if db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true)); err != nil {
			return err
		} else {
			b.db = db
		}
	}
	return nil
}

func (b *BadgerDB) OpenInMemory() error {
	if db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true)); err != nil {
		return err
	} else {
		b.db = db
	}
	return nil
}

func (b *BadgerDB) Close() error {
	return b.db.Close()
}

//func (b *BadgerDB) LoadProjects() (map[int64]*sss.Project, error) {
//	projects := make(map[int64]*sss.Project)
//	err := b.db.View(func(txn *badger.Txn) error {
//		it := txn.NewIterator(badger.DefaultIteratorOptions)
//		defer it.Close()
//		prefix := []byte("proj_")
//		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
//			item := it.Item()
//			var project sss.Project
//			err := item.Value(func(val []byte) error {
//				return sss.GenerateProjectFromProtoBytes(val, &project)
//			})
//			if err != nil {
//				return err
//			}
//			projects[project.ProjectID] = &project
//		}
//		return nil
//	})
//	return projects, err
//}
//
//func (b *BadgerDB) GetProjectByID(id int64) (*sss.Project, error) {
//	var project sss.Project
//	err := b.db.View(func(txn *badger.Txn) error {
//		item, err := txn.Get([]byte(sss.ProjectKey(id)))
//		if err != nil {
//			return err
//		}
//		return item.Value(func(val []byte) error {
//			return sss.GenerateProjectFromProtoBytes(val, &project)
//		})
//	})
//	return &project, err
//}
//
//func (b *BadgerDB) SaveProject(project *sss.Project) error {
//	err := b.db.Update(func(txn *badger.Txn) error {
//		bs, err := sss.GenerateProjectProtoBytes(project)
//		if err != nil {
//			return err
//		}
//		key := sss.ProjectKey(project.ProjectID)
//		return txn.Set([]byte(key), bs)
//	})
//	return err
//}

func (b *BadgerDB) SaveObject(object db.PBBytesTransfer) error {
	err := b.db.Update(func(txn *badger.Txn) error {
		bs, err := object.TransObjectToProtoBytes()
		if err != nil {
			return err
		}
		return txn.Set(object.Key(), bs)
	})
	return err
}

func (b *BadgerDB) GetObjectByID(id []byte, object db.PBBytesTransfer) error {
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return object.TransProtoBytesToObject(val)
		})
	})
	return err
}

func (b *BadgerDB) LoadObjectsByPrefix(prefix []byte, objects db.PBBytesAppender) error {
	err := b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			err := it.Item().Value(func(val []byte) error {
				return objects.AppendPBBytesObject(val)
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
