package sss

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"path"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/crypto"
	cmts "github.com/bnb-chain/tss-lib/crypto/commitments"
	"github.com/bnb-chain/tss-lib/crypto/vss"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	bolt "go.etcd.io/bbolt"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	crypto1 "github.com/lionstory/tsslib-grpc/pkg/crypto"
	"github.com/lionstory/tsslib-grpc/pkg/db"
	"github.com/lionstory/tsslib-grpc/pkg/db/boltdb"
	"github.com/lionstory/tsslib-grpc/pkg/logger"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
)

type VssServer struct {
	db db.DB
}

func NewVssServer(dbPath string) (pb.VssServiceServer, error) {
	logger.SetupLogger(zap.DebugLevel)
	SetupCurve()
	db := &boltdb.BoltDB{
		DBPath: path.Join(dbPath, "sss.db"),
		Options: bolt.Options{
			NoSync: false,
		},
		BucketName: []byte("sss"),
	}
	if err := db.Open(); err != nil {
		return nil, err
	}
	if err := db.LoadObjectsByPrefix([]byte("proj_"), PROJECTS); err != nil {
		return nil, err
	}
	return &VssServer{
		db: db,
	}, nil
}

func (v *VssServer) ProjectSetup(ctx context.Context, in *pb.ProjectSetupRequest) (*pb.CommonResonse, error) {
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	// 广播 project 到所有partner
	var result sync.Map
	var wg sync.WaitGroup
	for _, p := range in.Partners {
		wg.Add(1)
		go func(wg *sync.WaitGroup, partnerID int64, address string) {
			defer wg.Done()
			if resp, err := GrpcProjectSync(address, &pb.ProjectSetupRequest{ProjectID: in.ProjectID, PartID: partnerID, Partners: in.Partners}); err != nil {
				logger.Error("Project ID invalid")
				result.Store(partnerID, err.Error())
			} else {
				result.Store(partnerID, resp.Msg)
			}
		}(&wg, p.ID, p.Address)
	}
	wg.Wait()

	// 检查发送情况
	isSuccess := true
	result.Range(func(key, value interface{}) bool {
		if value != "success" {
			logger.Error("partner failed to receive cmt", zap.Any("partner id", key))
			isSuccess = false
			return false
		}
		return true
	})
	if !isSuccess {
		return &pb.CommonResonse{"500", "failed to send secret to all partner"}, errors.New("failed to send secret to all partner")
	}
	return &pb.CommonResonse{"200", "success"}, nil
}

func (v *VssServer) ProjectSync(ctx context.Context, in *pb.ProjectSetupRequest) (*pb.CommonResonse, error) {
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}
	partners := map[int64]*Partner{}
	for _, p := range in.Partners {
		partners[p.ID] = &Partner{
			ID:      p.ID,
			Address: p.Address,
			Status:  "ok",
		}
	}
	PROJECTS[in.ProjectID] = &Project{
		ProjectID: in.ProjectID,
		PartID:    in.PartID,
		Partners:  partners,
	}
	v.db.SaveObject(PROJECTS[in.ProjectID])
	return &pb.CommonResonse{"200", "success"}, nil
}

