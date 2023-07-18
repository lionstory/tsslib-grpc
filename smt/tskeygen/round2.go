package tskeygen

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/zk"
)

type Round2Info struct {
	FromID  int
	Xix     *big.Int
	Xiy     *big.Int
	Gammaix *big.Int
	Gammaiy *big.Int
	Rhoi    *big.Int
	Ui      *big.Int
}

func (p *Round2Info)MarshalString() (string, error){
	data, err := utils.Encode(p)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (p *Round2Info) UnshalString(data string) error{
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

func (p *Round2Info) DoSomething(party *network.Party){
	party.Mtx.Lock()
	party.Hash.Write(zk.BytesCombine(party.Data.Rtig.Bytes(),
		p.Xix.Bytes(), p.Xiy.Bytes(), p.Gammaix.Bytes(),
		p.Gammaiy.Bytes(), p.Rhoi.Bytes(), p.Ui.Bytes()))
	bytes := party.Hash.Sum(nil)

	Vi2 := new(big.Int).SetBytes(bytes)

	party.Hash.Reset()
	party.Mtx.Unlock()

	// 比较Vi2和Vi
	Vi3 := party.KeyGenTem.V[p.FromID]

	if Vi2.Cmp(Vi3) != 0{
		fmt.Println("Vi2 != Vi3 error", p.FromID)
	}

	party.Data.Rho.Add(party.Data.Rho, p.Rhoi)
}

func Rounds2(party *network.Party){
	//本地接受消息
	for i := 0; i < party.Params().PartyCount()-1; i++ {
		val := <-party.Recv // 出 chan
		//记录了每一个参与方的Vi
		val.MContent.DoSomething(party)
		//本地计算消息
	}

	MRoundContent := Round2Info{
		FromID: 	party.Params().PartyId().Index,
		Xix: 		party.KeyGenTem.Xix,
		Xiy:    	party.KeyGenTem.Xiy,
		Gammaix: 	party.KeyGenTem.Gammaix[party.Params().PartyId().Index],
		Gammaiy: 	party.KeyGenTem.Gammaiy[party.Params().PartyId().Index],
		Rhoi: 		party.KeyGenTem.Rhoi,
		Ui:			party.KeyGenTem.Ui,
	}

	//广播消息
	for _, _partyId := range party.Params().Parties().IDs(){
		if _partyId.Index != party.Params().PartyId().Index{
			Msg := network.Message{TaskName: "keygen_round2", FromID: party.Params().PartyId(), ToID: _partyId, MContent: &MRoundContent}
			party.Out <- &Msg
		}
	}
}