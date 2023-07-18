package ecdsa

import (
	"context"
	"fmt"
	"time"

	"github.com/lionstory/tsslib-grpc/pkg/config"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	"google.golang.org/grpc"
)

type KeyGenChan struct {
	KeygenStart    chan *pb.KeyGenPrepareRequest
	LocalpartyChan chan *keygen.LocalParty
	Router_table   chan *router.SortedRouterTable
	Message        chan tss.Message
	Out            chan tss.Message
	End            chan keygen.LocalPartySaveData
	ErrCh          chan *tss.Error
}

func NewKeyGenChan() *KeyGenChan {
	return &KeyGenChan{
		KeygenStart:    make(chan *pb.KeyGenPrepareRequest),
		LocalpartyChan: make(chan *keygen.LocalParty),
		Out:            make(chan tss.Message),
		Message:        make(chan tss.Message),
		Router_table:   make(chan *router.SortedRouterTable),
		End:            make(chan keygen.LocalPartySaveData),
		ErrCh:          make(chan *tss.Error),
	}
}

type KeyGenServer struct {
	Keychan     *KeyGenChan
	LocalParty  *keygen.LocalParty
	RouterTable *router.SortedRouterTable
	Running     bool
	conf        *config.TssConfig
}

func NewKeyGenServer(keyGenChan *KeyGenChan) *KeyGenServer {
	return &KeyGenServer{
		Keychan: keyGenChan,
		Running: false,
	}
}

func (k *KeyGenServer) SetConfig(conf *config.TssConfig) {
	k.conf = conf
}

func (k *KeyGenServer) GetConfig() *config.TssConfig {
	return k.conf
}

func (k *KeyGenServer) SetRouterTable(table *router.SortedRouterTable) {
	k.RouterTable = table
}

func (k *KeyGenServer) Prepare(urls []string, partyNum, threshold int) {
	pIDs := tss.GenerateTestPartyIDs(partyNum)
	//p2pCtx := tss.NewPeerContext(pIDs)

	table := router.NewSortedRouterTable(pIDs, urls)
	k.RouterTable = table
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
		_, err = client.KeyGenStart(context.Background(), &pb.KeyGenStartRequest{Party: encPId, Parties: encTable, Threshold: int32(threshold)})
		if err != nil {
			fmt.Printf("------%v\n", err)
			break
		}
		conn.Close()
	}
}

func (k *KeyGenServer) UpdateRound(pMsg tss.ParsedMessage) {
	go utils.UpdateRound(k.LocalParty, pMsg, k.Keychan.ErrCh)
}

func (k *KeyGenServer) Start(P *keygen.LocalParty) {
	fmt.Println("&&&&------------------------KeygenServer start--------->")
	dt_start := time.Now()
	fmt.Println("===>KeyGen Start time: ", dt_start.Format("2006-01-02 15:04:05"))

	k.Running = true
	go func(P *keygen.LocalParty) {
		if err := P.Start(); err != nil {
			k.Keychan.ErrCh <- err
		}
	}(P)

keyGen:
	for {
		select {
		case err := <-k.Keychan.ErrCh:
			fmt.Println(err)
			k.Running = false
			break keyGen
		case msg := <-k.Keychan.Out:
			fmt.Println("&&&&------------------------KeygenServer message--------->%T", msg)
			dest := msg.GetTo()
			if dest == nil {
				// broadcast
				for _, router_party := range k.RouterTable.Pids {
					if router_party.Index == msg.GetFrom().Index {
						continue
					}
					go utils.SharedPartyUpdater(k.LocalParty, router_party, msg, k.RouterTable.GetURLByID(router_party.Id), "keygen", k.Keychan.ErrCh)
				}
			} else {
				// point to point
				if dest[0].Index == msg.GetFrom().Index {
					return
				}
				if router := k.RouterTable.GetURLByID(dest[0].Id); router != "" {
					go utils.SharedPartyUpdater(k.LocalParty, dest[0], msg, router, "keygen", k.Keychan.ErrCh)
				}
			}
		case save := <-k.Keychan.End:
			//fmt.Println(save)
			utils.TryWriteTestFixtureFile(k.conf.SavePath, save)
			utils.TryWriteRouteTable(k.conf.SavePath, k.LocalParty.PartyID(), k.RouterTable, k.conf.Threshold)
			k.Running = false

			dt_end := time.Now()
			fmt.Println("===>KeyGen End time: ", dt_end.Format("2006-01-02 15:04:05"))
			fmt.Println("===>KeyGen Cost time: ", dt_end.Sub(dt_start))

			break keyGen
		}
	}
}
