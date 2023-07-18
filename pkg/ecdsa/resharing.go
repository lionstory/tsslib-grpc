package ecdsa

import (
	"context"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/config"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/ecdsa/resharing"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"log"
	"sync"
	"time"
)

type ReSharingChan struct {
	ReShareStart      chan *pb.ReSharingPrepareRequest
	OldLocalPartyChan chan *resharing.LocalParty
	NewLocalPartyChan chan *resharing.LocalParty
	OldRouterTable    chan *router.SortedRouterTable
	NewRouterTable    chan *router.SortedRouterTable
	MessageToOld      chan tss.Message
	MessageToNew      chan tss.Message
	Out               chan tss.Message
	End               chan keygen.LocalPartySaveData
	ErrCh             chan *tss.Error
}

func NewReSharingChan() *ReSharingChan {
	return &ReSharingChan{
		ReShareStart:      make(chan *pb.ReSharingPrepareRequest),
		OldLocalPartyChan: make(chan *resharing.LocalParty),
		NewLocalPartyChan: make(chan *resharing.LocalParty),
		Out:               make(chan tss.Message),
		MessageToOld:      make(chan tss.Message),
		MessageToNew:      make(chan tss.Message),
		OldRouterTable:    make(chan *router.SortedRouterTable),
		NewRouterTable:    make(chan *router.SortedRouterTable),
		End:               make(chan keygen.LocalPartySaveData),
		ErrCh:             make(chan *tss.Error),
	}
}

type ReSharingServer struct {
	ReSharingChan  *ReSharingChan
	OldLocalParty  *resharing.LocalParty
	NewLocalParty  *resharing.LocalParty
	OldRouterTable *router.SortedRouterTable
	NewRouterTable *router.SortedRouterTable
	Running        bool
	conf           *config.TssConfig
}

func NewReSharingServer(reSharingChan *ReSharingChan) *ReSharingServer {
	return &ReSharingServer{
		ReSharingChan: reSharingChan,
		Running:       false,
	}
}

func (r *ReSharingServer) SetConfig(conf *config.TssConfig) {
	r.conf = conf
}

func (r *ReSharingServer) GetConfig() *config.TssConfig {
	return r.conf
}

func (r *ReSharingServer) SetOldRouterTable(table *router.SortedRouterTable) {
	r.OldRouterTable = table
}

func (r *ReSharingServer) SetNewRouterTable(table *router.SortedRouterTable) {
	r.NewRouterTable = table
}

func (r *ReSharingServer) UpdateOldRound(pMsg tss.ParsedMessage) {
	go utils.UpdateRound(r.OldLocalParty, pMsg, r.ReSharingChan.ErrCh)
}

func (r *ReSharingServer) UpdateNewRound(pMsg tss.ParsedMessage) {
	go utils.UpdateRound(r.NewLocalParty, pMsg, r.ReSharingChan.ErrCh)
}

func (r *ReSharingServer) Prepare(urls []string, threshold, oldThreshold int) {
	fmt.Println("================resharing prepare============")

	maxRevision := 0
	revisionRouter := map[int][]*router.Router{}
	for index, url_path := range urls {
		conn, err := grpc.Dial(url_path, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			break
		}
		client := pb.NewTssServerClient(conn)
		if result, err := client.SignCollectParty(context.Background(), &pb.SignCollectRequest{Index: int32(index)}); err != nil {
			fmt.Println("----------ReSharingServer.SignCollectParty-----------", err)
			continue
		} else {
			var partyStatus router.PartyStatus
			if err = utils.Decode(result.Data, &partyStatus); err != nil {
				fmt.Println("----------ReSharingServer.SignCollectParty-----------", err)
				continue
			} else {
				// 只保留 revision 与max 相同的party
				r := &router.Router{PartyId: partyStatus.PartyId, Url: url_path}
				if _, ok := revisionRouter[partyStatus.KeyRevision]; ok {
					revisionRouter[partyStatus.KeyRevision] = append(revisionRouter[partyStatus.KeyRevision], r)
				} else {
					revisionRouter[partyStatus.KeyRevision] = []*router.Router{r}
				}

				if partyStatus.KeyRevision > maxRevision {
					maxRevision = partyStatus.KeyRevision
				}
			}
		}
		conn.Close()
	}

	var revisionParties tss.UnSortedPartyIDs
	oldRouteTable := map[string]string{}
	for _, v := range revisionRouter[maxRevision] {
		revisionParties = append(revisionParties, v.PartyId)
		oldRouteTable[v.PartyId.Id] = v.Url
	}
	if len(revisionParties) == 0 {
		return
	}
	sortedParties := tss.SortPartyIDs(revisionParties)
	oldSortedRouteTable := router.NewSortedRouterTable2(sortedParties, oldRouteTable)

	newPIDs := tss.GenerateTestPartyIDs(len(urls))
	newSortedRouteTable := router.NewSortedRouterTable(newPIDs, urls)

	r.NewRouterTable = newSortedRouteTable
	r.OldRouterTable = oldSortedRouteTable

	encNewTable, err := utils.Encode(newSortedRouteTable)
	if err != nil {
		fmt.Printf("------%v\n", err)
	}

	encOldParties, err := utils.Encode(oldSortedRouteTable)
	if err != nil {
		fmt.Printf("------%v\n", err)
	}

	var wg sync.WaitGroup
	var resps []*pb.CommonResonse
	for _, pid := range r.NewRouterTable.Pids {
		encPid, err := utils.Encode(pid)
		if err != nil {
			fmt.Println("============", err)
			continue
		}

		url := r.NewRouterTable.GetURLByID(pid.Id)
		if url == "" {
			fmt.Printf("can not find url to send for party %v\n", pid)
			continue
		}

		oldPid := r.OldRouterTable.GetPidByUrl(url)
		var encOldPid []byte
		if oldPid == nil {
			fmt.Printf("old pid doens't exist for party %v\n", pid)
			encOldPid = nil
		} else {
			if encOldPid, err = utils.Encode(oldPid); err != nil {
				fmt.Printf("can not encode oldpid for old party %v\n", oldPid)
				encOldPid = nil
			}
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, encPid, encOldPid []byte, url string) {
			defer wg.Done()

			conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				fmt.Println("---------------", err)
				return
			}
			defer conn.Close()
			client := pb.NewTssServerClient(conn)
			resp, err := client.ReSharingInit(context.Background(),
				&pb.ReSharingStartRequest{
					Party: encPid, Parties: encNewTable,
					Threshold:      int32(threshold),
					KeyRevision:    int32(maxRevision + 1),
					OldParty:       encOldPid,
					OldParties:     encOldParties,
					OldThreshold:   int32(oldThreshold),
					OldKeyRevision: int32(maxRevision)})
			if err != nil {
				fmt.Println("--------ReSharingInit Error---------", err)
				return
			}
			resps = append(resps, resp)

		}(&wg, encPid, encOldPid, url)
	}
	wg.Wait()

	allSuccess := true
	for _, r := range resps {
		if r.Msg != "success" {
			allSuccess = false
		}
	}

	if !allSuccess {
		fmt.Println("--------------failed to init all party--------------")
		return
	}
	for _, pid := range r.NewRouterTable.Pids {
		wg.Add(1)
		go func(wg *sync.WaitGroup, r *router.Router) {
			defer wg.Done()

			conn, err := grpc.Dial(r.Url, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				fmt.Println("---------------", err)
				return
			}
			defer conn.Close()

			client := pb.NewTssServerClient(conn)
			_, err = client.ReSharingStart(context.Background(), &empty.Empty{})
			if err != nil {
				fmt.Println("--------ReSharingStart Error---------", err)
				return
			}
		}(&wg, &router.Router{PartyId: pid, Url: r.NewRouterTable.GetURLByID(pid.Id)})
	}
	wg.Wait()
}

