package main

import (
	"fmt"
	"github.com/idena-network/idena-contract-runner/api"
	"github.com/idena-network/idena-contract-runner/chain"
	"github.com/idena-network/idena-go/common/hexutil"
	"github.com/idena-network/idena-go/core/mempool"
	"github.com/idena-network/idena-go/crypto"
	"github.com/idena-network/idena-go/ipfs"
	"github.com/idena-network/idena-go/log"
	"github.com/idena-network/idena-go/rpc"
	"net"
	"strings"
)

type Runner struct {
	chain        *chain.MemBlockchain
	stop         chan struct{}
	httpListener net.Listener
	httpServer   *rpc.Server
}

func NewRunner() *Runner {
	return &Runner{
		stop: make(chan struct{}),
	}
}

func (r *Runner) Start() error {
	key, _ := crypto.GenerateKey()
	log.Info("Generated god address", "addr", crypto.PubkeyToAddress(key.PublicKey).Hex(), "key", hexutil.Encode(crypto.FromECDSA(key)))

	r.chain = chain.NewMemBlockchain(key)
	if err := r.startRPC(); err != nil {
		return err
	}
	return nil
}

func (r *Runner) WaitForStop() {
	<-r.stop
}

func (r *Runner) startRPC() error {
	// Gather all the possible APIs to surface
	apis := r.apis()
	cfg := rpc.GetDefaultRPCConfig("localhost", 3333)
	cfg.HTTPModules = append(cfg.HTTPModules, "chain")
	if err := r.startHTTP(cfg.HTTPEndpoint(), apis, cfg.HTTPModules, cfg.HTTPCors, cfg.HTTPVirtualHosts, cfg.HTTPTimeouts, cfg.APIKey); err != nil {
		return err
	}
	return nil
}

func (r *Runner) startHTTP(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string, timeouts rpc.HTTPTimeouts, apiKey string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, httpServer, _, err := rpc.StartHTTPEndpoint(endpoint, apis, modules, cors, vhosts, timeouts, apiKey)
	if err != nil {
		return err
	}
	log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%s", endpoint), "cors", strings.Join(cors, ","), "vhosts", strings.Join(vhosts, ","))

	r.httpListener = listener
	r.httpServer = httpServer

	return nil
}

// apis returns the collection of RPC descriptors this node offers.
func (r *Runner) apis() []rpc.API {

	baseApi := api.NewBaseApi(r.chain, r.chain.KeyStore(), r.chain.SecStore(), ipfs.NewMemoryIpfsProxy(), r.TxPool())

	return []rpc.API{
		{
			Namespace: "contract",
			Version:   "1.0",
			Service:   api.NewContractApi(baseApi, r.chain),
			Public:    true,
		},
		{
			Namespace: "chain",
			Version:   "1.0",
			Service:   api.NewChainApi(baseApi, r.chain, r.TxPool()),
			Public:    true,
		},
	}
}

func (r *Runner) TxPool() *mempool.TxPool {
	return r.chain.TxPool()
}
