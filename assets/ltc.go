package assets

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/bisoncraft/utxowallet/netparams"
	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"golang.org/x/crypto/scrypt"
)

var (
	ltcMainPowLimit, _   = new(big.Int).SetString("0x0fffff000000000000000000000000000000000000000000000000000000", 0)
	ltcGenesisCoinbaseTx = wire.MsgTx{
		Version: 1,
		TxIn: []*wire.TxIn{
			{
				PreviousOutPoint: wire.OutPoint{
					Hash:  chainhash.Hash{},
					Index: 0xffffffff,
				},
				SignatureScript: []byte{
					0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x40, 0x4e, 0x59, 0x20, 0x54, 0x69, 0x6d, 0x65, 0x73, // |.......@NY Times|
					0x20, 0x30, 0x35, 0x2f, 0x4f, 0x63, 0x74, 0x2f, 0x32, 0x30, 0x31, 0x31, 0x20, 0x53, 0x74, 0x65, // | 05/Oct/2011 Ste|
					0x76, 0x65, 0x20, 0x4a, 0x6f, 0x62, 0x73, 0x2c, 0x20, 0x41, 0x70, 0x70, 0x6c, 0x65, 0xe2, 0x80, // |ve Jobs, Apple..|
					0x99, 0x73, 0x20, 0x56, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x61, 0x72, 0x79, 0x2c, 0x20, 0x44, 0x69, // |.s Visionary, Di|
					0x65, 0x73, 0x20, 0x61, 0x74, 0x20, 0x35, 0x36, // |es at 56|

				},
				Sequence: 0xffffffff,
			},
		},
		TxOut: []*wire.TxOut{
			{
				Value: 0x12a05f200,
				PkScript: []byte{
					0x41, 0x4, 0x1, 0x84, 0x71, 0xf, 0xa6, 0x89,
					0xad, 0x50, 0x23, 0x69, 0xc, 0x80, 0xf3, 0xa4,
					0x9c, 0x8f, 0x13, 0xf8, 0xd4, 0x5b, 0x8c, 0x85,
					0x7f, 0xbc, 0xbc, 0x8b, 0xc4, 0xa8, 0xe4, 0xd3,
					0xeb, 0x4b, 0x10, 0xf4, 0xd4, 0x60, 0x4f, 0xa0,
					0x8d, 0xce, 0x60, 0x1a, 0xaf, 0xf, 0x47, 0x2,
					0x16, 0xfe, 0x1b, 0x51, 0x85, 0xb, 0x4a, 0xcf,
					0x21, 0xb1, 0x79, 0xc4, 0x50, 0x70, 0xac, 0x7b,
					0x3, 0xa9, 0xac,
				},
			},
		},
		LockTime: 0,
	}
	ltcGenesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
		0xd9, 0xce, 0xd4, 0xed, 0x11, 0x30, 0xf7, 0xb7,
		0xfa, 0xad, 0x9b, 0xe2, 0x53, 0x23, 0xff, 0xaf,
		0xa3, 0x32, 0x32, 0xa1, 0x7c, 0x3e, 0xdf, 0x6c,
		0xfd, 0x97, 0xbe, 0xe6, 0xba, 0xfb, 0xdd, 0x97,
	})
)

