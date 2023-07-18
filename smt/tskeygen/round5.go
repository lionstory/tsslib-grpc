package tskeygen

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/zk"
)

type Round5Info struct {
	FromID  int
	Deltaix *big.Int
	Deltaiy *big.Int
	logp1   *zk.Logp
	logp2   *zk.Logp
	Deltai  *big.Int
}

func (p *Round5Info)MarshalString() (string, error){
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s", strconv.Itoa(p.FromID), hex.EncodeToString(p.Deltaix.Bytes()),
		hex.EncodeToString(p.Deltaiy.Bytes()), p.logp1.MarshalString(), p.logp2.MarshalString(),
		hex.EncodeToString(p.Deltai.Bytes()))
	return data, nil
}

func (p *Round5Info) UnshalString(data string) error{
	parts := strings.Split(data, "#")
	fromId, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	p.FromID = fromId
	ix, err := hex.DecodeString(parts[1])
	if err != nil {
		return err
	}
	p.Deltaix = new(big.Int).SetBytes(ix)
	iy, err := hex.DecodeString(parts[2])
	if err != nil {
		return err
	}
	p.Deltaiy = new(big.Int).SetBytes(iy)
	p.logp1 = &zk.Logp{}
	err = p.logp1.UnmarshalString(parts[3])
	if err != nil {
		return err
	}
	p.logp2 = &zk.Logp{}
	err = p.logp2.UnmarshalString(parts[4])
	if err != nil {
		return err
	}
	delta, err := hex.DecodeString(parts[5])
	if err != nil {
		return err
	}
	p.Deltai = new(big.Int).SetBytes(delta)
	return nil
}

func (p *Round5Info) DoSomething(party *network.Party){
	party.Mtx.Lock()
	flag := p.logp1.LogVerify(party.Hash, party.Params().EC(), party.Data.Yix[p.FromID], party.Data.Yiy[p.FromID])
	flag1 := p.logp2.LogVerify1(party.Hash, party.Params().EC(), p.Deltaix, p.Deltaiy, party.KeyGenTem.Gammax, party.KeyGenTem.Gammay)
	party.Mtx.Unlock()
	if flag != true {
		fmt.Println("error", p.FromID)
	}
	if flag1 != true {
		fmt.Println("error", p.FromID)
	}
	//计算Delta，验证Delta
	party.KeyGenTem.Delta = party.KeyGenTem.Delta.Add(party.KeyGenTem.Delta, p.Deltai)
	party.KeyGenTem.Delta = party.KeyGenTem.Delta.Mod(party.KeyGenTem.Delta, party.Params().EC().Params().N)
	party.KeyGenTem.Deltax, party.KeyGenTem.Deltay = party.Params().EC().Add(party.KeyGenTem.Deltax, party.KeyGenTem.Deltay, p.Deltaix, p.Deltaiy)
}

func Rounds5(party *network.Party){
	aibi := new(big.Int).Mul(party.KeyGenTem.Xi, party.KeyGenTem.Gammai)
	aibi = aibi.Mod(aibi, party.Params().EC().Params().N)
	party.KeyGenTem.Deltai = aibi

	for i := 0; i < party.Params().PartyCount()-1; i++ {
		val := <-party.Recv
		val.MContent.DoSomething(party)
	}

	Deltaix, Deltaiy := party.Params().EC().ScalarMult(party.KeyGenTem.Gammax, party.KeyGenTem.Gammay, party.KeyGenTem.Xi.Bytes())
	Deltaxx := new(big.Int).Set(Deltaix)
	Deltayy := new(big.Int).Set(Deltaiy)
	party.KeyGenTem.Deltaix, party.KeyGenTem.Deltaiy = Deltaxx, Deltayy

	party.Mtx.Lock()
	logp1 := zk.LogProve(party.Hash, party.Params().EC(), party.Data.Yix[party.Params().PartyId().Index], party.Data.Yiy[party.Params().PartyId().Index], party.Data.Y)
	logp2 := zk.LogProve1(party.Hash, party.Params().EC(), Deltaix, Deltaiy, party.KeyGenTem.Gammax, party.KeyGenTem.Gammay, party.KeyGenTem.Xi)
	party.Mtx.Unlock()
	//广播消息位置1
	Deltai := new(big.Int).Set(party.KeyGenTem.Deltai)

	MRoundContent := Round5Info{party.Params().PartyId().Index, Deltaix, Deltaiy, logp1, logp2, Deltai}
	//本地计算消息位置1，向每一个参与方广播相同消息的时候使用
	//Msg := smt.Message{FromID: party.Params().PartyId(), ToID: nil, MContent: &MRoundContent}

	for _, _partyId := range party.Params().Parties().IDs(){
		if _partyId.Index != party.Params().PartyId().Index{
			Msg := network.Message{TaskName: "keygen_round5", FromID: party.Params().PartyId(), ToID: _partyId, MContent: &MRoundContent}
			//Msg.ToID = _partyId
			party.Out <- &Msg
		}
	}
}
