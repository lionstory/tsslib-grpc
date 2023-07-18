package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"
	"os"
	"sync"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/tss"
	"google.golang.org/grpc"

	"github.com/lionstory/tsslib-grpc/pkg/config"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/resharing"
	smt_utils "github.com/lionstory/tsslib-grpc/smt/utils"
)

type TsReshareChan struct {
	TsReshareStart chan pb.SmtResharePrepareRequest
	LocalpartyChan chan *network.ReSharingParty
	Router_table   chan *router.SortedRouterTable
	Message        chan network.MessageResharing
	Recv           chan *network.MessageResharing
	Out            chan *network.MessageResharing
	End            chan network.LocalSaveData
}

func NewTsReshareChan() *TsReshareChan {
	return &TsReshareChan{
		TsReshareStart: make(chan pb.SmtResharePrepareRequest),
		LocalpartyChan: make(chan *network.ReSharingParty),
		Out:            make(chan *network.MessageResharing),
		Message:        make(chan network.MessageResharing),
		Recv:           make(chan *network.MessageResharing),
		Router_table:   make(chan *router.SortedRouterTable),
		End:            make(chan network.LocalSaveData),
	}
}

type TsReshareServer struct {
	ReshareChan  *TsReshareChan
	LocalParty   *network.ReSharingParty
	Router_table *router.SortedRouterTable
	Running      bool
	conf         *config.TssConfig
	MsgStore     map[string][]*network.MessageResharing
}

func NewTsReshareServer(tsChan *TsReshareChan) *TsReshareServer {
	return &TsReshareServer{
		ReshareChan: tsChan,
		Running:     false,
		MsgStore:    make(map[string][]*network.MessageResharing),
	}
}

func (ts *TsReshareServer) SetConfig(conf *config.TssConfig) {
	ts.conf = conf
}

func (ts *TsReshareServer) SetKeyRevision(KeyRevision int) {
	ts.conf.KeyRevision = KeyRevision
}

func (ts *TsReshareServer) SetPartyNum(PartyNum int) {
	ts.conf.PartyNum = PartyNum
}

func (ts *TsReshareServer) SetThreshold(Threshold int) {
	ts.conf.Threshold = Threshold
}

func (ts *TsReshareServer) GetConfig() *config.TssConfig {
	return ts.conf
}

func (ts *TsReshareServer) SetRouterTable(table *router.SortedRouterTable) {
	ts.Router_table = table
}

func (ts *TsReshareServer) Prepare(request pb.SmtResharePrepareRequest) {
	maxRevision := 0
	revisionRouter := map[int][]*tss.PartyID{}
	oldParty := tss.SortedPartyIDs{}
	partyIDs := make(tss.SortedPartyIDs, len(request.Urls))
	for index, url_path := range request.Urls {
		conn, err := grpc.Dial(url_path, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			fmt.Println("***********************************")
			break
		}
		client := pb.NewTssServerClient(conn)
		result, err := client.SmtSignCollectParty(context.Background(), &pb.SignCollectRequest{Index: int32(index)})
		if err != nil {
			fmt.Println("**************SmtSignCollectParty*********************", err)
			return
		}

		party_id := &tss.PartyID{}
		router_col := &router.PartyStatus{}
		if result.Data != nil {
			err = utils.Decode(result.Data, router_col)
			if err != nil {
				fmt.Println("**************decode PartyId*********************", err, result.Data)
				return
			}
			party_id = router_col.PartyId
			// oldParty = append(oldParty, party_id)
			if _, ok := revisionRouter[router_col.KeyRevision]; ok {
				revisionRouter[router_col.KeyRevision] = append(revisionRouter[router_col.KeyRevision], router_col.PartyId)
			} else {
				revisionRouter[router_col.KeyRevision] = []*tss.PartyID{router_col.PartyId}
			}
			if router_col.KeyRevision > maxRevision {
				maxRevision = router_col.KeyRevision
			}
		} else {
			msg_party := &tss.MessageWrapper_PartyID{
				Id:      fmt.Sprintf("%d", index+1),
				Moniker: fmt.Sprintf("P[%d]", index+1),
				Key:     common.MustGetRandomInt(256).Bytes(),
			}
			party_id.MessageWrapper_PartyID = msg_party
		}
		party_id.Index = index
		partyIDs[index] = party_id
		conn.Close()
	}
	for kvision, _parties := range revisionRouter {
		if kvision == maxRevision{
			for _, _mparty := range _parties{
				oldParty = append(oldParty, _mparty)
			}
		}
	}
	table := router.NewSortedRouterTable(partyIDs, request.Urls)
	ts.Router_table = table
	fmt.Println("=============smt prepare=================", oldParty, partyIDs, maxRevision, revisionRouter)
	encNewTable, err := utils.Encode(table)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	var resps []*pb.CommonResonse
	for _, pid := range ts.Router_table.Pids {
		encPid, err := utils.Encode(pid)
		if err != nil {
			fmt.Println("============", err)
			continue
		}
		url := ts.Router_table.GetURLByID(pid.Id)
		if url == "" {
			fmt.Printf("can not find url to send for party %v\n", pid)
			continue
		}
		OldParties, err := utils.Encode(oldParty)
		if err != nil {
			fmt.Println("============", err)
			continue
		}
		NewParties, err := utils.Encode(partyIDs)
		if err != nil {
			fmt.Println("============", err)
			continue
		}
		wg.Add(1)
		go func(wg *sync.WaitGroup, encPid []byte, url string) {
			defer wg.Done()

			conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				fmt.Println("---------------", err)
				return
			}
			defer conn.Close()
			client := pb.NewTssServerClient(conn)
			resp, err := client.SmtReshareStart(context.Background(),
				&pb.SmtReshareStartRequest{
					Party:        encPid,
					OldParties:   OldParties,
					NewParties:   NewParties,
					OldThreshold: request.OldThreshold,
					NewThreshold: request.NewThreshold,
					KeyRevision:  int32(maxRevision + 1),
					Table:        encNewTable,
				})
			if err != nil {
				fmt.Println("--------ReSharingInit Error---------", err)
				return
			}
			resps = append(resps, resp)

		}(&wg, encPid, url)
	}
	wg.Wait()
}

