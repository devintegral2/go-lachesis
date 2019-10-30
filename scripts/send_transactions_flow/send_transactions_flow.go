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
		AddrURL string

		SendCount        int64
		FirstSend        time.Time
		ErrorsCount      int64
		MissedCount		 int64
		TimeoutCount     int64
		RandomFlowFactor int

		LastSend time.Time

		client *ethclient.Client

		trxs         []common.Hash
		lastTrxClean time.Time
	}

	Account struct {
		Address *common.Address
		PvtKey  *ecdsa.PrivateKey
	}
)

var (
	nodes             []Node
	accounts          []Account
	maxFlowSpeedInMin int

	donorAddress       common.Address
	donorPvtKey        *ecdsa.PrivateKey
	mode               int
	trxBatchSize       int
	trxMaxCount        int
	waitConfirmTimeout = time.Minute
)

func main() {
	var (
		nodesListFile   string
		accCount        int
		donorAddrHex    string
		donorKeyHex     string
		randomFlowDelta int
	)

	flag.StringVar(&nodesListFile, "nodes", "", "Path of nodes (validators) list file")
	flag.IntVar(&trxBatchSize, "trx_batch", 10, "Count of transactions in batch")
	flag.IntVar(&accCount, "acc", 1, "Count of generated accounts")
	flag.IntVar(&trxMaxCount, "trx_count", 0, "Max count of transactions for sending to one node")
	flag.IntVar(&maxFlowSpeedInMin, "max_flow", 0, "Max send count in minute for one node")
	flag.IntVar(&randomFlowDelta, "rand_flow", 0, "Random delta for different flow to different nodes (only if -max_flow using)")
	flag.StringVar(&donorAddrHex, "donor_addr", "", "Address of donor (0xHEX)")
	flag.StringVar(&donorKeyHex, "donor_key", "", "Private key of donor (HEX)")
	flag.IntVar(&mode, "mode", 2, `Mode of waiting confirm transactions: 
0 - not wait
1 - detect finish transactions without waiting 
2 - wait all transaction finished after batch
`)
	flag.Parse()

	if nodesListFile == "" || donorAddrHex == "" || donorKeyHex == "" {
		flag.PrintDefaults()
		return
	}

	// Parse donor data
	// donorAddress = common.HexToAddress("0xf9352d0ca3820e4b16b5242c74adb0e26471fbea")
	// donorPvtKey, err = crypto.HexToECDSA("ae2037b61158065161bc5eeafe43b227663f1614c123ae5f378e8201d3a5f3e5")
	var err error
	donorAddress = common.HexToAddress(donorAddrHex)
	donorPvtKey, err = crypto.HexToECDSA(donorKeyHex)
	if err != nil {
		log.Panicf("Error decode donor private key: %s\n", err)
	}

	// Parse nodes file
	parseNodesFile(nodesListFile, randomFlowDelta)

	// Parse accounts file
	generateAccounts(accCount)

	// Transfer started funds
	transferFunds()

	// Run main loop of sending
	mainLoop()
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

// CalcAvgFlowSpeed calculate current average flow speed (transactions/minute)
func (n *Node) CalcAvgFlowSpeed() float64 {
	if n.SendCount <= 0 {
		return 0
	}
	return float64(n.SendCount) * 60 / (time.Now().UTC().Sub(n.FirstSend)).Seconds()
}

// DelayForNextSend calculate delay if flow speed great then max flow speed
func (n *Node) DelayForNextSend() time.Duration {
	delay := float64(n.SendCount*60/int64(maxFlowSpeedInMin+n.RandomFlowFactor)) - time.Now().UTC().Sub(n.FirstSend).Seconds()
	if delay < 0 {
		delay = 0
	}
	return time.Duration(delay) * time.Second
}

// SendRandomTransaction send transfer from random account to random account
func (n *Node) SendRandomTransaction() (*common.Hash, error) {
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

	gasLimit := uint64(21000) // in units
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
	trxInBatch := 0
	trxCount := 0
	if n.trxs == nil && mode != 0 {
		n.trxs = make([]common.Hash, 0)
	}
	n.lastTrxClean = time.Now().UTC()
	for {
		// Check max transactions count limit
		if trxMaxCount > 0 {
			if trxCount >= trxMaxCount {
				break
			}
		}
		trxCount++

		// Check MaxFlowSpeed
		if maxFlowSpeedInMin > 0 {
			currentFlowSpeed := n.CalcAvgFlowSpeed()
			if currentFlowSpeed > float64(maxFlowSpeedInMin+n.RandomFlowFactor) {
				delay := n.DelayForNextSend()
				if delay > 0 {
					fmt.Print(".")
					time.Sleep(delay)
				}
				continue
			}
		}

		trxHash, err := n.SendRandomTransaction()
		if err == nil && mode != 0 {
			n.trxs = append(n.trxs, *trxHash)
		}

		trxInBatch++
		switch mode {
		case 2:
			// Wait confirm all transactions
			if trxInBatch >= trxBatchSize {
				trxInBatch = 0
				for {
					if n.CleanFinishedTransactions(waitConfirmTimeout) == 0 {
						break
					}
					// log.Printf("Wait finish transactions for node %s (%d)\n", n.AddrURL, len(n.trxs))
					fmt.Print(".")
					time.Sleep(time.Second)
				}
				speed := int64(n.CalcAvgFlowSpeed())
				fmt.Printf("\n")
				log.Printf("Node %s: \tok = %d, \terrors = %d, missed = %d, \ttimeouts = %d, \tpending = %d (%d tx/min)\n",
					n.AddrURL, n.SendCount, n.ErrorsCount, n.MissedCount, n.TimeoutCount, len(n.trxs), speed)
			}
		case 1:
			// Check confirmed transactions, but not wait
			if trxInBatch >= trxBatchSize {
				trxInBatch = 0
				n.CleanFinishedTransactions(10 * waitConfirmTimeout)
				speed := int64(n.CalcAvgFlowSpeed())
				if maxFlowSpeedInMin > 0 {
					fmt.Printf("\n")
				}
				log.Printf("Node %s: \tok = %d (%d), \terrors = %d, missed = %d, \ttimeouts = %d, \tpending = %d (%d tx/min)\n",
					n.AddrURL, n.SendCount, int(n.SendCount)-len(n.trxs), n.ErrorsCount, n.MissedCount, n.TimeoutCount, len(n.trxs), speed)
			}
		case 0:
			// Not check confirmed transactions
			if trxInBatch >= trxBatchSize {
				trxInBatch = 0
				speed := int64(n.CalcAvgFlowSpeed())
				if maxFlowSpeedInMin > 0 {
					fmt.Printf("\n")
				}
				log.Printf("Node %s: \tok = %d, \terr = %d (%d tx/min)\n",
					n.AddrURL, n.SendCount, n.ErrorsCount, speed)
			}
		}
	}

	wg.Done()
}

func (n *Node) CleanFinishedTransactions(timeout time.Duration) int {
	for i := 0; i < len(n.trxs); i++ {
		// Check transaction exists
		t, _, _ := n.client.TransactionByHash(context.Background(), n.trxs[i])
		if t == nil {
			n.MissedCount++
		}

		// Check transaction receipt
		r, _ := n.client.TransactionReceipt(context.Background(), n.trxs[i])
		if r != nil || t == nil {
			n.lastTrxClean = time.Now().UTC()

			if i != (len(n.trxs) - 1) {
				n.trxs[i] = n.trxs[len(n.trxs)-1]
				i--
			}
			n.trxs = n.trxs[0 : len(n.trxs)-1]
		}
	}

	if timeout != 0 && time.Now().UTC().Sub(n.lastTrxClean) > timeout {
		// Force clean trxs by timeout
		for i := 0; i < len(n.trxs); i++ {
			n.TimeoutCount++
			log.Printf("Trx timeout: %s\n", n.trxs[i].Hex())
		}
		n.trxs = n.trxs[0:0]
		n.lastTrxClean = time.Now().UTC()
	}

	return len(n.trxs)
}

func parseNodesFile(fileName string, randomFlow int) {
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

		newNode := Node{
			AddrURL:          line,
			SendCount:        0,
			FirstSend:        time.Now().UTC(),
			LastSend:         time.Now().UTC(),
			RandomFlowFactor: randomFlow/2 - rand.Intn(randomFlow),
		}
		newNode.InitClient()
		nodes = append(nodes, newNode)
	}

	log.Printf("Read nodes: %d\n", len(nodes))
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
	log.Printf("Generate accounts: %d\n", len(accounts))
}

func transferFunds() {
	log.Printf("Create funds...\n")
	node := nodes[0]
	for _, acc := range accounts {
		from := Account{
			Address: &donorAddress,
			PvtKey:  donorPvtKey,
		}

		trxHash, err := node.SendTransfer(&from, &acc, big.NewInt(100000000))
		if err != nil {
			log.Panicf("Error transfer funds from donor: %s", err)
		}
		log.Printf("Create funds: %s\n", acc.Address.Hex())

		node.trxs = append(node.trxs, *trxHash)
	}
	log.Printf("Create funds done\n")
	log.Printf("Wait funds transactions finished...\n")
	for {
		if node.CleanFinishedTransactions(0) <= 0 {
			break
		}
		// log.Printf("Wait for finish %d transactions\n", len(node.trxs))
		fmt.Print(".")
		time.Sleep(3 * time.Second)
	}
	log.Printf("Funds transactions finished\n")
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
