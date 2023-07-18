package ecdsa

import (
	"fmt"
	"github.com/bnb-chain/tss-lib/tss"
)

type EcdsaChan struct {
	Keychan       *KeyGenChan
	Signchan      *SignChan
	ReSharingChan *ReSharingChan
}

func NewEcdsaChan() *EcdsaChan {
	return &EcdsaChan{
		Keychan:       NewKeyGenChan(),
		Signchan:      NewSignChan(),
		ReSharingChan: NewReSharingChan(),
	}
}

type EcdsaServer struct {
	KeygenServer    *KeyGenServer
	SignServer      *SignServer
	ReSharingServer *ReSharingServer
}

func NewEcdsaServer(ecdsachan *EcdsaChan) *EcdsaServer {
	return &EcdsaServer{
		KeygenServer:    NewKeyGenServer(ecdsachan.Keychan),
		SignServer:      NewSignServer(ecdsachan.Signchan),
		ReSharingServer: NewReSharingServer(ecdsachan.ReSharingChan),
	}
}

func (es *EcdsaServer) Start() {
	fmt.Println("start........")
	for {
		select {
		case request := <-es.KeygenServer.Keychan.KeygenStart:
			fmt.Println("keygen task is preparing to start--------->", request)
			go es.KeygenServer.Prepare(request.Urls, int(request.PartyNum), int(request.Threshold))
		case table := <-es.KeygenServer.Keychan.Router_table:
			es.KeygenServer.SetRouterTable(table)
		case party := <-es.KeygenServer.Keychan.LocalpartyChan:
			go es.KeygenServer.Start(party)
			es.KeygenServer.LocalParty = party
		case pmsg := <-es.KeygenServer.Keychan.Message:
			es.KeygenServer.UpdateRound(pmsg.(tss.ParsedMessage))
		case request := <-es.SignServer.Signchan.SignPrepare:
			fmt.Println("sign task is preparing to start--------->", request)
			go es.SignServer.Prepare(request.Urls, request.Message)
		case signParty := <-es.SignServer.Signchan.LocalpartyChan:
			go es.SignServer.Start(signParty)
			es.SignServer.LocalParty = signParty
		case signTable := <-es.SignServer.Signchan.Router_table:
			es.SignServer.SetRouterTable(signTable)
		case signPmsg := <-es.SignServer.Signchan.Message:
			es.SignServer.UpdateRound(signPmsg.(tss.ParsedMessage))
		case request := <-es.ReSharingServer.ReSharingChan.ReShareStart:
			fmt.Println("reSharing task is preparing to start--------->", request.Urls)
			go es.ReSharingServer.Prepare(request.Urls, int(request.Threshold), int(request.OldThreshold))
		case table := <-es.ReSharingServer.ReSharingChan.OldRouterTable:
			es.ReSharingServer.SetOldRouterTable(table)
		case table := <-es.ReSharingServer.ReSharingChan.NewRouterTable:
			es.ReSharingServer.SetNewRouterTable(table)
		case reSharingParty := <-es.ReSharingServer.ReSharingChan.OldLocalPartyChan:
			es.ReSharingServer.OldLocalParty = reSharingParty
		case reSharingParty := <-es.ReSharingServer.ReSharingChan.NewLocalPartyChan:
			es.ReSharingServer.NewLocalParty = reSharingParty
			//go es.ReSharingServer.Start()
		case msg := <-es.ReSharingServer.ReSharingChan.MessageToOld:
			es.ReSharingServer.UpdateOldRound(msg.(tss.ParsedMessage))
		case msg := <-es.ReSharingServer.ReSharingChan.MessageToNew:
			es.ReSharingServer.UpdateNewRound(msg.(tss.ParsedMessage))
		}
	}
}
