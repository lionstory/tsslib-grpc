package smt

import (
	"context"
	"fmt"

	"github.com/bnb-chain/tss-lib/tss"
	"google.golang.org/grpc"

	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/resharing"
	"github.com/lionstory/tsslib-grpc/smt/signing"
	"github.com/lionstory/tsslib-grpc/smt/tskeygen"
)

type SmtChan struct {
	Keychan     *TsKeyGenChan
	Signchan    *TsSignChan
	ReshareChan *TsReshareChan
}

func NewSmtChan() *SmtChan {
	return &SmtChan{
		Keychan:     NewTsKeyGenChan(),
		Signchan:    NewTsSignChan(),
		ReshareChan: NewTsReshareChan(),
	}
}

type SmtServer struct {
	KeygenServer  *TsKeyGenServer
	SignServer    *SignServer
	ReshareServer *TsReshareServer
}

func NewSmtServer(smtchan *SmtChan) *SmtServer {
	return &SmtServer{
		KeygenServer:  NewTsKeyGenServer(smtchan.Keychan),
		SignServer:    NewSignServer(smtchan.Signchan),
		ReshareServer: NewTsReshareServer(smtchan.ReshareChan),
	}
}

func (ss *SmtServer) Start() {
	fmt.Println("start........")
	for {
		select {
		case request := <-ss.KeygenServer.TsChan.TsKeygenStart:
			fmt.Println("sm2 keygen task is preparing to start--------->", request)
			go ss.KeygenServer.Prepare(request.Urls, int(request.PartyNum), int(request.Threshold))
		case table := <-ss.KeygenServer.TsChan.RouterTable:
			ss.KeygenServer.SetRouterTable(table)
		case party := <-ss.KeygenServer.TsChan.LocalpartyChan:
			go ss.KeygenServer.Start(party)
			ss.KeygenServer.LocalParty = party
		case pmsg := <-ss.KeygenServer.TsChan.Message:
			ss.KeygenServer.ReceiveMsg(&pmsg)
		case request := <-ss.SignServer.Signchan.SignPrepare:
			go ss.SignServer.Prepare(request.Urls, request.Message)
		case signParty := <-ss.SignServer.Signchan.LocalpartyChan:
			go ss.SignServer.Start(signParty)
			ss.SignServer.LocalParty = signParty
		case signTable := <-ss.SignServer.Signchan.RouterTable:
			ss.SignServer.SetRouterTable(signTable)
		case signPmsg := <-ss.SignServer.Signchan.Message:
			ss.SignServer.ReceiveMsg(&signPmsg)
		case resharePre := <-ss.ReshareServer.ReshareChan.TsReshareStart:
			go ss.ReshareServer.Prepare(resharePre)
		case table := <-ss.ReshareServer.ReshareChan.Router_table:
			ss.ReshareServer.SetRouterTable(table)
		case reshareParty := <-ss.ReshareServer.ReshareChan.LocalpartyChan:
			go ss.ReshareServer.Start(reshareParty)
			ss.ReshareServer.LocalParty = reshareParty
		case reshareMsg := <-ss.ReshareServer.ReshareChan.Message:
			ss.ReshareServer.ReceiveMsg(&reshareMsg)
		}
	}
}

func SendSmtMsg(msg *network.Message, url string, taskName string) error {
	fromId, toId, content, err := MessageByte(msg)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	client := pb.NewTssServerClient(conn)
	if taskName == "keygen" {
		_, err = client.SmtKeygenTransMsg(context.Background(), &pb.TransSmtMsgRequest{Content: content, FromId: fromId, ToId: toId, TaskName: msg.TaskName})
		if err != nil {
			fmt.Println("**********33**********", err)
			return err
		}
	} else if taskName == "signing" {
		_, err = client.SmtSignTransMsg(context.Background(), &pb.TransSmtMsgRequest{Content: content, FromId: fromId, ToId: toId, TaskName: msg.TaskName})
		if err != nil {
			return err
		}
	}
	return nil
}

func MessageByte(msg *network.Message) ([]byte, []byte, string, error) {
	fromId, err := utils.Encode(msg.FromID)
	if err != nil {
		return nil, nil, "", err
	}

	toId, err := utils.Encode(msg.ToID)
	if err != nil {
		return nil, nil, "", err
	}
	content, err := msg.MContent.MarshalString()
	if err != nil {
		return nil, nil, "", err
	}

	return fromId, toId, content, nil
}

func ToMessage(from_Id, to_Id []byte, content, taskName string) (*network.Message, error) {
	message := &network.Message{
		TaskName: taskName,
	}
	fromId := &tss.PartyID{}
	err := utils.Decode(from_Id, fromId)
	if err != nil {
		return nil, err
	}
	message.FromID = fromId

	toId := &tss.PartyID{}
	err = utils.Decode(to_Id, toId)
	if err != nil {
		return nil, err
	}
	message.ToID = toId
	switch {
	case taskName == "keygen_preround":
		con := &tskeygen.PreRoundInfo{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "keygen_round1":
		con := &tskeygen.Round1Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "keygen_round2":
		con := &tskeygen.Round2Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "keygen_round3":
		con := &tskeygen.Round3Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "keygen_round4":
		con := &tskeygen.Round4Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "keygen_round5":
		con := &tskeygen.Round5Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "signing_round1":
		con := &signing.Round1Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "signing_round2":
		con := &signing.Round2Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "signing_round3":
		con := &signing.Round3Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	}
	return message, nil
}

func ResharingMessageByte(msg *network.MessageResharing) ([]byte, []byte, string, error) {
	fromId, err := utils.Encode(msg.FromID)
	if err != nil {
		return nil, nil, "", err
	}

	toId, err := utils.Encode(msg.ToID)
	if err != nil {
		return nil, nil, "", err
	}
	content, err := msg.MContent.MarshalString()
	if err != nil {
		return nil, nil, "", err
	}

	return fromId, toId, content, nil
}

func ToResharingMessage(from_Id, to_Id []byte, content, taskName string) (*network.MessageResharing, error) {
	message := &network.MessageResharing{
		TaskName: taskName,
	}
	fromId := &tss.PartyID{}
	err := utils.Decode(from_Id, fromId)
	if err != nil {
		return nil, err
	}
	message.FromID = fromId

	toId := &tss.PartyID{}
	err = utils.Decode(to_Id, toId)
	if err != nil {
		return nil, err
	}
	message.ToID = toId
	switch {
	case taskName == "reshare_Round1":
		con := &resharing.Round1Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "reshare_Round2":
		con := &resharing.Round2Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	case taskName == "reshare_Round3":
		con := &resharing.Round3Info{}
		err := con.UnshalString(content)
		if err != nil {
			return nil, err
		}
		message.MContent = con
	}
	return message, nil
}

func SendSmtReshareMsg(msg *network.MessageResharing, url string) error {
	fromId, toId, content, err := ResharingMessageByte(msg)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	client := pb.NewTssServerClient(conn)
	_, err = client.SmtReshareTransMsg(context.Background(), &pb.TransSmtMsgRequest{Content: content, FromId: fromId, ToId: toId, TaskName: msg.TaskName})
	if err != nil {
		fmt.Println("**********33**********", err)
		return err
	}
	return nil
}
