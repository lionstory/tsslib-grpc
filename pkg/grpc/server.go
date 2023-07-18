package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"

	"github.com/lionstory/tsslib-grpc/pkg/ecdsa"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/smt"
	"github.com/lionstory/tsslib-grpc/pkg/sss"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
	smt_utils "github.com/lionstory/tsslib-grpc/smt/utils"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/ecdsa/resharing"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/tjfoc/gmsm/sm2"
	"google.golang.org/grpc"
)

type Service struct {
	Ecdsa *ecdsa.EcdsaServer
	Smt   *smt.SmtServer
}

func NewService(ecdsa *ecdsa.EcdsaServer, smt_server *smt.SmtServer) *Service {
	return &Service{
		Ecdsa: ecdsa,
		Smt:   smt_server,
	}
}

func (service *Service) KeyGenPrepare(ctx context.Context, request *pb.KeyGenPrepareRequest) (*pb.CommonResonse, error) {
	service.Ecdsa.KeygenServer.Keychan.KeygenStart <- request
	response := pb.CommonResonse{Code: "200", Msg: "success"}
	return &response, nil
}

func (service *Service) KeyGenStart(ctx context.Context, request *pb.KeyGenStartRequest) (*pb.CommonResonse, error) {
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	var table router.SortedRouterTable
	err = utils.Decode(request.Parties, &table)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "parties error"}, err
	}
	p2pCtx := tss.NewPeerContext(table.Pids)
	params := tss.NewParameters(tss.S256(), p2pCtx, pid, table.Pids.Len(), int(request.Threshold))

	service.Ecdsa.KeygenServer.GetConfig().Threshold = int(request.Threshold)
	service.Ecdsa.KeygenServer.Keychan.Router_table <- &table

	var P *keygen.LocalParty
	P = keygen.NewLocalParty(params, service.Ecdsa.KeygenServer.Keychan.Out, service.Ecdsa.KeygenServer.Keychan.End).(*keygen.LocalParty)
	service.Ecdsa.KeygenServer.Keychan.LocalpartyChan <- P

	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) KeygenTransMsg(ctx context.Context, request *pb.TransMsgRequest) (*pb.CommonResonse, error) {
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	pMsg, err := tss.ParseWireMessage(request.Message, pid, request.IsBroadcast)
	service.Ecdsa.KeygenServer.Keychan.Message <- pMsg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SignPrepare(ctx context.Context, request *pb.SignPrepareRequest) (*pb.CommonResonse, error) {
	party_ulr := request.Urls
	if len(party_ulr) == 0 {
		return &pb.CommonResonse{Code: "200", Msg: "urls num error"}, errors.New("url null")
	}
	service.Ecdsa.SignServer.Signchan.SignPrepare <- request
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SignCollectParty(ctx context.Context, request *pb.SignCollectRequest) (*pb.SignCollectResponse, error) {
	routeInfo, err := utils.LoadRouteInfo(service.Ecdsa.KeygenServer.GetConfig().SavePath)
	if err != nil {
		fmt.Printf("-------SignCollectParty-------%v\n", err)
		return &pb.SignCollectResponse{Code: "200", Msg: "load key failed"}, err
	}
	data, err := utils.Encode(&router.PartyStatus{routeInfo.PartyId, routeInfo.KeyRevision})
	if err != nil {
		fmt.Printf("-------SignCollectParty-------%v\n", err)
		return &pb.SignCollectResponse{Code: "200", Msg: "encode key failed"}, err
	}
	return &pb.SignCollectResponse{Code: "200", Msg: "success", Data: data}, nil
}

func (service *Service) SignStart(ctx context.Context, request *pb.SignStartRequest) (*pb.CommonResonse, error) {
	var table router.SortedRouterTable
	err := utils.Decode(request.Table, &table)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "decode parties failed"}, err
	}
	p2pCtx := tss.NewPeerContext(table.Pids)
	key, err := utils.LoadKeyInfo(service.Ecdsa.KeygenServer.GetConfig().SavePath)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "load key failed"}, err
	}
	routeInfo, err := utils.LoadRouteInfo(service.Ecdsa.KeygenServer.GetConfig().SavePath)
	routeInfo.PartyId.Index = int(request.Index)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "load routeinfo failed"}, err
	}
	service.Ecdsa.SignServer.Signchan.Router_table <- &table
	params := tss.NewParameters(tss.S256(), p2pCtx, routeInfo.PartyId, len(table.Pids), routeInfo.Threshold)
	msg := big.Int{}
	msg.SetBytes([]byte(request.Msg))
	P := signing.NewLocalParty(&msg, params, key, service.Ecdsa.SignServer.Signchan.Out, service.Ecdsa.SignServer.Signchan.End).(*signing.LocalParty)
	service.Ecdsa.SignServer.Signchan.LocalpartyChan <- P
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SignTransMsg(ctx context.Context, request *pb.TransMsgRequest) (*pb.CommonResonse, error) {
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	pMsg, err := tss.ParseWireMessage(request.Message, pid, request.IsBroadcast)
	service.Ecdsa.SignServer.Signchan.Message <- pMsg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) ReSharingPrepare(ctx context.Context, request *pb.ReSharingPrepareRequest) (*pb.CommonResonse, error) {
	if len(request.Urls) == 0 {
		return &pb.CommonResonse{Code: "200", Msg: "urls num error"}, errors.New("url null")
	}
	service.Ecdsa.ReSharingServer.ReSharingChan.ReShareStart <- request
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) ReSharingInit(ctx context.Context, request *pb.ReSharingStartRequest) (*pb.CommonResonse, error) {
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}

	oldpid := &tss.PartyID{}
	if request.OldParty != nil {
		err = utils.Decode(request.OldParty, oldpid)
		if err != nil {
			return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
		}
	} else {
		oldpid = nil
	}

	//pids := tss.SortedPartyIDs{}
	var newRouteTable router.SortedRouterTable
	err = utils.Decode(request.Parties, &newRouteTable)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "parties error"}, err
	}
	service.Ecdsa.ReSharingServer.ReSharingChan.NewRouterTable <- &newRouteTable

	var oldRouteTable router.SortedRouterTable
	err = utils.Decode(request.OldParties, &oldRouteTable)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "parties error"}, err
	}
	service.Ecdsa.ReSharingServer.ReSharingChan.OldRouterTable <- &oldRouteTable

	newP2PCtx := tss.NewPeerContext(newRouteTable.Pids)
	oldP2PCtx := tss.NewPeerContext(oldRouteTable.Pids)

	/* create old localparty
	   加载本地的key、route
	   * 如果加载不到说明本地是个全新的环境不需要localparty
	   * 加载到检查本地的信息
	   * 检查是否与request中的keyRevision相等
	     * 相等的说明当前revision是最新的revision，可以参与本次resharing
	     * 不等说明当前revision不是最新的revision，不参与本次resharing，同时可以删除之前的旧文件
	   * 检查是否与request中的oldpartyid 相等
	     * 相等说明包中的id 正确，使用包中的id 创建localparty，包中的partyid里的index在prepare时进行了重新排序。
	     * 不等说明包中的id 不正确，不创建localparty
	*/
	if key, err := utils.LoadKeyInfo(service.Ecdsa.ReSharingServer.GetConfig().SavePath); err != nil {
		fmt.Printf("failed to load key info , %v\n", err)
	} else if oldRouteInfo, err := utils.LoadRouteInfo(service.Ecdsa.ReSharingServer.GetConfig().SavePath); err != nil {
		fmt.Printf("failed to load route info , %v\n", err)
	} else if oldRouteInfo.KeyRevision != int(request.OldKeyRevision) {
		fmt.Println("KeyRevision doesn't match")
	} else if oldpid != nil && oldRouteInfo.PartyId.Id != oldpid.Id {
		fmt.Println("old party id doesn't match")
	} else if oldpid != nil && oldRouteInfo.PartyId.Id == oldpid.Id {
		oldRouteInfo.PartyId.Moniker = oldRouteInfo.PartyId.Id
		params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, oldpid, len(oldRouteTable.Pids), int(request.OldThreshold),
			len(newRouteTable.Pids), int(request.Threshold))
		fmt.Println("------NewReSharingParameters----1---------", service.Ecdsa.ReSharingServer.GetConfig().SavePath,
			params.NewPartyCount(),
			oldRouteInfo.KeyRevision,
			request.KeyRevision)
		t := &big.Int{}
		ts := []*big.Int{}
		for _, v := range oldRouteTable.Pids {
			ts = append(ts, t.SetBytes(v.Key))
		}
		fmt.Println("------NewReSharingParameters----1---------", ts)

		oldLocalParty := resharing.NewLocalParty(params, key, service.Ecdsa.ReSharingServer.ReSharingChan.Out,
			service.Ecdsa.ReSharingServer.ReSharingChan.End).(*resharing.LocalParty) // discard old key data
		service.Ecdsa.ReSharingServer.ReSharingChan.OldLocalPartyChan <- oldLocalParty
	}
	//else {
	//
	//	utils.CleanRouteAndKey(service.Ecdsa.ReSharingServer.GetConfig().SavePath)
	//}

	// create new localparty
	params := tss.NewReSharingParameters(tss.S256(), oldP2PCtx, newP2PCtx, pid, len(oldRouteTable.Pids), int(request.OldThreshold), len(newRouteTable.Pids), int(request.Threshold))
	fmt.Println("------NewReSharingParameters-----2--------", params.NewPartyCount())
	save := keygen.NewLocalPartySaveData(len(newRouteTable.Pids))
	if key, err := utils.LoadKeyInfo(service.Ecdsa.ReSharingServer.GetConfig().SavePath); err != nil {
		fmt.Printf("failed to load key info , %v\n", err)
	} else {
		save.LocalPreParams = key.LocalPreParams
	}
	newLocalParty := resharing.NewLocalParty(params, save, service.Ecdsa.ReSharingServer.ReSharingChan.Out, service.Ecdsa.ReSharingServer.ReSharingChan.End).(*resharing.LocalParty)
	service.Ecdsa.ReSharingServer.GetConfig().Threshold = int(request.Threshold)
	service.Ecdsa.ReSharingServer.GetConfig().KeyRevision = int(request.KeyRevision)
	service.Ecdsa.ReSharingServer.ReSharingChan.NewLocalPartyChan <- newLocalParty
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) ReSharingStart(ctx context.Context, empty *empty.Empty) (*pb.CommonResonse, error) {
	go service.Ecdsa.ReSharingServer.Start()
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) ReSharingTransMsgToOld(ctx context.Context, request *pb.TransMsgRequest) (*pb.CommonResonse, error) {
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		fmt.Println("-------------ReSharingTransMsgToOld--------------", err)
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	pMsg, err := tss.ParseWireMessage(request.Message, pid, request.IsBroadcast)
	if err != nil {
		fmt.Println("-------------ReSharingTransMsgToOld--------------", err)
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	service.Ecdsa.ReSharingServer.ReSharingChan.MessageToOld <- pMsg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) ReSharingTransMsgToNew(ctx context.Context, request *pb.TransMsgRequest) (*pb.CommonResonse, error) {
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		fmt.Println("-------------ReSharingTransMsgToNew--------------", err)
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	pMsg, err := tss.ParseWireMessage(request.Message, pid, request.IsBroadcast)
	if err != nil {
		fmt.Println("-------------ReSharingTransMsgToNew--------------", err)
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	service.Ecdsa.ReSharingServer.ReSharingChan.MessageToNew <- pMsg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtKeyGenPrepare(ctx context.Context, request *pb.KeyGenPrepareRequest) (*pb.CommonResonse, error) {
	service.Smt.KeygenServer.TsChan.TsKeygenStart <- request
	response := pb.CommonResonse{Code: "200", Msg: "success"}
	return &response, nil
}

func (service *Service) SmtKeyGenStart(ctx context.Context, request *pb.KeyGenStartRequest) (*pb.CommonResonse, error) {
	fmt.Println("==================sm2 keygen start==========================", len(request.Parties))
	pid := &tss.PartyID{}
	err := utils.Decode(request.Party, pid)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	var table router.SortedRouterTable
	err = utils.Decode(request.Parties, &table)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "parties error"}, err
	}

	p2pCtx := tss.NewPeerContext(table.Pids)
	params := network.NewParameters(sm2.P256Sm2(), p2pCtx, pid, len(table.Pids), int(request.Threshold))
	party := network.Newparty(params, service.Smt.KeygenServer.TsChan.Out, service.Smt.KeygenServer.TsChan.Recv, service.Smt.KeygenServer.TsChan.End)
	service.Smt.KeygenServer.TsChan.RouterTable <- &table
	service.Smt.KeygenServer.TsChan.LocalpartyChan <- party
	fmt.Println("==================sm2 keygen end==========================")
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtKeygenTransMsg(ctx context.Context, request *pb.TransSmtMsgRequest) (*pb.CommonResonse, error) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "==================sm2 keygen msg trans==========================", request.TaskName)
	//msg := network.Message{}
	//err := msg.SetBytes(request.FromId, request.ToId, request.Content)
	msg, err := smt.ToMessage(request.FromId, request.ToId, request.Content, request.TaskName)
	//err := utils.Decode(request.Message, &msg)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	// 存在party还没start，就会就收到其他方的round消息，导致报错
	for {
		if service.Smt.KeygenServer.Running{
			break
		}else{
			time.Sleep(20 * time.Millisecond)
		}
	}
	service.Smt.KeygenServer.TsChan.Message <- *msg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtSignPrepare(ctx context.Context, request *pb.SignPrepareRequest) (*pb.CommonResonse, error) {
	service.Smt.SignServer.Signchan.SignPrepare <- request
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtSignCollectParty(ctx context.Context, request *pb.SignCollectRequest) (*pb.SignCollectResponse, error) {
	routeInfo, err := utils.LoadRouteInfo(service.Smt.KeygenServer.GetConfig().SavePath)
	if err != nil {
		fmt.Printf("-------SignCollectParty-------%v\n", err)
		return &pb.SignCollectResponse{Code: "200", Msg: "load key failed", Data: nil}, nil
	}
	data, err := utils.Encode(&router.PartyStatus{routeInfo.PartyId, routeInfo.KeyRevision})
	if err != nil {
		fmt.Printf("-------SignCollectParty-------%v\n", err)
		return &pb.SignCollectResponse{Code: "200", Msg: "encode key failed"}, err
	}
	return &pb.SignCollectResponse{Code: "200", Msg: "success", Data: data}, nil
}

func (service *Service) SmtSignStart(ctx context.Context, request *pb.SignStartRequest) (*pb.CommonResonse, error) {
	var table router.SortedRouterTable
	err := utils.Decode(request.Table, &table)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "decode parties failed"}, err
	}
	p2pCtx := tss.NewPeerContext(table.Pids)
	key, err := smt_utils.LoadKeyInfo(service.Smt.KeygenServer.GetConfig().SavePath)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "load key failed"}, err
	}
	routeInfo, err := smt_utils.LoadRouteInfo(service.Smt.KeygenServer.GetConfig().SavePath)
	routeInfo.PartyId.Index = int(request.Index)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "load routeinfo failed"}, err
	}
	service.Smt.SignServer.Signchan.RouterTable <- &table
	params := network.NewParameters(sm2.P256Sm2(), p2pCtx, routeInfo.PartyId, len(table.Pids), routeInfo.Threshold)
	P := network.NewsignParty(params, service.Smt.SignServer.Signchan.Out, service.Smt.SignServer.Signchan.Recv, service.Smt.SignServer.Signchan.End)
	P.Data = network.BuildLocalSaveDataSubset(*key, p2pCtx.IDs())
	// P.Data = *key
	P.Msg = []byte(request.Msg)
	service.Smt.SignServer.Signchan.LocalpartyChan <- P
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtSignTransMsg(ctx context.Context, request *pb.TransSmtMsgRequest) (*pb.CommonResonse, error) {
	msg, err := smt.ToMessage(request.FromId, request.ToId, request.Content, request.TaskName)
	//err := utils.Decode(request.Message, &msg)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	// 存在party还没start，就会就收到其他方的round消息，导致报错
	for {
		if service.Smt.SignServer.Running{
			break
		}else{
			time.Sleep(20 * time.Millisecond)
		}
	}
	service.Smt.SignServer.Signchan.Message <- *msg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtResharePrepare(ctx context.Context, request *pb.SmtResharePrepareRequest) (*pb.CommonResonse, error) {
	party_ulr := request.Urls
	if len(party_ulr) == 0 {
		return &pb.CommonResonse{Code: "200", Msg: "urls num error"}, errors.New("url null")
	}
	service.Smt.ReshareServer.ReshareChan.TsReshareStart <- *request
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtReshareStart(ctx context.Context, request *pb.SmtReshareStartRequest) (*pb.CommonResonse, error) {
	var table router.SortedRouterTable
	err := utils.Decode(request.Table, &table)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "decode table failed"}, err
	}
	newPids := tss.SortedPartyIDs{}
	err = utils.Decode(request.NewParties, &newPids)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "decode NewParties failed"}, err
	}
	oldPids := tss.SortedPartyIDs{}
	err = utils.Decode(request.OldParties, &oldPids)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "decode OldParties failed"}, err
	}
	pid := &tss.PartyID{}
	err = utils.Decode(request.Party, pid)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "decode Party failed"}, err
	}
	key, err := smt_utils.LoadKeyInfo(service.Smt.ReshareServer.GetConfig().SavePath)
	if err != nil {
		// return &pb.CommonResonse{Code: "200", Msg: "load key failed"}, err
		newData := network.NewSaveData(len(newPids))
		key = &newData
	}
	service.Smt.ReshareServer.ReshareChan.Router_table <- &table
	p2pNewCtx := tss.NewPeerContext(newPids)
	p2pOldCtx := tss.NewPeerContext(oldPids)
	params := network.NewReSharingParameters(sm2.P256Sm2(), p2pOldCtx, p2pNewCtx, pid,
		len(oldPids), int(request.OldThreshold), len(newPids), int(request.NewThreshold))
	P := network.NewReSharingParty(params, service.Smt.ReshareServer.ReshareChan.Out, service.Smt.ReshareServer.ReshareChan.Recv, service.Smt.ReshareServer.ReshareChan.End, *key)
	service.Smt.ReshareServer.SetKeyRevision(int(request.KeyRevision))
	service.Smt.ReshareServer.SetPartyNum(len(newPids))
	service.Smt.ReshareServer.SetThreshold(int(request.NewThreshold))
	service.Smt.ReshareServer.ReshareChan.LocalpartyChan <- P
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

