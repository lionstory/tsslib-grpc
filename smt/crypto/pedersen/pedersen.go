package pedersen

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/cronokirby/safenum"
	"github.com/lionstory/tsslib-grpc/smt/crypto/math/arith"
	"github.com/lionstory/tsslib-grpc/smt/crypto/params"
)

type Error string

const (
	ErrNilFields    Error = "contains nil field"
	ErrSEqualT      Error = "S cannot be equal to T"
	ErrNotValidModN Error = "S and T must be in [1,…,N-1] and coprime to N"
)

func (e Error) Error() string {
	return fmt.Sprintf("pedersen: %s", string(e))
}

type Parameters struct {
	N    *arith.Modulus
	S, T *safenum.Nat
}

// New returns a new set of Pedersen parameters.
// Assumes ValidateParameters(n, s, t) returns nil.
func New(n *arith.Modulus, s, t *safenum.Nat) *Parameters {
	return &Parameters{
		S: s,
		T: t,
		N: n,
	}
}

// ValidateParameters check n, s and t, and returns an error if any of the following is true:
// - n, s, or t is nil.
// - s, t are not in [1, …,n-1].
// - s, t are not coprime to N.
// - s = t.
func ValidateParameters(n *safenum.Modulus, s, t *safenum.Nat) error {
	if n == nil || s == nil || t == nil {
		return ErrNilFields
	}
	// s, t ∈ ℤₙˣ
	if !arith.IsValidNatModN(n, s, t) {
		return ErrNotValidModN
	}
	// s ≡ t
	if _, eq, _ := s.Cmp(t); eq == 1 {
		return ErrSEqualT
	}
	return nil
}

// N = p•q, p ≡ q ≡ 3 mod 4.
//func (p Parameters) N() *safenum.Modulus { return p.n.Modulus }

// S = r² mod N.
//func (p Parameters) S() *safenum.Nat { return p.s }

// T = Sˡ mod N.
//func (p Parameters) T() *safenum.Nat { return p.t }

// Commit computes sˣ tʸ (mod N)
//
// x and y are taken as safenum.Int, because we want to keep these values in secret,
// in general. The commitment produced, on the other hand, hides their values,
// and can be safely shared.
func (p Parameters) Commit(x, y *safenum.Int) *safenum.Nat {
	sx := p.N.ExpI(p.S, x)
	ty := p.N.ExpI(p.T, y)

	result := sx.ModMul(sx, ty, p.N.Modulus)

	return result
}

// Verify returns true if sᵃ tᵇ ≡ S Tᵉ (mod N).
func (p Parameters) Verify(a, b, e *safenum.Int, S, T *safenum.Nat) bool {
	if a == nil || b == nil || S == nil || T == nil || e == nil {
		return false
	}
	nMod := p.N.Modulus
	if !arith.IsValidNatModN(nMod, S, T) {
		return false
	}

	sa := p.N.ExpI(p.S, a)         // sᵃ (mod N)
	tb := p.N.ExpI(p.T, b)         // tᵇ (mod N)
	lhs := sa.ModMul(sa, tb, nMod) // lhs = sᵃ⋅tᵇ (mod N)

	te := p.N.ExpI(T, e)          // Tᵉ (mod N)
	rhs := te.ModMul(te, S, nMod) // rhs = S⋅Tᵉ (mod N)
	return lhs.Eq(rhs) == 1
}

// WriteTo implements io.WriterTo and should be used within the hash.Hash function.
func (p *Parameters) WriteTo(w io.Writer) (int64, error) {
	if p == nil {
		return 0, io.ErrUnexpectedEOF
	}
	nAll := int64(0)
	buf := make([]byte, params.BytesIntModN)

	// write N, S, T
	for _, i := range []*safenum.Nat{p.N.Nat(), p.S, p.T} {
		i.FillBytes(buf)
		n, err := w.Write(buf)
		nAll += int64(n)
		if err != nil {
			return nAll, err
		}
	}
	return nAll, nil
}

// Domain implements hash.WriterToWithDomain, and separates this type within hash.Hash.
func (Parameters) Domain() string {
	return "Pedersen Parameters"
}

// todo
func (p *Parameters) MarshalString() string{
	data := fmt.Sprintf("%s|%s|%s", hex.EncodeToString(p.N.MarshalBytes()),
		hex.EncodeToString(p.S.Bytes()), hex.EncodeToString(p.T.Bytes()))
	return data
}

// todo
func (p *Parameters) UnmarshalString(data string) error{
	parts := strings.Split(data, "|")
	nByte, err := hex.DecodeString(parts[0])
	if err != nil {
		return err
	}
	n, err := arith.UnmarshalByByte(nByte)
	if err != nil {
		return err
	}
	p.N = n

	sByte, err := hex.DecodeString(parts[1])
	if err != nil {
		return err
	}
	s := &safenum.Nat{}
	s.SetBytes(sByte)
	p.S = s

	tByte, err := hex.DecodeString(parts[2])
	if err != nil {
		return err
	}
	t := &safenum.Nat{}
	t.SetBytes(tByte)
	p.T = t
	return nil
}
