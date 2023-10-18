// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package quaistats implements the network stats reporting service.
package quaistats

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"

	lru "github.com/hashicorp/golang-lru"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"

	"os/exec"

	"github.com/dominant-strategies/go-quai/common"
	"github.com/dominant-strategies/go-quai/consensus"
	"github.com/dominant-strategies/go-quai/core"
	"github.com/dominant-strategies/go-quai/core/types"
	"github.com/dominant-strategies/go-quai/eth/downloader"
	ethproto "github.com/dominant-strategies/go-quai/eth/protocols/eth"
	"github.com/dominant-strategies/go-quai/event"
	"github.com/dominant-strategies/go-quai/log"
	"github.com/dominant-strategies/go-quai/node"
	"github.com/dominant-strategies/go-quai/p2p"
	"github.com/dominant-strategies/go-quai/params"
	"github.com/dominant-strategies/go-quai/rpc"
)

const (
	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
	chainSideChanSize = 10

	// reportInterval is the time interval between two reports.
	reportInterval = 15

	c_alpha           = 8
	c_statsErrorValue = int64(-1)

	// Max number of stats objects to send in one batch
	c_queueBatchSize uint64 = 5
	// Number of blocks to include in one batch of transactions
	c_txBatchSize uint64 = 20
)

var (
	c_blocksPerMinute         uint64
	c_blocksPerHour           uint64
	c_txLookupCacheLimit      uint64
	c_txLookupCacheEvictLimit uint64
	chainID9000               = big.NewInt(9000)
	chainID12000              = big.NewInt(12000)
	chainID15000              = big.NewInt(15000)
	chainID17000              = big.NewInt(17000)
	chainID1337               = big.NewInt(1337)
)

// backend encompasses the bare-minimum functionality needed for quaistats reporting
type backend interface {
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
	SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription
	SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription
	CurrentHeader() *types.Header
	TotalLogS(header *types.Header) *big.Int
	HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error)
	Stats() (pending int, queued int)
	Downloader() *downloader.Downloader
	ChainConfig() *params.ChainConfig
	ProcessingState() bool
}

// fullNodeBackend encompasses the functionality necessary for a full node
// reporting to quaistats
type fullNodeBackend interface {
	backend
	BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error)
	CurrentBlock() *types.Block
}

// Service implements an Quai netstats reporting daemon that pushes local
// chain statistics up to a monitoring server.
type Service struct {
	server  *p2p.Server // Peer-to-peer server to retrieve networking infos
	backend backend
	engine  consensus.Engine // Consensus engine to retrieve variadic block fields

	node    string // Name of the node to display on the monitoring page
	pass    string // Password to authorize access to the monitoring page
	host    string // Remote address of the monitoring service
	trusted bool   // Whether the node is trusted or not

	pongCh  chan struct{} // Pong notifications are fed into this channel
	headSub event.Subscription
	sideSub event.Subscription

	transactionStatsQueue *StatsQueue
	detailStatsQueue      *StatsQueue
	appendTimeStatsQueue  *StatsQueue
	statsReadyCh          chan struct{}

	txLookupCache *lru.Cache

	chainID *big.Int

	instanceDir string // Path to the node's instance directory
}

// StatsQueue is a thread-safe queue designed for managing and processing stats data.
//
// The primary objective of the StatsQueue is to provide a safe mechanism for enqueuing,
// dequeuing, and requeuing stats objects concurrently across multiple goroutines.
//
// Key Features:
//   - Enqueue: Allows adding an item to the end of the queue.
//   - Dequeue: Removes and returns the item from the front of the queue.
//   - RequeueFront: Adds an item back to the front of the queue, useful for failed processing attempts.
//
// Concurrent Access:
//   - The internal state of the queue is protected by a mutex to prevent data races and ensure
//     that the operations are atomic. As a result, it's safe to use across multiple goroutines
//     without external synchronization.
type StatsQueue struct {
	data  []interface{}
	mutex sync.Mutex
}

func NewStatsQueue() *StatsQueue {
	return &StatsQueue{
		data: make([]interface{}, 0),
	}
}

func (q *StatsQueue) Enqueue(item interface{}) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.data = append(q.data, item)
}

func (q *StatsQueue) Dequeue() interface{} {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.data) == 0 {
		return nil
	}

	item := q.data[0]
	q.data = q.data[1:]
	return item
}

func (q *StatsQueue) EnqueueFront(item interface{}) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.data = append([]interface{}{item}, q.data...)
}

func (q *StatsQueue) Size() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return len(q.data)
}