var LTCParams = map[string]*netparams.ChainParams{
	"mainnet": {
		Chain:       "ltc",
		Name:        "mainnet",
		Net:         0xdbb6c0fb,
		DefaultPort: "9333",
		DNSSeeds: []chaincfg.DNSSeed{
			{"seed-a.litecoin.loshan.co.uk", true},
			{"dnsseed.thrasher.io", true},
			{"dnsseed.litecointools.com", false},
			{"dnsseed.litecoinpool.org", false},
		},
		GenesisBlock: &wire.MsgBlock{
			Header: wire.BlockHeader{
				Version:    1,
				PrevBlock:  chainhash.Hash{}, // 0000000000000000000000000000000000000000000000000000000000000000
				MerkleRoot: ltcGenesisMerkleRoot,
				Timestamp:  time.Unix(1317972665, 0),
				Bits:       0x1e0ffff0,
				Nonce:      2084524493,
			},
			Transactions: []*wire.MsgTx{&ltcGenesisCoinbaseTx},
		},
		GenesisHash: hashPointer([chainhash.HashSize]byte{ // Make go vet happy.
			0xe2, 0xbf, 0x04, 0x7e, 0x7e, 0x5a, 0x19, 0x1a,
			0xa4, 0xef, 0x34, 0xd3, 0x14, 0x97, 0x9d, 0xc9,
			0x98, 0x6e, 0x0f, 0x19, 0x25, 0x1e, 0xda, 0xba,
			0x59, 0x40, 0xfd, 0x1f, 0xe3, 0x65, 0xa7, 0x12,
		}),
		PowLimit:                 ltcMainPowLimit,
		TargetTimespan:           (time.Hour * 24 * 3) + (time.Hour * 12), // 3.5 days
		TargetTimePerBlock:       (time.Minute * 2) + (time.Second * 30),  // 2.5 minutes
		RetargetAdjustmentFactor: 4,                                       // 25% less, 400% more
		Checkpoints: []chaincfg.Checkpoint{
			{1500, newHashFromStr("841a2965955dd288cfa707a755d05a54e45f8bd476835ec9af4402a2b59a2967")},
			{4032, newHashFromStr("9ce90e427198fc0ef05e5905ce3503725b80e26afd35a987965fd7e3d9cf0846")},
			{8064, newHashFromStr("eb984353fc5190f210651f150c40b8a4bab9eeeff0b729fcb3987da694430d70")},
			{16128, newHashFromStr("602edf1859b7f9a6af809f1d9b0e6cb66fdc1d4d9dcd7a4bec03e12a1ccd153d")},
			{23420, newHashFromStr("d80fdf9ca81afd0bd2b2a90ac3a9fe547da58f2530ec874e978fce0b5101b507")},
			{50000, newHashFromStr("69dc37eb029b68f075a5012dcc0419c127672adb4f3a32882b2b3e71d07a20a6")},
			{80000, newHashFromStr("4fcb7c02f676a300503f49c764a89955a8f920b46a8cbecb4867182ecdb2e90a")},
			{120000, newHashFromStr("bd9d26924f05f6daa7f0155f32828ec89e8e29cee9e7121b026a7a3552ac6131")},
			{161500, newHashFromStr("dbe89880474f4bb4f75c227c77ba1cdc024991123b28b8418dbbf7798471ff43")},
			{179620, newHashFromStr("2ad9c65c990ac00426d18e446e0fd7be2ffa69e9a7dcb28358a50b2b78b9f709")},
			{240000, newHashFromStr("7140d1c4b4c2157ca217ee7636f24c9c73db39c4590c4e6eab2e3ea1555088aa")},
			{383640, newHashFromStr("2b6809f094a9215bafc65eb3f110a35127a34be94b7d0590a096c3f126c6f364")},
			{409004, newHashFromStr("487518d663d9f1fa08611d9395ad74d982b667fbdc0e77e9cf39b4f1355908a3")},
			{456000, newHashFromStr("bf34f71cc6366cd487930d06be22f897e34ca6a40501ac7d401be32456372004")},
			{638902, newHashFromStr("15238656e8ec63d28de29a8c75fcf3a5819afc953dcd9cc45cecc53baec74f38")},
			{721000, newHashFromStr("198a7b4de1df9478e2463bd99d75b714eab235a2e63e741641dc8a759a9840e5")},
		},
		Bech32HRPSegwit:         "ltc",                           // always ltc for main net
		PubKeyHashAddrID:        0x30,                            // starts with L
		ScriptHashAddrID:        0x32,                            // starts with M
		PrivateKeyID:            0xB0,                            // starts with 6 (uncompressed) or T (compressed)
		WitnessPubKeyHashAddrID: 0x06,                            // starts with p2
		WitnessScriptHashAddrID: 0x0A,                            // starts with 7Xh
		HDPrivateKeyID:          [4]byte{0x04, 0x88, 0xad, 0xe4}, // starts with xprv
		HDPublicKeyID:           [4]byte{0x04, 0x88, 0xb2, 0x1e}, // starts with xpub
		HDCoinType:              2,
		BIP0034Height:           710000,
		BIP0065Height:           918684,
		BIP0066Height:           811879,
		CheckPoW:                checkPoWScrypt,
		CoinbaseMaturity:        100,
		MaxSatoshi:              84e6 * btcutil.SatoshiPerBitcoin,
	},
	"testnet": {
		Chain:       "ltc",
		Name:        "testnet4",
		Net:         0xf1c8d2fd,
		DefaultPort: "19335",
		DNSSeeds: []chaincfg.DNSSeed{
			{"testnet-seed.litecointools.com", false},
			{"seed-b.litecoin.loshan.co.uk", true},
			{"dnsseed-testnet.thrasher.io", true},
		},
		GenesisBlock: &wire.MsgBlock{
			Header: wire.BlockHeader{
				Version:    1,
				PrevBlock:  chainhash.Hash{}, // 0000000000000000000000000000000000000000000000000000000000000000
				MerkleRoot: ltcGenesisMerkleRoot,
				Timestamp:  time.Unix(1486949366, 0),
				Bits:       0x1e0ffff0,
				Nonce:      293345,
			},
			Transactions: []*wire.MsgTx{&ltcGenesisCoinbaseTx},
		},
		GenesisHash: hashPointer([chainhash.HashSize]byte{ // Make go vet happy.
			0xa0, 0x29, 0x3e, 0x4e, 0xeb, 0x3d, 0xa6, 0xe6,
			0xf5, 0x6f, 0x81, 0xed, 0x59, 0x5f, 0x57, 0x88,
			0xd, 0x1a, 0x21, 0x56, 0x9e, 0x13, 0xee, 0xfd,
			0xd9, 0x51, 0x28, 0x4b, 0x5a, 0x62, 0x66, 0x49,
		}),
		PowLimit:                 ltcMainPowLimit,
		BIP0034Height:            76,
		BIP0065Height:            76,
		BIP0066Height:            76,
		TargetTimespan:           (time.Hour * 24 * 3) + (time.Hour * 12), // 3.5 days
		TargetTimePerBlock:       (time.Minute * 2) + (time.Second * 30),  // 2.5 minutes
		RetargetAdjustmentFactor: 4,
		Checkpoints: []chaincfg.Checkpoint{
			{26115, newHashFromStr("817d5b509e91ab5e439652eee2f59271bbc7ba85021d720cdb6da6565b43c14f")},
			{43928, newHashFromStr("7d86614c153f5ef6ad878483118ae523e248cd0dd0345330cb148e812493cbb4")},
			{69296, newHashFromStr("66c2f58da3cfd282093b55eb09c1f5287d7a18801a8ff441830e67e8771010df")},
			{99949, newHashFromStr("8dd471cb5aecf5ead91e7e4b1e932c79a0763060f8d93671b6801d115bfc6cde")},
			{159256, newHashFromStr("ab5b0b9968842f5414804591119d6db829af606864b1959a25d6f5c114afb2b7")},
			{2394367, newHashFromStr("bc5829f4973d0797755efee11313687b3c63ee2f70b60b62eebcd10283534327")},
		},
		Bech32HRPSegwit:         "tltc",                          // always tltc for test net
		PubKeyHashAddrID:        0x6f,                            // starts with m or n
		ScriptHashAddrID:        0x3a,                            // starts with Q
		WitnessPubKeyHashAddrID: 0x52,                            // starts with QW
		WitnessScriptHashAddrID: 0x31,                            // starts with T7n
		PrivateKeyID:            0xef,                            // starts with 9 (uncompressed) or c (compressed)
		HDPrivateKeyID:          [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
		HDPublicKeyID:           [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub
		HDCoinType:              1,
		CheckPoW:                checkPoWScrypt,
		CoinbaseMaturity:        100,
		MaxSatoshi:              84e6 * btcutil.SatoshiPerBitcoin,
	},
	"simnet": {
		Chain:       "ltc",
		Name:        "regtest",
		Net:         0xdab5bffa,
		DefaultPort: "18444",
		DNSSeeds:    []chaincfg.DNSSeed{},

		// Chain parameters
		GenesisBlock: &wire.MsgBlock{
			Header: wire.BlockHeader{
				Version:    1,
				PrevBlock:  chainhash.Hash{},
				MerkleRoot: ltcGenesisMerkleRoot,
				Timestamp:  time.Unix(1296688602, 0), // 2011-02-02 23:16:42 +0000 UTC
				Bits:       0x207fffff,               // 545259519 [7fffff0000000000000000000000000000000000000000000000000000000000]
				Nonce:      0,
			},
			Transactions: []*wire.MsgTx{&ltcGenesisCoinbaseTx},
		},
		GenesisHash: hashPointer([chainhash.HashSize]byte{ // Make go vet happy.
			0xf9, 0x16, 0xc4, 0x56, 0xfc, 0x51, 0xdf, 0x62,
			0x78, 0x85, 0xd7, 0xd6, 0x74, 0xed, 0x02, 0xdc,
			0x88, 0xa2, 0x25, 0xad, 0xb3, 0xf0, 0x2a, 0xd1,
			0x3e, 0xb4, 0x93, 0x8f, 0xf3, 0x27, 0x08, 0x53,
		}),
		PowLimit:                 new(big.Int).Sub(new(big.Int).Lsh(bigOne, 255), bigOne),
		BIP0034Height:            100000000,                               // Not active - Permit ver 1 blocks
		BIP0065Height:            1351,                                    // Used by regression tests
		BIP0066Height:            1251,                                    // Used by regression tests
		TargetTimespan:           (time.Hour * 24 * 3) + (time.Hour * 12), // 3.5 days
		TargetTimePerBlock:       (time.Minute * 2) + (time.Second * 30),  // 2.5 minutes
		RetargetAdjustmentFactor: 4,                                       // 25% less, 400% more

		// Checkpoints ordered from oldest to newest.
		Checkpoints:      nil,
		Bech32HRPSegwit:  "rltc",                          // always rltc for reg test net
		PubKeyHashAddrID: 0x6f,                            // starts with m or n
		ScriptHashAddrID: 0x3a,                            // starts with Q
		PrivateKeyID:     0xef,                            // starts with 9 (uncompressed) or c (compressed)
		HDPrivateKeyID:   [4]byte{0x04, 0x35, 0x83, 0x94}, // starts with tprv
		HDPublicKeyID:    [4]byte{0x04, 0x35, 0x87, 0xcf}, // starts with tpub
		HDCoinType:       1,
		CheckPoW:         checkPoWScrypt,
		CoinbaseMaturity: 100,
		MaxSatoshi:       84e6 * btcutil.SatoshiPerBitcoin,
	},
}

func checkPoWScrypt(hdr *wire.BlockHeader) error {
	var powHash chainhash.Hash

	buf := bytes.NewBuffer(make([]byte, 0, wire.MaxBlockHeaderPayload))
	hdr.Serialize(buf)

	scryptHash, _ := scrypt.Key(buf.Bytes(), buf.Bytes(), 1024, 1, 1, 32)
	copy(powHash[:], scryptHash)

	target := blockchain.CompactToBig(hdr.Bits)

	hashNum := blockchain.HashToBig(&powHash)
	if hashNum.Cmp(target) > 0 {
		return fmt.Errorf("block hash of %064x is higher than "+
			"expected max of %064x", hashNum, target)
	}
	return nil
}
