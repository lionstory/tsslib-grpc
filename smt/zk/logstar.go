package zk

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"hash"
	"strings"

	"github.com/cronokirby/safenum"
	"github.com/taurusgroup/multi-party-sig/pkg/math/sample"
	//"github.com/taurusgroup/multi-party-sig/pkg/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/paillier"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
	//"github.com/taurusgroup/multi-party-sig/pkg/pedersen"
	"math/big"
)

type Logstarp struct {
	//记得要该名称,算了，不改了，其他的都不变，增加了Y
	// S = sᵏtᵘ
	S *safenum.Nat
	// A = Enc₀ (α, r)
	A *paillier.Ciphertext
	// C = sᵃtᵍ
	C  *safenum.Nat
	Yx *big.Int
	Yy *big.Int
	// Z₁ = α + e⋅k
	Z1 *safenum.Int
	// Z₂ = r ⋅ ρᵉ mod N₀
	Z2 *safenum.Nat
	// Z₃ = γ + e⋅μ
	Z3 *safenum.Int
}

// 输入hash，aux，PK,证明的K和k,rho
func LogstarProve(hash hash.Hash, curve elliptic.Curve, Aux *pedersen.Parameters, PK *paillier.PublicKey, K *paillier.Ciphertext, Xx *big.Int, Xy *big.Int, k *safenum.Int, rho *safenum.Nat) *Logstarp {
	N := PK.N
	NModulus := PK.Modulus()
	//这里，让alpha永不为负,如此智障的想法，然后花了两个小时，活该你写的慢，而且丑。
	alpha1 := sample.IntervalLEps(rand.Reader)
	alpha2 := alpha1.Abs()
	alpha := new(safenum.Int).SetNat(alpha2)

	r := sample.UnitModN(rand.Reader, N.Modulus)
	mu := sample.IntervalLN(rand.Reader)
	gamma := sample.IntervalLEpsN(rand.Reader)

	S := Aux.Commit(k, mu)
	A := PK.EncWithNonce(alpha, r)
	C := Aux.Commit(alpha, gamma)
	x := alpha.Abs().Big()
	Yx, Yy := curve.ScalarBaseMult(x.Bytes())

	hash.Write(BytesCombine(Aux.N.Bytes(), Aux.S.Bytes(), Aux.T.Bytes(), PK.Modulus().Bytes(), K.Nat().Bytes(), S.Bytes(), A.Nat().Bytes(), C.Bytes(), Yx.Bytes(), Yy.Bytes()))
	bytes := hash.Sum(nil)
	e := new(safenum.Int).SetBytes(bytes)
	//注意这里没有控制e的范围，可能会出事请。
	//	e = (*safenum.Int)(e.Mod(N))
	hash.Reset()

	z1 := new(safenum.Int).SetInt(k)
	z1.Mul(e, z1, -1)
	z1.Add(z1, alpha, -1)

	z2 := NModulus.ExpI(rho, e)
	z2.ModMul(z2, r, N.Modulus)

	z3 := new(safenum.Int).Mul(e, mu, -1)
	z3.Add(z3, gamma, -1)

	return &Logstarp{
		S:  S,
		A:  A,
		C:  C,
		Yx: Yx,
		Yy: Yy,
		Z1: z1,
		Z2: z2,
		Z3: z3,
	}
}

func (zkp *Logstarp) LogstarVerify(hash hash.Hash, curve elliptic.Curve, Aux *pedersen.Parameters, PK *paillier.PublicKey, K *paillier.Ciphertext, Xx *big.Int, Xy *big.Int) bool {
	//缺了一个什么呢，缺了范围验证。

	hash.Write(BytesCombine(Aux.N.Bytes(), Aux.S.Bytes(), Aux.T.Bytes(), PK.Modulus().Bytes(), K.Nat().Bytes(), zkp.S.Bytes(), zkp.A.Nat().Bytes(), zkp.C.Bytes(), zkp.Yx.Bytes(), zkp.Yy.Bytes()))
	bytes := hash.Sum(nil)
	hash.Reset()
	e := new(safenum.Int).SetBytes(bytes)
	//注意这里没有控制e的范围，可能会出事请。
	//	e = (*safenum.Int)(e.Mod(N))

	if !Aux.Verify(zkp.Z1, zkp.Z3, e, zkp.C, zkp.S) {
		return false
	}

	{
		// lhs = Enc(z₁;z₂)
		lhs := PK.EncWithNonce(zkp.Z1, zkp.Z2)

		// rhs = (e ⊙ K) ⊕ A
		rhs := K.Clone().Mul(PK, e).Add(PK, zkp.A)
		if !lhs.Equal(rhs) {
			return false
		}
	}
	//不知道这个是不是多此一举
	//	zkp.Z1.Abs()
	e2 := e.Abs().Big()
	e2.Mod(e2, curve.Params().N)
	z := zkp.Z1.Abs().Big()
	z.Mod(z, curve.Params().N)

	zGx, zGy := curve.ScalarBaseMult(z.Bytes())
	epkx, epky := curve.ScalarMult(Xx, Xy, e2.Bytes())
	z2Gx, z2Gy := curve.Add(zkp.Yx, zkp.Yy, epkx, epky)

	return zGx.Cmp(z2Gx) == 0 && zGy.Cmp(z2Gy) == 0
}

func (zkp *Logstarp) MarshalString() (string, error){
	z1, err := zkp.Z1.MarshalBinary()
	if err != nil {
		return "", err
	}
	z3, err := zkp.Z3.MarshalBinary()
	if err != nil {
		return "", err
	}
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s", hex.EncodeToString(zkp.S.Bytes()), zkp.A.MarshalString(),
		hex.EncodeToString(zkp.C.Bytes()), hex.EncodeToString(zkp.Yx.Bytes()), hex.EncodeToString(zkp.Yy.Bytes()),
		hex.EncodeToString(z1), hex.EncodeToString(zkp.Z2.Bytes()), hex.EncodeToString(z3))

	return data, nil
}

func (zkp *Logstarp) UnmarshalString(data string) error{
	parts := strings.Split(data, "|")
	s, err := hex.DecodeString(parts[0])
	if err != nil {
		return err
	}
	zkp.S.SetBytes(s)
	a := &paillier.Ciphertext{}
	err = a.UnmarshalString(parts[1])
	if err != nil {
		return err
	}
	zkp.A = a
	c, err := hex.DecodeString(parts[2])
	if err != nil {
		return err
	}
	zkp.C.SetBytes(c)
	yx, err := hex.DecodeString(parts[3])
	if err != nil {
		return err
	}
	zkp.Yx.SetBytes(yx)
	yy, err := hex.DecodeString(parts[4])
	if err != nil {
		return err
	}
	zkp.Yy.SetBytes(yy)
	z1 := &safenum.Int{}
	z1_byte, err := hex.DecodeString(parts[5])
	if err != nil {
		return err
	}
	z1.UnmarshalBinary(z1_byte)
	zkp.Z1 = z1
	z2_byte, err := hex.DecodeString(parts[6])
	if err != nil {
		return err
	}
	zkp.Z2.SetBytes(z2_byte)
	z3 := &safenum.Int{}
	z3_byte, err := hex.DecodeString(parts[7])
	if err != nil {
		return err
	}
	z3.UnmarshalBinary(z3_byte)
	zkp.Z3 = z3
	return nil
}