// parseEthstatsURL parses the netstats connection url.
// URL argument should be of the form <nodename:secret@host:port>
// If non-erroring, the returned slice contains 3 elements: [nodename, pass, host]
func parseEthstatsURL(url string) (parts []string, err error) {
	err = fmt.Errorf("invalid netstats url: \"%s\", should be nodename:secret@host:port", url)

	hostIndex := strings.LastIndex(url, "@")
	if hostIndex == -1 || hostIndex == len(url)-1 {
		return nil, err
	}
	preHost, host := url[:hostIndex], url[hostIndex+1:]

	passIndex := strings.LastIndex(preHost, ":")
	if passIndex == -1 {
		return []string{preHost, "", host}, nil
	}
	nodename, pass := preHost[:passIndex], ""
	if passIndex != len(preHost)-1 {
		pass = preHost[passIndex+1:]
	}

	return []string{nodename, pass, host}, nil
}

// New returns a monitoring service ready for stats reporting.
func New(node *node.Node, backend backend, engine consensus.Engine, url string, trustedstatsprovider bool) error {
	parts, err := parseEthstatsURL(url)
	if err != nil {
		return err
	}

	chainID := backend.ChainConfig().ChainID
	var durationLimit *big.Int

	switch {
	case chainID.Cmp(chainID9000) == 0:
		durationLimit = params.DurationLimit
	case chainID.Cmp(chainID12000) == 0:
		durationLimit = params.GardenDurationLimit
	case chainID.Cmp(chainID15000) == 0:
		durationLimit = params.OrchardDurationLimit
	case chainID.Cmp(chainID17000) == 0:
		durationLimit = params.LighthouseDurationLimit
	case chainID.Cmp(chainID1337) == 0:
		durationLimit = params.LocalDurationLimit
	default:
		durationLimit = params.DurationLimit
	}

	durationLimitInt := durationLimit.Uint64()

	c_blocksPerMinute = 60 / durationLimitInt
	c_blocksPerHour = 60 * c_blocksPerMinute
	c_txLookupCacheEvictLimit = c_blocksPerHour / c_txBatchSize
	c_txLookupCacheLimit = 2 * c_txLookupCacheEvictLimit

	txLookupCache, _ := lru.New(int(c_txLookupCacheLimit * 2))

	quaistats := &Service{
		backend:               backend,
		engine:                engine,
		server:                node.Server(),
		node:                  parts[0],
		pass:                  parts[1],
		host:                  parts[2],
		pongCh:                make(chan struct{}),
		chainID:               backend.ChainConfig().ChainID,
		transactionStatsQueue: NewStatsQueue(),
		detailStatsQueue:      NewStatsQueue(),
		appendTimeStatsQueue:  NewStatsQueue(),
		statsReadyCh:          make(chan struct{}),
		trusted:               trustedstatsprovider,
		txLookupCache:         txLookupCache,
		instanceDir:           node.InstanceDir(),
	}

	node.RegisterLifecycle(quaistats)
	return nil
}

// Start implements node.Lifecycle, starting up the monitoring and reporting daemon.
func (s *Service) Start() error {
	// Subscribe to chain events to execute updates on
	chainHeadCh := make(chan core.ChainHeadEvent, chainHeadChanSize)
	chainSideCh := make(chan core.ChainSideEvent, chainSideChanSize)

	s.headSub = s.backend.SubscribeChainHeadEvent(chainHeadCh)
	s.sideSub = s.backend.SubscribeChainSideEvent(chainSideCh)

	go s.loopBlocks(chainHeadCh, chainSideCh)
	go s.loopSender(s.initializeURLMap())

	log.Info("Stats daemon started")
	return nil
}

// Stop implements node.Lifecycle, terminating the monitoring and reporting daemon.
func (s *Service) Stop() error {
	s.headSub.Unsubscribe()
	s.sideSub.Unsubscribe()
	log.Info("Stats daemon stopped")
	return nil
}

func (s *Service) loopBlocks(chainHeadCh chan core.ChainHeadEvent, chainSideCh chan core.ChainSideEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("Stats process crashed", "error", r)
			go s.loopBlocks(chainHeadCh, chainSideCh)
		}
	}()

	quitCh := make(chan struct{})
	headCh := make(chan *types.Block, chainHeadChanSize)
	sideCh := make(chan *types.Block, chainSideChanSize)

	go func() {
	HandleLoop:
		for {
			select {
			// Notify of chain head events, but drop if too frequent
			case head := <-chainHeadCh:
				select {
				case headCh <- head.Block:
				default:
				}
			// Notify of chain side events, but drop if too frequent
			case sideEvent := <-chainSideCh:
				select {
				case sideCh <- sideEvent.Block:
				default:
				}
			case <-s.headSub.Err():
				break HandleLoop
			}
		}
		close(quitCh)
	}()

	for {
		select {
		case <-quitCh:
			return
		default:
			s.handleBlock(headCh)
		}
	}
}

