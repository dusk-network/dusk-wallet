package mlsag

import (
	"bytes"
	"errors"
	"fmt"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Signature struct {
	c         ristretto.Scalar
	r         []Responses
	PubKeys   []PubKeys
	KeyImages []ristretto.Point
	Msg       []byte
}

func (proof *Proof) Prove() (*Signature, error) {
	// Shuffle the PubKeys and update the
	// 	index for our corresponding key
	err := proof.shuffleSet()
	if err != nil {
		return nil, err
	}

	keyImages := proof.calculateKeyImages()
	nonces := generateNonces(len(proof.privKeys))

	numUsers := len(proof.pubKeysMatrix)
	numKeysPerUser := len(proof.privKeys)

	// We will overwrite the signers responses
	responses := generateResponses(numUsers, numKeysPerUser, proof.index)

	// Let secretIndex = index of signer
	secretIndex := proof.index

	// Generate C_{secretIndex+1}
	buf := &bytes.Buffer{}
	buf.Write(proof.msg)
	signersPubKeys := proof.pubKeysMatrix[secretIndex]

	for i := 0; i < len(nonces); i++ {

		nonce := nonces[i]

		// P = nonce * G
		var P ristretto.Point
		P.ScalarMultBase(&nonce)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return nil, err
		}

		// P = nonce * H(K)
		var hK ristretto.Point
		hK.Derive(signersPubKeys.keys[i].Bytes())
		P.ScalarMult(&hK, &nonce)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return nil, err
		}
	}
	var CjPlusOne ristretto.Scalar
	CjPlusOne.Derive(buf.Bytes())

	// generate challenges
	challenges := make([]ristretto.Scalar, numUsers)
	challenges[(secretIndex+1)%numUsers] = CjPlusOne

	var prevChallenge ristretto.Scalar
	prevChallenge.Set(&CjPlusOne)

	for k := secretIndex + 2; k != (secretIndex+1)%numUsers; k = (k + 1) % numUsers {
		i := k % numUsers

		prevIndex := (i - 1) % numUsers
		if prevIndex < 0 {
			prevIndex = prevIndex + numUsers
		}
		fakeResponses := responses[prevIndex]
		decoyPubKeys := proof.pubKeysMatrix[prevIndex]

		c, err := generateChallenge(proof.msg, fakeResponses, keyImages, decoyPubKeys, prevChallenge)
		if err != nil {
			return nil, err
		}

		challenges[i].Set(&c)
		prevChallenge.Set(&c)
	}

	// Set the real response for signer
	var realResponse Responses
	for i := 0; i < numKeysPerUser; i++ {
		challenge := challenges[proof.index]
		privKey := proof.privKeys[i]
		nonce := nonces[i]
		var r ristretto.Scalar

		// r = nonce - challenge*privKey
		r.Mul(&challenge, &privKey)
		r.Neg(&r)
		r.Add(&r, &nonce)
		realResponse.AddResponse(r)
	}

	// replace real response in responses array
	responses[proof.index] = realResponse

	sig := &Signature{
		c:         challenges[0],
		r:         responses,
		PubKeys:   proof.pubKeysMatrix,
		KeyImages: keyImages,
		Msg:       proof.msg,
	}

	return sig, nil
}

func (sig *Signature) Verify() (bool, error) {

	if len(sig.PubKeys) == 0 || len(sig.r) == 0 || len(sig.KeyImages) == 0 {
		return false, errors.New("cannot have zero length for responses, pubkeys or key images")
	}

	numUsers := len(sig.r)
	index := 0

	var prevChallenge = sig.c

	keyImages := sig.KeyImages
	for k := index + 1; k != (index)%numUsers; k = (k + 1) % numUsers {
		i := k % numUsers
		prevIndex := (i - 1) % numUsers
		if prevIndex < 0 {
			prevIndex = prevIndex + numUsers
		}

		fakeResponses := sig.r[prevIndex]
		decoyPubKeys := sig.PubKeys[prevIndex]
		challenge, err := generateChallenge(sig.Msg, fakeResponses, keyImages, decoyPubKeys, prevChallenge)
		if err != nil {
			return false, err
		}
		prevChallenge = challenge
	}

	// Calculate c'
	prevIndex := (index - 1) % numUsers
	if prevIndex < 0 {
		prevIndex = prevIndex + numUsers
	}
	fakeResponses := sig.r[prevIndex]
	decoyPubKeys := sig.PubKeys[prevIndex]

	challenge, err := generateChallenge(sig.Msg, fakeResponses, keyImages, decoyPubKeys, prevChallenge)
	if err != nil {
		return false, err
	}

	if !challenge.Equals(&sig.c) {
		return false, fmt.Errorf("c'0 does not equal c0, %s != %s", challenge.String(), sig.c.String())
	}

	return true, nil
}

func generateNonces(n int) []ristretto.Scalar {
	var nonces []ristretto.Scalar
	for i := 0; i < n; i++ {
		var nonce ristretto.Scalar
		nonce.Rand()
		nonces = append(nonces, nonce)
	}
	return nonces
}

// XXX: Test should check that random numbers are not all zero
//A bug in ristretto lib that may not be fixed
// Check the same for points too
// skip skips the singers responses
func generateResponses(m int, n, skip int) []Responses {
	var matrixResponses []Responses
	for i := 0; i < m; i++ {
		if i == skip {
			matrixResponses = append(matrixResponses, Responses{})
			continue
		}
		var resp Responses
		for i := 0; i < n; i++ {
			var r ristretto.Scalar
			r.Rand()
			resp.AddResponse(r)
		}
		matrixResponses = append(matrixResponses, resp)
	}
	return matrixResponses
}

func generateChallenge(
	msg []byte,
	respsonses Responses,
	keyImages []ristretto.Point,
	pubKeys PubKeys,
	prevChallenge ristretto.Scalar) (ristretto.Scalar, error) {

	buf := &bytes.Buffer{}
	_, err := buf.Write(msg)
	if err != nil {
		return ristretto.Scalar{}, err
	}

	for i := 0; i < len(keyImages); i++ {

		r := respsonses[i]

		// P = r * G + c * PubKey
		var P, cK ristretto.Point
		P.ScalarMultBase(&r)
		cK.ScalarMult(&pubKeys.keys[i], &prevChallenge)
		P.Add(&P, &cK)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return ristretto.Scalar{}, err
		}

		// P = r * H(K) + c * Ki
		var hK ristretto.Point
		hK.Derive(pubKeys.keys[i].Bytes())
		P.ScalarMult(&hK, &r)
		cK.ScalarMult(&keyImages[i], &prevChallenge)
		P.Add(&P, &cK)
		_, err = buf.Write(P.Bytes())
		if err != nil {
			return ristretto.Scalar{}, err
		}
	}

	var challenge ristretto.Scalar
	challenge.Derive(buf.Bytes())

	return challenge, nil
}

func (proof *Proof) calculateKeyImages() []ristretto.Point {
	var keyImages []ristretto.Point

	privKeys := proof.privKeys
	pubKeys := proof.pubKeysMatrix[proof.index]

	for i := 0; i < len(privKeys); i++ {
		var point ristretto.Point
		point.Set(&pubKeys.keys[i])
		// P = H(xG)
		point.Derive(point.Bytes())
		// P = xH(P)
		point.ScalarMult(&point, &privKeys[i])

		keyImages = append(keyImages, point)
	}
	return keyImages
}
