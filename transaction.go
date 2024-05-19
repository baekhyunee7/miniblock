package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
)

type TxOut struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
}

type TxIn struct {
	TxOutId    string `json:"tx_out_id"`
	TxOutIndex int    `json:"tx_out_index"`
	Signature  string `json:"signature"`
}

type Transaction struct {
	Id     string  `json:"id"`
	TxIns  []TxIn  `json:"txins"`
	TxOuts []TxOut `json:"txouts"`
}

func (t *Transaction) getTxId() string {
	bs := []byte{}
	for _, in := range t.TxIns {
		bs = append(bs, []byte(in.TxOutId)...)
		bs = append(bs, int64ToBytes(int64(in.TxOutIndex))...)
	}
	for _, out := range t.TxOuts {
		bs = append(bs, []byte(out.Address)...)
		bs = append(bs, int64ToBytes(out.Amount)...)
	}
	h := sha256.New()
	if _, err := h.Write(bs); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

type UnSpentTxOut struct {
	TxOutId    string
	TxOutIndex int
	Address    string
	Amount     int64
}

func signTxIn(tx *Transaction, inIndex int, privateKey *ecdsa.PrivateKey, unSpent map[string]*UnSpentTxOut) string {
	txIn := tx.TxIns[inIndex]
	data := tx.Id
	unSpentTxOut := findUnspentTxOut(txIn.TxOutId, txIn.TxOutIndex, unSpent)
	if unSpentTxOut == nil {
		// todo
	}
	// addr := unSpentTxOut.Address
	hash := crypto.Keccak256Hash([]byte(data))
	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		log.Fatal(err)
	}
	return string(signature)
}

func findUnspentTxOut(outId string, outIndex int, unSpent map[string]*UnSpentTxOut) *UnSpentTxOut {
	key := txOutKey(outId, outIndex)
	return unSpent[key]
}

func txOutKey(outId string, outIndex int) string {
	return fmt.Sprintf("%s:%d", outId, outIndex)
}
