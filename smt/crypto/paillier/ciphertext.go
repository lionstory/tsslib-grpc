package paillier

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"github.com/lionstory/tsslib-grpc/smt/crypto/params"

	"github.com/cronokirby/safenum"
	"github.com/taurusgroup/multi-party-sig/pkg/math/sample"
)

// Ciphertext represents an integer of the for (1+N)ᵐρᴺ (mod N²), representing the encryption of m ∈ ℤₙˣ.
type Ciphertext struct {
	C *safenum.Nat
}

// Add sets ct to the homomorphic sum ct ⊕ ct₂.
// ct ← ct•ct₂ (mod N²).
func (ct *Ciphertext) Add(pk *PublicKey, ct2 *Ciphertext) *Ciphertext {
	if ct2 == nil {
		return ct
	}

	ct.C.ModMul(ct.C, ct2.C, pk.NSquared.Modulus)

	return ct
}

// Mul sets ct to the homomorphic multiplication of k ⊙ ct.
// ct ← ctᵏ (mod N²).
func (ct *Ciphertext) Mul(pk *PublicKey, k *safenum.Int) *Ciphertext {
	if k == nil {
		return ct
	}

	ct.C = pk.NSquared.ExpI(ct.C, k)

	return ct
}

// Equal check whether ct ≡ ctₐ (mod N²).
func (ct *Ciphertext) Equal(ctA *Ciphertext) bool {
	return ct.C.Eq(ctA.C) == 1
}

// Clone returns a deep copy of ct.
func (ct Ciphertext) Clone() *Ciphertext {
	c := new(safenum.Nat)
	c.SetNat(ct.C)
	return &Ciphertext{C: c}
}

// Randomize multiplies the ciphertext's nonce by a newly generated one.
// ct ← ct ⋅ nonceᴺ (mod N²).
// If nonce is nil, a random one is generated.
// The receiver is updated, and the nonce update is returned.
func (ct *Ciphertext) Randomize(pk *PublicKey, nonce *safenum.Nat) *safenum.Nat {
	if nonce == nil {
		nonce = sample.UnitModN(rand.Reader, pk.N.Modulus)
	}
	// c = c*r^N
	tmp := pk.NSquared.Exp(nonce, pk.NNat)
	ct.C.ModMul(ct.C, tmp, pk.NSquared.Modulus)
	return nonce
}

// WriteTo implements io.WriterTo and should be used within the hash.Hash function.
func (ct *Ciphertext) WriteTo(w io.Writer) (int64, error) {
	if ct == nil {
		return 0, io.ErrUnexpectedEOF
	}
	buf := make([]byte, params.BytesCiphertext)
	ct.C.FillBytes(buf)
	n, err := w.Write(buf)
	return int64(n), err
}

// Domain implements hash.WriterToWithDomain, and separates this type within hash.Hash.
func (*Ciphertext) Domain() string {
	return "Paillier Ciphertext"
}

func (ct *Ciphertext) MarshalBinary() ([]byte, error) {
	return ct.C.MarshalBinary()
}

func (ct *Ciphertext) UnmarshalBinary(data []byte) error {
	ct.C = new(safenum.Nat)
	return ct.C.UnmarshalBinary(data)
}

func (ct *Ciphertext) Nat() *safenum.Nat {
	return new(safenum.Nat).SetNat(ct.C)
}

// todo
func (ct *Ciphertext) MarshalString() string{
	data := ct.C.Bytes()
	return hex.EncodeToString(data)
}

// todo
func (ct *Ciphertext) UnmarshalString(data string) error{
	ct.C = new(safenum.Nat)
	c, err := hex.DecodeString(data)
	if err != nil{
		return err
	}
	ct.C.SetBytes(c)
	return nil
}