package resharing

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
)

type Round3Info struct {
	FromID            int
	Yix               *big.Int
	Yiy               *big.Int
}

func (p *Round3Info) MarshalString() (string, error){
	data, err := utils.Encode(p)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (p *Round3Info) UnshalString(data string) error{
	content, err := hex.DecodeString(data)
	if err != nil{
		return err
	}
	err = utils.Decode(content, p)
	if err != nil{
		return err
	}
	return nil
}

func (p *Round3Info) DoSomething(party *network.ReSharingParty){
	party.Save.Yix[p.FromID] = p.Yix
	party.Save.Yiy[p.FromID] = p.Yiy
}

func Round3(party *network.ReSharingParty){
	if !party.Params().IsNewCommittee(){
		return
	}
	for i := 0; i < party.Params().NewPartyCount()-1; i++ {
		val := <-party.Recv
		if val.TaskName != "reshare_Round2" {
			fmt.Println("message taskName error ", val.TaskName)
		}
		val.MContent.DoSomething(party)
		party.Temp.DgRound2Message[party.Params().PartyId().Index] = *val
	}
	// 计算 Y
	Y := new(big.Int)
	ax, ay := new(big.Int), new(big.Int)
	Ax, Ay := new(big.Int), new(big.Int)
	Xx, Xy := new(big.Int), new(big.Int)
	for _, msg := range party.Temp.DgRound1Message {
		p := msg.MContent.(*Round1Info)
		Y.Add(Y, p.Y)
		ax, ay = party.Params().EC().Add(ax, ay, p.Wix, p.Wiy)
		Ax = p.Ax
		Ay = p.Ay
		Xx = p.Xx
		Xy = p.Xy
	}
	//fmt.Println("===============", Ax, Ay, ax, ay)
	party.Save.Y = Y
	yix, yiy := party.Params().EC().ScalarBaseMult(Y.Bytes())
	party.Save.Yix[party.Params().PartyId().Index] = yix
	party.Save.Yiy[party.Params().PartyId().Index] = yiy

	// 验证Ax，Ay
	if Ax.Cmp(ax) != 0 || Ay.Cmp(ay) != 0 {
		fmt.Println("error,please run resharing Ax|AY")
	}
	party.Save.Ax = Ax
	party.Save.Ay = Ay
	party.Save.Xx = Xx
	party.Save.Xy = Xy
	// 更新 Rho
	Rho := new(big.Int)
	for _, rhoi := range party.Temp.Rhoi {
		Rho.Add(Rho, rhoi)
	}
	party.Save.Rho = Rho
	for _, _partyId := range party.Params().NewParties().IDs() {
		Round3Content := Round3Info{
			FromID: party.Params().PartyId().Index,
			Yix: yix,
			Yiy: yiy,
		}
		if _partyId.Index != party.Params().PartyId().Index {
			Msg := network.MessageResharing{
				TaskName: "reshare_Round3",
				FromID: party.Params().PartyId(),
				ToID: _partyId,
				MContent: &Round3Content,
			}
			party.Out <- &Msg
		}
	}
}