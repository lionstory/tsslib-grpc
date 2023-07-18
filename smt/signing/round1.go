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
	"github.com/lionstory/tsslib-grpc/smt/vss"
	"github.com/lionstory/tsslib-grpc/smt/zk"
	"github.com/cronokirby/safenum"
)

type Round1Info struct {
	FromID int
	//MtA要发送的消息,Ax,Ay用于验证zk
	Ax *big.Int
	Ay *big.Int
	//Bx,By用于合成R
	Bx *big.Int
	By *big.Int
	//将Gi存储下来，用于后续的MtA
	Gi             *paillier.Ciphertext //ENC(wi)
	Round1logstarp *zk.Logstarp
}

func (p *Round1Info) MarshalString() (string, error) {
	logstrap, err := p.Round1logstarp.MarshalString()
	if err != nil {
		return "", err
	}
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%s", strconv.Itoa(p.FromID), hex.EncodeToString(p.Ax.Bytes()),
		hex.EncodeToString(p.Ay.Bytes()), hex.EncodeToString(p.Bx.Bytes()), hex.EncodeToString(p.By.Bytes()),
		p.Gi.MarshalString(), logstrap)
	return data, nil
}

func (p *Round1Info) UnshalString(data string) error {
	parts := strings.Split(data, "#")
	fromId, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	p.FromID = fromId
	ax, err := hex.DecodeString(parts[1])
	if err != nil {
		return err
	}
	p.Ax = new(big.Int).SetBytes(ax)
	ay, err := hex.DecodeString(parts[2])
	if err != nil {
		return err
	}
	p.Ay = new(big.Int).SetBytes(ay)
	bx, err := hex.DecodeString(parts[3])
	if err != nil {
		return err
	}
	p.Bx = new(big.Int).SetBytes(bx)
	by, err := hex.DecodeString(parts[4])
	if err != nil {
		return err
	}
	p.By = new(big.Int).SetBytes(by)
	p.Gi = &paillier.Ciphertext{}
	err = p.Gi.UnmarshalString(parts[5])
	if err != nil {
		return err
	}
	p.Round1logstarp = &zk.Logstarp{}
	p.Round1logstarp.S = &safenum.Nat{}
	p.Round1logstarp.A = &paillier.Ciphertext{}
	p.Round1logstarp.C = &safenum.Nat{}
	p.Round1logstarp.Yx = new(big.Int)
	p.Round1logstarp.Yy = new(big.Int)
	p.Round1logstarp.Z1 = &safenum.Int{}
	p.Round1logstarp.Z2 = &safenum.Nat{}
	p.Round1logstarp.Z3 = &safenum.Int{}
	err = p.Round1logstarp.UnmarshalString(parts[6])
	if err != nil {
		return err
	}
	return nil
}

func (p *Round1Info) DoSomething(party *network.Party) {
	//验证zklogstar
	party.Mtx.Lock()
	flag2 := p.Round1logstarp.LogstarVerify(party.Hash, party.Params().EC(), party.Data.Aux[p.FromID], party.Data.PaillierPks[p.FromID], p.Gi, p.Ax, p.Ay)
	party.Mtx.Unlock()

	if flag2 != true {
		fmt.Println("error", p.FromID)
	}
	//计算得到Rx，Ry
	party.Temp.Rx, party.Temp.Ry = party.Params().EC().Add(party.Temp.Rx, party.Temp.Ry, p.Bx, p.By)
	//将Gi缓存起来。
	party.Temp.MtAEncW[p.FromID] = p.Gi
}

func Rounds1(party *network.Party) {
	fmt.Println("=>Rounds1 Start time: ", time.Now().Format("2006-01-02 15:04:05"))
	//生成随机数k，和kG
	ki, _ := modfiysm2.RandFieldElement(party.Params().EC(), nil)
	party.Temp.Ki = ki

	Kix, Kiy := party.Params().EC().ScalarBaseMult(ki.Bytes())
	party.Temp.Kix, party.Temp.Kiy = Kix, Kiy

	//计算wi
	//lambda := vss.Lagrange(net, party.ID, party.T)
	lambda := vss.Lagrange1(party, party.Params().PartyId().Index)
	wi := new(big.Int).Mul(lambda, party.Data.Y)
	party.Temp.Wi = wi

	Wix, Wiy := party.Params().EC().ScalarBaseMult(wi.Bytes())
	party.Temp.Wix, party.Temp.Wiy = Wix, Wiy

	//是否需要存储这些数据

	//验证A=sum(lambdai*Yi)
	Wx := new(big.Int)
	Wy := new(big.Int)
	for _, mparty := range party.Params().Parties().IDs() {
		if mparty.Index != party.Params().PartyId().Index {
			lambda := vss.Lagrange1(party, mparty.Index)
			Wix, Wiy := party.Params().EC().ScalarMult(party.Data.Yix[mparty.Index], party.Data.Yiy[mparty.Index], lambda.Bytes())
			Wx, Wy = party.Params().EC().Add(Wx, Wy, Wix, Wiy)
		}
	}
	Wx, Wy = party.Params().EC().Add(Wx, Wy, Wix, Wiy)
	//	fmt.Println("前两个应该都一样，后面也应该都一样", party.ID, party.Ax, party.Ay, Wx, Wy)
	flag := party.Data.Ax.Cmp(Wx) == 0 && party.Data.Ay.Cmp(Wy) == 0
	if flag != true {
		fmt.Println("error,please run presigning checken", party.Params().PartyId().Index, Wx, party.Data.Ax, Wy, party.Data.Ay)
	}

	//接下来就是MtA的事情了。当时为什么没有写成函数，这样不是更好调用吗。
	//Wix,Wiy,Kix,Kiy,wi,ki。其中都是他们的椭圆曲线点
	//Ai,Bi,ai,bi
	//最后生成签名的随机点R，和chi=xiki的加共享
	//这里和上面的wix
	x := new(safenum.Int).SetBig(wi, wi.BitLen())
	ct, v := party.Data.PaillierPks[party.Params().PartyId().Index].Enc(x)
	//MtAEncW := make([]*paillier.Ciphertext, party.Params().PartyCount())
	MtAEncW := make(map[int]*paillier.Ciphertext)
	MtAEncW[party.Params().PartyId().Index] = ct
	party.Temp.MtAEncW = MtAEncW

	//生成zkencp
	party.Mtx.Lock()
	Round1logstarp := zk.LogstarProve(party.Hash, party.Params().EC(), party.Data.Aux[party.Params().PartyId().Index], party.Data.PaillierPks[party.Params().PartyId().Index], ct, Wix, Wiy, x, v)
	party.Mtx.Unlock()

	//将Ai,Bi,ct广播出去
	Round1Content := Round1Info{party.Params().PartyId().Index, Wix, Wiy, Kix, Kiy, ct, Round1logstarp}
	//Msg := smt.Message{FromID: party.Params().PartyId(), ToID: "", MContent: &Round1Content}

	//广播消息,不失去一般性，这里只考虑前T个参与方
	for _, _partyId := range party.Params().Parties().IDs() {
		if _partyId.Index != party.Params().PartyId().Index {
			Msg := network.Message{TaskName: "signing_round1", FromID: party.Params().PartyId(), ToID: _partyId, MContent: &Round1Content}
			party.Out <- &Msg
		}
	}

	//Round1结束
	fmt.Println("=>Rounds1 End time: ", time.Now().Format("2006-01-02 15:04:05"))

}
