package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

var (
	httPort int
	wsPort  int
)

func init() {
	flag.IntVar(&httPort, "hp", 3001, "http port")
	flag.IntVar(&wsPort, "wp", 3002, "ws port")
}

func main() {
	flag.Parse()
	app := gin.Default()
	app.GET("/blocks", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/json", GBlockChain.marshal())
	})
	app.POST("/mineBlock", func(ctx *gin.Context) {
		data, err := ctx.GetRawData()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		newBlock := GBlockChain.generateNextBlock(data)
		ctx.JSON(http.StatusOK, newBlock)
	})
	app.GET("/peers", func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/json", GPeers.marshal())
	})
	app.POST("/addPeer", func(ctx *gin.Context) {
		data, err := ctx.GetRawData()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		GPeers.addPeer(string(data))
		ctx.Data(http.StatusOK, "application/json", nil)
	})
	go func() {
		http.HandleFunc("/ws", ws)
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", wsPort), nil))
	}()
	app.Run(fmt.Sprintf(":%d", httPort))
}
