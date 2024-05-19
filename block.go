package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
)

var (
	GBlockChain                      = newBlockChain()
	DIFFICULTY_ADJUST_BLOCK_INTERVAL = 10
	DIFFICULTY_ADJUST_TIME_INTERVAL  = 10
	MINE_AMOUNT                      = 100
)

type Block struct {
	Index        int64          `json:"index"`
	Data         []*Transaction `json:"data"`
	Ts           int64          `json:"ts"`
	Hash         string         `json:"hash"`
	PreviousHash string         `json:"previousHash"`
	Difficulty   int            `json:"diffculty"`
	Nonce        int64          `json:"nonce"`
}

func (b *Block) equal(o *Block) bool {
	txData, _ := json.Marshal(b.Data)
	oTxData, _ := json.Marshal(o.Data)
	return b.Index == o.Index &&
		string(txData) == string(oTxData) &&
		b.Ts == o.Ts &&
		b.Hash == o.Hash &&
		b.PreviousHash == o.PreviousHash &&
		b.Difficulty == o.Difficulty
}

func calculateHash(index int64, previousHash string, ts int64, data []*Transaction, difficulty int, nonce int64) string {
	bs := []byte{}
	bs = append(bs, int64ToBytes(index)...)
	bs = append(bs, []byte(previousHash)...)
	bs = append(bs, int64ToBytes(ts)...)
	bs = append(bs, int64ToBytes(int64(difficulty))...)
	bs = append(bs, int64ToBytes(nonce)...)
	for _, tx := range data {
		bs = append(bs, []byte(tx.Id)...)
	}
	h := sha256.New()
	_, err := h.Write(bs)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func newBlock(index int64, previousHash string, ts int64, difficulty int, nonce int64, data []*Transaction) *Block {
	b := &Block{
		Index:        index,
		Data:         data,
		Ts:           ts,
		PreviousHash: previousHash,
		Hash:         calculateHash(index, previousHash, ts, data, difficulty, nonce),
		Difficulty:   difficulty,
		Nonce:        nonce,
	}
	return b
}

func int64ToBytes(n int64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, uint64(n))
	return bytes
}

type BlockChain struct {
	mux           sync.Mutex
	blocks        []*Block
	unSpentTxOuts map[string]*UnSpentTxOut
}

func newBlockChain() *BlockChain {
	genesisblock := newBlock(0, "", 0, 1, 0, []*Transaction{})
	return &BlockChain{
		blocks:        []*Block{genesisblock},
		unSpentTxOuts: map[string]*UnSpentTxOut{},
	}
}

func (c *BlockChain) generateNextBlock(data []*Transaction) *Block {
	c.mux.Lock()
	lastBlock := c.blocks[len(c.blocks)-1]
	diffculty := c.getDifficulty()
	c.mux.Unlock()
	ts := time.Now().Unix()
	index := lastBlock.Index + 1
	newBlock := getBlock(index, lastBlock.Hash, ts, diffculty, data)
	c.mux.Lock()
	defer c.mux.Unlock()
	c.addBlock(newBlock)
	c.broadcastLatest()
	return newBlock
}

func (c *BlockChain) addBlock(b *Block) bool {
	lastBlock := c.blocks[len(c.blocks)-1]
	if isValidBlock(lastBlock, b) {
		if !c.processTxs(b.Data, b.Index) {
			return false
		}
		log.Println("add block success")
		c.blocks = append(c.blocks, b)
		return true
	} else {
		log.Println("invalid block")
		return false
	}
}

func getBlock(index int64, previousHash string, ts int64, difficulty int, data []*Transaction) *Block {
	nonce := int64(0)
	for {
		hash := calculateHash(index, previousHash, ts, data, difficulty, nonce)
		if hashMatchDiffculty(hash, difficulty) {
			return newBlock(index, previousHash, ts, difficulty, nonce, data)
		}
		nonce++
	}
}

func hashMatchDiffculty(hash string, diffculty int) bool {
	if diffculty == 0 {
		return true
	}
	for _, b := range hash {
		if b == '0' {
			diffculty -= 1
			if diffculty <= 0 {
				return true
			}
		}
	}
	return false
}

func isValidBlock(preBlock *Block, b *Block) bool {
	if preBlock.Index+1 != b.Index {
		return false
	}
	if preBlock.Hash != b.PreviousHash {
		return false
	}
	if !(time.Now().Unix() > b.Ts-10 && b.Ts > preBlock.Ts-10) {
		return false
	}
	if b.Hash != calculateHash(b.Index, b.PreviousHash, b.Ts, b.Data, b.Difficulty, b.Nonce) {
		return false
	}
	return true
}

