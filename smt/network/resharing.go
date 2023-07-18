package network

import (
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"github.com/bnb-chain/tss-lib/tss"
	"hash"
	"math/big"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
	"sync"
)

type ReSharingParameters struct {
	*Parameters
	newParties    *tss.PeerContext
	newPartyCount int
	newThreshold  int
}

func NewReSharingParameters(curve elliptic.Curve, oldctx, newCtx *tss.PeerContext, partyID *tss.PartyID,
	partyCount, threshold, newPartyCount, newThreshold int) *ReSharingParameters {

	params := NewParameters(curve, oldctx, partyID, partyCount, threshold)
	return &ReSharingParameters{
		Parameters:    params,
		newParties:    newCtx,
		newPartyCount: newPartyCount,
		newThreshold:  newThreshold,
	}
}

func (rgParams *ReSharingParameters) OldParties() *tss.PeerContext {
	return rgParams.Parties() // wr use the original method for old parties
}

func (rgParams *ReSharingParameters) OldPartyCount() int {
	return rgParams.partyCount
}

func (rgParams *ReSharingParameters) NewParties() *tss.PeerContext {
	return rgParams.newParties
}

func (rgParams *ReSharingParameters) NewPartyCount() int {
	return rgParams.newPartyCount
}

func (rgParams *ReSharingParameters) NewThreshold() int {
	return rgParams.newThreshold
}

func (rgParams *ReSharingParameters) IsOldCommittee() bool {
	partyID := rgParams.partyID
	for _, Pj := range rgParams.parties.IDs() {
		if partyID.KeyInt().Cmp(Pj.KeyInt()) == 0 {
			return true
		}
	}
	return false
}

func (rgParams *ReSharingParameters) IsNewCommittee() bool {
	partyID := rgParams.partyID
	for _, Pj := range rgParams.newParties.IDs() {
		if partyID.KeyInt().Cmp(Pj.KeyInt()) == 0 {
			return true
		}
	}
	return false
}

type ReSharingParty struct {
	params  	*ReSharingParameters
	Out         chan *MessageResharing
	Recv 		chan *MessageResharing
	End    		chan LocalSaveData
	Temp        LocaltempData
	Input       LocalSaveData
	Save        LocalSaveData
	Hash 		hash.Hash
	Mtx  		sync.Mutex
}

type LocalMessageStore struct {
	DgRound1Message,
	DgRound2Message []MessageResharing
}

type LocaltempData struct {
	LocalMessageStore
	//NewY     *big.Int
	//NewKs    []*big.Int
	Rhoi     []*big.Int
}

func NewReSharingParty(params *ReSharingParameters, out, recv chan *MessageResharing, end chan LocalSaveData, key LocalSaveData) *ReSharingParty{
	subset := key
	if params.IsOldCommittee() {
		subset = BuildLocalSaveDataSubset(key, params.OldParties().IDs())
	}
	p := &ReSharingParty{
		params: params,
		Out: out,
		Recv: recv,
		Temp: LocaltempData{},
		End: end,
		Input: subset,
		Save: NewSaveData(params.newPartyCount),
	}

	// msgs init
	p.Temp.DgRound1Message = make([]MessageResharing, len(params.Parties().IDs()))
	p.Temp.DgRound2Message = make([]MessageResharing, params.NewPartyCount())
	p.Temp.Rhoi 		   = make([]*big.Int, params.NewPartyCount())
	return p
}


func (party *ReSharingParty)Params() *ReSharingParameters{
	return party.params
}

func NewSaveData(partyCount int) (save LocalSaveData){
	save.Aux = make([]*pedersen.Parameters, partyCount)
	save.PaillierPks = make([]*paillier.PublicKey, partyCount)
	save.Yix = make([]*big.Int, partyCount)
	save.Yiy = make([]*big.Int, partyCount)
	save.Ks = make([]*big.Int, partyCount)
	return
}

func BuildLocalSaveDataSubset(sourceData LocalSaveData, sortedIDs tss.SortedPartyIDs) LocalSaveData{
	keysToIndices := make(map[string]int, len(sourceData.Ks))
	for j, kj := range sourceData.Ks {
		keysToIndices[hex.EncodeToString(kj.Bytes())] = j
	}
	newData := NewSaveData(sortedIDs.Len())
	newData.PaillierSK = sourceData.PaillierSK
	newData.Lambda = sourceData.Lambda
	newData.Rtigi = sourceData.Rtigi
	newData.Rtig = sourceData.Rtig
	newData.Y = sourceData.Y
	newData.Rho = sourceData.Rho
	newData.Xx = sourceData.Xx
	newData.Xy = sourceData.Xy
	newData.Ax = sourceData.Ax
	newData.Ay = sourceData.Ay
	for j, id := range sortedIDs {
		savedIdx, ok := keysToIndices[hex.EncodeToString(id.Key)]
		if !ok {
			panic(errors.New("BuildLocalSaveDataSubset: unable to find a signer party in the local save data"))
		}
		newData.Ks[j] = sourceData.Ks[savedIdx]
		newData.PaillierPks[j] = sourceData.PaillierPks[savedIdx]
		newData.Aux[j] = sourceData.Aux[savedIdx]
		newData.Yix[j] = sourceData.Yix[savedIdx]
		newData.Yiy[j] = sourceData.Yiy[savedIdx]
	}
	return newData
}

