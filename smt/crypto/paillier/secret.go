package paillier

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"github.com/lionstory/tsslib-grpc/smt/crypto/params"

	"github.com/cronokirby/safenum"
	"github.com/taurusgroup/multi-party-sig/pkg/math/sample"
	"github.com/taurusgroup/multi-party-sig/pkg/pool"
	"github.com/lionstory/tsslib-grpc/smt/crypto/math/arith"
	"github.com/lionstory/tsslib-grpc/smt/crypto/pedersen"
)

var (
	ErrPrimeBadLength = errors.New("prime factor is not the right length")
	ErrNotBlum        = errors.New("prime factor is not equivalent to 3 (mod 4)")
	ErrNotSafePrime   = errors.New("supposed prime factor is not a safe prime")
	ErrPrimeNil       = errors.New("prime is nil")
)

// SecretKey is the secret key corresponding to a Public Paillier Key.
//
// A public key is a modulus N, and the secret key contains the information
// needed to factor N into two primes, P and Q. This allows us to decrypt
// values encrypted using this modulus.
type SecretKey struct {
	*PublicKey
	// p, q such that N = p⋅q
	P, Q *safenum.Nat
	// phi = ϕ = (p-1)(q-1)
	Phi *safenum.Nat
	// phiInv = ϕ⁻¹ mod N
	PhiInv *safenum.Nat
}


// Phi returns ϕ = (P-1)(Q-1).
//
// This is the result of the totient function ϕ(N), where N = P⋅Q
// is our public key. This function counts the number of units mod N.
//
// This quantity is useful in ZK proofs.


// KeyGen generates a new PublicKey and it's associated SecretKey.
func KeyGen(pl *pool.Pool) (pk *PublicKey, sk *SecretKey) {
	sk = NewSecretKey(pl)
	pk = sk.PublicKey
	return
}

// NewSecretKey generates primes p and q suitable for the scheme, and returns the initialized SecretKey.
func NewSecretKey(pl *pool.Pool) *SecretKey {
	// TODO maybe we could take the reader as argument?
	return NewSecretKeyFromPrimes(sample.Paillier(rand.Reader, pl))
}

// NewSecretKeyFromPrimes generates a new SecretKey. Assumes that P and Q are prime.
func NewSecretKeyFromPrimes(P, Q *safenum.Nat) *SecretKey {
	oneNat := new(safenum.Nat).SetUint64(1)

	n := arith.ModulusFromFactors(P, Q)

	nNat := n.Nat()
	nPlusOne := new(safenum.Nat).Add(nNat, oneNat, -1)
	// Tightening is fine, since n is public
	nPlusOne.Resize(nPlusOne.TrueLen())

	pMinus1 := new(safenum.Nat).Sub(P, oneNat, -1)
	qMinus1 := new(safenum.Nat).Sub(Q, oneNat, -1)
	phi := new(safenum.Nat).Mul(pMinus1, qMinus1, -1)
	// ϕ⁻¹ mod N
	phiInv := new(safenum.Nat).ModInverse(phi, n.Modulus)

	pSquared := pMinus1.Mul(P, P, -1)
	qSquared := qMinus1.Mul(Q, Q, -1)
	nSquared := arith.ModulusFromFactors(pSquared, qSquared)

	return &SecretKey{
		P:      P,
		Q:      Q,
		Phi:    phi,
		PhiInv: phiInv,
		PublicKey: &PublicKey{
			N:        n,
			NSquared: nSquared,
			NNat:     nNat,
			NPlusOne: nPlusOne,
		},
	}
}

// Dec decrypts c and returns the plaintext m ∈ ± (N-2)/2.
// It returns an error if gcd(c, N²) != 1 or if c is not in [1, N²-1].
func (sk *SecretKey) Dec(ct *Ciphertext) (*safenum.Int, error) {
	oneNat := new(safenum.Nat).SetUint64(1)

	n := sk.PublicKey.N.Modulus

	if !sk.PublicKey.ValidateCiphertexts(ct) {
		return nil, errors.New("paillier: failed to decrypt invalid ciphertext")
	}

	phi := sk.Phi
	phiInv := sk.PhiInv

	// r = c^Phi 						(mod N²)
	result := sk.PublicKey.NSquared.Exp(ct.C, phi)
	// r = c^Phi - 1
	result.Sub(result, oneNat, -1)
	// r = [(c^Phi - 1)/N]
	result.Div(result, n, -1)
	// r = [(c^Phi - 1)/N] • Phi^-1		(mod N)
	result.ModMul(result, phiInv, n)

	// see 6.1 https://www.iacr.org/archive/crypto2001/21390136.pdf
	return new(safenum.Int).SetModSymmetric(result, n), nil
}

