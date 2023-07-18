package tskeygen

import (
	"encoding/hex"
	"fmt"
	"github.com/cronokirby/safenum"
	"math/big"
	"strconv"
	"strings"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/lionstory/tsslib-grpc/smt/vss"
	"github.com/lionstory/tsslib-grpc/smt/zk"
)

type Round3Info struct {
	FromID  int
	//VSS要发送的内容Cij和A
	VssEncyi   *paillier.Ciphertext
	VssAx      []*big.Int
	VssAy      []*big.Int
	Round3logp *zk.Logp

	//MtA要发送的消息
	Bx             *big.Int
	By             *big.Int
	Gi             *paillier.Ciphertext
	Round3logstarp *zk.Logstarp

	Xix            *big.Int
	Xiy            *big.Int
	Aux            *pedersen.Parameters
	PaillierPublickey *paillier.PublicKey
}

func (p *Round3Info)MarshalString() (string, error){
	vssax, err := utils.Encode(p.VssAx)
	if err != nil {
		return "", err
	}
	vssay, err := utils.Encode(p.VssAy)
	if err != nil {
		return "", err
	}
	logstrap, err := p.Round3logstarp.MarshalString()
	if err != nil {
		return "", err
	}
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%s#%s#%s#%s#%s#%s#%s", strconv.Itoa(p.FromID), p.VssEncyi.MarshalString(),
		hex.EncodeToString(vssax), hex.EncodeToString(vssay), p.Round3logp.MarshalString(), hex.EncodeToString(p.Bx.Bytes()),
		hex.EncodeToString(p.By.Bytes()), p.Gi.MarshalString(), logstrap, hex.EncodeToString(p.Xix.Bytes()),
		hex.EncodeToString(p.Xiy.Bytes()), p.Aux.MarshalString(), hex.EncodeToString(p.PaillierPublickey.MarshalBytes()))
	return data, nil
}

func (p *Round3Info) UnshalString(data string) error{
	parts := strings.Split(data, "#")
	fromId, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	p.FromID = fromId
	p.VssEncyi = &paillier.Ciphertext{}
	err = p.VssEncyi.UnmarshalString(parts[1])
	if err != nil {
		return err
	}
	vssax := []*big.Int{}
	ax, err := hex.DecodeString(parts[2])
	if err != nil {
		return err
	}
	err = utils.Decode(ax, &vssax)
	if err != nil {
		return err
	}
	p.VssAx = vssax
	vssay := []*big.Int{}
	ay, err := hex.DecodeString(parts[3])
	if err != nil {
		return err
	}
	err = utils.Decode(ay, &vssay)
	if err != nil {
		return err
	}
	p.VssAy = vssay

	p.Round3logp = &zk.Logp{}
	err = p.Round3logp.UnmarshalString(parts[4])
	if err != nil {
		return err
	}
	bx, err := hex.DecodeString(parts[5])
	if err != nil {
		return err
	}
	p.Bx = new(big.Int).SetBytes(bx)
	by, err := hex.DecodeString(parts[6])
	if err != nil {
		return err
	}
	p.By = new(big.Int).SetBytes(by)

	p.Gi = &paillier.Ciphertext{}
	err = p.Gi.UnmarshalString(parts[7])
	if err != nil {
		return err
	}

	p.Round3logstarp = &zk.Logstarp{}
	p.Round3logstarp.S = &safenum.Nat{}
	p.Round3logstarp.A = &paillier.Ciphertext{}
	p.Round3logstarp.C = &safenum.Nat{}
	p.Round3logstarp.Z1 = &safenum.Int{}
	p.Round3logstarp.Z2 = &safenum.Nat{}
	p.Round3logstarp.Z3 = &safenum.Int{}
	p.Round3logstarp.Yx = new(big.Int)
	p.Round3logstarp.Yy = new(big.Int)
	err = p.Round3logstarp.UnmarshalString(parts[8])
	if err != nil {
		return err
	}
	xix, err := hex.DecodeString(parts[9])
	if err != nil {
		return err
	}
	p.Xix = new(big.Int).SetBytes(xix)
	xiy, err := hex.DecodeString(parts[10])
	if err != nil {
		return err
	}
	p.Xiy = new(big.Int).SetBytes(xiy)

	p.Aux = &pedersen.Parameters{}
	err = p.Aux.UnmarshalString(parts[11])
	if err != nil {
		return err
	}
	pub, err := hex.DecodeString(parts[12])
	if err != nil {
		return err
	}
	puk, err := paillier.UnmarshalPublicKeyByByte(pub)
	if err != nil {
		return err
	}
	p.PaillierPublickey = puk
	return nil
}

