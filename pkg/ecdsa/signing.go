package ecdsa

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/lionstory/tsslib-grpc/pkg/config"
	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
	"google.golang.org/grpc"
)

type SignChan struct {
	SignPrepare    chan *pb.SignPrepareRequest
	LocalpartyChan chan *signing.LocalParty
	Router_table   chan *router.SortedRouterTable
	Message        chan tss.Message
	Out            chan tss.Message
	End            chan common.SignatureData
	ErrCh          chan *tss.Error
}

func NewSignChan() *SignChan {
	return &SignChan{
		SignPrepare:    make(chan *pb.SignPrepareRequest),
		LocalpartyChan: make(chan *signing.LocalParty),
		Out:            make(chan tss.Message),
		Message:        make(chan tss.Message),
		Router_table:   make(chan *router.SortedRouterTable),
		End:            make(chan common.SignatureData),
		ErrCh:          make(chan *tss.Error),
	}
}

type SignServer struct {
	Signchan    *SignChan
	LocalParty  *signing.LocalParty
	RouterTable *router.SortedRouterTable
	Running     bool
	conf        *config.TssConfig
}

func NewSignServer(signchan *SignChan) *SignServer {
	sign_server := SignServer{
		Signchan: signchan,
		Running:  false,
	}
	return &sign_server
}

func (k *SignServer) SetConfig(conf *config.TssConfig) {
	k.conf = conf
}

func (k *SignServer) GetConfig() *config.TssConfig {
	return k.conf
}

func (s *SignServer) SetRouterTable(table *router.SortedRouterTable) {
	s.RouterTable = table
}

func (s *SignServer) UpdateRound(pMsg tss.ParsedMessage) {
	go utils.UpdateRound(s.LocalParty, pMsg, s.Signchan.ErrCh)
}

func (s *SignServer) Prepare(urls []string, msg string) {
	var partyIDs tss.UnSortedPartyIDs
	routerTable := map[string]string{}
	for index, url_path := range urls {
		conn, err := grpc.Dial(url_path, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			break
		}
		client := pb.NewTssServerClient(conn)
		result, err := client.SignCollectParty(context.Background(), &pb.SignCollectRequest{Index: int32(index)})
		if err != nil {
			fmt.Println("========SignServer.Prepare==========", err)
			continue
		}
		var partyStatus router.PartyStatus
		err = utils.Decode(result.Data, &partyStatus)
		if err != nil {
			fmt.Println("========SignServer.Prepare==========", err)
			return
		}
		routerTable[partyStatus.PartyId.Id] = url_path
		partyIDs = append(partyIDs, partyStatus.PartyId)
		conn.Close()
	}

	if len(partyIDs) < len(urls) {
		return
	}
	sort_part := tss.SortPartyIDs(partyIDs)

	table := router.NewSortedRouterTable2(sort_part, routerTable)
	s.RouterTable = table

	PIDS, err := utils.Encode(sort_part)
	if err != nil {
		fmt.Println("-------------PIDS------------------------", err)
		return
	}

	table_data, err := utils.Encode(table)

	for _, v := range table.Pids {
		url_path := table.GetURLByID(v.Id)
		if url_path == "" {
			continue
		}
		conn, err := grpc.Dial(url_path, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			break
		}
		client := pb.NewTssServerClient(conn)
		_, err = client.SignStart(context.Background(), &pb.SignStartRequest{Index: int32(v.Index), Parties: PIDS, Table: table_data, Msg: msg})
		if err != nil {
			fmt.Println("-------------PIDS------------------------", err)
			break
		}
	}
}

func (s *SignServer) Start(P *signing.LocalParty) {
	dt_start := time.Now()
	fmt.Println("===>Signing Start time: ", dt_start.Format("2006-01-02 15:04:05"))

	go func(P *signing.LocalParty) {
		if err := P.Start(); err != nil {
			s.Signchan.ErrCh <- err
		}
	}(P)

signPhrase:
	for {
		select {
		case err := <-s.Signchan.ErrCh:
			fmt.Println(err)
			break signPhrase
		case msg := <-s.Signchan.Out:
			fmt.Println("&&&&------------------------SignServer message--------->%T", msg)
			dest := msg.GetTo()
			if dest == nil {
				// broadcast
				for _, router_party := range s.RouterTable.Pids {
					if router_party.Index == msg.GetFrom().Index {
						continue
					}
					go utils.SharedPartyUpdater(s.LocalParty, router_party, msg, s.RouterTable.GetURLByID(router_party.Id), "signing", s.Signchan.ErrCh)
				}
			} else {
				// point to point
				if dest[0].Index == msg.GetFrom().Index {
					return
				}
				if router := s.RouterTable.GetURLByID(dest[0].Id); router != "" {
					go utils.SharedPartyUpdater(s.LocalParty, dest[0], msg, router, "signing", s.Signchan.ErrCh)
				}
			}
		case save := <-s.Signchan.End:
			key, err := utils.LoadKeyInfo(s.conf.SavePath)
			if err != nil {
				fmt.Println(err)
			}
			pk := ecdsa.PublicKey{
				Curve: tss.EC(),
				X:     key.ECDSAPub.X(),
				Y:     key.ECDSAPub.Y(),
			}
			ok := ecdsa.Verify(&pk, big.NewInt(42).Bytes(), new(big.Int).SetBytes(save.R), new(big.Int).SetBytes(save.S))
			fmt.Println("ECDSA signing test done.", ok)

			dt_end := time.Now()
			fmt.Println("===>Signing End time: ", dt_end.Format("2006-01-02 15:04:05"))
			fmt.Println("===>Signing Cost time: ", dt_end.Sub(dt_start))

			break signPhrase
		}
	}
}
