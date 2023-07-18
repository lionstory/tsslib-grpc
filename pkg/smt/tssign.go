package smt

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/bnb-chain/tss-lib/tss"
	"google.golang.org/grpc"

	"github.com/lionstory/tsslib-grpc/pkg/config"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/modfiysm2"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/signing"
)

type TsSignChan struct {
	SignPrepare    chan *pb.SignPrepareRequest
	LocalpartyChan chan *network.Party
	RouterTable    chan *router.SortedRouterTable
	Message        chan network.Message
	Recv           chan *network.Message
	Out            chan *network.Message
	End            chan network.SignatureData
}

func NewTsSignChan() *TsSignChan {
	return &TsSignChan{
		SignPrepare:    make(chan *pb.SignPrepareRequest),
		LocalpartyChan: make(chan *network.Party),
		Out:            make(chan *network.Message),
		Message:        make(chan network.Message),
		Recv:           make(chan *network.Message),
		RouterTable:    make(chan *router.SortedRouterTable),
		End:            make(chan network.SignatureData),
	}
}

type SignServer struct {
	Signchan    *TsSignChan
	LocalParty  *network.Party
	RouterTable *router.SortedRouterTable
	Running     bool
	conf        *config.TssConfig
	MsgStore    map[string][]*network.Message
}

func NewSignServer(signchan *TsSignChan) *SignServer {
	sign_server := SignServer{
		Signchan: signchan,
		Running:  false,
		MsgStore: make(map[string][]*network.Message),
	}
	return &sign_server
}

func (s *SignServer) SetConfig(conf *config.TssConfig) {
	s.conf = conf
}

func (s *SignServer) GetPartyNum() int {
	return s.conf.PartyNum
}

func (s *SignServer) GetThreshold() int {
	return s.conf.Threshold
}

func (s *SignServer) SetRouterTable(table *router.SortedRouterTable) {
	//if s.Router_table == nil{
	//	s.Router_table = table
	//}
	s.RouterTable = table
}

func (s *SignServer) Prepare(urls []string, msg string) {
	var partyIDs tss.SortedPartyIDs
	routerTable := map[string]string{}
	for index, url_path := range urls {
		fmt.Printf("SmtSignCollectParty: %v, %v\n", index, url_path)
		conn, err := grpc.Dial(url_path, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			fmt.Println("========SignServer.Prepare==========", err)
			break
		}
		client := pb.NewTssServerClient(conn)
		result, err := client.SmtSignCollectParty(context.Background(), &pb.SignCollectRequest{Index: int32(index)})
		if result.Data == nil {
			fmt.Println("========SignServer.Prepare not file==========")
			return
		}
		var partyStatus router.PartyStatus
		err = utils.Decode(result.Data, &partyStatus)
		if err != nil {
			fmt.Println("========SignServer.Prepare==========", err)
			return
		}
		partyStatus.PartyId.Index = index
		routerTable[partyStatus.PartyId.Id] = url_path
		partyIDs = append(partyIDs, partyStatus.PartyId)
		conn.Close()
	}
	if len(partyIDs) < len(urls) {
		return
	}
	// sort_part := tss.SortPartyIDs(partyIDs)

	// table := router.NewSortedRouterTable2(sort_part, routerTable)
	table := router.NewSortedRouterTable2(partyIDs, routerTable)
	s.RouterTable = table
	fmt.Println("======================", table)

	PIDS, err := utils.Encode(partyIDs)
	if err != nil {
		fmt.Println("-------------PIDS------------------------")
		return
	}

	table_data, err := utils.Encode(table)

	for index, url_path := range urls {
		conn, err := grpc.Dial(url_path, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			break
		}
		client := pb.NewTssServerClient(conn)
		_, err = client.SmtSignStart(context.Background(), &pb.SignStartRequest{Index: int32(index), Parties: PIDS, Table: table_data, Msg: msg})
		if err != nil {
			break
		}
	}
}

func (s *SignServer) ReceiveMsg(msg *network.Message) {
	//s.LocalParty.Recv <- msg
	// s.MsgStore[msg.TaskName] = append(s.MsgStore[msg.TaskName], msg)
	if _, ok := s.MsgStore[msg.TaskName]; ok {
		s.MsgStore[msg.TaskName] = append(s.MsgStore[msg.TaskName], msg)
	} else {
		s.MsgStore[msg.TaskName] = []*network.Message{msg}
	}
	if len(s.MsgStore[msg.TaskName]) == len(s.LocalParty.Params().Parties().IDs())-1 {
		for _, msg_recv := range s.MsgStore[msg.TaskName] {
			s.LocalParty.Recv <- msg_recv
		}
	}
}

func (s *SignServer) InitMsg() {
	s.MsgStore = make(map[string][]*network.Message)
}

func (s *SignServer) Start(P *network.Party) {
	dt_start := time.Now()
	fmt.Println("===>Signing Start time: ", dt_start.Format("2006-01-02 15:04:05"))

	s.InitMsg()
	s.Running = true
	go func(p *network.Party) {
		signing.Rounds1(p)
		signing.Rounds2(p)
		signing.Rounds3(p)
		signing.Outputs(p)
	}(P)

sign:
	for {
		select {
		case msg := <-s.Signchan.Out:
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), reflect.TypeOf(msg.MContent), msg.FromID.Index, "---->", msg.ToID.Index)
			for _, router_party := range s.RouterTable.Pids {
				if router_party.Index == msg.ToID.Index {
					err := SendSmtMsg(msg, s.RouterTable.GetURLByID(router_party.Id), "signing")
					if err != nil {
						break sign
					}
				}
			}
		case save := <-s.Signchan.End:
			s.InitMsg()
			fmt.Println("sign end->", save.R, save.S)
			R := new(big.Int).Set(save.R)
			S := new(big.Int).Set(save.S)
			Z := modfiysm2.ComputeZ(P)
			//fmt.Println(R, S, Z)
			flag := modfiysm2.Verify(P.Params().EC(), P.Hash, P.Msg, Z, P.Data.Xx, P.Data.Xy, R, S)
			fmt.Println("签名验证结果", flag)

			dt_end := time.Now()
			fmt.Println("===>Signing End time: ", dt_end.Format("2006-01-02 15:04:05"))
			fmt.Println("===>Signing Cost time: ", dt_end.Sub(dt_start))
			s.Running = false
			break sign
		}
	}

}
