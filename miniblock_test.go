package main

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestTrasaction(t *testing.T) {
	privateKeys := [2]*ecdsa.PrivateKey{}
	addrs := [2]string{}
	txIds := [2]string{}
	for i := 0; i < 2; i++ {
		privateKey, err := crypto.GenerateKey()
		assert.Nil(t, err)
		publicKey := privateKey.Public()
		publicKeyECDSA := publicKey.(*ecdsa.PublicKey)
		privateKeys[i] = privateKey
		address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
		addrs[i] = address
		key, _ := serializeECDSAPrivateKey(privateKey)
		newBlock := GBlockChain.generateNextBlock(key)
		assert.NotNil(t, newBlock)
		txIds[i] = newBlock.Data[0].Id
	}
	tx := &Transaction{
		TxIns: []TxIn{
			{
				TxOutId:    txIds[0],
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
	GBlockChain.handleNewTx(tx)
	key, _ := serializeECDSAPrivateKey(privateKeys[0])
	newBlock := GBlockChain.generateNextBlock(key)
	assert.NotNil(t, newBlock)
}

func TestSerializeKey(t *testing.T) {
	priv, err := crypto.GenerateKey()
	assert.Nil(t, err)
	pemStr, err := serializeECDSAPrivateKey(priv)
	assert.Nil(t, err)
	_, err = deserializeECDSAPrivateKey(pemStr)
	assert.Nil(t, err)
}