func (v *VssServer) KeyGenerate(ctx context.Context, in *pb.KeyGenerateRequest) (*pb.CommonResonse, error) {
	dt_start := time.Now()
	defer func() {
		dt_end := time.Now()
		fmt.Println("===>KeyGenerate End time: ", dt_end.Format("2006-01-02 15:04:05"))
		fmt.Println("===>KeyGenerate Cost time: ", dt_end.Sub(dt_start))
	}()

	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	if _, ok := PROJECTS[in.ProjectID]; !ok {
		logger.Error("project doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("project doesn't exists")
	}

	curve, ok := crypto1.GetCurveByType(crypto1.CurveType(in.Curve))
	if !ok {
		logger.Error("curve doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("curve doesn't exists")
	}
	PROJECTS[in.ProjectID].Curve = curve

	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		logger.Error("failed to generate key", zap.Error(err))
		return &pb.CommonResonse{"400", "failed to generate key"}, err
	}

	var ids []*big.Int
	for _, v := range PROJECTS[in.ProjectID].Partners {
		ids = append(ids, big.NewInt(v.ID))
	}
	vs, shares, err := vss.Create(curve, int(in.Threshold), privateKey.D, ids)
	if err != nil {
		logger.Error("failed to create shares", zap.Error(err))
		return &pb.CommonResonse{"500", "failed to create shares"}, err
	}

	pGFlat, err := crypto.FlattenECPoints(vs)
	if err != nil {
		logger.Error("failed to flatten ecPoints", zap.Error(err))
		return &pb.CommonResonse{"500", "failed to flatten ecPoints"}, err
	}
	cmt := cmts.NewHashCommitment(pGFlat...)

	// 广播 cmt 到所有partner
	cmtBytes, err := json.Marshal(cmt)
	if err != nil {
		logger.Error("failed to flatten ecPoints", zap.Error(err))
		return &pb.CommonResonse{"500", "failed to flatten ecPoints"}, err
	}

	var result sync.Map
	var wg sync.WaitGroup
	for k, v := range PROJECTS[in.ProjectID].Partners {
		wg.Add(1)
		go func(wg *sync.WaitGroup, partnerID int64, addr string) {
			defer wg.Done()
			if resp, err := GrpcCmtProcess(addr, &pb.CmtMsg{ProjectID: in.ProjectID, Curve: in.Curve, Cmt: cmtBytes}); err != nil {
				logger.Error("failed to send secret to partner", zap.Int64("partnerID", partnerID), zap.Error(err))
				result.Store(partnerID, err.Error())
			} else {
				result.Store(partnerID, resp.Msg)
			}
		}(&wg, k, v.Address)
	}
	wg.Wait()

	// 检查发送情况
	isSuccess := true
	result.Range(func(key, value interface{}) bool {
		if value != "success" {
			logger.Error("failed to send secret to partner", zap.Any("partner id", key))
			isSuccess = false
			return false
		}
		return true
	})
	if !isSuccess {
		return &pb.CommonResonse{"500", "failed to send secret"}, errors.New("failed to send secret to partner id")
	}

	// 发送 shares 到 partner
	result = sync.Map{}
	for _, v := range shares {
		wg.Add(1)
		id := v.ID.Int64()
		address := PROJECTS[in.ProjectID].Partners[id].Address
		go func(wg *sync.WaitGroup, partnerID int64, address string, share *vss.Share) {
			defer wg.Done()
			bs, _ := json.Marshal(share)
			if resp, err := GrpcShareProcess(address, &pb.ShareMsg{
				ProjectID: in.ProjectID, Share: bs, X: privateKey.X.Bytes(), Y: privateKey.Y.Bytes()}); err != nil {
				logger.Error("failed to send share to partner", zap.Int64("partnerID", partnerID), zap.Error(err))
				result.Store(partnerID, err.Error())
			} else {
				result.Store(partnerID, resp.Msg)
			}
		}(&wg, id, address, v)
	}
	wg.Wait()

	// 检查发送情况
	isSuccess = true
	result.Range(func(key, value interface{}) bool {
		if value != "success" {
			logger.Error("failed to send secret share to partner", zap.Any("partner id", key))
			isSuccess = false
			return false
		}
		return true
	})
	if !isSuccess {
		return &pb.CommonResonse{"500", "failed to send secret share to partner"}, fmt.Errorf("failed to send secret share to partner id")
	}

	PROJECTS[in.ProjectID].PublicKey = &ecdsa.PublicKey{Curve: privateKey.Curve, X: privateKey.X, Y: privateKey.Y}
	v.db.SaveObject(PROJECTS[in.ProjectID])
	return &pb.CommonResonse{"200", "success"}, nil
}

func (v *VssServer) CmtProcess(ctx context.Context, in *pb.CmtMsg) (*pb.CommonResonse, error) {
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	if _, ok := PROJECTS[in.ProjectID]; !ok {
		logger.Error("project doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("project doesn't exists")
	}

	curve, ok := crypto1.GetCurveByType(crypto1.CurveType(in.Curve))
	if !ok {
		logger.Error("curve doesn't exists")
		return &pb.CommonResonse{"400", "curve doesn't exists"}, errors.New("curve doesn't exists")
	}

	var cmt cmts.HashCommitDecommit
	if err := json.Unmarshal(in.Cmt, &cmt); err != nil {
		logger.Error("failed to decode cmt")
		return &pb.CommonResonse{"500", "failed to decode cmt"}, err
	}

	ok, flatPolyGs := cmt.DeCommit()
	if !ok {
		logger.Error("failed to verify cmt")
		return &pb.CommonResonse{"500", "failed to verify cmt"}, errors.New("failed to verify cmt")
	}

	PjVs, err := crypto.UnFlattenECPoints(curve, flatPolyGs)
	if err != nil {
		logger.Error("failed to verify cmt", zap.Error(err))
		return &pb.CommonResonse{"500", "cmt can not flatten"}, errors.New("cmt can not flatten")
	}

	PROJECTS[in.ProjectID].Curve = curve
	PROJECTS[in.ProjectID].Vs = PjVs
	v.db.SaveObject(PROJECTS[in.ProjectID])
	return &pb.CommonResonse{"200", "success"}, nil
}

func (v *VssServer) ShareProcess(ctx context.Context, in *pb.ShareMsg) (*pb.CommonResonse, error) {
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	if _, ok := PROJECTS[in.ProjectID]; !ok {
		logger.Error("project doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("project doesn't exists")
	}

	var share vss.Share
	if err := json.Unmarshal(in.Share, &share); err != nil {
		logger.Error("failed to decode share")
		return &pb.CommonResonse{"500", "failed to decode share"}, err
	}

	if share.ID.Int64() != PROJECTS[in.ProjectID].PartID {
		logger.Error("share id not correct")
		return &pb.CommonResonse{"500", "share id not correct"}, errors.New("share id not correct")
	}

	if ok := share.Verify(PROJECTS[in.ProjectID].Curve, share.Threshold, PROJECTS[in.ProjectID].Vs); !ok {
		logger.Error("failed to verify share")
		return &pb.CommonResonse{"500", "failed to decode share"}, errors.New("failed to verify share")
	}

	PROJECTS[in.ProjectID].Share = &share
	PROJECTS[in.ProjectID].PublicKey = &ecdsa.PublicKey{
		Curve: PROJECTS[in.ProjectID].Curve, X: (&big.Int{}).SetBytes(in.X), Y: (&big.Int{}).SetBytes(in.Y)}
	v.db.SaveObject(PROJECTS[in.ProjectID])
	return &pb.CommonResonse{"200", "success"}, nil
}

func (v *VssServer) Reconstruct(ctx context.Context, in *pb.CommonRequest) (*pb.CommonResonse, error) {
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	if _, ok := PROJECTS[in.ProjectID]; !ok {
		logger.Error("project doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("project doesn't exists")
	}

	if js, err := json.Marshal(PROJECTS[in.ProjectID].Share); err != nil {
		logger.Error("failed to encode share", zap.Error(err))
		return &pb.CommonResonse{"400", "failed to encode share"}, err
	} else {
		return &pb.CommonResonse{Code: "200", Msg: string(js)}, nil
	}
}

func (v *VssServer) Encrypt(ctx context.Context, in *pb.CommonRequest) (*pb.CommonResonse, error) {
	dt_start := time.Now()
	defer func() {
		dt_end := time.Now()
		fmt.Println("===>Encrypt End time: ", dt_end.Format("2006-01-02 15:04:05"))
		fmt.Println("===>Encrypt Cost time: ", dt_end.Sub(dt_start))
	}()
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	if _, ok := PROJECTS[in.ProjectID]; !ok {
		logger.Error("project doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("project doesn't exists")
	}

	eciesPublicKey := ecies.ImportECDSAPublic(PROJECTS[in.ProjectID].PublicKey)
	if cipherBytes, err := ecies.Encrypt(rand.Reader, eciesPublicKey, []byte(in.Raw), nil, nil); err != nil {
		logger.Error("failed to encrypt", zap.Error(err))
		return &pb.CommonResonse{"500", "failed to encrypt"}, err
	} else {
		enc := base64.RawURLEncoding.EncodeToString(cipherBytes)
		return &pb.CommonResonse{"200", enc}, nil
	}
}

func (v *VssServer) Decrypt(ctx context.Context, in *pb.CommonRequest) (*pb.CommonResonse, error) {
	dt_start := time.Now()
	defer func() {
		dt_end := time.Now()
		fmt.Println("===>Decrypt End time: ", dt_end.Format("2006-01-02 15:04:05"))
		fmt.Println("===>Decrypt Cost time: ", dt_end.Sub(dt_start))
	}()
	if in.ProjectID <= 0 {
		logger.Error("Project ID invalid")
		return &pb.CommonResonse{"400", "Project ID invalid"}, errors.New("project id invalid")
	}

	if _, ok := PROJECTS[in.ProjectID]; !ok {
		logger.Error("project doesn't exists")
		return &pb.CommonResonse{"400", "project doesn't exists"}, errors.New("project doesn't exists")
	}

	rawMsg, err := base64.RawURLEncoding.DecodeString(in.Raw)
	if err != nil {
		logger.Error("failed to decode message", zap.Error(err))
		return &pb.CommonResonse{"500", "failed to decode message"}, errors.New("project doesn't exists")
	}

	// 进行 reconstruct
	var result sync.Map
	var wg sync.WaitGroup
	for k, v := range PROJECTS[in.ProjectID].Partners {
		wg.Add(1)
		go func(wg *sync.WaitGroup, partnerID int64, addr string) {
			defer wg.Done()
			if response, err := GrpcReconstruct(addr, &pb.CommonRequest{ProjectID: in.ProjectID}); err != nil {
				logger.Error("failed to get share from partner", zap.Int64("partnerID", partnerID), zap.Error(err))
				result.Store(partnerID, err.Error())
			} else {
				result.Store(partnerID, response.Msg)
			}
		}(&wg, k, v.Address)
	}
	wg.Wait()

	// 检查发送情况
	var shares vss.Shares
	result.Range(func(key, value interface{}) bool {
		var share vss.Share
		if err := json.Unmarshal([]byte(value.(string)), &share); err != nil {
			logger.Error("failed to parse share", zap.Any("partner id", key), zap.Any("raw", value))
		} else if share.ID.Int64() != PROJECTS[in.ProjectID].Partners[key.(int64)].ID {
			logger.Error("failed to verify shared recieved from partner, partner id not correct", zap.Any("partner id:", key), zap.Any("share id", share.ID.Int64()))
		} else if share.Verify(PROJECTS[in.ProjectID].Curve, share.Threshold, PROJECTS[in.ProjectID].Vs) {
			shares = append(shares, &share)
		} else {
			logger.Error("failed to verify shared recieved from partner", zap.Any("partner id", key))
		}
		return true
	})

	if len(shares) <= shares[0].Threshold {
		logger.Error("not enough shares to reconstruct")
		return &pb.CommonResonse{"500", "not enough shares to reconstruct"}, errors.New("not enough shares to reconstruct")
	}

	if secret, err := shares.ReConstruct(PROJECTS[in.ProjectID].Curve); err != nil {
		logger.Error("failed to reconstruct secret", zap.Error(err))
		return &pb.CommonResonse{"500", "failed to reconstruct secret"}, err
	} else {
		priv := ecdsa.PrivateKey{PublicKey: *PROJECTS[in.ProjectID].PublicKey, D: secret}
		eciesPrivateKey := ecies.ImportECDSA(&priv)
		if dec, err := eciesPrivateKey.Decrypt(rawMsg, nil, nil); err != nil {
			logger.Error("failed to decrypt message", zap.Error(err))
			return &pb.CommonResonse{"500", "failed to reconstruct secret"}, err
		} else {
			return &pb.CommonResonse{"200", string(dec)}, nil
		}
	}
}

func GrpcProjectSync(addr string, in *pb.ProjectSetupRequest) (*pb.CommonResonse, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return &pb.CommonResonse{"500", fmt.Sprintf("failed to connect addr: %s", addr)}, err
	}
	defer conn.Close()
	client := pb.NewVssServiceClient(conn)
	return client.ProjectSync(context.TODO(), in)

}

func GrpcCmtProcess(addr string, in *pb.CmtMsg) (*pb.CommonResonse, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return &pb.CommonResonse{"500", fmt.Sprintf("failed to connect addr: %s", addr)}, err
	}
	defer conn.Close()
	client := pb.NewVssServiceClient(conn)
	return client.CmtProcess(context.TODO(), in)
}

func GrpcShareProcess(addr string, in *pb.ShareMsg) (*pb.CommonResonse, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return &pb.CommonResonse{"500", fmt.Sprintf("failed to connect addr: %s", addr)}, err
	}
	defer conn.Close()
	client := pb.NewVssServiceClient(conn)
	return client.ShareProcess(context.TODO(), in)

}

func GrpcReconstruct(addr string, in *pb.CommonRequest) (*pb.CommonResonse, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second))
	if err != nil {
		return &pb.CommonResonse{"500", fmt.Sprintf("failed to connect addr: %s", addr)}, err
	}
	defer conn.Close()
	client := pb.NewVssServiceClient(conn)
	return client.Reconstruct(context.TODO(), in)

}
