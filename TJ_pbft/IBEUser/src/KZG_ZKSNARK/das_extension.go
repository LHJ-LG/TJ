package KZG_ZKSNARK

import "IBEUser/src/KZG_ZKSNARK/kzg_bls"

// warning: the values in `a` are modified in-place to become the outputs.
// Make a deep copy first if you need to use them later.
func (fs *FFTSettings) dASFFTExtension(ab []kzg_bls.Fr, domainStride uint64) {
	if len(ab) == 2 {
		aHalf0 := &ab[0]
		aHalf1 := &ab[1]
		var x kzg_bls.Fr
		kzg_bls.AddModFr(&x, aHalf0, aHalf1)
		var y kzg_bls.Fr
		kzg_bls.SubModFr(&y, aHalf0, aHalf1)
		var tmp kzg_bls.Fr
		kzg_bls.MulModFr(&tmp, &y, &fs.ExpandedRootsOfUnity[domainStride])
		kzg_bls.AddModFr(&ab[0], &x, &tmp)
		kzg_bls.SubModFr(&ab[1], &x, &tmp)
		return
	}

	if len(ab) < 2 {
		panic("bad usage")
	}

	half := uint64(len(ab))
	halfHalf := half >> 1
	abHalf0s := ab[:halfHalf]
	abHalf1s := ab[halfHalf:half]
	// Instead of allocating L0 and L1, just modify a in-place.
	//L0[i] = (((a_half0 + a_half1) % modulus) * inv2) % modulus
	//R0[i] = (((a_half0 - L0[i]) % modulus) * inverse_domain[i * 2]) % modulus
	var tmp1, tmp2 kzg_bls.Fr
	for i := uint64(0); i < halfHalf; i++ {
		aHalf0 := &abHalf0s[i]
		aHalf1 := &abHalf1s[i]
		kzg_bls.AddModFr(&tmp1, aHalf0, aHalf1)
		kzg_bls.SubModFr(&tmp2, aHalf0, aHalf1)
		kzg_bls.MulModFr(aHalf1, &tmp2, &fs.ReverseRootsOfUnity[i*2*domainStride])
		kzg_bls.CopyFr(aHalf0, &tmp1)
	}

	// L will be the left half of out
	fs.dASFFTExtension(abHalf0s, domainStride<<1)
	// R will be the right half of out
	fs.dASFFTExtension(abHalf1s, domainStride<<1)

	// The odd deduced outputs are written to the output array already, but then updated in-place
	// L1 = b[:halfHalf]
	// R1 = b[halfHalf:]

	// Half the work of a regular FFT: only deal with uneven-index outputs
	var yTimesRoot kzg_bls.Fr
	var x, y kzg_bls.Fr
	for i := uint64(0); i < halfHalf; i++ {
		// Temporary copies, so that writing to output doesn't conflict with input.
		// Note that one hand is from L1, the other R1
		kzg_bls.CopyFr(&x, &abHalf0s[i])
		kzg_bls.CopyFr(&y, &abHalf1s[i])
		root := &fs.ExpandedRootsOfUnity[(1+2*i)*domainStride]
		kzg_bls.MulModFr(&yTimesRoot, &y, root)
		// write outputs in place, avoid unnecessary list allocations
		kzg_bls.AddModFr(&abHalf0s[i], &x, &yTimesRoot)
		kzg_bls.SubModFr(&abHalf1s[i], &x, &yTimesRoot)
	}
}

// Takes vals as input, the values of the even indices.
// Then computes the values for the odd indices, which combined would make the right half of coefficients zero.
// Warning: the odd results are written back to the vals slice.
func (fs *FFTSettings) DASFFTExtension(vals []kzg_bls.Fr) {
	if uint64(len(vals))*2 > fs.MaxWidth {
		panic("domain too small for extending requested values")
	}
	fs.dASFFTExtension(vals, 1)
	// The above function didn't perform the divide by 2 on every layer.
	// So now do it all at once, by dividing by 2**depth (=length).
	var invLen kzg_bls.Fr
	kzg_bls.AsFr(&invLen, uint64(len(vals)))
	kzg_bls.InvModFr(&invLen, &invLen)
	for i := 0; i < len(vals); i++ {
		kzg_bls.MulModFr(&vals[i], &vals[i], &invLen)
	}
}