// loop keeps trying to connect to the netstats server, reporting chain events
// until termination.
func (s *Service) loopSender(urlMap map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Stats process crashed with error:", r)
			go s.loopSender(urlMap)
		}
	}()

	// Start a goroutine that exhausts the subscriptions to avoid events piling up
	var (
		quitCh = make(chan struct{})
	)

	nodeStatsMod := 0

	errTimer := time.NewTimer(0)
	defer errTimer.Stop()
	var authJwt = ""
	// Loop reporting until termination
	for {
		select {
		case <-quitCh:
			return
		case <-errTimer.C:
			// If we don't have a JWT or it's expired, get a new one
			isJwtExpiredResult, jwtIsExpiredErr := s.isJwtExpired(authJwt)
			if authJwt == "" || isJwtExpiredResult || jwtIsExpiredErr != nil {
				log.Info("Trying to login to quaistats")
				var err error
				authJwt, err = s.login2(urlMap["login"])
				if err != nil {
					log.Warn("Stats login failed", "err", err)
					errTimer.Reset(10 * time.Second)
					continue
				}
			}

			errs := make(map[string]error)

			// Authenticate the client with the server
			for key, url := range urlMap {
				switch key {
				case "login":
					continue
				case "nodeStats":
					if errs[key] = s.reportNodeStats(url, 0, authJwt); errs[key] != nil {
						log.Warn("Initial stats report failed for "+key, "err", errs[key])
						errTimer.Reset(0)
						continue
					}
				case "blockTransactionStats":
					if errs[key] = s.sendTransactionStats(url, authJwt); errs[key] != nil {
						log.Warn("Initial stats report failed for "+key, "err", errs[key])
						errTimer.Reset(0)
						continue
					}
				case "blockDetailStats":
					if errs[key] = s.sendDetailStats(url, authJwt); errs[key] != nil {
						log.Warn("Initial stats report failed for "+key, "err", errs[key])
						errTimer.Reset(0)
						continue
					}
				case "blockAppendTime":
					if errs[key] = s.sendAppendTimeStats(url, authJwt); errs[key] != nil {
						log.Warn("Initial stats report failed for "+key, "err", errs[key])
						errTimer.Reset(0)
						continue
					}
				}
			}

			// Keep sending status updates until the connection breaks
			fullReport := time.NewTicker(reportInterval * time.Second)

			var noErrs = true
			for noErrs {
				var err error
				select {
				case <-quitCh:
					fullReport.Stop()
					return

				case <-fullReport.C:
					nodeStatsMod ^= 1
					if err = s.reportNodeStats(urlMap["nodeStats"], nodeStatsMod, authJwt); err != nil {
						noErrs = false
						log.Warn("nodeStats full stats report failed", "err", err)
					}
				case <-s.statsReadyCh:
					if url, ok := urlMap["blockTransactionStats"]; ok {
						s.sendTransactionStats(url, authJwt)
					}
					if url, ok := urlMap["blockDetailStats"]; ok {
						s.sendDetailStats(url, authJwt)
					}
					if url, ok := urlMap["blockAppendTime"]; ok {
						s.sendAppendTimeStats(url, authJwt)
					}
				}
				errTimer.Reset(0)
			}
			fullReport.Stop()
		}
	}
}

func (s *Service) initializeURLMap() map[string]string {
	return map[string]string{
		"blockTransactionStats": fmt.Sprintf("http://%s/stats/blockTransactionStats", s.host),
		"blockAppendTime":       fmt.Sprintf("http://%s/stats/blockAppendTime", s.host),
		"blockDetailStats":      fmt.Sprintf("http://%s/stats/blockDetailStats", s.host),
		"nodeStats":             fmt.Sprintf("http://%s/stats/nodeStats", s.host),
		"login":                 fmt.Sprintf("http://%s/auth/login", s.host),
	}
}

func (s *Service) handleBlock(headCh chan *types.Block) {
	for head := range headCh {
		if s.trusted {
			dtlStats := s.assembleBlockDetailStats(head)
			s.detailStatsQueue.Enqueue(dtlStats)
		}

		appStats := s.assembleBlockAppendTimeStats(head)
		s.appendTimeStatsQueue.Enqueue(appStats)

		if head.NumberU64()%c_txBatchSize == 0 && s.trusted {
			txStats := s.assembleBlockTransactionStats(head)
			s.transactionStatsQueue.Enqueue(txStats)
		}

		// After handling a block and potentially adding to the queues, notify the sendStats goroutine
		// that stats are ready to be sent
		s.statsReadyCh <- struct{}{}
	}
}

