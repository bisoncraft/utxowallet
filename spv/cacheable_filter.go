package spv

import (
	"github.com/bisoncraft/utxowallet/spv/filterdb"
	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// FilterCacheKey represents the key used to access filters in the FilterCache.
type FilterCacheKey struct {
	BlockHash  chainhash.Hash
	FilterType filterdb.FilterType
}

// CacheableFilter is a wrapper around Filter type which provides a Size method
// used by the cache to target certain memory usage.
type CacheableFilter struct {
	*gcs.Filter
}

// Size returns size of this filter in bytes.
func (c *CacheableFilter) Size() (uint64, error) {
	f, err := c.Filter.NBytes()
	if err != nil {
		return 0, err
	}
	return uint64(len(f)), nil
}
