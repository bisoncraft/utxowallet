// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bisoncraft/utxowallet/assets"
	"github.com/bisoncraft/utxowallet/bisonwire"
	"github.com/bisoncraft/utxowallet/internal/cfgutil"
	"github.com/bisoncraft/utxowallet/internal/legacy/keystore"
	"github.com/bisoncraft/utxowallet/netparams"
	"github.com/bisoncraft/utxowallet/spv"
	"github.com/bisoncraft/utxowallet/wallet"
	"github.com/btcsuite/btcd/btcutil"
	flags "github.com/jessevdk/go-flags"
)

const (
	defaultConfigFilename = "utxowallet.conf"
	defaultLogLevel       = "info"
	defaultLogDirname     = "logs"
	defaultLogFilename    = "utxowallet.log"
)

var (
	defaultAppDataDir = btcutil.AppDataDir("utxowallet", false)
	defaultConfigFile = filepath.Join(defaultAppDataDir, defaultConfigFilename)
)

//nolint:lll
type config struct {
	// General application behavior
	Chain         string                  `long:"chain" description:"Blockchain (btc, ltc)"`
	ConfigFile    *cfgutil.ExplicitString `short:"C" long:"configfile" description:"Path to configuration file"`
	ShowVersion   bool                    `short:"V" long:"version" description:"Display version information and exit"`
	Create        bool                    `long:"create" description:"Create the wallet if it does not exist"`
	AppDataDir    *cfgutil.ExplicitString `short:"A" long:"appdata" description:"Application data directory for wallet config, databases and logs"`
	Testnet       bool                    `long:"testnet" description:"Use the test Bitcoin network (version 3) (default mainnet)"`
	Simnet        bool                    `long:"simnet" description:"Use the simulation test network (default mainnet)"`
	RegressionNet bool                    `long:"regtest" description:"Use the regression test network (default mainnet)"`
	NoInitialLoad bool                    `long:"noinitialload" description:"Defer wallet creation/opening on startup and enable loading wallets over RPC"`
	DebugLevel    string                  `short:"d" long:"debuglevel" description:"Logging level {trace, debug, info, warn, error, critical}"`
	LogDir        string                  `long:"logdir" description:"Directory to log output."`
	Profile       string                  `long:"profile" description:"Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65536"`
	DBTimeout     time.Duration           `long:"dbtimeout" description:"The timeout value to use when opening the wallet database."`

	// Wallet options
	WalletPass string `long:"walletpass" default-mask:"-" description:"The public wallet password -- Only required if the wallet was created with one"`

	// SPV client options
	UseSPV       bool          `long:"usespv" description:"Enables the experimental use of SPV rather than RPC for chain synchronization"`
	AddPeers     []string      `short:"a" long:"addpeer" description:"Add a peer to connect with at startup"`
	ConnectPeers []string      `long:"connect" description:"Connect only to the specified peers at startup"`
	MaxPeers     int           `long:"maxpeers" description:"Max number of inbound and outbound peers"`
	BanDuration  time.Duration `long:"banduration" description:"How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second"`
	BanThreshold uint32        `long:"banthreshold" description:"Maximum allowed ban score before disconnecting and banning misbehaving peers."`
}

// cleanAndExpandPath expands environement variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// NOTE: The os.ExpandEnv doesn't work with Windows cmd.exe-style
	// %VARIABLE%, but they variables can still be expanded via POSIX-style
	// $VARIABLE.
	path = os.ExpandEnv(path)

	if !strings.HasPrefix(path, "~") {
		return filepath.Clean(path)
	}

	// Expand initial ~ to the current user's home directory, or ~otheruser
	// to otheruser's home directory.  On Windows, both forward and backward
	// slashes can be used.
	path = path[1:]

	var pathSeparators string
	if runtime.GOOS == "windows" {
		pathSeparators = string(os.PathSeparator) + "/"
	} else {
		pathSeparators = string(os.PathSeparator)
	}

	userName := ""
	if i := strings.IndexAny(path, pathSeparators); i != -1 {
		userName = path[:i]
		path = path[i:]
	}

	homeDir := ""
	var u *user.User
	var err error
	if userName == "" {
		u, err = user.Current()
	} else {
		u, err = user.Lookup(userName)
	}
	if err == nil {
		homeDir = u.HomeDir
	}
	// Fallback to CWD if user lookup fails or user has no home directory.
	if homeDir == "" {
		homeDir = "."
	}

	return filepath.Join(homeDir, path)
}

