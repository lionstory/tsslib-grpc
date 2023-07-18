package resharing

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/cronokirby/safenum"
	"github.com/taurusgroup/multi-party-sig/pkg/hash"
	"github.com/taurusgroup/multi-party-sig/pkg/pool"
	"math/big"
	"strconv"
	"strings"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
	"github.com/lionstory/tsslib-grpc/smt/network"
	mod "github.com/lionstory/tsslib-grpc/smt/zk/mod"
	prm "github.com/lionstory/tsslib-grpc/smt/zk/prm"
)

type Round2Info struct {
	FromID            int
	Rtigi             *big.Int
	PaillierPublickey *paillier.PublicKey
	Aux               *pedersen.Parameters
	PrmPubic          *prm.Public
	PrmProof          *prm.Proof
	ModPubic          *mod.Public
	ModProof          *mod.Proof
	Rhoi              *big.Int
}

func (p *Round2Info)MarshalString() (string, error){
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
	data := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%s#%s#%s", strconv.Itoa(p.FromID), hex.EncodeToString(p.Rtigi.Bytes()),
		hex.EncodeToString(p.PaillierPublickey.MarshalBytes()), p.Aux.MarshalString(), prmpublic,
		hex.EncodeToString(prmproof), modpublic, hex.EncodeToString(modproof), hex.EncodeToString(p.Rhoi.Bytes()))
	return data, nil
}

func (p *Round2Info) UnshalString(data string) error{
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
	rhoi, err := hex.DecodeString(parts[8])
	if err != nil {
		return err
	}
	p.Rhoi = new(big.Int).SetBytes(rhoi)
	return nil
}

func (p *Round2Info) DoSomething(party *network.ReSharingParty){
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
	party.Save.Rtig.Add(party.Save.Rtig, p.Rtigi)
	//party.Save.Rho.Add(party.Save.Rho, p.Rhoi)
	party.Temp.Rhoi[p.FromID] = p.Rhoi
	party.Save.Aux[p.FromID] = p.Aux
	party.Save.PaillierPks[p.FromID] = p.PaillierPublickey
}

func Round2(party *network.ReSharingParty){
	pl := pool.NewPool(0)
	defer pl.TearDown()

	if !party.Params().IsNewCommittee(){
		return
	}

	count := len(party.Params().Parties().IDs()) - 1
	if !party.Params().IsOldCommittee(){
		count = len(party.Params().Parties().IDs())
	}
	for i := 0; i < count; i++ {
		val := <-party.Recv
		if val.TaskName != "reshare_Round1" {
			fmt.Println("message taskName error")
		}
		p := val.MContent.(*Round1Info)
		party.Temp.DgRound1Message[p.FromId] = *val

		if party.Params().PartyId().Index != val.ToID.Index{
			fmt.Println("-----+++++error+++++++++",party.Params().PartyId().Index ,val.TaskName, val.ToID.Index)
		}


	}

	var paillierSK  *paillier.SecretKey
	var aux     *pedersen.Parameters
	var lambda	*safenum.Nat
	if party.Input.ValidateWithProof(){
		paillierSK = party.Input.PaillierSK
		lambda = party.Input.Lambda
		aux    = party.Input.Aux[party.Params().PartyId().Index]
	}else{
		//生成paillier公私钥
		PaillierSecertKey := paillier.NewSecretKey(pl)
		//生成pederson参数
		ped, _lambda := PaillierSecertKey.GeneratePedersen()
		aux = ped
		lambda = _lambda
		paillierSK = PaillierSecertKey
	}
	party.Save.PaillierSK = paillierSK
	party.Save.Lambda = lambda
	party.Save.Aux[party.Params().PartyId().Index] = aux
	party.Save.PaillierPks[party.Params().PartyId().Index] = paillierSK.PublicKey

	//生成prm证明
	public1 := prm.Public{
		N: aux.N.Modulus,
		S: aux.S,
		T: aux.T,
	}
	Prmproof := prm.NewProof(prm.Private{
		Lambda: lambda,
		Phi:    paillierSK.Phi,
		P:      paillierSK.P,
		Q:      paillierSK.Q,
	}, hash.New(), public1, pl)
	//生成mod证明
	public2 := mod.Public{N: paillierSK.PublicKey.N.Modulus}
	Modproof := mod.NewProof(hash.New(), mod.Private{
		P:   paillierSK.P,
		Q:   paillierSK.Q,
		Phi: paillierSK.Phi,
	}, public2, pl)

	//生成随机数rhoi，ui
	bf := make([]byte, 32)
	rand.Read(bf)
	rhoi := new(big.Int).SetBytes(bf)

	party.Save.Rtigi = party.Params().PartyId().KeyInt()
	party.Save.Rtig  = party.Params().PartyId().KeyInt()
	party.Save.Rho   = rhoi
	party.Temp.Rhoi[party.Params().PartyId().Index] = rhoi
	// 更新Ks
	for _, partyId := range party.Params().NewParties().IDs() {
		party.Save.Ks[partyId.Index] = partyId.KeyInt()
	}

	// 将这些信息发送给其余的newparty
	for _, _partyId := range party.Params().NewParties().IDs() {
		Round2Content := Round2Info {
			FromID: party.Params().PartyId().Index,
			Rtigi:  party.Save.Rtigi,
			PaillierPublickey: paillierSK.PublicKey,
			Aux: aux,
			Rhoi: rhoi,
			PrmPubic: &public1,
			PrmProof: Prmproof,
			ModPubic: &public2,
			ModProof: Modproof,
		}
		if _partyId.Index != party.Params().PartyId().Index {
			Msg := network.MessageResharing{
				TaskName: "reshare_Round2",
				FromID: party.Params().PartyId(),
				ToID: _partyId,
				MContent: &Round2Content,
			}
			party.Out <- &Msg
		}else {
			party.Temp.DgRound2Message[party.Params().PartyId().Index] = network.MessageResharing{
				TaskName: "reshare_Round2",
				FromID: party.Params().PartyId(),
				ToID: _partyId,
				MContent: &Round2Content,
			}
		}
	}
}