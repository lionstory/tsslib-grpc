package network

import (
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
)

type Message struct {
	TaskName string
	FromID   *tss.PartyID
	ToID     *tss.PartyID
	MContent Content
}

type Content interface {
	DoSomething(*Party)
	MarshalString() (string, error)
	UnshalString(data string) error
}

func (message *Message) Bytes()([]byte, []byte, string, error){
	fromId, err := utils.Encode(message.FromID)
	if err != nil{
		return nil, nil, "", err
	}

	toId, err := utils.Encode(message.ToID)
	if err != nil{
		return nil, nil, "", err
	}
	content, err := message.MContent.MarshalString()
	if err != nil{
		return nil, nil, "", err
	}
	return fromId, toId, content, nil
}

func (message *Message) SetBytes(from_Id, to_Id []byte, content string) error{
	fromId := &tss.PartyID{}
	err := utils.Decode(from_Id, fromId)
	if err != nil{
		return err
	}
	message.FromID = fromId

	toId := &tss.PartyID{}
	err = utils.Decode(to_Id, toId)
	if err != nil{
		return err
	}
	message.ToID = toId
	err = message.MContent.UnshalString(content)
	if err != nil{
		return err
	}
	return nil
}

type MessageResharing struct {
	TaskName string
	FromID   *tss.PartyID
	ToID     *tss.PartyID
	MContent ReContent
}

type ReContent interface {
	DoSomething(*ReSharingParty)
	MarshalString() (string, error)
	UnshalString(data string) error
}