func (s *Service) reportNodeStats(url string, mod int, authJwt string) error {
	if url == "" {
		log.Warn("node stats url is empty")
		return errors.New("node stats connection is empty")
	}

	isRegion := strings.Contains(s.instanceDir, "region")
	isPrime := strings.Contains(s.instanceDir, "prime")

	if isRegion || isPrime {
		log.Debug("Skipping node stats for region or prime. Filtered out on backend")
		return nil
	}

	log.Trace("Quai Stats Instance Dir", "path", s.instanceDir+"/../..")

	// Don't send if dirSize < 1
	// Get disk usage (as a percentage)
	diskUsage, err := dirSize(s.instanceDir + "/../..")
	if err != nil {
		log.Warn("Error calculating directory sizes:", "error", err)
		diskUsage = c_statsErrorValue
	}

	diskSize, err := diskTotalSize()
	if err != nil {
		log.Warn("Error calculating disk size:", "error", err)
		diskUsage = c_statsErrorValue
	}

	diskUsagePercent := float64(c_statsErrorValue)
	if diskSize > 0 {
		diskUsagePercent = float64(diskUsage) / float64(diskSize)
	} else {
		log.Warn("Error calculating disk usage percent: disk size is 0")
	}

	// Usage in your main function
	ramUsage, err := getQuaiRAMUsage()
	if err != nil {
		log.Warn("Error getting Quai RAM usage:", "error", err)
		return err
	}
	var ramUsagePercent, ramFreePercent, ramAvailablePercent float64
	if vmStat, err := mem.VirtualMemory(); err == nil {
		ramUsagePercent = float64(ramUsage) / float64(vmStat.Total)
		ramFreePercent = float64(vmStat.Free) / float64(vmStat.Total)
		ramAvailablePercent = float64(vmStat.Available) / float64(vmStat.Total)
	} else {
		log.Warn("Error getting RAM stats:", "error", err)
		return err
	}

	// Get CPU usage
	cpuUsageQuai, err := getQuaiCPUUsage()
	if err != nil {
		log.Warn("Error getting Quai CPU percent usage:", "error", err)
		return err
	} else {
		cpuUsageQuai /= float64(100)
	}

	var cpuFree float32
	if cpuUsageTotal, err := cpu.Percent(0, false); err == nil {
		cpuFree = 1 - float32(cpuUsageTotal[0]/float64(100))
	} else {
		log.Warn("Error getting CPU free:", "error", err)
		return err
	}

	currentHeader := s.backend.CurrentHeader()

	if currentHeader == nil {
		log.Warn("Current header is nil")
		return errors.New("current header is nil")
	}
	// Get current block number
	currentBlockHeight := currentHeader.NumberArray()

	// Get location
	location := currentHeader.Location()

	// Get the first non-loopback MAC address
	var macAddress string
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, interf := range interfaces {
			if interf.HardwareAddr != nil && len(interf.HardwareAddr.String()) > 0 && (interf.Flags&net.FlagLoopback) == 0 {
				macAddress = interf.HardwareAddr.String()
				break
			}
		}
	} else {
		log.Warn("Error getting MAC address:", err)
		return err
	}

	// Hash the MAC address
	var hashedMAC string
	if macAddress != "" {
		hash := sha256.Sum256([]byte(macAddress))
		hashedMAC = hex.EncodeToString(hash[:])
	}

	// Assemble the new node stats
	log.Trace("Sending node details to quaistats")

	document := map[string]interface{}{
		"id": s.node,
		"nodeStats": &nodeStats{
			Name:                s.node,
			Timestamp:           big.NewInt(time.Now().Unix()), // Current timestamp
			RAMUsage:            int64(ramUsage),
			RAMUsagePercent:     float32(ramUsagePercent),
			RAMFreePercent:      float32(ramFreePercent),
			RAMAvailablePercent: float32(ramAvailablePercent),
			CPUUsagePercent:     float32(cpuUsageQuai),
			CPUFree:             float32(cpuFree),
			DiskUsageValue:      int64(diskUsage),
			DiskUsagePercent:    float32(diskUsagePercent),
			CurrentBlockNumber:  currentBlockHeight,
			RegionLocation:      location.Region(),
			ZoneLocation:        location.Zone(),
			NodeStatsMod:        mod,
			HashedMAC:           hashedMAC,
		},
	}

	jsonData, err := json.Marshal(document)
	if err != nil {
		log.Error("Failed to marshal node stats", "err", err)
		return err
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Failed to create new HTTP request", "err", err)
		return err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authJwt)

	// Send the request using the default HTTP client
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error("Failed to send node stats", "err", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("Failed to response body", "err", err)
			return err
		}
		log.Error("Received non-OK response", "status", resp.Status, "body", string(body))
		return errors.New("Received non-OK response: " + resp.Status)
	}
	log.Trace("Successfully sent node stats to quaistats")
	return nil
}