func (r *ReSharingServer) Start() {
	fmt.Println("================resharing start============")
	dt_start := time.Now()
	fmt.Println("===>resharing Start time: ", dt_start.Format("2006-01-02 15:04:05"))
	if r.NewLocalParty != nil {
		go func(P *resharing.LocalParty) {
			if err := P.Start(); err != nil {
				r.ReSharingChan.ErrCh <- err
			}
		}(r.NewLocalParty)
	}

	if r.OldLocalParty != nil {
		go func(P *resharing.LocalParty) {
			if err := P.Start(); err != nil {
				r.ReSharingChan.ErrCh <- err
			}
		}(r.OldLocalParty)
	}

	endCount := 0
resharing:
	for {
		select {
		case err := <-r.ReSharingChan.ErrCh:
			fmt.Println("---ReSharingChan.ErrCh---", err)
			break resharing
		case msg := <-r.ReSharingChan.Out:
			dest := msg.GetTo()
			if dest == nil {
				log.Fatal("did not expect a msg to have a nil destination during resharing")
			}
			/*
				todo:
				  这里使用 SharedPartyUpdater2 函数的原因是，新旧 party 的 index 是有可能相同的。
				  也就是可能出现 msg 中的 party.id.index与router.id中的一样。
				  在 SharedPartyUpdater 因为通过比较index 来达成少发消息，所以可能导致消息漏发
			*/
			if msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, p := range dest {
					if router := r.OldRouterTable.GetURLByID(p.Id); router != "" {
						go utils.SharedPartyUpdater2(r.OldLocalParty, p, msg, router, "reSharingOld", r.ReSharingChan.ErrCh)
					}
				}
			}
			if !msg.IsToOldCommittee() || msg.IsToOldAndNewCommittees() {
				for _, p := range dest {
					if router := r.NewRouterTable.GetURLByID(p.Id); router != "" {
						go utils.SharedPartyUpdater2(r.NewLocalParty, p, msg, router, "reSharingNew", r.ReSharingChan.ErrCh)
					}
				}
				//for _, router := range r.NewRouterTable.Table{
				//	go utils.SharedPartyUpdater(router.PartyId, msg, router.Url, "reSharingNew", r.ReSharingChan.ErrCh)
				//}
			}
		case save := <-r.ReSharingChan.End:
			// old committee members that aren't receiving a share have their Xi zeroed
			if save.Xi != nil {
				utils.WriteTestFixtureFile(r.conf.SavePath, save)
				utils.WriteRouteTable(r.conf.SavePath, r.NewLocalParty.PartyID(), r.NewRouterTable, r.conf.Threshold, r.conf.KeyRevision)
			}
			// save new partyid

			fmt.Printf("&&&&------------------------ReSharingServer save--------->\n")
			//break resharing
			// 一个全新的party，不会有oldlocalparty

			if r.OldLocalParty == nil || endCount == 1 {
				dt_end := time.Now()
				fmt.Println("===>Resharing End time: ", dt_end.Format("2006-01-02 15:04:05"))
				fmt.Println("===>Resharing Cost time: ", dt_end.Sub(dt_start))
				r.Running = false
				break resharing
			} else {
				endCount++
			}
		}
	}
}

func Keys(m map[string]*router.Router) string {
	t := make([]string, len(m))
	for k, _ := range m {
		t = append(t, k)
	}
	return fmt.Sprintf("%v", t)
}
