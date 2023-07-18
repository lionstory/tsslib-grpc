package tskeygen

import (
	"encoding/hex"
	"fmt"
	"github.com/cronokirby/safenum"
	"math/big"
	"strconv"
	"strings"

	//"github.com/taurusgroup/multi-party-sig/pkg/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/modfiysm2"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/zk"
)

type Round4Info struct {
	FromID   int
	Eji      *paillier.Ciphertext
	Dji      *paillier.Ciphertext
	Encstarp *zk.Proof
	Yix      *big.Int
	Yiy      *big.Int
}

func (p *Round4Info)MarshalString() (string, error){
	encstarp, err := p.Encstarp.MarshalString()
	if err != nil {
		return "", err
	}
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s",strconv.Itoa(p.FromID), p.Eji.MarshalString(), p.Dji.MarshalString(),
		encstarp, hex.EncodeToString(p.Yix.Bytes()), hex.EncodeToString(p.Yiy.Bytes()))
	return data, nil
}

func (p *Round4Info) UnshalString(data string) error{
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
	yix, err := hex.DecodeString(parts[4])
	if err != nil {
		return err
	}
	p.Yix = new(big.Int).SetBytes(yix)
	yiy, err := hex.DecodeString(parts[5])
	if err != nil {
		return err
	}
	p.Yiy = new(big.Int).SetBytes(yiy)
	return nil
}


func (p *Round4Info) DoSomething(party *network.Party) {
	public := zk.Public{
		Kv: party.KeyGenTem.MtAEncB[party.Params().PartyId().Index],
		Dv: p.Eji,
		Fp: p.Dji,
		Xx: party.KeyGenTem.Gammaix[p.FromID],
		Xy: party.KeyGenTem.Gammaiy[p.FromID],
		Prover: party.Data.PaillierPks[p.FromID],
		Verifier: party.Data.PaillierSK.PublicKey,
		Aux: party.Data.Aux[party.Params().PartyId().Index],
	}
	party.Mtx.Lock()
	flag := p.Encstarp.EncstarVerify(party.Hash, public)
	party.Mtx.Unlock()
	if flag != true {
		fmt.Println("EncstarVerify error", p.FromID)
	}

	//解密Eij
	alphaij, _ := party.Data.PaillierSK.Dec(p.Eji)

	//计算detai
	alphabeta := alphaij.Abs().Big()
	alphabeta = alphabeta.Add(alphabeta, party.KeyGenTem.Beta[p.FromID])
	party.KeyGenTem.Deltai = party.KeyGenTem.Deltai.Add(party.KeyGenTem.Deltai, alphabeta)
	party.KeyGenTem.Deltai = party.KeyGenTem.Deltai.Mod(party.KeyGenTem.Deltai, party.Params().EC().Params().N)

	party.Data.Yix[p.FromID] = p.Yix
	party.Data.Yiy[p.FromID] = p.Yiy
}

func Rounds4(party *network.Party) {
	Y := new(big.Int).Set(party.KeyGenTem.Vssy[party.Params().PartyId().Index])
	party.Data.Y = Y

	for i := 0; i < party.Params().PartyCount()-1; i++ {
		val := <-party.Recv
		val.MContent.DoSomething(party)
	}

	GammaX := new(big.Int)
	GammaY := new(big.Int)
	for key := 0; key < party.Params().PartyCount(); key ++{
		GammaX, GammaY = party.Params().EC().Add(GammaX, GammaY, party.KeyGenTem.Gammaix[key], party.KeyGenTem.Gammaiy[key])
	}
	party.KeyGenTem.Gammax, party.KeyGenTem.Gammay = GammaX, GammaY

	yix, yiy := party.Params().EC().ScalarBaseMult(party.Data.Y.Bytes())
	Yix := make([]*big.Int, party.Params().PartyCount())
	Yiy := make([]*big.Int, party.Params().PartyCount())
	Yix[party.Params().PartyId().Index] = yix
	Yiy[party.Params().PartyId().Index] = yiy
	party.Data.Yix, party.Data.Yiy = Yix, Yiy

	Beta := make(map[int]*big.Int)
	party.KeyGenTem.Beta = Beta

	for _, mparty := range party.Params().Parties().IDs() {
		if mparty.Index != party.Params().PartyId().Index {

			//随机Beta，然后加密
			Betaj, _ := modfiysm2.RandFieldElement(party.Params().EC(), nil)
			Betajneg := new(big.Int).Neg(Betaj)
			Betajnegsafe := new(safenum.Int).SetBig(Betajneg, Betajneg.BitLen())
			EBetajnegsafe, fij := party.Data.PaillierPks[mparty.Index].Enc(Betajnegsafe)

			//Beta应该存储到SecretInfo中。
			party.KeyGenTem.Beta[mparty.Index] = Betaj

			//点乘Gammai和Gj
			Gj := party.KeyGenTem.MtAEncB[mparty.Index]
			Eji := (*paillier.Ciphertext).Clone(Gj)
			Gammaisafe := new(safenum.Int).SetBig(party.KeyGenTem.Gammai, party.KeyGenTem.Gammai.BitLen())
			Eji = Eji.Mul(party.Data.PaillierPks[mparty.Index], Gammaisafe)
			//加法，然后Eji计算完毕
			Eji = Eji.Add(party.Data.PaillierPks[mparty.Index], EBetajnegsafe)
			//计算Dji
			Dji, gij := party.Data.PaillierSK.PublicKey.Enc(Betajnegsafe)
			//计算EncstarP

			public := zk.Public{
				Kv:       Gj,
				Dv:       Eji,
				Fp:       Dji,
				Xx:       party.KeyGenTem.Gammaix[party.Params().PartyId().Index],
				Xy:       party.KeyGenTem.Gammaiy[party.Params().PartyId().Index],
				Prover:   party.Data.PaillierPks[party.Params().PartyId().Index],
				Verifier: party.Data.PaillierPks[mparty.Index],
				Aux:      party.Data.Aux[mparty.Index],
			}
			private := zk.Private{
				X: Gammaisafe,
				Y: Betajnegsafe,
				S: fij,
				R: gij,
			}
			party.Mtx.Lock()
			proof := zk.EncstarProof(party.Hash, party.Params().EC(), public, private)
			party.Mtx.Unlock()
			MRoundContent := Round4Info{
				FromID: party.Params().PartyId().Index,
				Eji: Eji,
				Dji: Dji,
				Encstarp: proof,
				Yix: party.Data.Yix[party.Params().PartyId().Index],
				Yiy: party.Data.Yiy[party.Params().PartyId().Index],
			}

			//本地计算消息位置1，向每一个参与方广播相同消息的时候使用
			Msg := network.Message{TaskName: "keygen_round4", FromID: party.Params().PartyId(), ToID: nil, MContent: &MRoundContent}

			//这里也是单独的情况下

			Msg.ToID = mparty
			party.Out <- &Msg
		}

	}
}