func (s *Service) sendTransactionStats(url string, authJwt string) error {
	if len(s.transactionStatsQueue.data) == 0 {
		return nil
	}
	statsBatch := make([]*blockTransactionStats, 0, c_queueBatchSize)

	for i := 0; i < int(c_queueBatchSize) && len(s.transactionStatsQueue.data) > 0; i++ {
		stat := s.transactionStatsQueue.Dequeue()
		if stat == nil {
			break
		}
		statsBatch = append(statsBatch, stat.(*blockTransactionStats))
	}

	if len(statsBatch) == 0 {
		return nil
	}

	err := s.report(url, "blockTransactionStats", statsBatch, authJwt)
	if err != nil && strings.Contains(err.Error(), "Received non-OK response") {
		log.Warn("Failed to send transaction stats, requeuing stats", "err", err)
		// Re-enqueue the failed stats from end to beginning
		for i := len(statsBatch) - 1; i >= 0; i-- {
			s.transactionStatsQueue.EnqueueFront(statsBatch[i])
		}
		return err
	} else if err != nil {
		log.Warn("Failed to send transaction stats", "err", err)
		return err
	}
	return nil
}

func (s *Service) sendDetailStats(url string, authJwt string) error {
	if len(s.detailStatsQueue.data) == 0 {
		return nil
	}
	statsBatch := make([]*blockDetailStats, 0, c_queueBatchSize)

	for i := 0; i < int(c_queueBatchSize) && s.detailStatsQueue.Size() > 0; i++ {
		stat := s.detailStatsQueue.Dequeue()
		if stat == nil {
			break
		}
		statsBatch = append(statsBatch, stat.(*blockDetailStats))
	}

	if len(statsBatch) == 0 {
		return nil
	}

	err := s.report(url, "blockDetailStats", statsBatch, authJwt)
	if err != nil && strings.Contains(err.Error(), "Received non-OK response") {
		log.Warn("Failed to send detail stats, requeuing stats", "err", err)
		// Re-enqueue the failed stats from end to beginning
		for i := len(statsBatch) - 1; i >= 0; i-- {
			s.detailStatsQueue.EnqueueFront(statsBatch[i])
		}
		return err
	} else if err != nil {
		log.Warn("Failed to send detail stats", "err", err)
		return err
	}
	return nil
}

func (s *Service) sendAppendTimeStats(url string, authJwt string) error {
	if len(s.appendTimeStatsQueue.data) == 0 {
		return nil
	}

	statsBatch := make([]*blockAppendTime, 0, c_queueBatchSize)

	for i := 0; i < int(c_queueBatchSize) && s.appendTimeStatsQueue.Size() > 0; i++ {
		stat := s.appendTimeStatsQueue.Dequeue()
		if stat == nil {
			break
		}
		statsBatch = append(statsBatch, stat.(*blockAppendTime))
	}

	if len(statsBatch) == 0 {
		return nil
	}

	err := s.report(url, "blockAppendTime", statsBatch, authJwt)
	if err != nil && strings.Contains(err.Error(), "Received non-OK response") {
		log.Warn("Failed to send append time stats, requeuing stats", "err", err)
		// Re-enqueue the failed stats from end to beginning
		for i := len(statsBatch) - 1; i >= 0; i-- {
			s.appendTimeStatsQueue.EnqueueFront(statsBatch[i])
		}
		return err
	} else if err != nil {
		log.Warn("Failed to send append time stats", "err", err)
		return err
	}
	return nil
}

func (s *Service) report(url string, dataType string, stats interface{}, authJwt string) error {
	if url == "" {
		log.Warn(dataType + " url is empty")
		return errors.New(dataType + " url is empty")
	}

	if stats == nil {
		log.Warn(dataType + " stats are nil")
		return errors.New(dataType + " stats are nil")
	}

	log.Trace("Sending " + dataType + " stats to quaistats")

	document := map[string]interface{}{
		"id":     s.node,
		dataType: stats,
	}

	jsonData, err := json.Marshal(document)
	if err != nil {
		log.Error("Failed to marshal "+dataType+" stats", "err", err)
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("Failed to create new request for "+dataType+" stats", "err", err)
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authJwt) // Add this line for the Authorization header

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to send "+dataType+" stats", "err", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Error("Failed to response body", "err", err)
			return err
		}
		log.Error("Received non-OK response", "status", resp.Status, "body", string(body))
		return errors.New("Received non-OK response: " + resp.Status)
	}
	log.Trace("Successfully sent " + dataType + " stats to quaistats")
	return nil
}

// nodeInfo is the collection of meta information about a node that is displayed
// on the monitoring page.
type nodeInfo struct {
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Network  string `json:"net"`
	Protocol string `json:"protocol"`
	API      string `json:"api"`
	Os       string `json:"os"`
	OsVer    string `json:"os_v"`
	Client   string `json:"client"`
	History  bool   `json:"canUpdateHistory"`
	Chain    string `json:"chain"`
	ChainID  uint64 `json:"chainId"`
}

// authMsg is the authentication infos needed to login to a monitoring server.
type authMsg struct {
	ID     string      `json:"id"`
	Info   nodeInfo    `json:"info"`
	Secret loginSecret `json:"secret"`
}

