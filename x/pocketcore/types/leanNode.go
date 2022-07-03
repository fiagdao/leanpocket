package types

import (
	"fmt"
	"github.com/pokt-network/pocket-core/crypto"
	sdk "github.com/pokt-network/pocket-core/types"
	types "github.com/pokt-network/pocket-core/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/privval"
	"sync"
)

var GlobalEvidenceCache *CacheStorage
var GlobalSessionCache *CacheStorage

var GlobalPocketNodes = map[string]*PocketNode{}
var GlobalPocketNodesRWLock = sync.RWMutex{}

// PocketNode represents an entity in the network that is able to handle dispatches, servicing, challenges, and submit proofs/claims.
type PocketNode struct {
	PrivateKey      crypto.PrivateKey
	EvidenceStore   *CacheStorage
	SessionStore    *CacheStorage
	DoCacheInitOnce *sync.Once
}

func AddPocketNode(pk crypto.PrivateKey, logger log.Logger) *PocketNode {
	key := sdk.GetAddress(pk.PublicKey()).String()
	node, exists := GlobalPocketNodes[key]
	if exists {
		return node
	}
	node = &PocketNode{
		PrivateKey:      pk,
		DoCacheInitOnce: &sync.Once{},
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
		if c.PocketConfig.LeanPocket {
			evidenceDbName = evidenceDbName + "_" + sdk.GetAddress(node.PrivateKey.PublicKey()).String()
		}
		node.EvidenceStore.Init(c.PocketConfig.DataDir, evidenceDbName, GlobalTenderMintConfig.LevelDBOptions, GlobalPocketConfig.MaxEvidenceCacheEntires, false)
		node.SessionStore.Init(GlobalPocketConfig.DataDir, "", GlobalTenderMintConfig.LevelDBOptions, GlobalPocketConfig.MaxSessionCacheEntries, true)

		if GlobalSessionCache == nil {
			GlobalSessionCache = node.SessionStore
			GlobalEvidenceCache = node.EvidenceStore
		}
	})
}

func InitPocketNodeCaches(c types.Config, logger log.Logger) {

	// this statement is to allow for backwards compatibility for legacy by grabbing only the first added node
	if !c.PocketConfig.LeanPocket {
		node := GetPocketNode()
		InitPocketNodeCache(node, c, logger)
		return
	}

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
