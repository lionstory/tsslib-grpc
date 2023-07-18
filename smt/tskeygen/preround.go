package tskeygen

import (
	"encoding/hex"
	"fmt"
	"github.com/cronokirby/safenum"
	"github.com/taurusgroup/multi-party-sig/pkg/hash"
	"github.com/taurusgroup/multi-party-sig/pkg/pool"
	"strconv"
	"strings"
	"github.com/lionstory/tsslib-grpc/pkg/utils"

	"math/big"
	//mod "github.com/taurusgroup/multi-party-sig/pkg/zk/mod"
	//prm "github.com/taurusgroup/multi-party-sig/pkg/zk/prm"
	mod "github.com/lionstory/tsslib-grpc/smt/zk/mod"
	prm "github.com/lionstory/tsslib-grpc/smt/zk/prm"
	//"github.com/taurusgroup/multi-party-sig/pkg/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	//"github.com/taurusgroup/multi-party-sig/pkg/pedersen"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
	"github.com/lionstory/tsslib-grpc/smt/network"
)

type PreRoundInfo struct {
	FromID            int
	Rtigi             *big.Int
	PaillierPublickey *paillier.PublicKey
	Aux               *pedersen.Parameters
	PrmPubic          *prm.Public
	PrmProof          *prm.Proof
	ModPubic          *mod.Public
	ModProof          *mod.Proof
}

func (p *PreRoundInfo)MarshalString() (string, error){
	prmpublic, err := p.PrmPubic.MarshalString()
	if err != nil {
		return "", err
	}
	prmproof, err := utils.Encode(p.PrmProof)
	if err != nil {
		return "", err
	}
	modpublic, err := p.ModPubic.MarshalString()
	if err != nil {
		return "", err
	}
	modproof, err := utils.Encode(p.ModProof)
	if err != nil {
		return "", err
	}
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%s#%s", strconv.Itoa(p.FromID), hex.EncodeToString(p.Rtigi.Bytes()),
		hex.EncodeToString(p.PaillierPublickey.MarshalBytes()), p.Aux.MarshalString(), prmpublic,
		hex.EncodeToString(prmproof), modpublic, hex.EncodeToString(modproof))
	return data, nil
}

func (p *PreRoundInfo) UnshalString(data string) error{
	parts := strings.Split(data, "#")
	fromId, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	p.FromID = fromId
	rtigi, err := hex.DecodeString(parts[1])
	if err != nil {
		return err
	}
	p.Rtigi = new(big.Int).SetBytes(rtigi)
	pub, err := hex.DecodeString(parts[2])
	if err != nil {
		return err
	}
	paillier_pub, err := paillier.UnmarshalPublicKeyByByte(pub)
	if err != nil {
		return err
	}
	p.PaillierPublickey = paillier_pub
	p.Aux  = &pedersen.Parameters{}
	err = p.Aux.UnmarshalString(parts[3])
	if err != nil {
		return err
	}
	p.PrmPubic = &prm.Public{}
	p.PrmPubic.S = &safenum.Nat{}
	p.PrmPubic.T = &safenum.Nat{}
	err = p.PrmPubic.UnmarshalString(parts[4])
	if err != nil {
		return err
	}
	prmproof_byte, err := hex.DecodeString(parts[5])
	prmproof := &prm.Proof{}
	err = utils.Decode(prmproof_byte, prmproof)
	if err != nil {
		return err
	}
	p.PrmProof = prmproof

	p.ModPubic = &mod.Public{}
	p.ModPubic.N = &safenum.Modulus{}
	err = p.ModPubic.UnmarshalString(parts[6])
	if err != nil {
		return err
	}
	modprood_byte, err := hex.DecodeString(parts[7])
	modproof := &mod.Proof{}
	err = utils.Decode(modprood_byte, modproof)
	if err != nil {
		return err
	}
	p.ModProof = modproof
	return nil
}

func (p *PreRoundInfo) DoSomething(party *network.Party){
	pl := pool.NewPool(0)
	defer pl.TearDown()
	flag1 := p.PrmProof.Verify(*p.PrmPubic, hash.New(), pl)
	if flag1 != true {
		fmt.Println("the fails party is ", p.FromID)
		return
	}
	flag2 := p.ModProof.Verify(*p.ModPubic, hash.New(), pl)
	if flag2 != true {
		fmt.Println("the fails party is ", p.FromID)
		return
	}
	party.Data.Rtig.Add(party.Data.Rtig, p.Rtigi)
	party.Data.PaillierPks[p.FromID] = p.PaillierPublickey
	party.Data.Aux[p.FromID] = p.Aux
	party.Data.Ks[p.FromID] = p.Rtigi
}

func PreRound(party *network.Party){
	pl := pool.NewPool(0)
	defer pl.TearDown()
	//生成paillier公私钥
	PaillierSecertKey := paillier.NewSecretKey(pl)
	//生成pederson参数
	ped, lambda := PaillierSecertKey.GeneratePedersen()
	//生成prm证明
	public1 := prm.Public{
		N: ped.N.Modulus,
		S: ped.S,
		T: ped.T,
	}

	Prmproof := prm.NewProof(prm.Private{
		Lambda: lambda,
		Phi:    PaillierSecertKey.Phi,
		P:      PaillierSecertKey.P,
		Q:      PaillierSecertKey.Q,
	}, hash.New(), public1, pl)
	//生成mod证明
	public2 := mod.Public{N: PaillierSecertKey.PublicKey.N.Modulus}
	Modproof := mod.NewProof(hash.New(), mod.Private{
		P:   PaillierSecertKey.P,
		Q:   PaillierSecertKey.Q,
		Phi: PaillierSecertKey.Phi,
	}, public2, pl)

	// 保存信息
	aux := make([]*pedersen.Parameters, party.Params().PartyCount())
	aux[party.Params().PartyId().Index] = ped
	party.Data.Aux = aux
	paillierPks := make([]*paillier.PublicKey, party.Params().PartyCount())
	paillierPks[party.Params().PartyId().Index] = PaillierSecertKey.PublicKey
	party.Data.PaillierPks = paillierPks

	party.Data.PaillierSK = PaillierSecertKey

	party.Data.Lambda = lambda
	ks := make([]*big.Int, len(party.Params().Parties().IDs()))
	ks[party.Params().PartyId().Index] = party.Data.Rtigi
	party.Data.Ks = ks


	Round1Content := PreRoundInfo{party.Params().PartyId().Index, party.Data.Rtigi, PaillierSecertKey.PublicKey, ped, &public1, Prmproof, &public2, Modproof}

	//广播消息
	for _, _partyId := range party.Params().Parties().IDs(){
		if _partyId.Index != party.Params().PartyId().Index{
			Msg := network.Message{
				TaskName: "keygen_preround",
				FromID: party.Params().PartyId(),
				ToID: _partyId,
				MContent: &Round1Content,
			}
			//Msg.ToID = _partyId
			party.Out <- &Msg
		}
	}
}