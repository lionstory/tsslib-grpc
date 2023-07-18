package boltdb

import (
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/db"

	bolt "go.etcd.io/bbolt"
)

type BoltDB struct {
	DBPath     string
	BucketName []byte
	Options    bolt.Options
	db         *bolt.DB
}

func (b *BoltDB) Open() error {
	if db, err := bolt.Open(b.DBPath, 0600, &b.Options); err != nil {
		return err
	} else {
		b.db = db
		return db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(b.BucketName)
			return err
		})
	}
}

func (b *BoltDB) Close() error {
	return b.db.Close()
}

//func (b *BoltDB) LoadProjects() (map[int64]*sss.Project, error) {
//	projects := make(map[int64]*sss.Project)
//	err := b.db.View(func(tx *bolt.Tx) error {
//		bucket := tx.Bucket(b.BucketName)
//		if bucket == nil {
//			return fmt.Errorf("bucket %v doesn't exist", b.BucketName)
//		}
//
//		c := bucket.Cursor()
//		prefix := []byte("proj_")
//		for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
//			var project sss.Project
//			if err := sss.GenerateProjectFromProtoBytes(v, &project); err != nil {
//				return err
//			} else {
//				projects[project.ProjectID] = &project
//			}
//		}
//		return nil
//	})
//	return projects, err
//}
//
//func (b *BoltDB) GetProjectByID(id int64) (*sss.Project, error) {
//	var project sss.Project
//	err := b.db.View(func(tx *bolt.Tx) error {
//		bucket := tx.Bucket(b.BucketName)
//		if bucket == nil {
//			return fmt.Errorf("bucket %v doesn't exist", b.BucketName)
//		}
//		val := bucket.Get([]byte(sss.ProjectKey(id)))
//		if val == nil {
//			return fmt.Errorf("project %v doesn't exist", id)
//		}
//		return sss.GenerateProjectFromProtoBytes(val, &project)
//	})
//	return &project, err
//}
//
//func (b *BoltDB) SaveProject(project *sss.Project) error {
//	err := b.db.Update(func(tx *bolt.Tx) error {
//		bs, err := sss.GenerateProjectProtoBytes(project)
//		if err != nil {
//			return err
//		}
//		key := sss.ProjectKey(project.ProjectID)
//		bucket := tx.Bucket(b.BucketName)
//		if bucket == nil {
//			return fmt.Errorf("bucket %v doesn't exist", b.BucketName)
//		}
//		return bucket.Put([]byte(key), bs)
//	})
//	return err
//}

func (b *BoltDB) SaveObject(object db.PBBytesTransfer) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.BucketName)
		if bucket == nil {
			return fmt.Errorf("bucket %v doesn't exist", b.BucketName)
		}
		bs, err := object.TransObjectToProtoBytes()
		if err != nil {
			return err
		}
		return bucket.Put(object.Key(), bs)
	})
	return err
}

func (b *BoltDB) GetObjectByID(id []byte, object db.PBBytesTransfer) error {
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.BucketName)
		if bucket == nil {
			return fmt.Errorf("bucket %v doesn't exist", b.BucketName)
		}
		val := bucket.Get(id)
		if val == nil {
			return fmt.Errorf("project %v doesn't exist", id)
		}
		return object.TransProtoBytesToObject(val)
	})
	return err
}

func (b *BoltDB) LoadObjectsByPrefix(prefix []byte, objects db.PBBytesAppender) error {
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(b.BucketName)
		if bucket == nil {
			return fmt.Errorf("bucket %v doesn't exist", b.BucketName)
		}
		c := bucket.Cursor()
		for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
			if err := objects.AppendPBBytesObject(v); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
