package crypto

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateKey(t *testing.T) {
	hash := "ee71b697dc3cd771d13508cd1781ab83"
	key := CreateSessionKey(hash)
	assert.Equal(t, 32, len(key))
}

func TestSymetricEncryption(t *testing.T) {
	plainText := []byte("hello world")
	inputEncrypt := bytes.NewBuffer(plainText)
	outputEncrypt := &bytes.Buffer{}

	hash := "ee71b697dc3cd771d13508cd1781ab83"
	key := CreateSessionKey(hash)

	err := EncryptSym(key, inputEncrypt, outputEncrypt)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !assert.NotEqual(t, outputEncrypt.Bytes(), plainText) {
		t.FailNow()
	}

	outputDecrypt := &bytes.Buffer{}
	inputDecrypt := bytes.NewBuffer(outputEncrypt.Bytes())
	err = DecryptSym(key, inputDecrypt, outputDecrypt)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	if !reflect.DeepEqual(plainText, outputDecrypt.Bytes()) {
		t.Errorf("Original message defer from decryptred message")
	}
}

func TestAsymectricEncryption(t *testing.T) {
	hash := "ee71b697dc3cd771d13508cd1781ab83"
	sessionKey := CreateSessionKey(hash)

	privateKey := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAsUweFjQUDjQjBZOXLbdDTiczKSndO3A5Z5P2TAAK3uzICsNn
BYVp1CBdIJucIzFINfBJDlG5IccHJ07c9lQhe9dNRm035f+8M9d+5ESJhxZa+NTi
8NrNXxSQPOQgGaKmOB7w2mqGM8ir/jqNhiyetz20dioGsjVM/y2rX8WZT+0264eN
LmfHoHLgda4VtIApS96Ql1GV1zU0TLzKGrfF1CCDe01FUneRqoLP/h+B41sclokc
xThempykLhJQv5wRQvia7c/bR0moZEzRKPb4SmKHiOcGbRTR3SS08LjPNayW0OOI
UhWIBZQTpc2zv3oh32a0x8bEumV6TV7ME16++QIDAQABAoIBAELGY1KDfMY4trQD
+V1bd3r44pjvToZzZvtuy8WmAnIhhdof7C41KD2fjtOYJ/9NMWA3RpyhBPQGzNfu
KOSRnSbSWSVcP0BdyBlSYVVBxvZc4hhzvaFvFwhna0ezt69QBgB/DsGEe1UHkFeo
3+KX7ZMgJ1aVz33Q+1XkcnYYqvxhvwrcM20YEO/Tax1tdWm+r1RkLdxoL1ZlCu8D
TiYvUwAVCGX7Rrs+pkrRGst+KbGfhMQCZI+qraCbLVCvs3nYF6xwNyeSSZPux4z5
0+uMBOM2rqYXPbzMm+LYRovVAz5VYFFVBev7jBgWkG19FS3oQqpUkaruT/ic59FG
AkCwGzECgYEA6/t1QWBk1Lp1qODxkDyyBx3LJ1gC3/VbO5qibygch3PSXNDXcaSs
EY5EFvojWSFp89e5U+RetIuusKRSgkpDuh3SaSbh+Mcfd0BkZi/mZqldmGirVQ5e
Jg6otnsYZFyP/TNaE8rFHY1JsnAgxWw+czcBY7rWXh+j3QMiW7bVZcUCgYEAwFZF
EnJ8+s13mf90DAFPR18WGEAFyoFR4isQOyIXuHnMUMOAHk6PTHHrgXhNSyl/1gRH
RNku55mT8BNOcbXu1gZQ20/evaX8at7hrW3+Z2Im0bmbtCYsw6/9jQPIe1MoQDny
NxZyTIAACAjtcfiHmyyJaNPFgVfR4gJ/htmc+6UCgYEA2ObiWdsOEvHn3/gSUO9Y
+22JI3qj+dJ9rwVtNBp8ToxI2QMkY9JmTiSjtTLpdq1dw8GPGOsZmX2ibb48EIHO
Sq3KjtgscAwmgefv4HU6ozYdT0813BI+u2BR9piiTO0/dA3VR8fi8kzBZn/lv1DE
/gWbA13iV9VhOm39EKu27bkCgYAwG2PbYVdxQ8MOeZ6FAi7aIyZbmmfYZtAcSbkd
kUFtmslHyh5ZdjzRWg0VrQloK1EWLqvExK2+r+MYwTt1pZO/ZIUE1c1YkhO4h1bb
Eg/3u80J1+rh/EpmB7bbdn7Gmd4Pcm7q6GpeSAW5/MGnKAqC/XjBB3b3CwgsB4Pu
Lq/dIQKBgFAtzA669N7k6RqqH5Vp4qqxHKg/NmiOnSpN7bD9rC+U2cppybFeq+ti
iGvWog1P6Ekz7ZL3vW/AvokvOfgvN2jFW2nJhEyrgTeIVy24slsZ2H57hFwcI67G
OauPYiMoveEIny40isq7UoKsoC9vRYzGwX2836JjynqSdI4csy6U
-----END RSA PRIVATE KEY-----`)

	privKey, err := ReadPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Error reading private key: %v", err)
	}

	encryptedSessionKey, err := EncryptAsym(&privKey.PublicKey, sessionKey)
	if err != nil {
		t.Fatalf("Error encrypting session key: %v", err)
	}

	sessionKey2, err := DecryptAsym(privKey, encryptedSessionKey)
	if err != nil {
		t.Fatalf("Error decrypting session key: %v", err)
	}

	if !reflect.DeepEqual(sessionKey, sessionKey2) {
		t.Error("Original message defer from decryptred message")
	}
}
