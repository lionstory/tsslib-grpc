package utils

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"

	pb "github.com/lionstory/tsslib-grpc/pkg/proto"
	"github.com/lionstory/tsslib-grpc/pkg/router"
	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

func Encode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decode(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}

func SharedPartyUpdater(p tss.Party, party *tss.PartyID, msg tss.Message, url string, taskName string, errCh chan<- *tss.Error) {
	//if party.Index == msg.GetFrom().Index {
	//	return
	//}
	if party.KeyInt().Cmp(msg.GetFrom().KeyInt()) == 0 {
		fmt.Println("========SharedPartyUpdater==================", party.Id, party.KeyInt(), msg.GetFrom().Id, msg.GetFrom().KeyInt())
		return
	}

	bz, _, err := msg.WireBytes()
	if err != nil {
		errCh <- p.WrapError(err, party)
		return
	}

	pid, err := Encode(msg.GetFrom())
	if err != nil {
		errCh <- p.WrapError(err)
		return
	}

	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return
	}
	client := pb.NewTssServerClient(conn)
	if taskName == "keygen" {
		_, err = client.KeygenTransMsg(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	} else if taskName == "signing" {
		_, err = client.SignTransMsg(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	} else if taskName == "reSharingOld" {
		_, err = client.ReSharingTransMsgToOld(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	} else if taskName == "reSharingNew" {
		_, err = client.ReSharingTransMsgToNew(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	}

	if err != nil {
		errCh <- p.WrapError(err)
	}
}

func SharedPartyUpdater2(p tss.Party, party *tss.PartyID, msg tss.Message, url string, taskName string, errCh chan<- *tss.Error) {
	bz, _, err := msg.WireBytes()
	if err != nil {
		errCh <- p.WrapError(err, party)
		return
	}

	pid, err := Encode(msg.GetFrom())
	if err != nil {
		errCh <- p.WrapError(err)
		return
	}

	conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return
	}
	client := pb.NewTssServerClient(conn)
	if taskName == "keygen" {
		_, err = client.KeygenTransMsg(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	} else if taskName == "signing" {
		_, err = client.SignTransMsg(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	} else if taskName == "reSharingOld" {
		_, err = client.ReSharingTransMsgToOld(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	} else if taskName == "reSharingNew" {
		_, err = client.ReSharingTransMsgToNew(context.Background(), &pb.TransMsgRequest{Message: bz, Party: pid, IsBroadcast: msg.IsBroadcast()})
	}

	if err != nil {
		errCh <- p.WrapError(err)
	}
}

func UpdateRound(party tss.Party, pMsg tss.ParsedMessage, errCh chan *tss.Error) {
	_, err := party.Update(pMsg)
	if err != nil {
		errCh <- err
	}
}

func CheckDirectory(dataDir string) error {
	if stat, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dataDir, os.ModePerm)
		} else {
			return err
		}
	} else if !stat.IsDir() {
		return errors.Errorf("need a directory, but found a ")
	}
	return nil
}

func TryWriteTestFixtureFile(dataDir string, data keygen.LocalPartySaveData) {
	fixtureFileName := fmt.Sprintf("%s/keygen_data.json", dataDir)
	fmt.Println(">>>Key file path to be written: ", fixtureFileName)

	fd, err := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		common.Logger.Errorf("unable to open fixture file %s for writing, %s", fixtureFileName, err)
		panic(err)
	}
	bz, err := json.Marshal(&data)
	if err != nil {
		common.Logger.Errorf("unable to marshal save data for fixture file %s", fixtureFileName)
		panic(err)
	}
	_, err = fd.Write(bz)
	if err != nil {
		common.Logger.Errorf("unable to write to fixture file %s", fixtureFileName)
		panic(err)
	}
	common.Logger.Info(">>>Saved a test fixture file for party %s", fixtureFileName)

}

func LoadKeyInfo(dataDir string) (keygen.LocalPartySaveData, error) {
	var key keygen.LocalPartySaveData
	fixtureFilePath := fmt.Sprintf("%s/keygen_data.json", dataDir)
	bz, err := os.ReadFile(fixtureFilePath)
	if err != nil {
		return key, errors.Wrapf(err,
			"could not open the test fixture for party in the expected location: %s. run keygen tests first.",
			fixtureFilePath)
	}
	if err = json.Unmarshal(bz, &key); err != nil {
		return key, errors.Wrapf(err,
			"could not unmarshal fixture data for party located at: %s",
			fixtureFilePath)
	}
	for _, kbxj := range key.BigXj {
		kbxj.SetCurve(tss.S256())
	}
	return key, nil
}

type RouteInfo struct {
	PartyId     *tss.PartyID
	Router      *router.SortedRouterTable
	Threshold   int
	PIdKey      *big.Int
	KeyRevision int
}

func TryWriteRouteTable(dataDir string, pid *tss.PartyID, table *router.SortedRouterTable, threshold int) {
	routeFileName := fmt.Sprintf("%s/route_info.json", dataDir)

	fd, err := os.OpenFile(routeFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		common.Logger.Errorf("unable to open route file %s for writing", routeFileName)
	}
	defer fd.Close()

	var pidKey big.Int
	t := RouteInfo{pid, table, threshold, (&pidKey).SetBytes(pid.Key), 0}
	bs, err := json.Marshal(t)
	//bs, err := Encode(t)
	if err != nil {
		common.Logger.Errorf("unable to marshal save data for route file %s", routeFileName)
	}
	_, err = fd.Write(bs)
	if err != nil {
		common.Logger.Errorf("unable to write to route file %s", routeFileName)
	}
	common.Logger.Infof("Saved a test route file for party %d: %s", pid.Index, routeFileName)
}

func WriteTestFixtureFile(dataDir string, data keygen.LocalPartySaveData) {
	fixtureFileName := fmt.Sprintf("%s/keygen_data.json", dataDir)

	fd, err := os.OpenFile(fixtureFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		common.Logger.Errorf("unable to open fixture file %s for writing, %s", fixtureFileName, err)
	}
	bz, err := json.Marshal(&data)
	if err != nil {
		common.Logger.Errorf("unable to marshal save data for fixture file %s", fixtureFileName)
	}
	_, err = fd.Write(bz)
	if err != nil {
		common.Logger.Errorf("unable to write to fixture file %s", fixtureFileName)
	}
	common.Logger.Info("Saved a test fixture file for party %s", fixtureFileName)
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

func CleanRouteAndKey(dataDir string) error {
	if err := os.Remove(fmt.Sprintf("%s/keygen_data.json", dataDir)); err != nil {
		return err
	} else if err = os.Remove(fmt.Sprintf("%s/route_info.json", dataDir)); err != nil {
		return err
	}
	return nil
}

func MergeStruct(val1, val2 interface{}) error {
	t := reflect.TypeOf(val1)
	v := reflect.ValueOf(val1)
	v2 := reflect.ValueOf(val2)

	fmt.Println(v.Type())
	if v.Type() != v.Type() {
		return errors.New("val1, val2 is not same type")
	}

	for i := 0; i < v.Elem().NumField(); i++ {
		fmt.Println(t.Elem().Field(i).Name, t.Elem().Field(i).Type, v.Elem().Field(i).Type(), v.Elem().Field(i).Kind())
		kind := v.Elem().Field(i).Kind()
		switch kind {
		case reflect.Struct:
			if err := MergeStruct(v.Elem().Field(i).Addr().Interface(), v2.Elem().Field(i).Addr().Interface()); err != nil {
				return err
			}
		case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
			if v.Elem().Field(i).IsNil() && !v2.Elem().Field(i).IsNil() {
				v.Elem().Field(i).Set(v2.Elem().Field(i))
			}
		default:
			if v.Elem().Field(i).IsZero() && !v2.Elem().Field(i).IsZero() {
				v.Elem().Field(i).Set(v2.Elem().Field(i))
			}
		}
	}
	return nil
}
