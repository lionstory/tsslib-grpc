package signing

import (
	"math/big"
	"github.com/lionstory/tsslib-grpc/smt/network"
)

func Outputs(party *network.Party) {
	//defer wg.Done()

	party.SignData.S = new(big.Int).Set(party.Temp.S)

	for i := 0; i < len(party.Params().Parties().IDs())-1; i++ {
		val := <-party.Recv // 出 chan
		val.MContent.DoSomething(party)
		//本地计算消息
	}
	//s=sum(s)
	party.SignData.S.Mod(party.SignData.S, party.Params().EC().Params().N)
	//s=(s-r)modn
	party.SignData.S.Sub(party.SignData.S, party.SignData.R)
	party.SignData.S.Mod(party.SignData.S, party.Params().EC().Params().N)


	party.SignEnd <- party.SignData
}