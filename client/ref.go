package main

import (
	"encoding/hex"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/pkg/errors"
	"github.com/tjfoc/gmsm/sm2"
	"math/big"
	"reflect"
)

func main() {
	var data keygen.LocalPartySaveData
	var data2 keygen.LocalPartySaveData
	MergeStruct(&data, &data2)
	a, b := utils.LoadRouteInfo("data/party-1")
	fmt.Println(a, b)
	i := &big.Int{}
	i = i.SetBytes(a.PartyId.Key)
	fmt.Printf("%v\n", i)
	fmt.Println(a.PIdKey)

	bs, _ := utils.Encode(sm2.P256Sm2())
	fmt.Printf("---%v---\n", hex.EncodeToString(bs))
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
