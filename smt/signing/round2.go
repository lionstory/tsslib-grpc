package signing

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/modfiysm2"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/zk"
	"github.com/cronokirby/safenum"
)

type Round2Info struct {
	FromID   int
	Eji      *paillier.Ciphertext
	Dji      *paillier.Ciphertext
	Encstarp *zk.Proof
	Rix      *big.Int
	Riy      *big.Int
}

func (p *Round2Info) MarshalString() (string, error) {
	encstrap, err := p.Encstarp.MarshalString()
	if err != nil {
		return "", err
	}
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s", strconv.Itoa(p.FromID), p.Eji.MarshalString(),
		p.Dji.MarshalString(), encstrap, hex.EncodeToString(p.Rix.Bytes()), hex.EncodeToString(p.Riy.Bytes()))
	return data, nil
}

func (p *Round2Info) UnshalString(data string) error {
	parts := strings.Split(data, "#")
	fromId, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	p.FromID = fromId
	p.Eji = &paillier.Ciphertext{}
	err = p.Eji.UnmarshalString(parts[1])
	if err != nil {
		return err
	}
	p.Dji = &paillier.Ciphertext{}
	err = p.Dji.UnmarshalString(parts[2])
	if err != nil {
		return err
	}
	p.Encstarp = &zk.Proof{}
	err = p.Encstarp.UnmarshalString(parts[3])
	if err != nil {
		return err
	}
	rix, err := hex.DecodeString(parts[4])
	if err != nil {
		return err
	}
	p.Rix = new(big.Int).SetBytes(rix)
	riy, err := hex.DecodeString(parts[5])
	if err != nil {
		return err
	}
	p.Riy = new(big.Int).SetBytes(riy)
	return nil
}

func (p *Round2Info) DoSomething(party *network.Party) {

	public := zk.Public{
		Kv:       party.Temp.MtAEncW[party.Params().PartyId().Index],
		Dv:       p.Eji,
		Fp:       p.Dji,
		Xx:       p.Rix,
		Xy:       p.Riy,
		Prover:   party.Data.PaillierPks[p.FromID],
		Verifier: party.Data.PaillierSK.PublicKey,
		Aux:      party.Data.Aux[party.Params().PartyId().Index],
	}
	party.Mtx.Lock()
	flag := p.Encstarp.EncstarVerify(party.Hash, public)
	party.Mtx.Unlock()
	if flag != true {
		fmt.Println("Sign EncstarVerify error", p.FromID)
	}

	//解密Eij
	alphaij, _ := party.Data.PaillierSK.Dec(p.Eji)

	//计算detai
	alphabeta := alphaij.Abs().Big()
	alphabeta = alphabeta.Add(alphabeta, party.Temp.Beta2[p.FromID])
	party.Temp.Chi = party.Temp.Chi.Add(party.Temp.Chi, alphabeta)
	party.Temp.Chi = party.Temp.Chi.Mod(party.Temp.Chi, party.Params().EC().Params().N)

}

func Rounds2(party *network.Party) {
	fmt.Println("=>Rounds2 Start time: ", time.Now().Format("2006-01-02 15:04:05"))

	//这里是用来计算R
	Rx := new(big.Int).Set(party.Temp.Kix)
	Ry := new(big.Int).Set(party.Temp.Kiy)

	party.Temp.Rx, party.Temp.Ry = Rx, Ry

	//多存了Rix，有什么用处呢？？
	Rix := new(big.Int).Set(party.Temp.Kix)
	Riy := new(big.Int).Set(party.Temp.Kiy)

	party.Temp.Rix, party.Temp.Riy = Rix, Riy

	//注意这里呀，T+1个参与方
	for i := 0; i < len(party.Params().Parties().IDs())-1; i++ {
		fmt.Println("##########################", i, len(party.Params().Parties().IDs())-1)
		val := <-party.Recv // 出 chan
		val.MContent.DoSomething(party)
	}

	//make一个链接。存储Beta
	Beta2 := make(map[int]*big.Int)
	party.Temp.Beta2 = Beta2

	for _, _partyId := range party.Params().Parties().IDs() {
		if _partyId.Index != party.Params().PartyId().Index {
			//随机Beta，然后加密
			Betaj, _ := modfiysm2.RandFieldElement(party.Params().EC(), nil)
			Betajneg := new(big.Int).Neg(Betaj)
			Betajnegsafe := new(safenum.Int).SetBig(Betajneg, Betajneg.BitLen())
			EBetajnegsafe, fij := party.Data.PaillierPks[_partyId.Index].Enc(Betajnegsafe)
			//Beta2应该存储到SecretInfo中。
			party.Temp.Beta2[_partyId.Index] = Betaj

			//Wix,Wiy,Kix,Kiy,wi,ki。其中都是他们的椭圆曲线点
			//Ai,Bi,ai,bi

			//点乘Gammai和Gj，换了MtAEncB换成MtAEncW，和Gammai换成Ki
			Gj := party.Temp.MtAEncW[_partyId.Index]
			Eji := (*paillier.Ciphertext).Clone(Gj)
			Kisafe := new(safenum.Int).SetBig(party.Temp.Ki, party.Temp.Ki.BitLen())
			Eji = Eji.Mul(party.Data.PaillierPks[_partyId.Index], Kisafe)
			//加法，然后Eji计算完毕
			Eji = Eji.Add(party.Data.PaillierPks[_partyId.Index], EBetajnegsafe)
			//计算Dji
			Dji, gij := party.Data.PaillierSK.PublicKey.Enc(Betajnegsafe)
			//计算EncstarP,Gammaix,Gammaiy换成Rix,Riy
			public := zk.Public{
				Kv:       Gj,
				Dv:       Eji,
				Fp:       Dji,
				Xx:       new(big.Int).Set(party.Temp.Rix), //ki的kiG
				Xy:       new(big.Int).Set(party.Temp.Riy),
				Prover:   party.Data.PaillierSK.PublicKey,
				Verifier: party.Data.PaillierPks[_partyId.Index],
				Aux:      party.Data.Aux[_partyId.Index],
			}
			private := zk.Private{
				X: Kisafe,
				Y: Betajnegsafe,
				S: fij,
				R: gij,
			}
			party.Mtx.Lock()
			proof := zk.EncstarProof(party.Hash, party.Params().EC(), public, private)
			party.Mtx.Unlock()

			MRoundContent := Round2Info{
				FromID:   party.Params().PartyId().Index,
				Eji:      Eji,
				Dji:      Dji,
				Encstarp: proof,
				Rix:      party.Temp.Rix,
				Riy:      party.Temp.Riy,
			}
			//本地计算消息位置1，向每一个参与方广播相同消息的时候使用
			Msg := network.Message{TaskName: "signing_round2", FromID: party.Params().PartyId(), ToID: _partyId, MContent: &MRoundContent}

			party.Out <- &Msg
		}
	}

	fmt.Println("=>Rounds2 End time: ", time.Now().Format("2006-01-02 15:04:05"))

}
