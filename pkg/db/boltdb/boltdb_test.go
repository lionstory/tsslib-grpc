package boltdb_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"testing"

	crypto2 "github.com/bnb-chain/tss-lib/crypto"
	"github.com/bnb-chain/tss-lib/crypto/vss"
	bolt "go.etcd.io/bbolt"

	"github.com/lionstory/tsslib-grpc/pkg/crypto"
	"github.com/lionstory/tsslib-grpc/pkg/db/boltdb"
	"github.com/lionstory/tsslib-grpc/pkg/sss"
)

func TestBoltDB(t *testing.T) {
	boldDB := boltdb.BoltDB{
		DBPath:     "./bolt.db",
		Options:    bolt.Options{},
		BucketName: []byte("tss"),
	}
	if err := boldDB.Open(); err != nil {
		t.Fatalf("failed to open boltdb: %v\n", err)
	}

	curve := crypto.GetCurveByType2(crypto.P256SM2)
	if curve == nil {
		t.Fatal("failed to get curve for curveType: crypto.P256SM2")
	}

	privateKey, _ := ecdsa.GenerateKey(curve, rand.Reader)

	var vs []*crypto2.ECPoint
	for i := 0; i < 3; i++ {
		p, err := crypto2.NewECPoint(curve, privateKey.X, privateKey.Y)
		if err != nil {
			t.Fatalf("failed to create ecPoint, %v\n", err)
		}
		vs = append(vs, p)
	}

	project := sss.Project{
		ProjectID: 1,
		PartID:    1,
		Partners: map[int64]*sss.Partner{
			1: &sss.Partner{ID: 1, Address: ":1111", Status: "1 status"},
			2: &sss.Partner{ID: 2, Address: ":2222", Status: "2 status"},
			3: &sss.Partner{ID: 3, Address: ":3333", Status: "3 status"},
		},
		Curve:     curve,
		PublicKey: &privateKey.PublicKey,
		Share:     &vss.Share{Threshold: 1, ID: big.NewInt(123), Share: big.NewInt(456)},
		Vs:        vs,
	}

	if err := boldDB.SaveObject(&project); err != nil {
		t.Fatalf("failed to save project %v\n", err)
	}

	var project2 sss.Project
	if err := boldDB.GetObjectByID(project.Key(), &project2); err != nil {
		t.Fatalf("failed to execute GetProjectByID %v\n", err)
	}
	if reflect.DeepEqual(&project, &project2) != true {
		t.Fatal("project load from boltdb is not equal to original one")
	}

	//projects := make(map[int64]*sss.Project)
	projects := make(sss.Projects)
	if err := boldDB.LoadObjectsByPrefix([]byte("proj_"), projects); err != nil {
		t.Fatalf("failed to execute GetProjectByID %v\n", err)
	}
	if len(projects) == 0 {
		t.Fatal("failed get load projects, projects length is zero")
	}

	for k, v := range projects {
		fmt.Println("=====", k)
		if k != project.ProjectID {
			t.Fatalf("map key is not correct, need %v, but found %v\n", sss.ProjectKey(project.ProjectID), k)
		}

		if reflect.DeepEqual(&project, v) != true {
			t.Fatal("project load from boltdb is not equal to original one")
		}
	}

	if err := boldDB.Close(); err != nil {
		t.Fatalf("failed to close boltdb: %v\n", err)
	}

	if err := os.Remove(boldDB.DBPath); err != nil {
		t.Fatalf("can not delete boltdb file %v, %v\n", boldDB.DBPath, err)
	}
}
