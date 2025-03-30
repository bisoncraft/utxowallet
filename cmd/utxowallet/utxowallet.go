// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // nolint:gosec
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/bisoncraft/utxowallet/bisonwire"
	"github.com/bisoncraft/utxowallet/chain"
	"github.com/bisoncraft/utxowallet/netparams"
	"github.com/bisoncraft/utxowallet/spv"
	"github.com/bisoncraft/utxowallet/waddrmgr"
	"github.com/bisoncraft/utxowallet/wallet"
	"github.com/bisoncraft/utxowallet/walletdb"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var (
	cfg *config
)

func main() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Work around defer not working after os.Exit.
	if err := walletMain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// walletMain is a work-around main function that is required since deferred
// functions (such as log flushing) are not called with calls to os.Exit.
// Instead, main runs this function and checks for a non-nil error, at which
// point any defers have already run, and if the error is non-nil, the program
// can be exited with an error exit status.
func walletMain() error {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	tcfg, netDir, netParams, err := loadConfig()
	if err != nil {
		return err
	}
	cfg = tcfg
	defer func() {
		if logRotator != nil {
			logRotator.Close()
		}
	}()

	// Show version at startup.
	log.Infof("Version %s", version())

	if cfg.Profile != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Profile)
			log.Infof("Profile server listening on %s", listenAddr)
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			log.Errorf("%v", http.ListenAndServe(listenAddr, nil))
		}()
	}

	loader := wallet.NewLoader(
		netParams, netDir, true, cfg.DBTimeout, 250,
	)

	// Create and start chain RPC client so it's ready to connect to
	// the wallet when loaded later.
	if !cfg.NoInitialLoad {
		go run(loader, netDir, netParams, cfg.DevRPC)
	}

	if !cfg.NoInitialLoad {
		// Load the wallet database.  It must have been created already
		// or this will return an appropriate error.
		_, err = loader.OpenExistingWallet([]byte(cfg.WalletPass), true)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	// Add interrupt handlers to shutdown the various process components
	// before exiting.  Interrupt handlers run in LIFO order, so the wallet
	// (which should be closed last) is added first.
	addInterruptHandler(func() {
		err := loader.UnloadWallet()
		if err != nil && err != wallet.ErrNotLoaded {
			log.Errorf("Failed to close wallet: %v", err)
		}
	})

	<-interruptHandlersDone
	log.Info("Shutdown complete")
	return nil
}

func run(loader *wallet.Loader, netDir string, chainParams *netparams.ChainParams, devRPC bool) {

	for {
		spvdb, err := walletdb.Create(
			"bdb", filepath.Join(netDir, "spv.db"),
			true, cfg.DBTimeout,
		)
		if err != nil {
			log.Errorf("Unable to create Neutrino DB: %s", err)
			continue
		}
		defer spvdb.Close()
		chainService, err := spv.NewChainService(
			spv.Config{
				Chain:        bisonwire.Chain(cfg.Chain),
				DataDir:      netDir,
				Database:     spvdb,
				ChainParams:  chainParams,
				ConnectPeers: cfg.ConnectPeers,
				AddPeers:     cfg.AddPeers,
			})
		if err != nil {
			log.Errorf("Couldn't create Neutrino ChainService: %s", err)
			continue
		}
		chainClient := chain.NewNeutrinoClient(chainParams, chainService)
		err = chainClient.Start()
		if err != nil {
			log.Errorf("Couldn't start Neutrino client: %s", err)
		}

		// Rather than inlining this logic directly into the loader
		// callback, a function variable is used to avoid running any of
		// this after the client disconnects by setting it to nil.  This
		// prevents the callback from associating a wallet loaded at a
		// later time with a client that has already disconnected.  A
		// mutex is used to make this concurrent safe.
		associateRPCClient := func(w *wallet.Wallet) {
			w.Synchronize(chainClient)
		}
		mu := new(sync.Mutex)
		loader.RunAfterLoad(func(w *wallet.Wallet) {
			mu.Lock()
			associate := associateRPCClient
			mu.Unlock()
			if associate != nil {
				associate(w)
			}
			// Run the dev rpc is not on mainnet
			if devRPC {
				go runDevRPC(w, chainService)
			}
		})

		chainClient.WaitForShutdown()

		mu.Lock()
		associateRPCClient = nil
		mu.Unlock()

		loadedWallet, ok := loader.LoadedWallet()
		if ok {
			// Do not attempt a reconnect when the wallet was
			// explicitly stopped.
			if loadedWallet.ShuttingDown() {
				return
			}

			loadedWallet.SetChainSynced(false)

			// TODO: Rework the wallet so changing the RPC client
			// does not require stopping and restarting everything.
			loadedWallet.Stop()
			loadedWallet.WaitForShutdown()
			loadedWallet.Start()
		}
	}
}

