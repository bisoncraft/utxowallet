// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof" // nolint:gosec
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/bisoncraft/utxowallet/bisonwire"
	"github.com/bisoncraft/utxowallet/chain"
	"github.com/bisoncraft/utxowallet/netparams"
	"github.com/bisoncraft/utxowallet/spv"
	"github.com/bisoncraft/utxowallet/wallet"
	"github.com/bisoncraft/utxowallet/walletdb"
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
		go run(loader, netDir, netParams)
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

func run(loader *wallet.Loader, netDir string, netParams *netparams.ChainParams) {

	for {
		var (
			chainClient chain.Interface
			err         error
		)

		var (
			chainService *spv.ChainService
			spvdb        walletdb.DB
		)
		spvdb, err = walletdb.Create(
			"bdb", filepath.Join(netDir, "spv.db"),
			true, cfg.DBTimeout,
		)
		if err != nil {
			log.Errorf("Unable to create Neutrino DB: %s", err)
			continue
		}
		defer spvdb.Close()
		chainService, err = spv.NewChainService(
			spv.Config{
				Chain:        bisonwire.Chain(cfg.Chain),
				DataDir:      netDir,
				Database:     spvdb,
				ChainParams:  netParams,
				ConnectPeers: cfg.ConnectPeers,
				AddPeers:     cfg.AddPeers,
			})
		if err != nil {
			log.Errorf("Couldn't create Neutrino ChainService: %s", err)
			continue
		}
		chainClient = chain.NewNeutrinoClient(netParams, chainService)
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