func (service *Service) SmtReshareTransMsg(ctx context.Context, request *pb.TransSmtMsgRequest) (*pb.CommonResonse, error) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "==================sm2 reshare msg trans==========================", request.TaskName)
	msg, err := smt.ToResharingMessage(request.FromId, request.ToId, request.Content, request.TaskName)
	if err != nil {
		return &pb.CommonResonse{Code: "200", Msg: "party error"}, err
	}
	// 存在party还没start，就会就收到其他方的round消息，导致报错
	for {
		if service.Smt.ReshareServer.Running{
			break
		}else{
			time.Sleep(20 * time.Millisecond)
		}
	}
	service.Smt.ReshareServer.ReshareChan.Message <- *msg
	return &pb.CommonResonse{Code: "200", Msg: "success"}, nil
}

type GrpcServer struct {
	Grpcserver *grpc.Server
}

func NewGrpcServer(ecdsa *ecdsa.EcdsaServer, smt_server *smt.SmtServer, savePath string) *GrpcServer {
	grpc_server := grpc.NewServer()

	vssServer, err := sss.NewVssServer(savePath)
	if err != nil {
		log.Fatal(err)
	}

	pb.RegisterTssServerServer(grpc_server, NewService(ecdsa, smt_server))
	pb.RegisterVssServiceServer(grpc_server, vssServer)
	return &GrpcServer{
		Grpcserver: grpc_server,
	}
}

func (server *GrpcServer) Start(port int64) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {

	}
	fmt.Println("Grpc Server Start Success", fmt.Sprintf(":%d", port))
	if err = server.Grpcserver.Serve(lis); err != nil {

	}
}
