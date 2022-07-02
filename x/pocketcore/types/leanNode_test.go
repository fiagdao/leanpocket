package types

import (
	sdk "github.com/pokt-network/pocket-core/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	"testing"
)

func TestLeanNodeAdd(t *testing.T) {
	key := GetRandomPrivateKey()
	address := sdk.GetAddress(key.PublicKey())
	InitPocketNodeCache(key, log.NewNopLogger())
	_, ok := GlobalPocketNodes[address.String()]
	assert.True(t, ok)
}

func TestLeanNodeAddByAddress(t *testing.T) {
	key := GetRandomPrivateKey()
	address := sdk.GetAddress(key.PublicKey())
	InitPocketNodeCache(key, log.NewNopLogger())
	node, err := GetPocketNodeByAddress(&address)
	assert.Nil(t, err)
	assert.NotNil(t, node)
}

func TestLeanNodeGet(t *testing.T) {
	key := GetRandomPrivateKey()
	InitPocketNodeCache(key, log.NewNopLogger())
	node, err := GetPocketNode()
	assert.Nil(t, err)
	assert.NotNil(t, node)
}