// validLogLevel returns whether or not logLevel is a valid debug log level.
func validLogLevel(logLevel string) bool {
	switch logLevel {
	case "trace":
		fallthrough
	case "debug":
		fallthrough
	case "info":
		fallthrough
	case "warn":
		fallthrough
	case "error":
		fallthrough
	case "critical":
		return true
	}
	return false
}

// supportedSubsystems returns a sorted slice of the supported subsystems for
// logging purposes.
func supportedSubsystems() []string {
	// Convert the subsystemLoggers map keys to a slice.
	subsystems := make([]string, 0, len(subsystemLoggers))
	for subsysID := range subsystemLoggers {
		subsystems = append(subsystems, subsysID)
	}

	// Sort the subsytems for stable display.
	sort.Strings(subsystems)
	return subsystems
}

// parseAndSetDebugLevels attempts to parse the specified debug level and set
// the levels accordingly.  An appropriate error is returned if anything is
// invalid.
func parseAndSetDebugLevels(debugLevel string) error {
	// When the specified string doesn't have any delimters, treat it as
	// the log level for all subsystems.
	if !strings.Contains(debugLevel, ",") && !strings.Contains(debugLevel, "=") {
		// Validate debug log level.
		if !validLogLevel(debugLevel) {
			str := "the specified debug level [%v] is invalid"
			return fmt.Errorf(str, debugLevel)
		}

		// Change the logging level for all subsystems.
		setLogLevels(debugLevel)

		return nil
	}

	// Split the specified string into subsystem/level pairs while detecting
	// issues and update the log levels accordingly.
	for _, logLevelPair := range strings.Split(debugLevel, ",") {
		if !strings.Contains(logLevelPair, "=") {
			str := "the specified debug level contains an invalid " +
				"subsystem/level pair [%v]"
			return fmt.Errorf(str, logLevelPair)
		}

		// Extract the specified subsystem and log level.
		fields := strings.Split(logLevelPair, "=")
		subsysID, logLevel := fields[0], fields[1]

		// Validate subsystem.
		if _, exists := subsystemLoggers[subsysID]; !exists {
			str := "the specified subsystem [%v] is invalid -- " +
				"supported subsytems %v"
			return fmt.Errorf(str, subsysID, supportedSubsystems())
		}

		// Validate log level.
		if !validLogLevel(logLevel) {
			str := "the specified debug level [%v] is invalid"
			return fmt.Errorf(str, logLevel)
		}

		setLogLevel(subsysID, logLevel)
	}

	return nil
}

// loadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
//  1. Start with a default config with sane settings
//  2. Pre-parse the command line to check for an alternative config file
//  3. Load configuration file overwriting defaults with any specified options
//  4. Parse CLI options and overwrite/add any specified options
//
// The above results in utxowallet functioning properly without any config
// settings while still allowing the user to override settings with config files
// and command line options.  Command line options always take precedence.
func loadConfig() (*config, string, *netparams.ChainParams, error) {
	// Default config.
	cfg := config{
		DebugLevel:   defaultLogLevel,
		ConfigFile:   cfgutil.NewExplicitString(defaultConfigFile),
		AppDataDir:   cfgutil.NewExplicitString(defaultAppDataDir),
		WalletPass:   wallet.InsecurePubPassphrase,
		UseSPV:       false,
		AddPeers:     []string{},
		ConnectPeers: []string{},
		MaxPeers:     spv.MaxPeers,
		BanDuration:  spv.BanDuration,
		BanThreshold: spv.BanThreshold,
		DBTimeout:    wallet.DefaultDBTimeout,
	}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.
	preCfg := cfg
	preParser := flags.NewParser(&preCfg, flags.Default)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			preParser.WriteHelp(os.Stderr)
		}
		return nil, "", nil, err
	}

	// For now, the chain has to be set at the command-line.
	if preCfg.Chain == "" {
		return nil, "", nil, fmt.Errorf("no chain defined")
	}

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version())
		os.Exit(0)
	}

	// Load additional config from file.
	var configFileError error
	parser := flags.NewParser(&cfg, flags.Default)
	configFilePath := preCfg.ConfigFile.Value
	if preCfg.ConfigFile.ExplicitlySet() {
		configFilePath = cleanAndExpandPath(configFilePath)
	} else if preCfg.AppDataDir.Value != defaultAppDataDir {
		configFilePath = filepath.Join(preCfg.AppDataDir.Value, defaultConfigFilename)
	}
	err = flags.NewIniParser(parser).ParseFile(configFilePath)
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			fmt.Fprintln(os.Stderr, err)
			parser.WriteHelp(os.Stderr)
			return nil, "", nil, err
		}
		configFileError = err
	}

	// Parse command line options again to ensure they take precedence.
	_, err = parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stderr)
		}
		return nil, "", nil, err
	}

	// They must choose a blockchain.
	if _, err := bisonwire.ChainFromString(cfg.Chain); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}

	net := "mainnet"
	switch {
	case cfg.Simnet:
		net = "simnet"
	case cfg.Testnet:
		net = "testnet"
	}
	netParams, err := assets.NetParams(cfg.Chain, net)
	if err != nil {
		return nil, "", nil, err
	}

	// Special show command to list supported subsystems and exit.
	if cfg.DebugLevel == "show" {
		fmt.Println("Supported subsystems", supportedSubsystems())
		os.Exit(0)
	}

	// Parse, validate, and set debug log level(s).
	if err := parseAndSetDebugLevels(cfg.DebugLevel); err != nil {
		err := fmt.Errorf("%s: %w", "loadConfig", err)
		fmt.Fprintln(os.Stderr, err)
		parser.WriteHelp(os.Stderr)
		return nil, "", nil, err
	}

	// Ensure the wallet exists or create it when the create flag is set.
	netDir := filepath.Join(cfg.AppDataDir.Value, cfg.Chain, netParams.Name)
	dbPath := filepath.Join(netDir, wallet.WalletDBName)
	cfg.LogDir = filepath.Join(netDir, "logs")

	// Initialize log rotation.  After log rotation has been initialized, the
	// logger variables may be used.
	initLogRotator(filepath.Join(cfg.LogDir, defaultLogFilename))

	dbFileExists, err := cfgutil.FileExists(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil, "", nil, err
	}

	if cfg.Create {
		// Error if the create flag is set and the wallet already
		// exists.
		if dbFileExists {
			err := fmt.Errorf("the wallet database file `%v` "+
				"already exists", dbPath)
			fmt.Fprintln(os.Stderr, err)
			return nil, "", nil, err
		}

		// Ensure the data directory for the network exists.
		if err := checkCreateDir(netDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, "", nil, err
		}

		// Perform the initial wallet creation wizard.
		if err := createWallet(&cfg, netDir, netParams); err != nil {
			fmt.Fprintln(os.Stderr, "Unable to create wallet:", err)
			return nil, "", nil, err
		}

		// Created successfully, so exit now with success.
		os.Exit(0)
	} else if !dbFileExists && !cfg.NoInitialLoad {
		keystorePath := filepath.Join(netDir, keystore.Filename)
		keystoreExists, err := cfgutil.FileExists(keystorePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return nil, "", nil, err
		}
		if !keystoreExists {
			err = fmt.Errorf("the wallet does not exist, run with " +
				"the --create option to initialize and create it")
		} else {
			err = fmt.Errorf("the wallet is in legacy format, run " +
				"with the --create option to import it")
		}
		fmt.Fprintln(os.Stderr, err)
		return nil, "", nil, err
	}

	spv.MaxPeers = cfg.MaxPeers
	spv.BanDuration = cfg.BanDuration
	spv.BanThreshold = cfg.BanThreshold
	// Warn about missing config file after the final command line parse
	// succeeds.  This prevents the warning on help messages and invalid
	// options.
	if configFileError != nil {
		log.Warnf("%v", configFileError)
	}

	return &cfg, netDir, netParams, nil
}
