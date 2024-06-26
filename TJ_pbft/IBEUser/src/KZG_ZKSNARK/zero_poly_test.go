package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
	"math/rand"
	"testing"
)

func TestFFTSettings_reduceLeaves(t *testing.T) {
	fs := NewFFTSettings(4)

	var fromTreeReduction []kzg_bls.Fr

	{
		// prepare some leaves
		leaves := [][]kzg_bls.Fr{make([]kzg_bls.Fr, 3), make([]kzg_bls.Fr, 3), make([]kzg_bls.Fr, 3), make([]kzg_bls.Fr, 3)}
		leafIndices := [][]uint64{{1, 3}, {7, 8}, {9, 10}, {12, 13}}
		for i := 0; i < 4; i++ {
			fs.makeZeroPolyMulLeaf(leaves[i], leafIndices[i], 1)
		}

		dst := make([]kzg_bls.Fr, 16, 16)
		scratch := make([]kzg_bls.Fr, 16*3, 16*3)
		fs.reduceLeaves(scratch, dst, leaves)
		fromTreeReduction = dst[:2*4+1]
	}

	var fromDirect []kzg_bls.Fr
	{
		dst := make([]kzg_bls.Fr, 9, 9)
		indices := []uint64{1, 3, 7, 8, 9, 10, 12, 13}
		fs.makeZeroPolyMulLeaf(dst, indices, 1)
		fromDirect = dst
	}

	if len(fromDirect) != len(fromTreeReduction) {
		t.Fatal("length mismatch")
	}
	for i := 0; i < len(fromDirect); i++ {
		a, b := &fromDirect[i], &fromTreeReduction[i]
		if !kzg_bls.EqualFr(a, b) {
			t.Errorf("zero poly coeff %d is different. direct: %s, tree: %s", i, kzg_bls.FrStr(a), kzg_bls.FrStr(b))
		}
	}
	//debugFrs("zero poly (tree reduction)", fromTreeReduction)
	//debugFrs("zero poly (direct slow)", fromDirect)
}

func TestFFTSettings_reduceLeaves_parametrized(t *testing.T) {
	ratios := []float64{0.01, 0.1, 0.2, 0.4, 0.5, 0.7, 0.9, 0.99}
	for scale := uint8(5); scale < 13; scale++ {
		t.Run(fmt.Sprintf("scale_%d", scale), func(t *testing.T) {
			for i, ratio := range ratios {
				t.Run(fmt.Sprintf("ratio_%.3f", ratio), func(t *testing.T) {
					seed := int64(1000*int(scale) + i)
					testReduceLeaves(scale, ratio, seed, t)
				})
			}
		})
	}
}

func testReduceLeaves(scale uint8, missingRatio float64, seed int64, t *testing.T) {
	fs := NewFFTSettings(scale)
	rng := rand.New(rand.NewSource(seed))
	pointCount := uint64(1) << scale
	missingCount := uint64(int(float64(pointCount) * missingRatio))
	if missingCount == 0 {
		return // nothing missing
	}

	// select the missing points
	missing := make([]uint64, pointCount, pointCount)
	for i := uint64(0); i < pointCount; i++ {
		missing[i] = i
	}
	rng.Shuffle(int(pointCount), func(i, j int) {
		missing[i], missing[j] = missing[j], missing[i]
	})
	missing = missing[:missingCount]

	// build the leaves
	pointsPerLeaf := uint64(63)
	leafCount := (missingCount + pointsPerLeaf - 1) / pointsPerLeaf
	leaves := make([][]kzg_bls.Fr, leafCount, leafCount)
	for i := uint64(0); i < leafCount; i++ {
		start := i * pointsPerLeaf
		end := start + pointsPerLeaf
		if end > missingCount {
			end = missingCount
		}
		leafSize := end - start
		leaf := make([]kzg_bls.Fr, leafSize+1, leafSize+1)
		indices := make([]uint64, leafSize, leafSize)
		for j := uint64(0); j < leafSize; j++ {
			indices[j] = missing[i*pointsPerLeaf+j]
		}
		fs.makeZeroPolyMulLeaf(leaf, indices, 1)
		leaves[i] = leaf
	}

	var fromTreeReduction []kzg_bls.Fr

	{
		dst := make([]kzg_bls.Fr, pointCount, pointCount)
		scratch := make([]kzg_bls.Fr, pointCount*3, pointCount*3)
		fs.reduceLeaves(scratch, dst, leaves)
		fromTreeReduction = dst[:missingCount+1]
	}

	var fromDirect []kzg_bls.Fr
	{
		dst := make([]kzg_bls.Fr, missingCount+1, missingCount+1)
		fs.makeZeroPolyMulLeaf(dst, missing, fs.MaxWidth/pointCount)
		fromDirect = dst
	}

	if len(fromDirect) != len(fromTreeReduction) {
		t.Fatal("length mismatch")
	}
	for i := 0; i < len(fromDirect); i++ {
		a, b := &fromDirect[i], &fromTreeReduction[i]
		if !kzg_bls.EqualFr(a, b) {
			t.Errorf("zero poly coeff %d is different. direct: %s, tree: %s", i, kzg_bls.FrStr(a), kzg_bls.FrStr(b))
		}
	}
	//debugFrs("zero poly (tree reduction)", fromTreeReduction)
	//debugFrs("zero poly (direct slow)", fromDirect)
}

