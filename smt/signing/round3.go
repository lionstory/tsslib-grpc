package signing

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lianghuiqiang9/smt/zk"
)

type Round3Info struct {
	FromID int
	S      *big.Int
}

func (p *Round3Info) MarshalString() (string, error) {
	data, err := utils.Encode(p)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func (p *Round3Info) UnshalString(data string) error {
	content, err := hex.DecodeString(data)
	if err != nil {
		return err
	}
	err = utils.Decode(content, p)
	if err != nil {
		return err
	}
	return nil
}

func (p *Round3Info) DoSomething(party *network.Party) {
	party.SignData.S.Add(party.SignData.S, p.S)
}

func Rounds3(party *network.Party) {
	fmt.Println("=>Rounds3 Start time: ", time.Now().Format("2006-01-02 15:04:05"))

	aibi := new(big.Int).Mul(party.Temp.Ki, party.Temp.Wi)

	aibi = aibi.Mod(aibi, party.Params().EC().Params().N)
	party.Temp.Chi = aibi

	for i := 0; i < len(party.Params().Parties().IDs())-1; i++ {
		val := <-party.Recv // 出 chan
		val.MContent.DoSomething(party)
		//本地计算消息
	}

	//设置签名消息
	msg := party.Msg
	party.Mtx.Lock()
	//计算Z
	party.Hash.Write(zk.BytesCombine(party.Data.Rtig.Bytes(), party.Data.Rho.Bytes(), party.Data.Xx.Bytes(), party.Data.Xy.Bytes()))
	bytes := party.Hash.Sum(nil)
	//将hash映射到椭圆曲线阶上。
	Z := new(big.Int).SetBytes(bytes)
	party.Hash.Reset()

	//计算e
	party.Hash.Write(zk.BytesCombine(Z.Bytes(), msg))
	bytes2 := party.Hash.Sum(nil)
	//将hash映射到椭圆曲线阶上。
	e := new(big.Int).SetBytes(bytes2)

	party.Hash.Reset()
	party.Mtx.Unlock()

	//计算r
	e.Add(e, party.Temp.Rx)
	r := new(big.Int).Mod(e, party.Params().EC().Params().N)

	party.SignData.R = r

	//计算s
	s := new(big.Int).Mul(party.Temp.Wi, r)
	s.Mod(s, party.Params().EC().Params().N)
	s.Add(s, new(big.Int).Set(party.Temp.Chi))
	s.Mod(s, party.Params().EC().Params().N)
	party.Temp.S = s

	Round1Content := Round3Info{party.Params().PartyId().Index, s}
	//Msg := smt.Message{FromID: party.ID, ToID: "", MContent: &Round1Content}

	//广播消息,不失去一般性，这里只考虑前T个参与方
	for _, _partyId := range party.Params().Parties().IDs() {
		if _partyId.Index != party.Params().PartyId().Index {
			Msg := network.Message{TaskName: "signing_round3", FromID: party.Params().PartyId(), ToID: _partyId, MContent: &Round1Content}
			//Msg.ToID = _partyId
			party.Out <- &Msg
		}
	}

	fmt.Println("=>Rounds3 End time: ", time.Now().Format("2006-01-02 15:04:05"))

}
