// Original: https://github.com/ethereum/research/blob/master/kzg_data_availability/fk20_multi.py
package KZG_ZKSNARK

import (
	"IBEUser/src/KZG_ZKSNARK/kzg_bls"
	"fmt"
)

// FK20 Method to compute all proofs
// Toeplitz multiplication via http://www.netlib.org/utk/people/JackDongarra/etemplates/node384.html
// Multi proof method

// For a polynomial of size n, let w be a n-th root of unity. Then this method will return
// k=n/l KZG proofs for the points
//
//	proof[0]: w^(0*l + 0), w^(0*l + 1), ... w^(0*l + l - 1)
//	proof[0]: w^(0*l + 0), w^(0*l + 1), ... w^(0*l + l - 1)
//	...
//	proof[i]: w^(i*l + 0), w^(i*l + 1), ... w^(i*l + l - 1)
//	...
func (ks *FK20MultiSettings) FK20Multi(polynomial []kzg_bls.Fr) []kzg_bls.G1Point {
	n := uint64(len(polynomial))
	n2 := n * 2
	if ks.MaxWidth < n2 {
		panic(fmt.Errorf("KZGSettings are set to MaxWidth %d but got half polynomial of length %d",
			ks.MaxWidth, n))
	}

	hExtFFT := make([]kzg_bls.G1Point, n2, n2)
	for i := uint64(0); i < n2; i++ {
		kzg_bls.CopyG1(&hExtFFT[i], &kzg_bls.ZeroG1)
	}

	var tmp kzg_bls.G1Point
	for i := uint64(0); i < ks.chunkLen; i++ {
		toeplitzCoeffs := ks.toeplitzCoeffsStepStrided(polynomial, i, ks.chunkLen)
		hExtFFTFile := ks.ToeplitzPart2(toeplitzCoeffs, ks.xExtFFTFiles[i])
		for j := uint64(0); j < n2; j++ {
			kzg_bls.AddG1(&tmp, &hExtFFT[j], &hExtFFTFile[j])
			kzg_bls.CopyG1(&hExtFFT[j], &tmp)
		}
	}
	h := ks.ToeplitzPart3(hExtFFT)

	out, err := ks.FFTG1(h, false)
	if err != nil {
		panic(err)
	}
	return out
}

// FK20 multi-proof method, optimized for dava availability where the top half of polynomial
// coefficients == 0
func (ks *FK20MultiSettings) FK20MultiDAOptimized(polynomial []kzg_bls.Fr) []kzg_bls.G1Point {
	n2 := uint64(len(polynomial))
	if ks.MaxWidth < n2 {
		panic(fmt.Errorf("KZGSettings are set to MaxWidth %d but got polynomial of length %d",
			ks.MaxWidth, n2))
	}
	n := n2 / 2
	for i := n; i < n2; i++ {
		if !kzg_bls.EqualZero(&polynomial[i]) {
			panic("bad input, second half should be zeroed")
		}
	}

	k := n / ks.chunkLen
	k2 := k * 2
	hExtFFT := make([]kzg_bls.G1Point, k2, k2)
	for i := uint64(0); i < k2; i++ {
		kzg_bls.CopyG1(&hExtFFT[i], &kzg_bls.ZeroG1)
	}

	reducedPoly := polynomial[:n]
	var tmp kzg_bls.G1Point
	for i := uint64(0); i < ks.chunkLen; i++ {
		toeplitzCoeffs := ks.toeplitzCoeffsStepStrided(reducedPoly, i, ks.chunkLen)
		//debugFrs(fmt.Sprintf("toeplitz_coefficients %d:", i), toeplitzCoeffs)
		//DebugG1s(fmt.Sprintf("xext_fft file %d:", i), ks.xExtFFTFiles[i])
		hExtFFTFile := ks.ToeplitzPart2(toeplitzCoeffs, ks.xExtFFTFiles[i])
		//DebugG1s(fmt.Sprintf("hext_fft file %d:", i), hExtFFTFile)
		for j := uint64(0); j < k2; j++ {
			kzg_bls.AddG1(&tmp, &hExtFFT[j], &hExtFFTFile[j])
			kzg_bls.CopyG1(&hExtFFT[j], &tmp)
		}
		//DebugG1s(fmt.Sprintf("hext_fft %d:", i), hExtFFT)
	}
	//DebugG1s("hext_fft final", hExtFFT)
	h := ks.ToeplitzPart3(hExtFFT)
	//DebugG1s("h", h)

	// TODO: maybe use a G1 version of the DAS extension FFT to perform the h -> output conversion?

	// Now redo the padding before final step.
	// Instead of copying h into a new extended array, just reuse the old capacity.
	h = h[:k2]
	for i := k; i < k2; i++ {
		kzg_bls.CopyG1(&h[i], &kzg_bls.ZeroG1)
	}
	out, err := ks.FFTG1(h, false)
	if err != nil {
		panic(err)
	}
	return out
}

// Computes all the KZG proofs for data availability checks. This involves sampling on the double domain
// and reordering according to reverse bit order
func (ks *FK20MultiSettings) DAUsingFK20Multi(polynomial []kzg_bls.Fr) []kzg_bls.G1Point {
	n := uint64(len(polynomial))
	if n > ks.MaxWidth/2 {
		panic("expected poly contents not bigger than half the size of the FK20-multi settings")
	}
	if !kzg_bls.IsPowerOfTwo(n) {
		panic("expected poly length to be power of two")
	}
	n2 := n * 2
	extendedPolynomial := make([]kzg_bls.Fr, n2, n2)
	for i := uint64(0); i < n; i++ {
		kzg_bls.CopyFr(&extendedPolynomial[i], &polynomial[i])
	}
	for i := n; i < n2; i++ {
		kzg_bls.CopyFr(&extendedPolynomial[i], &kzg_bls.ZERO)
	}
	allProofs := ks.FK20MultiDAOptimized(extendedPolynomial)
	// change to reverse bit order.
	reverseBitOrderG1(allProofs)
	return allProofs
}
