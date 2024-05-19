package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	GBlockChain                      = newBlockChain()
	DIFFICULTY_ADJUST_BLOCK_INTERVAL = 10
	DIFFICULTY_ADJUST_TIME_INTERVAL  = 10
)

type Block struct {
	Index        int64  `json:"index"`
	Data         []byte `json:"data"`
	Ts           int64  `json:"ts"`
	Hash         string `json:"hash"`
	PreviousHash string `json:"previousHash"`
	Difficulty   int    `json:"diffculty"`
	Nonce        int64  `json:"nonce"`
}

func (b *Block) equal(o *Block) bool {
	return b.Index == o.Index &&
		string(b.Data) == string(o.Data) &&
		b.Ts == o.Ts &&
		b.Hash == o.Hash &&
		b.PreviousHash == o.PreviousHash &&
		b.Difficulty == o.Difficulty
}

func calculateHash(index int64, previousHash string, ts int64, data []byte, difficulty int, nonce int64) string {
	bs := []byte{}
	bs = append(bs, int64ToBytes(index)...)
	bs = append(bs, []byte(previousHash)...)
	bs = append(bs, int64ToBytes(ts)...)
	bs = append(bs, int64ToBytes(int64(difficulty))...)
	bs = append(bs, int64ToBytes(nonce)...)
	bs = append(bs, data...)
	h := sha256.New()
	_, err := h.Write(bs)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func newBlock(index int64, previousHash string, ts int64, difficulty int, nonce int64, data []byte) *Block {
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
	mux    sync.Mutex
	blocks []*Block
}

func newBlockChain() *BlockChain {
	genesisblock := newBlock(0, "", 0, 1, 0, []byte("genesis"))
	return &BlockChain{
		blocks: []*Block{genesisblock},
	}
}

func (c *BlockChain) generateNextBlock(data []byte) *Block {
	c.mux.Lock()
	lastBlock := c.blocks[len(c.blocks)-1]
	diffculty := c.getDifficulty()
	c.mux.Unlock()
	ts := time.Now().Unix()
	index := lastBlock.Index + 1
	newBlock := getBlock(index, lastBlock.Hash, ts, diffculty, data)
	c.addBlock(newBlock)
	c.broadcastLatest()
	return newBlock
}

func (c *BlockChain) addBlock(b *Block) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.addBlockUnsafe(b)
}

func (c *BlockChain) addBlockUnsafe(b *Block) bool {
	lastBlock := c.blocks[len(c.blocks)-1]
	if isValidBlock(lastBlock, b) {
		log.Println("add block success")
		c.blocks = append(c.blocks, b)
		return true
	} else {
		log.Println("invalid block")
		return false
	}
}

func getBlock(index int64, previousHash string, ts int64, difficulty int, data []byte) *Block {
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

// 已经上锁
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

func (c *BlockChain) marshalLastBlock() []byte {
	c.mux.Lock()
	defer c.mux.Unlock()
	ret, err := json.Marshal(c.blocks[len(c.blocks)-1])
	if err != nil {
		log.Fatalf("json marshal err: %v", err)
	}
	return ret
}

// 在锁中
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
			if c.addBlockUnsafe(b) {
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