type loginSecret struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type Credentials struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
}

func (s *Service) login2(url string) (string, error) {
	// Substitute with your actual service address and port

	infos := s.server.NodeInfo()

	var protocols []string
	for _, proto := range s.server.Protocols {
		protocols = append(protocols, fmt.Sprintf("%s/%d", proto.Name, proto.Version))
	}
	var network string
	if info := infos.Protocols["eth"]; info != nil {
		network = fmt.Sprintf("%d", info.(*ethproto.NodeInfo).Network)
	}

	var secretUser string
	if s.trusted {
		secretUser = "admin"
	} else {
		secretUser = s.node
	}

	auth := &authMsg{
		ID: s.node,
		Info: nodeInfo{
			Name:     s.node,
			Node:     infos.Name,
			Port:     infos.Ports.Listener,
			Network:  network,
			Protocol: strings.Join(protocols, ", "),
			API:      "No",
			Os:       runtime.GOOS,
			OsVer:    runtime.GOARCH,
			Client:   "0.1.1",
			History:  true,
			Chain:    common.NodeLocation.Name(),
			ChainID:  s.chainID.Uint64(),
		},
		Secret: loginSecret{
			Name:     secretUser,
			Password: s.pass,
		},
	}

	authJson, err := json.Marshal(auth)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(authJson))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Failed to response body", "err", err)
		return "", err
	}

	var authResponse AuthResponse
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		return "", err
	}

	if authResponse.Success {
		return authResponse.Token, nil
	}

	return "", fmt.Errorf("login failed")
}

// isJwtExpired checks if the JWT token is expired
func (s *Service) isJwtExpired(authJwt string) (bool, error) {
	if authJwt == "" {
		return false, errors.New("token is nil")
	}

	parts := strings.Split(authJwt, ".")
	if len(parts) != 3 {
		return false, errors.New("invalid token")
	}

	claims := jwt.MapClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(authJwt, claims)
	if err != nil {
		return false, err
	}

	if exp, ok := claims["exp"].(float64); ok {
		return time.Now().Unix() >= int64(exp), nil
	}

	return false, errors.New("exp claim not found in token")
}

// Trusted Only
type blockTransactionStats struct {
	Timestamp             *big.Int `json:"timestamp"`
	TotalNoTransactions1h uint64   `json:"totalNoTransactions1h"`
	TPS1m                 uint64   `json:"tps1m"`
	TPS1hr                uint64   `json:"tps1hr"`
	Chain                 string   `json:"chain"`
}

// Trusted Only
type blockDetailStats struct {
	Timestamp    *big.Int `json:"timestamp"`
	ZoneHeight   uint64   `json:"zoneHeight"`
	RegionHeight uint64   `json:"regionHeight"`
	PrimeHeight  uint64   `json:"primeHeight"`
	Chain        string   `json:"chain"`
	Entropy      string   `json:"entropy"`
	Difficulty   string   `json:"difficulty"`
}

// Everyone sends every block
type blockAppendTime struct {
	AppendTime  time.Duration `json:"appendTime"`
	BlockNumber *big.Int      `json:"number"`
	Chain       string        `json:"chain"`
}

type nodeStats struct {
	Name                string     `json:"name"`
	Timestamp           *big.Int   `json:"timestamp"`
	RAMUsage            int64      `json:"ramUsage"`
	RAMUsagePercent     float32    `json:"ramUsagePercent"`
	RAMFreePercent      float32    `json:"ramFreePercent"`
	RAMAvailablePercent float32    `json:"ramAvailablePercent"`
	CPUUsagePercent     float32    `json:"cpuPercent"`
	CPUFree             float32    `json:"cpuFree"`
	DiskUsagePercent    float32    `json:"diskUsagePercent"`
	DiskUsageValue      int64      `json:"diskUsageValue"`
	CurrentBlockNumber  []*big.Int `json:"currentBlockNumber"`
	RegionLocation      int        `json:"regionLocation"`
	ZoneLocation        int        `json:"zoneLocation"`
	NodeStatsMod        int        `json:"nodeStatsMod"`
	HashedMAC           string     `json:"hashedMAC"`
}

type tps struct {
	TPS1m                     uint64
	TPS1hr                    uint64
	TotalNumberTransactions1h uint64
}

type BatchObject struct {
	TotalNoTransactions uint64
	OldestBlockTime     uint64
}

func (s *Service) evictOutdatedEntries(currentMaxBlock int) {
	for {
		key, _, ok := s.txLookupCache.GetOldest()
		if !ok {
			break
		}

		keyInt, ok := key.(int)
		if !ok {
			return
		}

		if keyInt < currentMaxBlock {
			s.txLookupCache.Remove(key)
		} else {
			return
		}
	}
}

