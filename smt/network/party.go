package network

import (
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/cronokirby/safenum"
	"hash"
	"math/big"
	"sync"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
)

type Parameters struct {
	ec             			elliptic.Curve
	partyID  				*tss.PartyID
	parties  				*tss.PeerContext
	partyCount              int
	threshold               int
}

func NewParameters(curve elliptic.Curve,ctx *tss.PeerContext, partId *tss.PartyID, partyCount , threshold int)*Parameters{
	return &Parameters{
		ec: curve,
		parties: ctx,
		partyID: partId,
		partyCount: partyCount,
		threshold: threshold,
	}
}

func (params *Parameters) EC() elliptic.Curve{
	return params.ec
}

func (p *Parameters) PartyId() *tss.PartyID{
	return p.partyID
}

func (p *Parameters) Parties() *tss.PeerContext{
	return p.parties
}

func (p *Parameters) PartyCount() int {
	return p.partyCount
}

func (p *Parameters) SetPartyCount(count int){
	p.partyCount = count
}

func (p *Parameters) Threshold() int {
	return p.threshold
}

type Party struct {
	params      *Parameters
	Data        LocalSaveData
	Temp        TempSignData
	KeyGenTem   TempKeyGenData
	SignData    SignatureData
	Out 		chan *Message
	End         chan LocalSaveData
	SignEnd     chan SignatureData
	Recv 		chan *Message
	Hash 		hash.Hash
	Mtx  		sync.Mutex
	Msg  		[]byte
	HandleErr 	error
}

func Newparty(params *Parameters, out, recv chan *Message, end chan LocalSaveData) *Party{
	party := Party{
		params: params,
		Out: out,
		Recv: recv,
		End: end,
		Hash: sha256.New(),
	}
	shareId := params.partyID.KeyInt()
	party.Data.Rtigi = params.partyID.KeyInt()
	party.Data.Rtig = new(big.Int).Set(shareId)
	//party.Data.Index = params.PartyId().Index
	return &party
}

func NewsignParty(params *Parameters, out, recv chan *Message, end chan SignatureData) *Party{
	party := Party{
		params: params,
		Out: out,
		Recv: recv,
		SignEnd: end,
		Hash: sha256.New(),
	}
	return &party
}

func (party *Party)Params() *Parameters{
	return party.params
}

type LocalSaveData struct {
	PaillierSK 		*paillier.SecretKey
	PaillierPks     []*paillier.PublicKey
	Aux             []*pedersen.Parameters
	Rtigi           *big.Int
	Ks              []*big.Int
	Rtig            *big.Int
	Lambda			*safenum.Nat
	//Index           int
	Y               *big.Int
	Rho             *big.Int
	Xx              *big.Int
	Xy              *big.Int
	Yix				[]*big.Int
	Yiy             []*big.Int
	Ax              *big.Int
	Ay              *big.Int
}

func (preParam LocalSaveData) ValidateWithProof() bool{
	return preParam.PaillierSK != nil && len(preParam.Aux) > 0 &&
		len(preParam.PaillierPks) > 0 && preParam.Xx != nil &&
		preParam.Xy != nil && preParam.Ax != nil && preParam.Ay != nil
}

//用于json序列化中间过渡
type SaveData struct {
	PaillierSK 		string
	PaillierPks     []string
	Aux             []string
	Lambda          string
	Rtigi           *big.Int
	Ks              []*big.Int
	Rtig            *big.Int
	//Index           int
	Y               *big.Int
	Rho             *big.Int
	Xx              *big.Int
	Xy              *big.Int
	Yix				[]*big.Int
	Yiy             []*big.Int
	Ax              *big.Int
	Ay              *big.Int
}

