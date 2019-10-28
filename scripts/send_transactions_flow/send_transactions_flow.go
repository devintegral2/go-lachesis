package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"flag"
	"log"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
)

const (
	ChainID = 1555
)

type (
	Node struct {
		AddrURL		string

		SendCount	int64
		FirstSend	time.Time
		ErrorsCount int64

		LastSend	time.Time

		client *ethclient.Client
	}

	Account struct {
		Address		*common.Address
		PvtKey		*ecdsa.PrivateKey
	}
)

var (
	nodes				[]Node
	accounts			[]Account
	maxFlowSpeedInSec 	int64
)

// InitClient initialize node client connection
func (n *Node) InitClient() {
	var err error
	n.client, err = ethclient.Dial(n.AddrURL)
	if err != nil {
		log.Panicf("Error connect to node %s: %s", n.AddrURL, err)
	}
}

// CalcAvgFlowSpeed calculate current average flow speed
func (n *Node) CalcAvgFlowSpeed() int64 {
	if n.SendCount <= 0 {
		return 0
	}
	return int64(float64(n.SendCount) / (time.Now().UTC().Sub(n.FirstSend)).Seconds())
}

// DelayForNextSend calculate delay if flow speed great then max flow speed
func (n *Node) DelayForNextSend() time.Duration {
	delay := float64( node.SendCount / maxFlowSpeedInSec ) - time.Now().UTC().Sub(node.FirstSend).Seconds()
	return time.Duration(delay) * time.Second
}

// SendRandomTransaction send transfer from random account to random account
func  (n *Node) SendRandomTransaction() {
	from := getRandomAccount()
	to := getRandomAccount()

	nonce, err := n.client.PendingNonceAt(context.Background(), *from.Address)
	if err != nil {
		log.Printf("PendingNonceAt error: %s", err)
		n.ErrorsCount++
		return
	}

	value := big.NewInt(100000000000000000) // in wei (0.1 eth)
	gasLimit := uint64(21000)               // in units
	gasPrice := big.NewInt(20000000)

	tx := types.NewTransaction(nonce, *to.Address, value, gasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(ChainID)), from.PvtKey)

	err = n.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Printf("Send transaction error: %s", err)
		n.ErrorsCount++
		return
	}

	n.SendCount++
	n.LastSend = time.Now().UTC()
}

// SendLoopForNode create infinity loop of sending random transactions on node
func (n *Node) SendLoopForNode(wg *sync.WaitGroup) {
	for {
		// Check MaxFlowSpeed
		if maxFlowSpeedInSec > 0 {
			currentFlowSpeed := n.CalcAvgFlowSpeed()
			if currentFlowSpeed > maxFlowSpeedInSec {
				time.Sleep(n.DelayForNextSend())
				continue
			}
		}

		n.SendRandomTransaction()
	}

	wg.Done()
}

func main() {
	var (
		nodesListFile   string
		count			int
	)

	flag.StringVar(&nodesListFile, 		"nodes",		"", 	"Path of nodes (validators) list file")
	flag.IntVar(&count, 				"count", 		1, 	"Count of generated accounts")
	flag.Int64Var(&maxFlowSpeedInSec, 	"max_flow",	0,	"Max send count in second for one node")
	flag.Parse()

	// Parse nodes file
	parseNodesFile(nodesListFile)

	// Parse accounts file
	generateAccounts(count)

	// Run main loop of sending
	mainLoop()
}

func parseNodesFile(fileName string) {
	nodes := make([]Node, 0, 30)
	nodesFile, err := os.Open(fileName)
	if err != nil {
		log.Panicf("Can not open nodes file: %s", err)
	}
	defer nodesFile.Close()

	nodesBuf := bufio.NewReader(nodesFile)
	for {
		line, err := nodesBuf.ReadString('\n')
		if err != nil {
			break
		}

		newNode := Node{
			AddrURL:   line,
			SendCount: 0,
			FirstSend: time.Now().UTC(),
			LastSend:  time.Now().UTC(),
		}
		newNode.InitClient()
		nodes = append(nodes, newNode)
	}
}

func generateAccounts(count int) {
	accounts := make([]Account, count, count)
	for i := 0; i < count; i++ {
		key, err := crypto.GenerateKey()
		if err != nil {
			log.Printf("GenerateKey error: %s", err)
			i--
			continue
		}

		address := crypto.PubkeyToAddress(key.PublicKey)
		accounts[i].Address = &address
		accounts[i].PvtKey = key
	}
}

func mainLoop() {
	// Create goroutine for every node
	wg := sync.WaitGroup{}

	for _, n := range nodes {
		wg.Add(1)
		go n.SendLoopForNode(&wg)
	}

	wg.Wait()
}

func getRandomAccount() *Account {
	i := rand.Intn(len(accounts) - 1)
	return &accounts[i]
}