func (s *Service) calculateTPS(block *types.Block) *tps {
	var totalTransactions1h uint64
	var totalTransactions1m uint64

	currentBlock := block
	startBlockTime := block.Time()
	var batchesNeeded uint64 = c_blocksPerHour / c_txBatchSize
	var oldestKeyUsed uint64
	var withinMinute = true
	startBlockNum := currentBlock.NumberU64()

	// Get the parent block for the next iteration
	fullBackend := s.backend.(fullNodeBackend)

	for i := 0; i < int(batchesNeeded); i++ {
		if currentBlock == nil || currentBlock.NumberU64() == 0 {
			log.Error("Encountered a nil block, stopping iteration")
			break
		}

		if oldestKeyUsed == 0 {
			oldestKeyUsed = startBlockNum
		} else {
			oldestKeyUsed = min(oldestKeyUsed, startBlockNum)
		}

		// Try to get the data from the LRU cache
		cachedBatchObject, ok := s.txLookupCache.Get(uint64(startBlockNum))
		if !ok {
			// Not in cache, so we need to calculate the transaction count for this batch
			txCount := uint64(0)
			oldestBlockTimeInBatch := uint64(0)

			for j := 0; j < int(c_txBatchSize); j++ {
				currentNumber := currentBlock.NumberU64()
				if currentNumber == 0 {
					log.Trace("Current block number is 0, stopping iteration")
					break
				}

				// Add the number of transactions in the current block to the total
				txCount += uint64(len(currentBlock.Transactions()))

				if withinMinute && startBlockTime < currentBlock.Time()+60 {
					totalTransactions1m += uint64(len(currentBlock.Transactions()))
				} else {
					withinMinute = false
				}

				if oldestBlockTimeInBatch == 0 {
					oldestBlockTimeInBatch = currentBlock.Time()
				} else {
					oldestBlockTimeInBatch = min(oldestBlockTimeInBatch, currentBlock.Time())
				}

				var err error
				currentBlock, err = fullBackend.BlockByNumber(context.Background(), rpc.BlockNumber(currentNumber-1))
				if err != nil {
					log.Error(fmt.Sprintf("Error getting block number %d: %s", currentNumber-1, err.Error()))
					break
				}
				if currentBlock == nil {
					log.Error(fmt.Sprintf("No block found at number %d", currentNumber-1))
					break
				}
			}

			batchObject := &BatchObject{
				TotalNoTransactions: txCount,
				OldestBlockTime:     oldestBlockTimeInBatch,
			}

			// Store the sum in the cache
			s.txLookupCache.Add(startBlockNum, batchObject)

			cachedBatchObject = batchObject
		}

		// Add the transactions from this batch
		totalTransactions1h += cachedBatchObject.(*BatchObject).TotalNoTransactions
		if startBlockNum == 0 {
			break
		}
		startBlockNum -= uint64(c_txBatchSize)
	}

	if s.txLookupCache.Len() > int(c_txLookupCacheEvictLimit) {
		s.evictOutdatedEntries(int(block.NumberU64() - c_blocksPerHour))
	}

	// Find the oldest batch object in the cache
	// and use that oldest block time to calculate the TPS
	found := false
	notFoundCount := 0
	var batchObject *BatchObject
	var ok bool

	for !found {
		if notFoundCount >= int(c_txBatchSize) {
			log.Error("Could not find any batch object in cache returning estimations")
			return &tps{
				TPS1m:                     totalTransactions1m / 60,
				TPS1hr:                    totalTransactions1h / 3600,
				TotalNumberTransactions1h: totalTransactions1h,
			}
		}

		// Retrieve the batch object from the cache
		value, exists := s.txLookupCache.Get(oldestKeyUsed)
		if !exists {
			log.Warn("Could not find batch object in cache")
			notFoundCount += 1
			oldestKeyUsed += c_txBatchSize
			continue
		}

		found = true

		// Type assert the value to a *BatchObject
		batchObject, ok = value.(*BatchObject)
		if !ok {
			log.Warn("Error casting value to *BatchObject")
			return &tps{
				TPS1m:                     totalTransactions1m / 60,
				TPS1hr:                    totalTransactions1h / 3600,
				TotalNumberTransactions1h: totalTransactions1h,
			}
		}
	}

	delta1hr := startBlockTime - batchObject.OldestBlockTime

	// Now use the BatchObject to get the oldest block time
	TPS1hr := totalTransactions1h / delta1hr
	TPS1m := totalTransactions1m / 60

	log.Trace("Generated tx stats", "tps1m", TPS1m, "tps1hr", TPS1hr, "totalTransactions1h", totalTransactions1h, "totalTransactions1m", totalTransactions1m, "oldest1hBlockTime", batchObject.OldestBlockTime, "startBlockTime", startBlockTime, "difference1h", delta1hr, "difference1m", 60, "startBlockNum", uint64(startBlockNum))

	// Now totalTransactions1h and totalTransactions1m have the transaction counts for the last c_blocksPerHour and c_txBatchSize blocks respectively
	return &tps{
		TPS1m:                     TPS1m,
		TPS1hr:                    TPS1hr,
		TotalNumberTransactions1h: totalTransactions1h,
	}
}

