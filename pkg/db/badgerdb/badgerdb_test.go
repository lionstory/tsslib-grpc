package badgerdb_test

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	crypto2 "github.com/bnb-chain/tss-lib/crypto"
	"github.com/bnb-chain/tss-lib/crypto/vss"
	"github.com/dgraph-io/badger/v3"

	"github.com/lionstory/tsslib-grpc/pkg/crypto"
	"github.com/lionstory/tsslib-grpc/pkg/db/badgerdb"
	"github.com/lionstory/tsslib-grpc/pkg/sss"
)

func TestBadgerDB(t *testing.T) {
	options := badger.DefaultOptions("./")
	badgerDB := badgerdb.BadgerDB{
		Options: &options,
	}
	if err := badgerDB.Open(); err != nil {
		t.Fatalf("failed to open badger: %v\n", err)
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

	if err := badgerDB.SaveObject(&project); err != nil {
		t.Fatalf("failed to save project %v\n", err)
	}

	var project2 sss.Project
	if err := badgerDB.GetObjectByID(project.Key(), &project2); err != nil {
		t.Fatalf("failed to execute GetProjectByID %v\n", err)
	}
	if reflect.DeepEqual(project, project2) != true {
		t.Fatal("project load from badger is not equal to original one")
	}

	projects := make(map[int64]*sss.Project)
	//projects := make(sss.Projects)
	if err := badgerDB.LoadObjectsByPrefix([]byte("proj_"), sss.Projects(projects)); err != nil {
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
			t.Fatal("project load from badger is not equal to original one")
		}
	}

	if err := badgerDB.Close(); err != nil {
		t.Fatalf("failed to close badger: %v\n", err)
	}

	if dirEntry, err := os.ReadDir(badgerDB.Options.Dir); err != nil {
		t.Fatalf("can not clean dir %v, %v\n", badgerDB.Options.Dir, err)
	} else {
		for _, v := range dirEntry {
			if !v.IsDir() && !strings.HasSuffix(v.Name(), ".go") {
				if err := os.Remove(v.Name()); err != nil {
					t.Errorf("failed to delete file %v, %v\n", v.Name(), err)
				}
			}
		}
	}
}
