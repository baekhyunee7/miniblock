package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

func serializeECDSAPrivateKey(priv *ecdsa.PrivateKey) (string, error) {
	privBytes := crypto.FromECDSA(priv)
	return hex.EncodeToString(privBytes), nil
}

func deserializeECDSAPrivateKey(hexKey string) (*ecdsa.PrivateKey, error) {
	privBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, err
	}
	return crypto.ToECDSA(privBytes)
}

func txOutKey(outId string, outIndex int) string {
	return fmt.Sprintf("%s:%d", outId, outIndex)
}
