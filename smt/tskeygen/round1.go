package tskeygen

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/modfiysm2"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/zk"
)

type Round1Info struct {
	FromID 	int
	Gammaix *big.Int
	Gammaiy *big.Int
	V       *big.Int
}

func (p *Round1Info)MarshalString() (string, error){
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

func (p *Round1Info) DoSomething(party *network.Party){
	party.KeyGenTem.V[p.FromID] = p.V
	party.KeyGenTem.Gammaix[p.FromID] = p.Gammaix
	party.KeyGenTem.Gammaiy[p.FromID] = p.Gammaiy
}


func Rounds1(party *network.Party){

	for i:=0; i < party.Params().PartyCount()-1; i++{
		val := <-party.Recv
		val.MContent.DoSomething(party)
	}

	xi, _ := modfiysm2.RandFieldElement(party.Params().EC(), nil)
	gammai, _ := modfiysm2.RandFieldElement(party.Params().EC(), nil)

	Xix, Xiy := party.Params().EC().ScalarBaseMult(xi.Bytes())
	Gammaix, Gammaiy := party.Params().EC().ScalarBaseMult(gammai.Bytes())

	//生成随机数rhoi，ui
	bf := make([]byte, 32)
	rand.Read(bf)
	rhoi := new(big.Int).SetBytes(bf)

	bf2 := make([]byte, 32)
	rand.Read(bf2)
	ui := new(big.Int).SetBytes(bf2)

	party.Mtx.Lock()
	party.Hash.Write(zk.BytesCombine(party.Data.Rtig.Bytes(), Xix.Bytes(), Xiy.Bytes(), Gammaix.Bytes(), Gammaiy.Bytes(), rhoi.Bytes(), ui.Bytes()))
	bytes := party.Hash.Sum(nil)
	// 计算hash承诺
	Vi := new(big.Int).SetBytes(bytes)
	party.Hash.Reset()
	party.Mtx.Unlock()

	party.KeyGenTem.Xi = xi
	party.KeyGenTem.Gammai = gammai
	party.KeyGenTem.Xix = Xix
	party.KeyGenTem.Xiy = Xiy
	party.KeyGenTem.Rhoi = rhoi
	party.Data.Rho = new(big.Int).SetBytes(bf)
	party.KeyGenTem.Ui = ui

	ax, ay := Xix, Xiy
	party.Data.Ax = new(big.Int).Set(ax)
	party.Data.Ay = new(big.Int).Set(ay)

	Vs := make([]*big.Int, party.Params().PartyCount())
	Vs[party.Params().PartyId().Index] = Vi
	party.KeyGenTem.V = Vs

	Gammaixs := make([]*big.Int, party.Params().PartyCount())
	Gammaixs[party.Params().PartyId().Index] = Gammaix
	party.KeyGenTem.Gammaix = Gammaixs

	Gammaiys := make([]*big.Int, party.Params().PartyCount())
	Gammaiys[party.Params().PartyId().Index] = Gammaiy
	party.KeyGenTem.Gammaiy = Gammaiys

	// 将hash值广播出去
	Round1Content := Round1Info{
		FromID: party.Params().PartyId().Index,
		V : Vi,
		Gammaix: Gammaix,
		Gammaiy: Gammaiy,
	}

	for _, _partyId := range party.Params().Parties().IDs(){
		if _partyId.Index != party.Params().PartyId().Index{
			Msg := network.Message{TaskName: "keygen_round1", FromID: party.Params().PartyId(), ToID: _partyId, MContent: &Round1Content}
			party.Out <- &Msg
		}
	}
}