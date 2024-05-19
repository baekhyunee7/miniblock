package main

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestTrasaction(t *testing.T) {
	txs := [2]*Transaction{}
	privateKeys := [2]*ecdsa.PrivateKey{}
	addrs := [2]string{}
	for i := 0; i < 2; i++ {
		privateKey, err := crypto.GenerateKey()
		assert.Nil(t, err)
		publicKey := privateKey.Public()
		publicKeyECDSA := publicKey.(*ecdsa.PublicKey)
		privateKeys[i] = privateKey
		address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
		addrs[i] = address
		tx := &Transaction{
			TxIns: []TxIn{
				{
					TxOutId:    "test_out_id",
					TxOutIndex: i + 1,
				},
			},
			TxOuts: []TxOut{
				{
					Address: address,
					Amount:  int64(MINE_AMOUNT),
				},
			},
		}
		tx.Id = tx.getTxId()
		txs[i] = tx
		newBlock := GBlockChain.generateNextBlock([]*Transaction{tx})
		assert.NotNil(t, newBlock)
	}
	coinBaseTx := &Transaction{
		TxIns: []TxIn{
			{
				TxOutId:    "test_out_id",
				TxOutIndex: 3,
			},
		},
		TxOuts: []TxOut{
			{
				Address: addrs[0],
				Amount:  int64(MINE_AMOUNT),
			},
		},
	}
	coinBaseTx.Id = coinBaseTx.getTxId()
	tx := &Transaction{
		TxIns: []TxIn{
			{
				TxOutId:    txs[0].Id,
				TxOutIndex: 0,
			},
		},
		TxOuts: []TxOut{
			{
				Address: addrs[0],
				Amount:  20,
			},
			{
				Address: addrs[1],
				Amount:  80,
			},
		},
	}
	tx.Id = tx.getTxId()
	hash := crypto.Keccak256Hash([]byte(tx.Id))
	signature, err := crypto.Sign(hash.Bytes(), privateKeys[0])
	assert.Nil(t, err)
	tx.TxIns[0].Signature = string(signature)
	newBlock := GBlockChain.generateNextBlock([]*Transaction{coinBaseTx, tx})
	assert.NotNil(t, newBlock)
}
