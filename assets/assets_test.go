package assets

import "testing"

func TestParams(t *testing.T) {
	for chain, nets := range map[string][]string{
		"btc": {"mainnet", "testnet", "simnet"},
		"ltc": {"mainnet", "testnet", "simnet"},
	} {
		for _, net := range nets {
			t.Run(chain+"."+net, func(t *testing.T) {
				chainParams, _ := NetParams(chain, net)
				h := chainParams.GenesisBlock.BlockHash()
				if h != *chainParams.GenesisHash {
					t.Fatalf("Genesis hash mismatch")
				}
				if chainParams.Chain != chain {
					t.Fatalf("Wrong chain. %s != %s", chainParams.Chain, chain)
				}
			})
		}
	}
}