func (l LocalSaveData) Marshal() SaveData{
	paillierSk := l.PaillierSK.MarshalBytes()
	save := SaveData{
		PaillierSK: paillierSk,
		Rtig: l.Rtig,
		Rtigi: l.Rtigi,
		Ks: l.Ks,
		//Index: l.Index,
		Rho: l.Rho,
		Y: l.Y,
		Xx: l.Xx,
		Xy: l.Xy,
		Yix: l.Yix,
		Yiy: l.Yiy,
		Ax: l.Ax,
		Ay: l.Ay,
	}
	paillierPks := make([]string, len(l.PaillierPks))
	for i := 0; i < len(l.PaillierPks); i++{
		publicKey := l.PaillierPks[i]
		paillierPks[i] = string(publicKey.MarshalBytes())
	}
	save.PaillierPks = paillierPks
	aux := make([]string, len(l.Aux))
	for j := 0; j < len(l.Aux); j++{
		aux[j] = l.Aux[j].MarshalString()
	}
	save.Aux = aux
	save.Lambda = hex.EncodeToString(l.Lambda.Bytes())
	return save
}

func (save SaveData) Unmarshal() (*LocalSaveData, error){
	local := &LocalSaveData{
		Rtig: 	save.Rtig,
		Rtigi:  save.Rtigi,
		//Index:  save.Index,
		Ks:     save.Ks,
		Rho: 	save.Rho,
		Y: 		save.Y,
		Xx: 	save.Xx,
		Xy: 	save.Xy,
		Yix: 	save.Yix,
		Yiy: 	save.Yiy,
		Ax: 	save.Ax,
		Ay: 	save.Ay,
	}
	paillierSK, err := paillier.UnMarshalSecretkeyByByte(save.PaillierSK)
	if err != nil {
		return nil, err
	}
	local.PaillierSK = paillierSK

	paillierPks := make([]*paillier.PublicKey, len(save.PaillierPks))
	for i := 0; i < len(save.PaillierPks); i++{
		publicKey, err := paillier.UnmarshalPublicKeyByByte([]byte(save.PaillierPks[i]))
		if err != nil{
			return nil, err
		}
		paillierPks[i] = publicKey
	}
	local.PaillierPks = paillierPks

	aux := make([]*pedersen.Parameters, len(save.Aux))
	for j := 0; j < len(save.Aux); j++ {
		ped := &pedersen.Parameters{}
		err := ped.UnmarshalString(save.Aux[j])
		if err != nil{
			return nil, err
		}
		aux[j] = ped
	}
	local.Aux = aux
	labmda, err := hex.DecodeString(save.Lambda)
	if err != nil {
		return nil, err
	}
	s := &safenum.Nat{}
	s.SetBytes(labmda)
	local.Lambda = s
	return local, nil
}


type TempKeyGenData struct {
	Xi      		  *big.Int
	Xix				  *big.Int
	Xiy               *big.Int
	Gammai  		  *big.Int
	Gammaix           []*big.Int
	Gammaiy           []*big.Int
	Rhoi    		  *big.Int
	Ui      		  *big.Int
	V                 []*big.Int

	// vss
	Vssa            []*big.Int
	VssAx           []*big.Int
	VssAy           []*big.Int
	Vssy            []*big.Int
	VssEncy         []*paillier.Ciphertext

	Beta              map[int]*big.Int
	MtAEncB           map[int]*paillier.Ciphertext

	Gammax            *big.Int
	Gammay            *big.Int

	Deltai  		  *big.Int
	Deltaix           *big.Int
	Deltaiy           *big.Int
	Delta             *big.Int
	Deltax            *big.Int
	Deltay            *big.Int
}

type SignatureData struct {
	R         		*big.Int
	S         		*big.Int
}

type TempSignData struct {
	Ki                *big.Int
	Kix               *big.Int
	Kiy               *big.Int
	Wi                *big.Int
	Wix				  *big.Int
	Wiy               *big.Int
	MtAEncW           map[int]*paillier.Ciphertext
	Rx                *big.Int
	Ry                *big.Int
	Rix               *big.Int
	Riy               *big.Int
	Chi               *big.Int
	Beta2             map[int]*big.Int
	S                 *big.Int
}