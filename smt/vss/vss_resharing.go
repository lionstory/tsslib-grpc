package vss

import (
	"crypto/elliptic"
	"github.com/bnb-chain/tss-lib/tss"
	"math/big"
	"github.com/lionstory/tsslib-grpc/smt/modfiysm2"
)


func Lagrange(IDs tss.SortedPartyIDs, id int, N *big.Int) *big.Int{
	//N := party.Params().EC().Params().N

	xi := new(big.Int)
	for _, _mparty := range IDs{
		if _mparty.Index == id {
			xi = _mparty.KeyInt()
		}
	}

	//计算系数
	xj := new(big.Int)
	A, _ := new(big.Int).SetString("1", 0)
	B, _ := new(big.Int).SetString("1", 0)

	for _, mparty := range IDs{
		if mparty.Index != id{
			//计算每一项
			xj = xj.Neg(mparty.KeyInt())
			//			fmt.Println(id, xj)
			A.Mul(A, xj)
			A.Mod(A, N)
			xj.Add(xj, xi)
			B.Mul(B, xj)
			B.Mod(B, N)
		}
	}
	B.ModInverse(B, N)
	B.Mul(B, A)
	B.Mod(B, N)
	return B
}

func Vssshare(curve elliptic.Curve, secret *big.Int, threshold, partyNum int, IDs tss.SortedPartyIDs) ([]*big.Int, []*big.Int, []*big.Int, []*big.Int){
	//新建一些VSS信息
	Vssa := make([]*big.Int, threshold)
	VssAx := make([]*big.Int, threshold)
	VssAy := make([]*big.Int, threshold)
	for i := 0; i < threshold; i++ {
		Vssa[i], _ = modfiysm2.RandFieldElement(curve, nil)
		VssAx[i], VssAy[i] = curve.ScalarBaseMult(Vssa[i].Bytes())
	}

	//计算分享值,显然，这里需要对每一个party计算一个，每一次计算有一个T循环，所以有两层循环
	Vssy := make([]*big.Int, partyNum)
	for _, partyi := range IDs{
		yi := new(big.Int)
		di := new(big.Int).Set(partyi.KeyInt())
		temp := new(big.Int)
		for key := 0; key < threshold; key++ {
			temp.Mul(di, Vssa[key])
			temp.Mod(temp, curve.Params().N)
			yi.Add(yi, temp)
			di.Mul(di, partyi.KeyInt())
			di.Mod(di, curve.Params().N)
		}
		yi.Add(yi, secret)
		yi.Mod(yi, curve.Params().N)
		Vssy[partyi.Index] = yi
	}
	return Vssa, VssAx, VssAy, Vssy
}