// DecWithRandomness returns the underlying plaintext, as well as the randomness used.
func (sk *SecretKey) DecWithRandomness(ct *Ciphertext) (*safenum.Int, *safenum.Nat, error) {
	m, err := sk.Dec(ct)
	if err != nil {
		return nil, nil, err
	}
	mNeg := new(safenum.Int).SetInt(m).Neg(1)

	// x = C(N+1)⁻ᵐ (mod N)
	x := sk.N.ExpI(sk.NPlusOne, mNeg)
	x.ModMul(x, ct.C, sk.N.Modulus)

	// r = xⁿ⁻¹ (mod N)
	nInverse := new(safenum.Nat).ModInverse(sk.NNat, safenum.ModulusFromNat(sk.Phi))
	r := sk.N.Exp(x, nInverse)
	return m, r, nil
}

func (sk SecretKey) GeneratePedersen() (*pedersen.Parameters, *safenum.Nat) {
	s, t, lambda := sample.Pedersen(rand.Reader, sk.Phi, sk.N.Modulus)
	ped := pedersen.New(sk.N, s, t)
	return ped, lambda
}

// ValidatePrime checks whether p is a suitable prime for Paillier.
// Checks:
// - log₂(p) ≡ params.BitsBlumPrime.
// - p ≡ 3 (mod 4).
// - q := (p-1)/2 is prime.
func ValidatePrime(p *safenum.Nat) error {
	if p == nil {
		return ErrPrimeNil
	}
	// check bit lengths
	const bitsWant = params.BitsBlumPrime
	// Technically, this leaks the number of bits, but this is fine, since returning
	// an error asserts this number statically, anyways.
	if bits := p.TrueLen(); bits != bitsWant {
		return fmt.Errorf("invalid prime size: have: %d, need %d: %w", bits, bitsWant, ErrPrimeBadLength)
	}
	// check == 3 (mod 4)
	if p.Byte(0)&0b11 != 3 {
		return ErrNotBlum
	}

	// check (p-1)/2 is prime
	pMinus1Div2 := new(safenum.Nat).Rsh(p, 1, -1)

	if !pMinus1Div2.Big().ProbablyPrime(1) {
		return ErrNotSafePrime
	}
	return nil
}

//todo
func (s *SecretKey) MarshalBytes() string{
	data := fmt.Sprintf("%s|%s|%s|%s|%s", hex.EncodeToString(s.PublicKey.MarshalBytes()),
		hex.EncodeToString(s.P.Bytes()), hex.EncodeToString(s.Q.Bytes()),
		hex.EncodeToString(s.Phi.Bytes()), hex.EncodeToString(s.PhiInv.Bytes()))
	return data
}

// todo
func UnMarshalSecretkeyByByte(data string) (*SecretKey, error){
	//data := string(bytes)
	parts := strings.Split(data, "|")
	publicKeyByte, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	publicKey, err := UnmarshalPublicKeyByByte(publicKeyByte)
	if err != nil {
		return nil, err
	}
	pByte, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	p := &safenum.Nat{}
	p.SetBytes(pByte)
	qByte, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	q := &safenum.Nat{}
	q.SetBytes(qByte)
	phiByte, err := hex.DecodeString(parts[3])
	if err != nil {
		return nil, err
	}
	phi := &safenum.Nat{}
	phi.SetBytes(phiByte)
	phiInvByte, err := hex.DecodeString(parts[4])
	if err != nil {
		return nil, err
	}
	phiInv := &safenum.Nat{}
	phiInv.SetBytes(phiInvByte)
	return &SecretKey{
		PublicKey: publicKey,
		P: p,
		Q: q,
		Phi: phi,
		PhiInv: phiInv,
	}, nil
}