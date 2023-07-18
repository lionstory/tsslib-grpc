package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/bnb-chain/tss-lib/tss"
	"google.golang.org/grpc"

	"github.com/lionstory/tsslib-grpc/pkg/config"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/tskeygen"
	smt_utils "github.com/lionstory/tsslib-grpc/smt/utils"
)

type TsKeyGenChan struct {
	TsKeygenStart  chan *pb.KeyGenPrepareRequest
	LocalpartyChan chan *network.Party
	RouterTable    chan *router.SortedRouterTable
	Message        chan network.Message
	Recv           chan *network.Message
	Out            chan *network.Message
	End            chan network.LocalSaveData
}

func NewTsKeyGenChan() *TsKeyGenChan {
	return &TsKeyGenChan{
		TsKeygenStart:  make(chan *pb.KeyGenPrepareRequest),
		LocalpartyChan: make(chan *network.Party),
		Out:            make(chan *network.Message),
		Message:        make(chan network.Message),
		Recv:           make(chan *network.Message),
		RouterTable:    make(chan *router.SortedRouterTable),
		End:            make(chan network.LocalSaveData),
		//ErrCh : make(chan *tss.Error),
	}
}

type TsKeyGenServer struct {
	TsChan      *TsKeyGenChan
	LocalParty  *network.Party
	RouterTable *router.SortedRouterTable
	Running     bool
	conf        *config.TssConfig
	MsgStore    map[string][]*network.Message
}

func NewTsKeyGenServer(tsChan *TsKeyGenChan) *TsKeyGenServer {
	return &TsKeyGenServer{
		TsChan:  tsChan,
		Running: false,
		MsgStore: make(map[string][]*network.Message),
	}
}

func (ts *TsKeyGenServer) SetConfig(conf *config.TssConfig) {
	ts.conf = conf
}

func (ts *TsKeyGenServer) GetConfig() *config.TssConfig {
	return ts.conf
}

func (ts *TsKeyGenServer) SetRouterTable(table *router.SortedRouterTable) {
	if ts.RouterTable == nil {
		ts.RouterTable = table
	}
}

func (ts *TsKeyGenServer) Prepare(urls []string, partyNum, threshold int) {
	pIDs := tss.GenerateTestPartyIDs(partyNum)

	table := router.NewSortedRouterTable(pIDs, urls)
	fmt.Println("=====TsKeyGenServer, Prepare===--------------", table, pIDs)
	// partyId 与 路由表绑定
	for _, pid := range table.Pids {
		encPId, err := utils.Encode(pid)
		if err != nil {
			fmt.Println("-----", err)
			break
		}
		encTable, err := utils.Encode(table)
		if err != nil {
			break
		}

		conn, err := grpc.Dial(table.GetURLByID(pid.Id), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			break
		}
		client := pb.NewTssServerClient(conn)
		_, err = client.SmtKeyGenStart(context.Background(), &pb.KeyGenStartRequest{Party: encPId, Parties: encTable, Threshold: int32(threshold)})
		if err != nil {
			fmt.Println("=========222==========", err)
			break
		}
		conn.Close()
	}
}

func (ts *TsKeyGenServer) ReceiveMsg(msg *network.Message) {
	//ts.LocalParty.Recv <- msg
	ts.MsgStore[msg.TaskName] = append(ts.MsgStore[msg.TaskName], msg)
	if len(ts.MsgStore[msg.TaskName]) == ts.LocalParty.Params().PartyCount()-1 {
		for _, msg_recv := range ts.MsgStore[msg.TaskName] {
			ts.LocalParty.Recv <- msg_recv
		}
	}
}

func (ts *TsKeyGenServer) InitMsg() {
	ts.MsgStore = make(map[string][]*network.Message)
}

func (ts *TsKeyGenServer) Start(party *network.Party) {
	fmt.Println("===============================smt Start=====================================")
	dt_start := time.Now()
	fmt.Println("===>KeyGen Start time: ", dt_start.Format("2006-01-02 15:04:05"))

	ts.InitMsg()
	ts.Running = true
	go func(p *network.Party) {
		tskeygen.PreRound(p)
		tskeygen.Rounds1(p)
		tskeygen.Rounds2(p)
		tskeygen.Rounds3(p)
		tskeygen.Rounds4(p)
		tskeygen.Rounds5(p)
		tskeygen.Outputs(p)
	}(party)

keygen:
	for {
		select {
		case msg := <-ts.TsChan.Out:
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), reflect.TypeOf(msg.MContent), msg.FromID.Index, "---->", msg.ToID.Index)
			for _, router_party := range ts.RouterTable.Pids {
				if router_party.Index == msg.ToID.Index {
					err := SendSmtMsg(msg, ts.RouterTable.GetURLByID(router_party.Id), "keygen")
					if err != nil {
						fmt.Println("--------trans--------", err)
						break keygen
					}
				}
			}
		case save := <-ts.TsChan.End:
			ts.InitMsg()
			fmt.Println("end------------------------------------>")
			fixtureFileName := fmt.Sprintf("%s/ts_keygen_data.json", ts.conf.SavePath)

			fd, _ := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			bz, _ := json.Marshal(save.Marshal())

			_, err := fd.Write(bz)
			if err != nil {
				fmt.Printf("unable to write to key file %s", fixtureFileName)
				os.Exit(-1)
			}
			fmt.Printf("Saved a key file : %s", fixtureFileName)
			smt_utils.WriteRouteTable(ts.conf.SavePath, ts.LocalParty.Params().PartyId(), ts.RouterTable, ts.conf.Threshold, 0)

			dt_end := time.Now()
			fmt.Println("===>KeyGen End time: ", dt_end.Format("2006-01-02 15:04:05"))
			fmt.Println("===>KeyGen Cost time: ", dt_end.Sub(dt_start))
			ts.Running = false
			break keygen
		}
	}

}
