package resharing

import (
	"encoding/hex"
	"math/big"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/vss"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
)

type Round1Info struct {
	FromId   int
	Y        *big.Int
	Wix      *big.Int
	Wiy      *big.Int
	Xx       *big.Int
	Xy       *big.Int
	Ax       *big.Int
	Ay       *big.Int
}

func (p *Round1Info) DoSomething(party *network.ReSharingParty){

}

func (p *Round1Info) MarshalString() (string, error){
	data, err := utils.Encode(p)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (p *Round1Info) UnshalString(data string) error{
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

func Round1(party *network.ReSharingParty) {
	if !party.Params().IsOldCommittee(){
		return
	}

	lambda := vss.Lagrange(party.Params().OldParties().IDs(), party.Params().PartyId().Index, party.Params().EC().Params().N)
	wi := new(big.Int).Mul(lambda, party.Input.Y)
	_, _, _, vssy := vss.Vssshare(party.Params().EC(), wi, party.Params().NewThreshold(),
		party.Params().NewPartyCount(), party.Params().NewParties().IDs())
	//fmt.Println(vssa, vssAx, vssAy, vssy)

	wix, wiy := party.Params().EC().ScalarBaseMult(wi.Bytes())

	for _, _partyId := range party.Params().NewParties().IDs(){
		//fmt.Println("@@@@@@@@@@@@@@@@@@@@@@@@@@", party.Params().PartyId().Index, _partyId.String(), _partyId.Index)
		if _partyId.Index != party.Params().PartyId().Index{
			Round1Content := Round1Info{
				FromId: party.Params().PartyId().Index,
				Y: vssy[_partyId.Index],
				Wix: wix,
				Wiy: wiy,
				Xx: party.Input.Xx,
				Xy: party.Input.Xy,
				Ax: party.Input.Ax,
				Ay: party.Input.Ay,
			}
			Msg := network.MessageResharing{
				TaskName: "reshare_Round1",
				FromID: party.Params().PartyId(),
				ToID: _partyId,
				MContent: &Round1Content,
			}
			//fmt.Println("==========>>>>>>>>>>>", )
			party.Out <- &Msg
		}else{
			Round1Content := Round1Info{
				FromId: party.Params().PartyId().Index,
				Y: vssy[_partyId.Index],
				Wix: wix,
				Wiy: wiy,
				Xx: party.Input.Xx,
				Xy: party.Input.Xy,
				Ax: party.Input.Ax,
				Ay: party.Input.Ay,
			}
			party.Temp.DgRound1Message[party.Params().PartyId().Index] = network.MessageResharing{
				TaskName: "reshare_Round1",
				FromID: party.Params().PartyId(),
				ToID: _partyId,
				MContent: &Round1Content,
			}
		}
	}

}