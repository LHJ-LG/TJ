package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"testing"
)

func TestKZGSettings_CommitToEvalPoly(t *testing.T) {
	fs := NewFFTSettings(4)
	s1, s2 := GenerateTestingSetup("1927409816240961209460912649124", 16+1)
	ks := NewKZGSettings(fs, s1, s2)
	polynomial := testPoly(1, 2, 3, 4, 7, 7, 7, 7, 13, 13, 13, 13, 13, 13, 13, 13)
	evalPoly, err := fs.FFT(polynomial, false)
	if err != nil {
		t.Fatal(err)
	}
	secretG1IFFT, err := fs.FFTG1(ks.SecretG1[:16], true)
	if err != nil {
		t.Fatal(err)
	}

	commitmentByCoeffs := ks.CommitToPoly(polynomial)
	commitmentByEval := CommitToEvalPoly(secretG1IFFT, evalPoly)
	if !kzg_bls.EqualG1(commitmentByEval, commitmentByCoeffs) {
		t.Fatalf("expected commitments to be equal, but got:\nby eval: %s\nby coeffs: %s",
			commitmentByEval, commitmentByCoeffs)
	}
}

func TestKZGSettings_CheckProofSingle(t *testing.T) {
	fs := NewFFTSettings(4)
	s1, s2 := GenerateTestingSetup("1927409816240961209460912649124", 16+1)
	ks := NewKZGSettings(fs, s1, s2)
	for i := 0; i < len(ks.SecretG1); i++ {
		t.Logf("secret g1 %d: %s", i, kzg_bls.StrG1(&ks.SecretG1[i]))
	}

	polynomial := testPoly(1, 2, 3, 4, 7, 7, 7, 7, 13, 13, 13, 13, 13, 13, 13, 13)
	for i := 0; i < len(polynomial); i++ {
		t.Logf("poly coeff %d: %s", i, kzg_bls.FrStr(&polynomial[i]))
	}

	commitment := ks.CommitToPoly(polynomial)
	t.Log("commitment\n", kzg_bls.StrG1(commitment))

	proof := ks.ComputeProofSingle(polynomial, 17)
	t.Log("proof\n", kzg_bls.StrG1(proof))

	var x kzg_bls.Fr
	kzg_bls.AsFr(&x, 17)
	var value kzg_bls.Fr
	kzg_bls.EvalPolyAt(&value, polynomial, &x)
	t.Log("value\n", kzg_bls.FrStr(&value))

	if !ks.CheckProofSingle(commitment, proof, &x, &value) {
		t.Fatal("could not verify proof")
	}
}

func testPoly(polynomial ...uint64) []kzg_bls.Fr {
	n := len(polynomial)
	polynomialFr := make([]kzg_bls.Fr, n, n)
	for i := 0; i < n; i++ {
		kzg_bls.AsFr(&polynomialFr[i], polynomial[i])
	}
	return polynomialFr
}
