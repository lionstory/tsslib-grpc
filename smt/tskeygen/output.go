package tskeygen

import (
	"fmt"
	"math/big"
	"github.com/lionstory/tsslib-grpc/smt/network"
)

func Outputs(party *network.Party) {
	//defer wg.Done()

	Delta := new(big.Int).Set(party.KeyGenTem.Deltai)
	party.KeyGenTem.Delta = Delta

	Deltax := new(big.Int).Set(party.KeyGenTem.Deltaix)
	Deltay := new(big.Int).Set(party.KeyGenTem.Deltaiy)
	party.KeyGenTem.Deltax, party.KeyGenTem.Deltay = Deltax, Deltay

	for i := 0; i < party.Params().PartyCount()-1; i++ {
		val := <-party.Recv // 出 chan
		val.MContent.DoSomething(party)
		//本地计算消息
	}
	Dx, Dy := party.Params().EC().ScalarBaseMult(party.KeyGenTem.Delta.Bytes())
	flag := Dx.Cmp(party.KeyGenTem.Deltax) == 0 && Dy.Cmp(party.KeyGenTem.Deltay) == 0
	if flag != true {
		fmt.Println("error")
	}
	//计算X=delta^-1Gamma-G
	//delta为party.Delta
	//gamma为party.Gammax,party.Gammay.

	//不要忘记求逆呀。
	party.KeyGenTem.Delta.ModInverse(party.KeyGenTem.Delta, party.Params().EC().Params().N)

	party.Data.Xx, party.Data.Xy = party.Params().EC().ScalarMult(party.KeyGenTem.Gammax, party.KeyGenTem.Gammay, party.KeyGenTem.Delta.Bytes())
	var one = new(big.Int).SetInt64(1) //将1变成大数
	oneNeg := new(big.Int).Sub(party.Params().EC().Params().N, one)

	NegGx, NegGy := party.Params().EC().ScalarBaseMult(oneNeg.Bytes())
	party.Data.Xx, party.Data.Xy = party.Params().EC().Add(party.Data.Xx, party.Data.Xy, NegGx, NegGy)

	//	fmt.Println(party.ID, party.Xx, party.Xy)
	party.End <- party.Data

}