func runDevRPC(wllt *wallet.Wallet, chainSvc *spv.ChainService) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Use POST", http.StatusMethodNotAllowed)
			return
		}

		var args []string
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		resp, err := handleRPCCall(wllt, chainSvc, args)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		b, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("JSON encode error: %v", err)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(append(b, byte('\n')))
		if err != nil {
			log.Errorf("Write error: %v", err)
		}
	})
	http.ListenAndServe(":44825", nil)
}

func handleRPCCall(w *wallet.Wallet, chainSvc *spv.ChainService, args []string) (any, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("no api method specified")
	}

	method := args[0]
	args = args[1:]
	const acctNum, minConf, feePerKB = 0, 0, 10_000
	scope := waddrmgr.KeyScopeBIP0084
	switch method {
	case "getbalance":
		return w.CalculateAccountBalances(acctNum, minConf)
	case "showaddress":
		addr, err := w.CurrentAddress(acctNum, scope)
		if err != nil {
			return nil, err
		}
		return addr.String(), nil
	case "getnewaddress":
		addr, err := w.NewAddress(acctNum, scope)
		if err != nil {
			return nil, err
		}
		return addr.String(), nil
	case "sendtoaddress":
		if len(args) != 2 {
			return nil, fmt.Errorf("wrong number of args. expected [address, amount]")
		}
		addrStr, amtStr := args[0], args[1]
		addr, err := btcutil.DecodeAddress(addrStr, w.ChainParams())
		if err != nil {
			return nil, fmt.Errorf("error decoding address %q: %w", addrStr, err)
		}
		pkScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, fmt.Errorf("error generating pk script: %w", err)
		}
		amt, err := strconv.ParseFloat(amtStr, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing amount from %q: %w", amtStr, err)
		}
		v, err := btcutil.NewAmount(amt)
		if err != nil {
			return nil, fmt.Errorf("error creating amount from float: %w", err)
		}
		txOut := &wire.TxOut{
			Value:    int64(v),
			PkScript: pkScript,
		}
		msgTx, err := w.SendOutputs([]*wire.TxOut{txOut}, &scope, acctNum, minConf, feePerKB, wallet.CoinSelectionRandom, "dev-rpc")
		if err != nil {
			return nil, fmt.Errorf("error sending: %w", err)
		}
		h := msgTx.TxHash()
		return h.String(), nil
	case "gettransaction":
		if len(args) != 1 {
			return nil, fmt.Errorf("wrong number of args. expected [txid]")
		}
		txID := args[0]
		txHash, err := chainhash.NewHashFromStr(txID)
		if err != nil {
			return nil, fmt.Errorf("error parsing tx hash from %q: %w", txID, err)
		}
		return w.GetTransaction(*txHash)
	case "unlock":
		if len(args) != 1 {
			return nil, fmt.Errorf("wrong number of args. expected [password]")
		}
		if err := w.Unlock([]byte(args[0]), nil); err != nil {
			return nil, fmt.Errorf("error unlocking: %w", err)
		}
		return "ok", nil
	case "unbanall":
		if err := chainSvc.UnbanPeers(); err != nil {
			return nil, err
		}
		return "ok", nil
	default:
		return nil, fmt.Errorf("unknown method %q", method)
	}
}