func (p *Round3Info) DoSomething(party *network.Party){
	party.Mtx.Lock()
	flag := p.Round3logp.LogVerify(party.Hash, party.Params().EC(), p.Xix, p.Xiy)
	party.Mtx.Unlock()
	//	fmt.Println(flag)
	if flag != true {
		fmt.Println("LogVerify error", p.FromID)
	}
	plaintxt, _ := party.Data.PaillierSK.Dec(p.VssEncyi)
	yij := plaintxt.Big()
	//vssverify
	vss.VssVerify1(p.FromID, yij, p.VssAx, p.VssAy, party, p.Xix, p.Xiy)
	//计算yi
	party.Data.Y.Add(party.Data.Y, yij)

	// 验证logstarp
	party.Mtx.Lock()
	flag2 := p.Round3logstarp.LogstarVerify(party.Hash, party.Params().EC(), p.Aux, p.PaillierPublickey, p.Gi, p.Xix, p.Xiy)
	party.Mtx.Unlock()
	if flag2 != true {
		fmt.Println("flag2 error", p.FromID)
	}

	party.KeyGenTem.MtAEncB[p.FromID] = p.Gi
	party.Data.Ax, party.Data.Ay = party.Params().EC().Add(party.Data.Ax, party.Data.Ay, p.Xix, p.Xiy)
}

func Rounds3(party *network.Party){

	for i := 0; i < party.Params().PartyCount()-1; i++ {
		val := <-party.Recv
		val.MContent.DoSomething(party)
	}

	// 进行分片
	vss.Vssshare1(party)

	party.Mtx.Lock()
	Round3logp := zk.LogProve(party.Hash, party.Params().EC(), party.KeyGenTem.Xix, party.KeyGenTem.Xiy, party.KeyGenTem.Xi)
	party.Mtx.Unlock()

	// 运行MtA协议
	x := new(safenum.Int).SetBig(party.KeyGenTem.Xi, party.KeyGenTem.Xi.BitLen())
	ct, v := party.Data.PaillierSK.PublicKey.Enc(x)
	MtAEncB := make(map[int]*paillier.Ciphertext)
	MtAEncB[party.Params().PartyId().Index] = ct
	party.KeyGenTem.MtAEncB = MtAEncB

	party.Mtx.Lock()
	Round3logstarp := zk.LogstarProve(party.Hash, party.Params().EC(), party.Data.Aux[party.Params().PartyId().Index], party.Data.PaillierSK.PublicKey, ct, party.KeyGenTem.Xix, party.KeyGenTem.Xiy, x, v)
	party.Mtx.Unlock()

	for _, _partyId := range party.Params().Parties().IDs(){
		if _partyId.Index != party.Params().PartyId().Index{
			MRoundContent := Round3Info{
				FromID: party.Params().PartyId().Index,
				VssEncyi: party.KeyGenTem.VssEncy[_partyId.Index],
				VssAx: party.KeyGenTem.VssAx,
				VssAy: party.KeyGenTem.VssAy,
				Round3logp: Round3logp,
				Bx: party.KeyGenTem.Gammaix[party.Params().PartyId().Index],
				By: party.KeyGenTem.Gammaiy[party.Params().PartyId().Index],
				Gi: ct,
				Round3logstarp: Round3logstarp,
				Xix: party.KeyGenTem.Xix,
				Xiy: party.KeyGenTem.Xiy,
				PaillierPublickey: party.Data.PaillierSK.PublicKey,
				Aux: party.Data.Aux[party.Params().PartyId().Index],
			}
			//本地计算消息位置1，向每一个参与方广播相同消息的时候使用
			Msg := network.Message{TaskName: "keygen_round3", FromID: party.Params().PartyId(), ToID: nil, MContent: &MRoundContent}

			//这里也是单独的情况下
			Msg.ToID = _partyId
			party.Out <- &Msg
		}
	}
}

