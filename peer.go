package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	GPeers = &Peers{
		peers: map[*websocket.Conn]struct{}{},
	}
)

const (
	QueryLatest = iota
	ResponseLatest
	QueryAll
	ResponseAll
)

type Peers struct {
	mux   sync.Mutex
	peers map[*websocket.Conn]struct{}
}

func (p *Peers) marshal() []byte {
	p.mux.Lock()
	defer p.mux.Unlock()
	addrs := []string{}
	for peer := range p.peers {
		addrs = append(addrs, peer.RemoteAddr().String())
	}
	data, err := json.Marshal(addrs)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func (p *Peers) addPeer(url string) {
	url = fmt.Sprintf("%s/ws", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("add peer %s fail: %v", url, err)
		return
	}
	p.addConn(conn)
	go p.readLoop(conn)
	conn.WriteJSON(&PeerMsg{
		Id: QueryLatest,
	})
}

func (p *Peers) addConn(conn *websocket.Conn) {
	p.mux.Lock()
	defer p.mux.Unlock()
	p.peers[conn] = struct{}{}
}

func (p *Peers) removeConn(conn *websocket.Conn) {
	log.Printf("ws disconencted %s", conn.RemoteAddr().String())
	conn.Close()
	p.mux.Lock()
	defer p.mux.Unlock()
	delete(p.peers, conn)
}

type PeerMsg struct {
	Id   int    `json:"id"`
	Data []byte `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ws(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade err: %v", err)
		return
	}
	log.Printf("ws accept connection %s", conn.RemoteAddr().String())
	GPeers.addConn(conn)
	go GPeers.readLoop(conn)
}

func (p *Peers) readLoop(conn *websocket.Conn) {
	defer func() {
		p.removeConn(conn)
	}()
	for {
		var msg PeerMsg
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("read err: %v", err)
			break
		}
		log.Printf("recv: %+v", msg)
		switch msg.Id {
		case QueryLatest:
			conn.WriteJSON(&PeerMsg{
				Id:   ResponseLatest,
				Data: GBlockChain.marshalLastBlock(),
			})
		case ResponseLatest:
			var block Block
			err := json.Unmarshal(msg.Data, &block)
			if err != nil {
				log.Printf("unmarshal fail: %v", err)
				continue
			}
			GBlockChain.handleResponseLatest(&block, conn)
		case QueryAll:
			conn.WriteJSON(&PeerMsg{
				Id:   ResponseAll,
				Data: GBlockChain.marshal(),
			})
		case ResponseAll:
			blocks := []*Block{}
			err := json.Unmarshal(msg.Data, &blocks)
			if err != nil {
				log.Printf("unmarshal fail: %v", err)
				continue
			}
			GBlockChain.replaceChain(blocks)
		}
	}
}

func (p *Peers) broadcast(msg *PeerMsg) {
	p.mux.Lock()
	defer p.mux.Unlock()
	for peer := range p.peers {
		err := peer.WriteJSON(msg)
		if err != nil {
			log.Printf("write message fail, err: %v", err)
		}
	}
}
