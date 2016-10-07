package util

import "io"
import "errors"
import "crypto/rand"
import "crypto/sha512"
import "golang.org/x/crypto/scrypt"
import "crypto/md5"
import "encoding/hex"

func RandomSalt(size int) string {

	buf := make([]byte, size)
	io.ReadFull(rand.Reader, buf)
	salt := EncodeBase64(buf)
	return salt[0:size]
}

// Recommended password generator
func Scrypt(password string, salt string) (string, error) {

	data, err := scrypt.Key([]byte(password), []byte(salt), 16384, 8, 1, 64)

	if err != nil {
		return "", err
	}
	hstr := EncodeBase64(data)
	return hstr, nil
}

// Linux shadow compatible sha-512 password
func Sha512Crypt(password string, salt string) (string, error) {
	rounds := 5000

	passwordb := []byte(password)
	saltb := []byte(salt)

	if len(saltb) > 16 {
		return "", errors.New("salt must not exceed 16 bytes")
	}

	// B
	b := sha512.New()
	b.Write(passwordb)
	b.Write(saltb)
	b.Write(passwordb)
	bsum := b.Sum(nil)

	// A
	a := sha512.New()
	a.Write(passwordb)
	a.Write(saltb)
	repeat(a, bsum, len(passwordb))

	plen := len(passwordb)
	for plen != 0 {
		if (plen & 1) != 0 {
			a.Write(bsum)
		} else {
			a.Write(passwordb)
		}
		plen = plen >> 1
	}

	asum := a.Sum(nil)

	// DP
	dp := sha512.New()
	for i := 0; i < len(passwordb); i++ {
		dp.Write(passwordb)
	}

	dpsum := dp.Sum(nil)

	// P
	p := make([]byte, len(passwordb))
	repeatTo(p, dpsum)

	// DS
	ds := sha512.New()
	for i := 0; i < (16 + int(asum[0])); i++ {
		ds.Write(saltb)
	}

	dssum := ds.Sum(nil)[0:len(saltb)]

	// S
	s := make([]byte, len(saltb))
	repeatTo(s, dssum)

	// C
	cur := asum[:]
	for i := 0; i < rounds; i++ {
		c := sha512.New()
		if (i & 1) != 0 {
			c.Write(p)
		} else {
			c.Write(cur)
		}
		if (i % 3) != 0 {
			c.Write(s)
		}
		if (i % 7) != 0 {
			c.Write(p)
		}
		if (i & 1) == 0 {
			c.Write(p)
		} else {
			c.Write(cur)
		}
		cur = c.Sum(nil)[:]
	}

	// Transposition
	transpose512(cur)

	// Hash
	//hstr := base64.RawStdEncoding.EncodeToString(cur)
	hstr := EncodeBase64(cur)
	return hstr, nil
}

func repeat(w io.Writer, b []byte, sz int) {
	var i int
	for i = 0; (i + len(b)) <= sz; i += len(b) {
		w.Write(b)
	}
	w.Write(b[0 : sz-i])
}

func repeatTo(out []byte, b []byte) {
	if len(b) == 0 {
		return
	}

	var i int
	for i = 0; (i + len(b)) <= len(out); i += len(b) {
		copy(out[i:], b)
	}
	copy(out[i:], b)
}

func transpose512(b []byte) {
	b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19], b[20], b[21], b[22], b[23], b[24], b[25], b[26], b[27], b[28], b[29], b[30], b[31], b[32], b[33], b[34], b[35], b[36], b[37], b[38], b[39], b[40], b[41], b[42], b[43], b[44], b[45], b[46], b[47], b[48], b[49], b[50], b[51], b[52], b[53], b[54], b[55], b[56], b[57], b[58], b[59], b[60], b[61], b[62], b[63] =
		b[42], b[21], b[0], b[1], b[43], b[22], b[23], b[2], b[44], b[45], b[24], b[3], b[4], b[46], b[25], b[26], b[5], b[47], b[48], b[27], b[6], b[7], b[49], b[28], b[29], b[8], b[50], b[51], b[30], b[9], b[10], b[52], b[31], b[32], b[11], b[53], b[54], b[33], b[12], b[13], b[55], b[34], b[35], b[14], b[56], b[57], b[36], b[15], b[16], b[58], b[37], b[38], b[17], b[59], b[60], b[39], b[18], b[19], b[61], b[40], b[41], b[20], b[62], b[63]
}

const bmap = "./0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Encodes a byte string using the sha2-crypt base64 variant.
func EncodeBase64(b []byte) string {
	o := make([]byte, len(b)/3*4+4)

	for i, j := 0, 0; i < len(b); {
		b1 := b[i]
		b2 := byte(0)
		b3 := byte(0)

		if (i + 1) < len(b) {
			b2 = b[i+1]
		}
		if (i + 2) < len(b) {
			b3 = b[i+2]
		}

		o[j] = bmap[(b1 & 0x3F)]
		o[j+1] = bmap[((b1&0xC0)>>6)|((b2&0x0F)<<2)]
		o[j+2] = bmap[((b2&0xF0)>>4)|((b3&0x03)<<4)]
		o[j+3] = bmap[(b3&0xFC)>>2]
		i += 3
		j += 4
	}

	s := string(o)
	return s[0 : len(b)*4/3-(len(b)%4)+1]
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
