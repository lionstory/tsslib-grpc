package router

import (
	"fmt"
	"github.com/bnb-chain/tss-lib/tss"
)

type Router struct {
	Url     string
	PartyId *tss.PartyID
}

type RouterTable struct {
	Table []*Router
}

type PartyStatus struct {
	PartyId     *tss.PartyID
	KeyRevision int
}

func NewRouterTable(pids tss.SortedPartyIDs, urls []string) *RouterTable {
	table := make([]*Router, 0, len(pids))
	for i, P := range pids {
		router := &Router{
			Url:     urls[i],
			PartyId: P,
		}
		table = append(table, router)
	}
	return &RouterTable{
		Table: table,
	}
}

type SortedRouterTable struct {
	Pids  tss.SortedPartyIDs
	Table map[string]string
}

func NewSortedRouterTable(pids tss.SortedPartyIDs, urls []string) *SortedRouterTable {
	table := map[string]string{}
	for i, P := range pids {
		table[P.Id] = urls[i]
	}
	return &SortedRouterTable{
		Pids:  pids,
		Table: table,
	}
}

func NewSortedRouterTable2(pids tss.SortedPartyIDs, urls map[string]string) *SortedRouterTable {
	return &SortedRouterTable{
		Pids:  pids,
		Table: urls,
	}
}

func (s *SortedRouterTable) GetURLByID(id string) string {
	if url, ok := s.Table[id]; ok {
		return url
	}
	return ""
}

func (s *SortedRouterTable) GetPidByUrl(url string) *tss.PartyID {
	for k, v := range s.Table {
		if v == url {
			for _, pid := range s.Pids {
				if pid.Id == k {
					return pid
				}
			}
		}
	}
	return nil
}

func (s *SortedRouterTable) Display() string {
	for _, v := range s.Pids {
		fmt.Printf("======SortedRouterTable.ID======id:%v, key:%v, moniker:%v\n", v.Id, v.Key, v.Moniker)
	}
	for k, v := range s.Table {
		fmt.Printf("======SortedRouterTable.URL======id:%v, url:%v\n", k, v)
	}
	return ""
}

//func (table *RouterTable)BroadCast(method string, data map[string]string) error{
//	values := url.Values{}
//	for key, value := range(data){
//		values.Set(key, value)
//	}
//	for _, router := range table.Table{
//		fmt.Println(router.Url+"/"+method)
//		_, err := utils.PostForm(router.Url+"/"+method, values)
//		if err != nil{
//			return err
//		}
//	}
//	return nil
//}
