package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

type (
	Node struct {
		AddrURL		string

		SendCount	int64
		FirstSend	time.Time
		ErrorsCount int64
		TimeoutCount int64

		LastSend	time.Time

		client *ethclient.Client

		trxs			[]common.Hash
		lastTrxClean	time.Time
	}

	Account struct {
		Address		*common.Address
		PvtKey		*ecdsa.PrivateKey
	}
)

const (
	cleanTrxCounter = 10
)

var (
	nodes				[]Node
	accounts			[]Account
	maxFlowSpeedInSec 	int64

	DonorAddress 		common.Address
	DonorPvtKey			*ecdsa.PrivateKey
)

func init() {
	var err error
	DonorAddress = common.HexToAddress("0xf9352d0ca3820e4b16b5242c74adb0e26471fbea")
	DonorPvtKey, err = crypto.HexToECDSA("ae2037b61158065161bc5eeafe43b227663f1614c123ae5f378e8201d3a5f3e5")
	if err != nil {
		log.Panicf("Error decode donor private key: %s\n", err)
	}
}

// InitClient initialize node client connection
func (n *Node) InitClient() {
	var err error
	n.client, err = ethclient.Dial(n.AddrURL)
	if err != nil {
		log.Panicf("Error connect to node [%s]: %s", n.AddrURL, err)
	}
	n.lastTrxClean = time.Now().UTC()
}

// CalcAvgFlowSpeed calculate current average flow speed
func (n *Node) CalcAvgFlowSpeed() float64 {
	if n.SendCount <= 0 {
		return 0
	}
	return float64(n.SendCount) / (time.Now().UTC().Sub(n.FirstSend)).Seconds()
}

// DelayForNextSend calculate delay if flow speed great then max flow speed
func (n *Node) DelayForNextSend() time.Duration {
	delay := float64( n.SendCount / maxFlowSpeedInSec ) - time.Now().UTC().Sub(n.FirstSend).Seconds()
	return time.Duration(delay) * time.Second
}

// SendRandomTransaction send transfer from random account to random account
func  (n *Node) SendRandomTransaction() (*common.Hash, error) {
	from := getRandomAccount()
	to := getRandomAccount()

	value := big.NewInt(1)
	trxHash, err := n.SendTransfer(from, to, value)
	if err != nil {
		n.ErrorsCount++
		return nil, err
	}

	n.SendCount++
	n.LastSend = time.Now().UTC()

	return trxHash, nil
}

func (n *Node) SendTransfer(from, to *Account, amount *big.Int) (*common.Hash, error) {
	nonce, err := n.client.PendingNonceAt(context.Background(), *from.Address)
	if err != nil {
		// log.Printf("PendingNonceAt error: %s", err)
		return nil, err
	}

	gasLimit := uint64(21000)               // in units
	gasPrice := big.NewInt(0)

	var data []byte
	tx := types.NewTransaction(nonce, *to.Address, amount, gasLimit, gasPrice, data)

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(params.AllEthashProtocolChanges.ChainID), from.PvtKey)
	if err != nil {
		log.Printf("Sign transaction error: %s", from.Address.Hex(), err)
		return nil, err
	}

	err = n.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		// log.Printf("Send transaction error from %s: %s", from.Address.Hex(), err)
		return nil, err
	}
	// fmt.Printf("OK: Transfer %d coins: %s -> %s Trx: %s\n", amount, from.Address.Hex(), to.Address.Hex(), signedTx.Hash().Hex())

	trxHash := signedTx.Hash()
	return &trxHash, nil
}

// SendLoopForNode create infinity loop of sending random transactions on node
func (n *Node) SendLoopForNode(wg *sync.WaitGroup) {
	trxN := 0
	if n.trxs == nil {
		n.trxs = make([]common.Hash, 0)
	}
	n.lastTrxClean = time.Now().UTC()
	for {
		// Check MaxFlowSpeed
		if maxFlowSpeedInSec > 0 {
			currentFlowSpeed := n.CalcAvgFlowSpeed()
			if currentFlowSpeed > float64(maxFlowSpeedInSec) {
				time.Sleep(n.DelayForNextSend())
				continue
			}
		}

		trxHash, err := n.SendRandomTransaction()
		if err == nil {
			n.trxs = append(n.trxs, *trxHash)
		}

		trxN++
		if trxN >= cleanTrxCounter {
			trxN = 0
			for {
				if n.CleanFinishedTransactions() == 0 {
					break
				}
				// log.Printf("Wait finish transactions for node %s (%d)\n", n.AddrURL, len(n.trxs))
				fmt.Print(".")
				time.Sleep(time.Second)
			}
			speed := int64(n.CalcAvgFlowSpeed() * 60)
			fmt.Printf("\n")
			log.Printf("Node %s: ok = %d, \terr = %d, \ttimeouts = %d, \tpending = %d (%d tx/min)\n",
				n.AddrURL, n.SendCount, n.ErrorsCount, n.TimeoutCount, len(n.trxs), speed)
		}
	}

	wg.Done()
}

