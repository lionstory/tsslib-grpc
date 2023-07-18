package resharing

import (
	"github.com/lionstory/tsslib-grpc/smt/network"
)

func Outputs(party *network.ReSharingParty) {
	for i := 0; i < party.Params().NewPartyCount()-1; i++ {
		val := <-party.Recv // 出 chan
		val.MContent.DoSomething(party)
		//本地计算消息
	}
	//fmt.Println("***************************", party.Save.Rho)
	party.End <- party.Save
}
