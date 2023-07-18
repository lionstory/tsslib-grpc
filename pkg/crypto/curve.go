// Copyright © 2019 Binance
//
// This file is part of Binance. The full Binance copyright notice, including
// terms governing use, modification, and redistribution, is contained in the
// file LICENSE at the root of the source code distribution tree.

package crypto

import (
	"crypto/elliptic"
	"reflect"

	s256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/decred/dcrd/dcrec/edwards/v2"
	"github.com/emmansun/gmsm/sm2"
)

type CurveType int

const (
	Secp256k1 CurveType = 0
	Ed25519   CurveType = 1
	P256SM2   CurveType = 2
	P256      CurveType = 3
	None                = -1
)

func (c CurveType) String() string {
	switch c {
	case Secp256k1:
		return "secp256k1"
	case Ed25519:
		return "ed25519"
	case P256SM2:
		return "p256sm2"
	case P256:
		return "p256"
	}
	return "none"
}

func ParseCurveType(c string) CurveType {
	switch c {
	case "secp256k1":
		return Secp256k1
	case "ed25519":
		return Ed25519
	case "p256sm2":
		return P256SM2
	case "p256":
		return P256
	}
	return None
}

var (
	ec       elliptic.Curve
	registry map[CurveType]elliptic.Curve
)

// Init default curve (secp256k1)
func init() {
	registry = make(map[CurveType]elliptic.Curve)
	registry[P256] = elliptic.P256()
	registry[Secp256k1] = s256k1.S256()
	registry[Ed25519] = edwards.Edwards()
	registry[P256SM2] = sm2.P256()
	ec = registry[Secp256k1]
}

func RegisterCurve(name string, curve elliptic.Curve) {
	t := ParseCurveType(name)
	registry[t] = curve
}

// return curve, exist(bool)
func GetCurveByName(name string) (elliptic.Curve, bool) {
	t := ParseCurveType(name)
	if val, exist := registry[t]; exist {
		return val, true
	}

	return nil, false
}

func GetCurveByType(t CurveType) (elliptic.Curve, bool) {
	if val, exist := registry[t]; exist {
		return val, true
	}

	return nil, false
}

func GetCurveByType2(t CurveType) elliptic.Curve {
	if val, exist := registry[t]; exist {
		return val
	}

	return nil
}

// return name, exist(bool)
func GetCurveType(curve elliptic.Curve) (CurveType, bool) {
	for name, e := range registry {
		if reflect.TypeOf(curve) == reflect.TypeOf(e) {
			return name, true
		}
	}

	return None, false
}

func GetCurveType2(curve elliptic.Curve) CurveType {
	for name, e := range registry {
		if reflect.TypeOf(curve) == reflect.TypeOf(e) {
			return name
		}
	}

	return None
}

// EC returns the current elliptic curve in use. The default is secp256k1
func EC() elliptic.Curve {
	return ec
}