func (n *Node) CleanFinishedTransactions() int {
	for i := 0; i < len(n.trxs); i++ {
		// Check transaction exists
		t, _, _ := n.client.TransactionByHash(context.Background(), n.trxs[i])
		if t == nil {
			n.ErrorsCount++
		}

		// Check transaction receipt
		r, _ := n.client.TransactionReceipt(context.Background(), n.trxs[i])
		if r != nil || t == nil {
			n.lastTrxClean = time.Now().UTC()

			if i != (len(n.trxs) - 1) {
				n.trxs[i] = n.trxs[len(n.trxs) - 1]
				i--
			}
			n.trxs = n.trxs[0:len(n.trxs) - 1]
		}
	}

	if time.Now().UTC().Sub(n.lastTrxClean) > (time.Minute) {
		// Force clean trxs
		for i := 0; i < len(n.trxs); i++ {
			n.TimeoutCount++
			log.Printf("Trx timeout: %s\n", n.trxs[i].Hex())
		}
		n.trxs = n.trxs[0:0]
		n.lastTrxClean = time.Now().UTC()
	}

	return len(n.trxs)
}


func main() {
	var (
		nodesListFile   string
		count			int
	)

	flag.StringVar(&nodesListFile, 		"nodes",		"", 	"Path of nodes (validators) list file")
	flag.IntVar(&count, 				"acc", 		1, 	"Count of generated accounts")
	flag.Int64Var(&maxFlowSpeedInSec, 	"max_flow",	0,	"Max send count in second for one node")
	flag.Parse()

	if nodesListFile == "" {
		flag.PrintDefaults()
		return
	}

	// Parse nodes file
	parseNodesFile(nodesListFile)

	// Parse accounts file
	generateAccounts(count)

	// Transfer started funds
	transferFunds()

	// Run main loop of sending
	mainLoop()
}

func parseNodesFile(fileName string) {
	nodes = make([]Node, 0, 30)
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
		line = strings.TrimRight(line, "\n")
		line = strings.TrimRight(line, "\r")

		log.Printf("Read node: %s\n", line)

		newNode := Node{
			AddrURL:   line,
			SendCount: 0,
			FirstSend: time.Now().UTC(),
			LastSend:  time.Now().UTC(),
		}
		newNode.InitClient()

		log.Printf("Append node: %+v\n", newNode)
		nodes = append(nodes, newNode)
	}
}

func generateAccounts(count int) {
	accounts = make([]Account, count, count)
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
	log.Printf("Account addr: %x\n", accounts[0].Address)
	log.Printf("Account pvtkey: %x\n", accounts[0].PvtKey.D.Bytes())
}

func transferFunds() {
	fmt.Printf("Create funds...\n")
	node := nodes[0]
	for _, acc := range accounts {
		from := Account{
			Address: &DonorAddress,
			PvtKey:  DonorPvtKey,
		}

		trxHash, err := node.SendTransfer(&from, &acc, big.NewInt(100000000))
		if err != nil {
			log.Panicf("Error transfer funds from donor: %s", err)
		}
		fmt.Printf("Create funds: %s\n", acc.Address.Hex())

		node.trxs = append(node.trxs, *trxHash)
	}
	fmt.Printf("Create funds done\n")
	fmt.Printf("Wait funds transactions finished...\n")
	for {
		if node.CleanFinishedTransactions() <= 0 {
			break
		}
		// log.Printf("Wait for finish %d transactions\n", len(node.trxs))
		fmt.Print(".")
		time.Sleep(3*time.Second)
	}
	fmt.Printf("Funds transactions finished\n")
}

func mainLoop() {
	log.Println("Main loop")
	// Create goroutine for every node
	wg := &sync.WaitGroup{}

	for _, n := range nodes {
		log.Println("New node")
		wg.Add(1)
		node := n
		go node.SendLoopForNode(wg)
	}

	log.Println("Wait finish...")
	wg.Wait()
	log.Println("Finished")
}

func getRandomAccount() *Account {
	i := rand.Intn(len(accounts) - 1)
	return &accounts[i]
}