func min(a uint64, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func (s *Service) assembleBlockDetailStats(block *types.Block) *blockDetailStats {
	if block == nil {
		log.Error("Block is nil")
		return nil
	}
	header := block.Header()
	difficulty := header.Difficulty().String()

	// Assemble and return the block stats
	return &blockDetailStats{
		Timestamp:    new(big.Int).SetUint64(header.Time()),
		ZoneHeight:   header.NumberU64(2),
		RegionHeight: header.NumberU64(1),
		PrimeHeight:  header.NumberU64(0),
		Chain:        common.NodeLocation.Name(),
		Entropy:      common.BigBitsToBits(s.backend.TotalLogS(block.Header())).String(),
		Difficulty:   difficulty,
	}
}

func (s *Service) assembleBlockAppendTimeStats(block *types.Block) *blockAppendTime {
	if block == nil {
		log.Error("Block is nil")
		return nil
	}
	header := block.Header()
	appendTime := block.GetAppendTime()

	log.Info("Raw Block Append Time", "appendTime", appendTime.Microseconds())

	// Assemble and return the block stats
	return &blockAppendTime{
		AppendTime:  appendTime,
		BlockNumber: header.Number(),
		Chain:       common.NodeLocation.Name(),
	}
}

func (s *Service) assembleBlockTransactionStats(block *types.Block) *blockTransactionStats {
	if block == nil {
		log.Error("Block is nil")
		return nil
	}
	header := block.Header()
	tps := s.calculateTPS(block)

	// Assemble and return the block stats
	return &blockTransactionStats{
		Timestamp:             new(big.Int).SetUint64(header.Time()),
		TotalNoTransactions1h: tps.TotalNumberTransactions1h,
		TPS1m:                 tps.TPS1m,
		TPS1hr:                tps.TPS1hr,
		Chain:                 common.NodeLocation.Name(),
	}
}

func getQuaiCPUUsage() (float64, error) {
	// 'ps' command options might vary depending on your OS
	cmd := exec.Command("ps", "aux")
	numCores := runtime.NumCPU()

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	var totalCpuUsage float64
	var cpuUsage float64
	for _, line := range lines {
		if strings.Contains(line, "go-quai") {
			fields := strings.Fields(line)
			if len(fields) > 2 {
				// Assuming %CPU is the third column, command is the eleventh
				cpuUsage, err = strconv.ParseFloat(fields[2], 64)
				if err != nil {
					return 0, err
				}
				totalCpuUsage += cpuUsage
			}
		}
	}

	if totalCpuUsage == 0 {
		return 0, errors.New("quai process not found")
	}

	return totalCpuUsage / float64(numCores), nil
}

func getQuaiRAMUsage() (uint64, error) {
	// Get a list of all running processes
	processes, err := process.Processes()
	if err != nil {
		return 0, err
	}

	var totalRam uint64

	// Debug: log number of processes
	log.Trace("Number of processes", "number", len(processes))

	for _, p := range processes {
		cmdline, err := p.Cmdline()
		if err != nil {
			// Debug: log error
			log.Trace("Error getting process cmdline", "error", err)
			continue
		}

		if strings.Contains(cmdline, "go-quai") {
			memInfo, err := p.MemoryInfo()
			if err != nil {
				return 0, err
			}
			totalRam += memInfo.RSS
		}
	}

	if totalRam == 0 {
		return 0, errors.New("go-quai process not found")
	}

	return totalRam, nil
}

// dirSize returns the size of a directory in bytes.
func dirSize(path string) (int64, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("du", "-sk", path)
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("du", "-bs", path)
	} else {
		return -1, errors.New("unsupported OS")
	}
	// Execute command
	output, err := cmd.Output()
	if err != nil {
		return -1, err
	}

	// Split the output and parse the size.
	sizeStr := strings.Split(string(output), "\t")[0]
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return -1, err
	}

	// If on macOS, convert size from kilobytes to bytes.
	if runtime.GOOS == "darwin" {
		size *= 1024
	}

	return size, nil
}

// diskTotalSize returns the total size of the disk in bytes.
func diskTotalSize() (int64, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("df", "-k", "/")
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("df", "--block-size=1K", "/")
	} else {
		return 0, errors.New("unsupported OS")
	}

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, errors.New("unexpected output from df command")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 2 {
		return 0, errors.New("unexpected output from df command")
	}

	totalSize, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, err
	}

	return totalSize * 1024, nil // convert from kilobytes to bytes
}
