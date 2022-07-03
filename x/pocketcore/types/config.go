package types

import (
	"encoding/hex"
	"fmt"
	"github.com/pokt-network/pocket-core/crypto"
	"github.com/pokt-network/pocket-core/types"
	"github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/libs/log"
	"time"
)

const (
	DefaultRPCTimeout = 3000
	MaxRPCTimeout     = 1000000
	MinRPCTimeout     = 1
)

var (
	globalRPCTimeout       time.Duration
	GlobalPocketConfig     types.PocketConfig
	GlobalTenderMintConfig config.Config
)

func InitConfig(chains *HostedBlockchains, logger log.Logger, c types.Config) {
	configOnce.Do(func() {
		InitPocketNodeCaches(c, logger)
		InitGlobalServiceMetric(chains, logger, c.PocketConfig.PrometheusAddr, c.PocketConfig.PrometheusMaxOpenfiles)
	})
	GlobalPocketConfig = c.PocketConfig
	GlobalTenderMintConfig = c.TendermintConfig
	if GlobalPocketConfig.LeanPocket {
		GlobalTenderMintConfig.PrivValidatorState = types.DefaultPVSNameLean
		GlobalTenderMintConfig.PrivValidatorKey = types.DefaultPVKNameLean
		GlobalTenderMintConfig.NodeKey = types.DefaultPVSNameLean
	}
	SetRPCTimeout(c.PocketConfig.RPCTimeout)
}

func ConvertEvidenceToProto(config types.Config) error {
	// we have to add a random pocket node so that way lean pokt can still support getting the legacy evidence cache
	node := AddPocketNode(crypto.GenerateEd25519PrivKey().GenPrivateKey(), log.NewNopLogger())

	InitConfig(nil, log.NewNopLogger(), config)


	gec := node.EvidenceStore
	it, err := gec.Iterator()
	if err != nil {
		return fmt.Errorf("error creating evidence iterator: %s", err.Error())
	}
	defer it.Close()
	for ; it.Valid(); it.Next() {
		ev, err := Evidence{}.LegacyAminoUnmarshal(it.Value())
		if err != nil {
			return fmt.Errorf("error amino unmarshalling evidence: %s", err.Error())
		}
		k, err := ev.Key()
		if err != nil {
			return fmt.Errorf("error creating key from evidence object: %s", err.Error())
		}
		gec.SetWithoutLockAndSealCheck(hex.EncodeToString(k), ev)
	}
	err = gec.FlushToDBWithoutLock()
	if err != nil {
		return fmt.Errorf("error flushing evidence objects to the database: %s", err.Error())
	}
	return nil
}

func FlushSessionCache() {
	if GlobalPocketNodes == nil {
		return
	}
	for _, k := range GlobalPocketNodes {
		err := k.SessionStore.FlushToDB()
		if err != nil {
			fmt.Printf("unable to flush sessions to the database before shutdown!! %s\n", err.Error())
		}
		err = k.EvidenceStore.FlushToDB()
		if err != nil {
			fmt.Printf("unable to flush GOBEvidence to the database before shutdown!! %s\n", err.Error())
		}
	}
}

func GetRPCTimeout() time.Duration {
	return globalRPCTimeout
}

func SetRPCTimeout(timeout int64) {
	if timeout < MinRPCTimeout || timeout > MaxRPCTimeout {
		timeout = DefaultRPCTimeout
	}

	globalRPCTimeout = time.Duration(timeout)
}
