package sss

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"math/big"

	crypto2 "github.com/bnb-chain/tss-lib/crypto"
	"github.com/bnb-chain/tss-lib/crypto/vss"
	"github.com/emmansun/gmsm/sm3"
	"github.com/emmansun/gmsm/sm4"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/golang/protobuf/proto"

	"github.com/lionstory/tsslib-grpc/pkg/crypto"
	"github.com/lionstory/tsslib-grpc/pkg/db"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
)

type Partner struct {
	ID      int64
	Address string
	Status  string
}

type Project struct {
	ProjectID int64
	PartID    int64
	Partners  map[int64]*Partner
	Curve     elliptic.Curve
	PublicKey *ecdsa.PublicKey
	Share     *vss.Share
	Vs        vss.Vs
}

const PREFIX = "proj_"

type Projects map[int64]*Project

var PROJECTS = make(Projects)

func (ps Projects) AppendPBBytesObject(bs []byte) error {
	p := Project{}
	if err := p.TransProtoBytesToObject(bs); err != nil {
		return err
	} else {
		ps[p.ProjectID] = &p
	}
	return nil
}

func SetupCurve() {
	curve, _ := crypto.GetCurveByType(crypto.Secp256k1)
	ecies.AddParamsForCurve(curve, ecies.ECIES_AES128_SHA256)
	curve, _ = crypto.GetCurveByType(crypto.Ed25519)
	ecies.AddParamsForCurve(curve, ecies.ECIES_AES128_SHA256)
	curve, _ = crypto.GetCurveByType(crypto.P256SM2)
	ecies.AddParamsForCurve(curve, &ecies.ECIESParams{
		Hash:      sm3.New,
		Cipher:    sm4.NewCipher,
		BlockSize: sm4.BlockSize,
		KeyLen:    16,
	})
}

func (p *Project) Key() []byte {
	buff := bytes.NewBufferString(PREFIX)
	buff.Write(db.Int64ToBytes(p.ProjectID))
	return buff.Bytes()
}

func (p *Project) Prefix() []byte {
	return []byte(PREFIX)
}

func (p *Project) TransObjectToProtoBytes() ([]byte, error) {
	partners := make(map[int64]*pb.Partner)
	for k, v := range p.Partners {
		partners[k] = &pb.Partner{ID: v.ID, Address: v.Address, Status: v.Status}
	}
	vsPoints, err := crypto2.FlattenECPoints(p.Vs)
	if err != nil {
		return nil, err
	}
	var vsBytes [][]byte
	for _, p := range vsPoints {
		vsBytes = append(vsBytes, p.Bytes())
	}
	var pbCurveType pb.CurveType
	switch crypto.GetCurveType2(p.Curve) {
	case crypto.Secp256k1:
		pbCurveType = pb.CurveType_secp256k1
	case crypto.Ed25519:
		pbCurveType = pb.CurveType_ed25519
	case crypto.P256SM2:
		pbCurveType = pb.CurveType_p256sm2
	case crypto.P256:
		pbCurveType = pb.CurveType_p256
	case crypto.None:
		return nil, fmt.Errorf("curve %v doesn't exist", p.Curve)
	}

	var publicKey *pb.PublicKey
	if p.PublicKey != nil {
		publicKey = &pb.PublicKey{X: p.PublicKey.X.Bytes(), Y: p.PublicKey.Y.Bytes()}
	}
	var share *pb.Share
	if p.Share != nil {
		share = &pb.Share{Threshold: int32(p.Share.Threshold), ID: p.Share.ID.Bytes(), Share: p.Share.Share.Bytes()}
	}

	project := pb.Project{
		ProjectID: p.ProjectID,
		PartID:    p.PartID,
		Partners:  partners,
		Curve:     pbCurveType,
		PublicKey: publicKey,
		Share:     share,
		VS:        vsBytes,
	}
	return proto.Marshal(&project)
}

