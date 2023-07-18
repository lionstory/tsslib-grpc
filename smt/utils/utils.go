package utils

import (
	"encoding/json"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/lionstory/tsslib-grpc/smt/network"
	"github.com/pkg/errors"
	"math/big"
	"os"
)

func LoadKeyInfo(dataDir string) (*network.LocalSaveData, error){
	var key network.SaveData
	fixtureFilePath := fmt.Sprintf("%s/ts_keygen_data.json", dataDir)
	bz, err := os.ReadFile(fixtureFilePath)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(bz, &key); err != nil {
		return nil, err
	}
	data, err := key.Unmarshal()
	if err != nil {
		return nil, err
	}

	return data, nil
}

type RouteInfo struct {
	PartyId     *tss.PartyID
	Router      *router.SortedRouterTable
	Threshold   int
	PIdKey      *big.Int
	KeyRevision int
}

func WriteRouteTable(dataDir string, pid *tss.PartyID, table *router.SortedRouterTable, threshold int, revision int) {
	routeFileName := fmt.Sprintf("%s/route_info.json", dataDir)

	fd, err := os.OpenFile(routeFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		common.Logger.Errorf("unable to open route file %s for writing", routeFileName)
	}
	defer fd.Close()

	var pidKey big.Int
	t := RouteInfo{pid, table, threshold, (&pidKey).SetBytes(pid.Key), revision}
	bs, err := json.Marshal(t)
	//bs, err := Encode(t)
	if err != nil {
		common.Logger.Errorf("unable to marshal save data for route file %s", routeFileName)
	}
	_, err = fd.Write(bs)
	if err != nil {
		common.Logger.Errorf("unable to write to route file %s", routeFileName)
	}
	common.Logger.Info("Saved a test route file for party %d: %s", pid.Index, routeFileName)
}

func LoadRouteInfo(dataDir string) (*RouteInfo, error) {
	routeFileName := fmt.Sprintf("%s/route_info.json", dataDir)
	bz, err := os.ReadFile(routeFileName)
	if err != nil {
		return nil, errors.Wrapf(err,
			"could not open the route for party in the expected location: %s. ",
			routeFileName)
	}
	var t RouteInfo
	if err = json.Unmarshal(bz, &t); err != nil {
		return nil, errors.Wrapf(err,
			"could not unmarshal route data for party located at: %s",
			routeFileName)
	}
	return &t, nil
}