func (c *BlockChain) isValidChain(blocks []*Block) bool {
	if len(blocks) < 1 {
		return false
	}
	if !c.blocks[0].equal(blocks[0]) {
		return false
	}
	for i := 1; i < len(blocks); i++ {
		if !isValidBlock(blocks[i-1], blocks[i]) {
			return false
		}
	}
	return true
}

func (c *BlockChain) replaceChain(blocks []*Block) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.isValidChain(blocks) && len(blocks) > len(c.blocks) {
		c.blocks = blocks
		log.Printf("repalce chain success")
		c.broadcastLatest()
	} else {
		log.Println("received chain invalid")
	}
}

func (c *BlockChain) processTxs(txs []*Transaction, index int64) bool {
	if c.txsIsValid(txs, index) {
		c.updateUnspentTxOut(txs)
		return true
	}
	log.Println("transactions is not valid")
	return false
}

func (c *BlockChain) broadcastLatest() {
	data := c.marshalLastBlock()
	GPeers.broadcast(&PeerMsg{
		Id:   ResponseLatest,
		Data: data,
	})
}

func (c *BlockChain) marshal() []byte {
	c.mux.Lock()
	defer c.mux.Unlock()
	ret, err := json.Marshal(c.blocks)
	if err != nil {
		log.Fatalf("json marshal err: %v", err)
	}
	return ret
}

func (c *BlockChain) marshalLastBlockSafe() []byte {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.marshalLastBlock()
}

func (c *BlockChain) marshalLastBlock() []byte {
	ret, err := json.Marshal(c.blocks[len(c.blocks)-1])
	if err != nil {
		log.Fatalf("json marshal err: %v", err)
	}
	return ret
}

func (c *BlockChain) getDifficulty() int {
	lastBlock := c.blocks[len(c.blocks)-1]
	if lastBlock.Index != 0 && lastBlock.Index%int64(DIFFICULTY_ADJUST_BLOCK_INTERVAL) == 0 {
		previousBlock := c.blocks[len(c.blocks)-DIFFICULTY_ADJUST_BLOCK_INTERVAL]
		timeSpent := lastBlock.Ts - previousBlock.Ts
		timeExpected := DIFFICULTY_ADJUST_BLOCK_INTERVAL * DIFFICULTY_ADJUST_TIME_INTERVAL
		if timeSpent < int64(timeExpected)/2 {
			return lastBlock.Difficulty + 1
		}
		if timeSpent > int64(timeExpected)*2 {
			return lastBlock.Difficulty - 1
		}
	}
	return lastBlock.Difficulty
}

func (c *BlockChain) handleResponseLatest(b *Block, conn *websocket.Conn) {
	c.mux.Lock()
	defer c.mux.Unlock()
	latestBlock := c.blocks[len(c.blocks)-1]
	if latestBlock.Index < b.Index {
		if b.PreviousHash == latestBlock.Hash {
			if c.addBlock(b) {
				conn.WriteJSON(&PeerMsg{
					Id:   ResponseLatest,
					Data: c.marshalLastBlock(),
				})
			}
		} else {
			conn.WriteJSON(&PeerMsg{
				Id: QueryAll,
			})
		}
	} else {
		log.Printf("received block is not ahead of local")
	}
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

func (c *BlockChain) txsIsValid(txs []*Transaction, index int64) bool {
	if !coinbaseTxIsValid(txs[0], index) {
		return false
	}
	for _, tx := range txs[1:] {
		spent := int64(0)
		if tx.getTxId() != tx.Id {
			return false
		}
		for _, in := range tx.TxIns {
			key := txOutKey(in.TxOutId, in.TxOutIndex)
			unSpentTxOut, ok := c.unSpentTxOuts[key]
			if !ok {
				return false
			}
			addr := unSpentTxOut.Address
			hash := crypto.Keccak256Hash([]byte(tx.Id))
			pubKey, err := crypto.SigToPub(hash.Bytes(), []byte(in.Signature))
			if err != nil {
				return false
			}
			recoveredAddr := crypto.PubkeyToAddress(*pubKey)
			if addr != recoveredAddr.Hex() {
				return false
			}
			spent += unSpentTxOut.Amount
		}
		for _, out := range tx.TxOuts {
			spent -= out.Amount
		}
		if spent != 0 {
			return false
		}
	}
	return true
}

func (c *BlockChain) updateUnspentTxOut(txs []*Transaction) {
	for _, tx := range txs {
		for _, in := range tx.TxIns {
			delete(c.unSpentTxOuts, txOutKey(in.TxOutId, in.TxOutIndex))
		}
		for i, out := range tx.TxOuts {
			key := txOutKey(tx.Id, i)
			c.unSpentTxOuts[key] = &UnSpentTxOut{
				TxOutId:    tx.Id,
				TxOutIndex: i,
				Address:    out.Address,
				Amount:     out.Amount,
			}
		}
	}
}
