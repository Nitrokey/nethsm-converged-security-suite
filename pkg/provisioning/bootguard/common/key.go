//go:generate manifestcodegen

package common

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"math/big"
)

// Key is a public key of an asymmetric crypto keypair.
type Key struct {
	KeyAlg  Algorithm `json:"keyAlg"`
	Version uint8     `require:"0x10"  json:"keyVersion"`
	KeySize BitSize   `json:"keyBitsize"`
	Data    []byte    `countValue:"keyDataSize()" json:"keyData"`
}

// BitSize is a size in bits.
type BitSize uint16

// InBits returns the size in bits.
func (ks BitSize) InBits() uint16 {
	return uint16(ks)
}

// InBytes returns the size in bytes.
func (ks BitSize) InBytes() uint16 {
	return uint16(ks >> 3)
}

// SetInBits sets the size in bits.
func (ks *BitSize) SetInBits(amountOfBits uint16) {
	*ks = BitSize(amountOfBits)
}

// SetInBytes sets the size in bytes.
func (ks *BitSize) SetInBytes(amountOfBytes uint16) {
	*ks = BitSize(amountOfBytes << 3)
}

// keyDataSize returns the expected length of Data for specified
// KeyAlg and KeySize.
func (k Key) keyDataSize() int64 {
	switch k.KeyAlg {
	case AlgRSA:
		return int64(k.KeySize.InBytes()) + 4
	}
	return -1
}

// PubKey parses Data into crypto.PublicKey.
func (k Key) PubKey() (crypto.PublicKey, error) {
	expectedSize := int(k.keyDataSize())
	if expectedSize < 0 {
		return nil, fmt.Errorf("unexpected algorithm: %s", k.KeyAlg)
	}
	if len(k.Data) != expectedSize {
		return nil, fmt.Errorf("unexpected size: expected:%d, received %d", expectedSize, len(k.Data))
	}

	switch k.KeyAlg {
	case AlgRSA:
		result := &rsa.PublicKey{
			N: new(big.Int).SetBytes(reverseBytes(k.Data[4:])),
			E: int(binaryOrder.Uint32(k.Data)),
		}
		return result, nil
	}

	return nil, fmt.Errorf("unexpected TPM algorithm: %s", k.KeyAlg)
}

func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for idx := range b {
		r[idx] = b[len(b)-idx-1]
	}
	return r
}

// SetPubKey sets Data the value corresponding to passed `key`.
func (k *Key) SetPubKey(key crypto.PublicKey) error {
	k.Version = 0x10

	switch key := key.(type) {
	case *rsa.PublicKey:
		k.KeyAlg = AlgRSA
		n := key.N.Bytes()
		k.KeySize.SetInBytes(uint16(len(n)))
		k.Data = make([]byte, 4+len(n))
		binaryOrder.PutUint32(k.Data, uint32(key.E))
		copy(k.Data[4:], reverseBytes(n))
		return nil
	}

	return fmt.Errorf("unexpected key type: %T", key)
}

//PrintBPMPubKey prints the BPM public signing key hash to fuse into the Intel ME
func (k *Key) PrintBPMPubKey(bpmAlg Algorithm) error {
	buf := new(bytes.Buffer)
	if len(k.Data) > 1 {
		hash, err := bpmAlg.Hash()
		if err != nil {
			return err
		}
		if k.KeyAlg == AlgRSA {
			if err := binary.Write(buf, binary.LittleEndian, k.Data[4:]); err != nil {
				return err
			}
			hash.Write(buf.Bytes())
			fmt.Printf("   Boot Policy Manifest Pubkey Hash: 0x%x\n", hash.Sum(nil))
		} else {
			fmt.Printf("   Boot Policy Manifest Pubkey Hash: Unknown Algorithm\n")
		}
	} else {
		fmt.Printf("   Boot Policy Pubkey Hash: No km public key set in KM\n")
	}

	return nil
}

//PrintKMPubKey prints the KM public signing key hash to fuse into the Intel ME
func (k *Key) PrintKMPubKey(kmAlg Algorithm) error {
	buf := new(bytes.Buffer)
	if len(k.Data) > 1 {
		if k.KeyAlg == AlgRSA {
			if err := binary.Write(buf, binary.LittleEndian, k.Data[4:]); err != nil {
				return err
			}
			if err := binary.Write(buf, binary.LittleEndian, k.Data[:4]); err != nil {
				return err
			}
			if kmAlg != AlgSHA256 {
				return fmt.Errorf("KM public key hash algorithm must be SHA256")
			}
			hash, err := kmAlg.Hash()
			if err != nil {
				return err
			}
			hash.Write(buf.Bytes())
			fmt.Printf("   Key Manifest Pubkey Hash: 0x%x\n", hash.Sum(nil))
		} else {
			fmt.Printf("   Key Manifest Pubkey Hash: Unsupported Algorithm\n")
		}
	} else {
		fmt.Printf("   Key Manifest Pubkey Hash: No km public key set in KM\n")
	}

	return nil
}
