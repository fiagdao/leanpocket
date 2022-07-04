package types

import (
	"fmt"
	"github.com/pokt-network/pocket-core/crypto"
	"github.com/pokt-network/pocket-core/types"
	sdk "github.com/pokt-network/pocket-core/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/privval"
	"sync"
)

// GlobalEvidenceCache & GlobalSessionCache is used for the first pocket node and acts as backwards-compatibility for pre-lean pocket
var GlobalEvidenceCache *CacheStorage
var GlobalSessionCache *CacheStorage

var GlobalPocketNodes = map[string]*PocketNode{}

// PocketNode represents an entity in the network that is able to handle dispatches, servicing, challenges, and submit proofs/claims.
type PocketNode struct {
	PrivateKey      crypto.PrivateKey
	EvidenceStore   *CacheStorage
	SessionStore    *CacheStorage
	DoCacheInitOnce sync.Once
}

func AddPocketNode(pk crypto.PrivateKey, logger log.Logger) *PocketNode {
	key := sdk.GetAddress(pk.PublicKey()).String()
	node, exists := GlobalPocketNodes[key]
	if exists {
		return node
	}
	node = &PocketNode{
		PrivateKey: pk,
	}
	GlobalPocketNodes[key] = node
	return node
}

func AddPocketNodeByFilePVKey(fpvKey privval.FilePVKey, logger log.Logger) {
	key, err := crypto.PrivKeyToPrivateKey(fpvKey.PrivKey)
	if err != nil {
		return
	}
	AddPocketNode(key, logger)
}

// InitPocketNodeCache adds a PocketNode with its SessionStore and EvidenceStore initialized
func InitPocketNodeCache(node *PocketNode, c types.Config, logger log.Logger) {
	node.DoCacheInitOnce.Do(func() {
		evidenceDbName := GlobalPocketConfig.EvidenceDBName

		// In LeanPocket, we create a evidence store on disk with suffix of the node's address
		if c.PocketConfig.LeanPocket {
			evidenceDbName = evidenceDbName + "_" + sdk.GetAddress(node.PrivateKey.PublicKey()).String()
		}

		node.EvidenceStore = new(CacheStorage)
		node.SessionStore = new(CacheStorage)
		node.EvidenceStore.Init(c.PocketConfig.DataDir, evidenceDbName, c.TendermintConfig.LevelDBOptions, c.PocketConfig.MaxEvidenceCacheEntires, false)
		node.SessionStore.Init(c.PocketConfig.DataDir, "", c.TendermintConfig.LevelDBOptions, c.PocketConfig.MaxSessionCacheEntries, true)

		// Set the GOBSession and GOBEvidence Caches for legacy compatibility for pre-leanpocket
		if GlobalSessionCache == nil {
			GlobalSessionCache = node.SessionStore
			GlobalEvidenceCache = node.EvidenceStore
		}
	})
}

func InitPocketNodeCaches(c types.Config, logger log.Logger) {
	for _, node := range GlobalPocketNodes {
		InitPocketNodeCache(node, c, logger)
	}
}

// GetPocketNodeByAddress returns a PocketNode from global map GlobalPocketNodes
func GetPocketNodeByAddress(address *sdk.Address) (*PocketNode, error) {
	node, ok := GlobalPocketNodes[address.String()]
	if !ok {
		return nil, fmt.Errorf("failed to find private key for %s", address.String())
	}
	return node, nil
}

// GetPocketNode returns a PocketNode from global map GlobalPocketNodes, it does not guarantee order
func GetPocketNode() *PocketNode {
	for _, r := range GlobalPocketNodes {
		if r != nil {
			return r
		}
	}
	return nil
}