func (p *Project) TransProtoBytesToObject(bs []byte) error {
	var pbProject pb.Project
	if err := proto.Unmarshal(bs, &pbProject); err != nil {
		return err
	}
	partners := make(map[int64]*Partner)
	for k, v := range pbProject.Partners {
		partners[k] = &Partner{ID: v.ID, Address: v.Address, Status: v.Status}
	}
	curve, _ := crypto.GetCurveByType(crypto.CurveType(pbProject.Curve))
	var points []*big.Int
	for _, v := range pbProject.VS {
		points = append(points, (&big.Int{}).SetBytes(v))
	}
	vs, err := crypto2.UnFlattenECPoints(curve, points)
	if err != nil {
		return err
	}
	p.ProjectID = pbProject.ProjectID
	p.PartID = pbProject.PartID
	p.Partners = partners
	p.Curve = curve
	p.PublicKey = &ecdsa.PublicKey{Curve: curve, X: (&big.Int{}).SetBytes(pbProject.PublicKey.X), Y: (&big.Int{}).SetBytes(pbProject.PublicKey.Y)}
	p.Share = &vss.Share{Threshold: int(pbProject.Share.Threshold), ID: (&big.Int{}).SetBytes(pbProject.Share.ID), Share: (&big.Int{}).SetBytes(pbProject.Share.Share)}
	p.Vs = vs
	return nil
}

func GenerateProjectProtoBytes(p *Project) ([]byte, error) {
	partners := make(map[int64]*pb.Partner)
	for k, v := range p.Partners {
		partners[k] = &pb.Partner{ID: v.ID, Address: v.Address, Status: v.Status}
	}
	vsPoints, err := crypto2.FlattenECPoints(p.Vs)
	if err != nil {
		return nil, err
	}
	var vsBytes [][]byte
	for _, p := range vsPoints {
		vsBytes = append(vsBytes, p.Bytes())
	}
	var pbCurveType pb.CurveType
	switch crypto.GetCurveType2(p.Curve) {
	case crypto.Secp256k1:
		pbCurveType = pb.CurveType_secp256k1
	case crypto.Ed25519:
		pbCurveType = pb.CurveType_ed25519
	case crypto.P256SM2:
		pbCurveType = pb.CurveType_p256sm2
	case crypto.P256:
		pbCurveType = pb.CurveType_p256
	case crypto.None:
		return nil, fmt.Errorf("curve %v doesn't exist", p.Curve)
	}

	project := pb.Project{
		ProjectID: p.ProjectID,
		PartID:    p.PartID,
		Partners:  partners,
		Curve:     pbCurveType,
		PublicKey: &pb.PublicKey{X: p.PublicKey.X.Bytes(), Y: p.PublicKey.Y.Bytes()},
		Share:     &pb.Share{Threshold: int32(p.Share.Threshold), ID: p.Share.ID.Bytes(), Share: p.Share.Share.Bytes()},
		VS:        vsBytes,
	}
	return proto.Marshal(&project)
}

func GenerateProjectFromProtoBytes(bytes []byte, p *Project) error {
	var pbProject pb.Project
	if err := proto.Unmarshal(bytes, &pbProject); err != nil {
		return err
	}
	partners := make(map[int64]*Partner)
	for k, v := range pbProject.Partners {
		partners[k] = &Partner{ID: v.ID, Address: v.Address, Status: v.Status}
	}
	curve, _ := crypto.GetCurveByType(crypto.CurveType(pbProject.Curve))
	var points []*big.Int
	for _, v := range pbProject.VS {
		points = append(points, (&big.Int{}).SetBytes(v))
	}
	vs, err := crypto2.UnFlattenECPoints(curve, points)
	if err != nil {
		return err
	}
	p.ProjectID = pbProject.ProjectID
	p.PartID = pbProject.PartID
	p.Partners = partners
	p.Curve = curve
	p.PublicKey = &ecdsa.PublicKey{Curve: curve, X: (&big.Int{}).SetBytes(pbProject.PublicKey.X), Y: (&big.Int{}).SetBytes(pbProject.PublicKey.Y)}
	p.Share = &vss.Share{Threshold: int(pbProject.Share.Threshold), ID: (&big.Int{}).SetBytes(pbProject.Share.ID), Share: (&big.Int{}).SetBytes(pbProject.Share.Share)}
	p.Vs = vs
	return nil
}

func ProjectKey(id int64) string {
	return fmt.Sprintf("proj_%d", id)
}
