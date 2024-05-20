package main

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
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

func coinbaseTxIsValid(tx *Transaction, index int64) bool {
	if tx.getTxId() != tx.Id {
		return false
	}
	if len(tx.TxIns) != 1 {
		return false
	}
	if tx.TxIns[0].TxOutIndex != int(index) {
		return false
	}
	if len(tx.TxOuts) != 1 {
		return false
	}
	if tx.TxOuts[0].Amount != int64(MINE_AMOUNT) {
		return false
	}
	return true
}
