package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
	"math/rand"
	"testing"
)

func TestErasureCodeRecoverSimple(t *testing.T) {
	// Create some random data, with padding...
	fs := NewFFTSettings(5)
	poly := make([]kzg_bls.Fr, fs.MaxWidth, fs.MaxWidth)
	for i := uint64(0); i < fs.MaxWidth/2; i++ {
		kzg_bls.AsFr(&poly[i], i)
	}
	for i := fs.MaxWidth / 2; i < fs.MaxWidth; i++ {
		poly[i] = kzg_bls.ZERO
	}
	debugFrs("poly", poly)
	// Get data for polynomial SLOW_INDICES
	data, err := fs.FFT(poly, false)
	if err != nil {
		t.Fatal(err)
	}
	debugFrs("data", data)

	// copy over the 2nd half, leave the first half as nils
	subset := make([]*kzg_bls.Fr, fs.MaxWidth, fs.MaxWidth)
	half := fs.MaxWidth / 2
	for i := half; i < fs.MaxWidth; i++ {
		subset[i] = &data[i]
	}

	debugFrPtrs("subset", subset)
	recovered, err := fs.ErasureCodeRecover(subset)
	if err != nil {
		t.Fatal(err)
	}
	debugFrs("recovered", recovered)
	for i := range recovered {
		if got := &recovered[i]; !kzg_bls.EqualFr(got, &data[i]) {
			t.Errorf("recovery at index %d got %s but expected %s", i, kzg_bls.FrStr(got), kzg_bls.FrStr(&data[i]))
		}
	}
	// And recover the original coeffs for good measure
	back, err := fs.FFT(recovered, true)
	if err != nil {
		t.Fatal(err)
	}
	debugFrs("back", back)
	for i := uint64(0); i < half; i++ {
		if got := &back[i]; !kzg_bls.EqualFr(got, &poly[i]) {
			t.Errorf("coeff at index %d got %s but expected %s", i, kzg_bls.FrStr(got), kzg_bls.FrStr(&poly[i]))
		}
	}
	for i := half; i < fs.MaxWidth; i++ {
		if got := &back[i]; !kzg_bls.EqualZero(got) {
			t.Errorf("expected zero padding in index %d", i)
		}
	}
}

func TestErasureCodeRecover(t *testing.T) {
	// Create some random poly, with padding so we get redundant data
	fs := NewFFTSettings(7)
	poly := make([]kzg_bls.Fr, fs.MaxWidth, fs.MaxWidth)
	for i := uint64(0); i < fs.MaxWidth/2; i++ {
		kzg_bls.AsFr(&poly[i], i)
	}
	for i := fs.MaxWidth / 2; i < fs.MaxWidth; i++ {
		poly[i] = kzg_bls.ZERO
	}
	debugFrs("poly", poly)
	// Get coefficients for polynomial SLOW_INDICES
	data, err := fs.FFT(poly, false)
	if err != nil {
		t.Fatal(err)
	}
	debugFrs("data", data)

	// Util to pick a random subnet of the values
	randomSubset := func(known uint64, rngSeed uint64) []*kzg_bls.Fr {
		withMissingValues := make([]*kzg_bls.Fr, fs.MaxWidth, fs.MaxWidth)
		for i := range data {
			withMissingValues[i] = &data[i]
		}
		rng := rand.New(rand.NewSource(int64(rngSeed)))
		missing := fs.MaxWidth - known
		pruned := rng.Perm(int(fs.MaxWidth))[:missing]
		for _, i := range pruned {
			withMissingValues[i] = nil
		}
		return withMissingValues
	}

	// Try different amounts of known indices, and try it in multiple random ways
	var lastKnown uint64 = 0
	for knownRatio := 0.7; knownRatio < 1.0; knownRatio += 0.05 {
		known := uint64(float64(fs.MaxWidth) * knownRatio)
		if known == lastKnown {
			continue
		}
		lastKnown = known
		for i := 0; i < 3; i++ {
			t.Run(fmt.Sprintf("random_subset_%d_known_%d", i, known), func(t *testing.T) {
				subset := randomSubset(known, uint64(i))

				debugFrPtrs("subset", subset)
				recovered, err := fs.ErasureCodeRecover(subset)
				if err != nil {
					t.Fatal(err)
				}
				debugFrs("recovered", recovered)
				for i := range recovered {
					if got := &recovered[i]; !kzg_bls.EqualFr(got, &data[i]) {
						t.Errorf("recovery at index %d got %s but expected %s", i, kzg_bls.FrStr(got), kzg_bls.FrStr(&data[i]))
					}
				}
				// And recover the original coeffs for good measure
				back, err := fs.FFT(recovered, true)
				if err != nil {
					t.Fatal(err)
				}
				debugFrs("back", back)
				half := uint64(len(back)) / 2
				for i := uint64(0); i < half; i++ {
					if got := &back[i]; !kzg_bls.EqualFr(got, &poly[i]) {
						t.Errorf("coeff at index %d got %s but expected %s", i, kzg_bls.FrStr(got), kzg_bls.FrStr(&poly[i]))
					}
				}
				for i := half; i < fs.MaxWidth; i++ {
					if got := &back[i]; !kzg_bls.EqualZero(got) {
						t.Errorf("expected zero padding in index %d", i)
					}
				}
			})
		}
	}
}