func (ts *TsReshareServer) ReceiveMsg(msg *network.MessageResharing) {
	//s.LocalParty.Recv <- msg
	// ts.MsgStore[msg.TaskName] = append(ts.MsgStore[msg.TaskName], msg)
	if _, ok := ts.MsgStore[msg.TaskName]; ok {
		ts.MsgStore[msg.TaskName] = append(ts.MsgStore[msg.TaskName], msg)
	} else {
		ts.MsgStore[msg.TaskName] = []*network.MessageResharing{msg}
	}
	var count int
	switch {
	case msg.TaskName == "reshare_Round1":
		if !ts.LocalParty.Params().IsOldCommittee() {
			count = len(ts.LocalParty.Params().Parties().IDs())
		} else {
			count = len(ts.LocalParty.Params().Parties().IDs()) - 1
		}
	case msg.TaskName == "reshare_Round2":
		count = ts.LocalParty.Params().NewPartyCount() - 1
	case msg.TaskName == "reshare_Round3":
		count = ts.LocalParty.Params().NewPartyCount() - 1
	}
	if len(ts.MsgStore[msg.TaskName]) == count {
		for _, msg_recv := range ts.MsgStore[msg.TaskName] {
			ts.LocalParty.Recv <- msg_recv
		}
	}
}

func (s *TsReshareServer) InitMsg() {
	s.MsgStore = make(map[string][]*network.MessageResharing)
}

func (ts *TsReshareServer) Start(party *network.ReSharingParty) {
	dt_start := time.Now()
	fmt.Println("===>Smtreshare Start time: ", dt_start.Format("2006-01-02 15:04:05"))
	ts.InitMsg()
	ts.Running = true
	go func(p *network.ReSharingParty) {
		resharing.Round1(p)
		resharing.Round2(p)
		resharing.Round3(p)
		resharing.Outputs(p)
	}(party)

Rekeygen:
	for {
		select {
		case msg := <-ts.ReshareChan.Out:
			fmt.Println(time.Now().Format("2006-01-02 15:04:05"), reflect.TypeOf(msg.MContent), msg.FromID.Index, "---->", msg.ToID.Index)
			for _, router_party := range ts.Router_table.Pids {
				if router_party.Index == msg.ToID.Index {
					err := SendSmtReshareMsg(msg, ts.Router_table.GetURLByID(router_party.Id))
					if err != nil {
						break Rekeygen
					}
				}
			}
		case save := <-ts.ReshareChan.End:
			ts.InitMsg()
			fmt.Println(ts.conf)
			fixtureFileName := fmt.Sprintf("%s/ts_keygen_data.json", ts.conf.SavePath)

			fd, _ := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
			bz, _ := json.Marshal(save.Marshal())

			_, err := fd.Write(bz)
			if err != nil {
				fmt.Printf("unable to write to key file %s", fixtureFileName)
				os.Exit(-1)
			}
			fmt.Printf("Saved a key file : %s", fixtureFileName)
			smt_utils.WriteRouteTable(ts.conf.SavePath, ts.LocalParty.Params().PartyId(), ts.Router_table, ts.conf.Threshold, ts.conf.KeyRevision)
			ts.Running = false

			dt_end := time.Now()
			fmt.Println("===>Smt reshare End time: ", dt_end.Format("2006-01-02 15:04:05"))
			fmt.Println("===>smt reshare Cost time: ", dt_end.Sub(dt_start))

			break Rekeygen
		}
	}
}