func TestFFTSettings_ZeroPolyViaMultiplication_Python(t *testing.T) {
	fs := NewFFTSettings(4)

	exists := []bool{
		true, false, false, true,
		false, true, true, false,
		false, false, true, true,
		false, true, false, true,
	}
	var missingIndices []uint64
	for i, v := range exists {
		if !v {
			missingIndices = append(missingIndices, uint64(i))
		}
	}

	zeroEval, zeroPoly := fs.ZeroPolyViaMultiplication(missingIndices, uint64(len(exists)))

	// produced from python implementation, check it's exactly correct.
	expectedEval := []kzg_bls.Fr{
		kzg_bls.ToFr("40868503138626303263713448452028063093974861640573380501185290423282553381059"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("9059493333851894280622930192031068801018187410981018272280547403745554404951"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("589052107338478098858761185551735055781651813398303959420821217298541933174"),
		kzg_bls.ToFr("1980700778768058987161339158728243463014673552245301202287722613196911807966"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("48588946696503834689243119316363329218956542308951664733900338765742108388091"),
		kzg_bls.ToFr("17462668815085674001076443909983570919844170615339489499875900337907893054793"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("32986316229085390499922301497961243665601583888595873281538162159212447231217"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("31340620128536760059637470141592017333700483773455661424257920684057136952965"),
	}
	for i := range zeroEval {
		if !kzg_bls.EqualFr(&expectedEval[i], &zeroEval[i]) {
			t.Errorf("at eval %d, expected: %s, got: %s", i, kzg_bls.FrStr(&expectedEval[i]), kzg_bls.FrStr(&zeroEval[i]))
		}
	}
	expectedPoly := []kzg_bls.Fr{
		kzg_bls.ToFr("37647706414300369857238608619982937390838535937985112215973498325246987289395"),
		kzg_bls.ToFr("2249310547870908874251949653552971443359134481191188461034956129255788965773"),
		kzg_bls.ToFr("14214218681578879810156974734536988864583938194339599855352132142401756507144"),
		kzg_bls.ToFr("11562429031388751544281783289945994468702719673309534612868555280828261838388"),
		kzg_bls.ToFr("38114263339263944057999429128256535679768370097817780187577397655496877536510"),
		kzg_bls.ToFr("21076784030567214561538347586500535789557219054084066119912281151549494675620"),
		kzg_bls.ToFr("9111875896859243625633322505516518368332415340935654725595105138403527134249"),
		kzg_bls.ToFr("11763665547049371891508513950107512764213633861965719968078681999977021803005"),
		kzg_bls.ToFr("1"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
		kzg_bls.ToFr("0"),
	}
	for i := range zeroPoly {
		if !kzg_bls.EqualFr(&expectedPoly[i], &zeroPoly[i]) {
			t.Errorf("at poly %d, expected: %s, got: %s", i, kzg_bls.FrStr(&expectedPoly[i]), kzg_bls.FrStr(&zeroPoly[i]))
		}
	}
}

func testZeroPoly(t *testing.T, scale uint8, seed int64) {
	fs := NewFFTSettings(scale)

	rng := rand.New(rand.NewSource(seed))

	exists := make([]bool, fs.MaxWidth, fs.MaxWidth)
	var missingIndices []uint64
	missingStr := ""
	for i := 0; i < len(exists); i++ {
		if rng.Intn(2) == 0 {
			exists[i] = true
		} else {
			missingIndices = append(missingIndices, uint64(i))
			missingStr += fmt.Sprintf(" %d", i)
		}
	}
	//t.Logf("missing indices:%s", missingStr)

	zeroEval, zeroPoly := fs.ZeroPolyViaMultiplication(missingIndices, uint64(len(exists)))

	//debugFrs("zero eval", zeroEval)
	//debugFrs("zero poly", zeroPoly)

	for i, v := range exists {
		if !v {
			var at kzg_bls.Fr
			kzg_bls.CopyFr(&at, &fs.ExpandedRootsOfUnity[i])
			var out kzg_bls.Fr
			kzg_bls.EvalPolyAt(&out, zeroPoly, &at)
			if !kzg_bls.EqualZero(&out) {
				t.Errorf("expected zero at %d, but got: %s", i, kzg_bls.FrStr(&out))
			}
		}
	}

	p, err := fs.FFT(zeroEval, true)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(zeroPoly); i++ {
		if !kzg_bls.EqualFr(&p[i], &zeroPoly[i]) {
			t.Errorf("fft not correct, i: %v, a: %s, b: %s", i, kzg_bls.FrStr(&p[i]), kzg_bls.FrStr(&zeroPoly[i]))
		}
	}
	for i := len(zeroPoly); i < len(p); i++ {
		if !kzg_bls.EqualZero(&p[i]) {
			t.Errorf("fft not correct, i: %v, a: %s, b: 0", i, kzg_bls.FrStr(&p[i]))
		}
	}
}

func TestFFTSettings_ZeroPolyViaMultiplication_Parametrized(t *testing.T) {
	for i := uint8(3); i < 12; i++ {
		t.Run(fmt.Sprintf("scale_%d", i), func(t *testing.T) {
			for j := int64(0); j < 3; j++ {
				t.Run(fmt.Sprintf("case_%d", j), func(t *testing.T) {
					testZeroPoly(t, i, int64(i)*1000+j)
				})
			}
		})
	}
}
