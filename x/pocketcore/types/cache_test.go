package types

import (
	"encoding/hex"
	sdk "github.com/pokt-network/pocket-core/types"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/libs/log"
	"os"
	"reflect"
	"testing"
)

func InitCacheTest() {
	logger := log.NewNopLogger()
	testingConfig := sdk.DefaultTestingPocketConfig()
	AddPocketNode(GetRandomPrivateKey(), log.NewNopLogger())
	InitConfig(&HostedBlockchains{
		M: make(map[string]HostedBlockchain),
	}, logger, testingConfig)
}

func TestMain(m *testing.M) {
	InitCacheTest()
	m.Run()
	err := os.RemoveAll("data")
	if err != nil {
		panic(err)
	}
	os.Exit(0)
}

func TestIsUniqueProof(t *testing.T) {
	h := SessionHeader{
		ApplicationPubKey:  "0",
		Chain:              "0001",
		SessionBlockHeight: 0,
	}
	e, _ := GetEvidence(h, RelayEvidence, sdk.NewInt(100000), GlobalEvidenceCacheLegacy)
	p := RelayProof{
		Entropy: 1,
	}
	p1 := RelayProof{
		Entropy: 2,
	}
	assert.True(t, IsUniqueProof(p, e), "p is unique")
	e.AddProof(p)
	SetEvidence(e, GlobalEvidenceCacheLegacy)
	e, err := GetEvidence(h, RelayEvidence, sdk.ZeroInt(), GlobalEvidenceCacheLegacy)
	assert.Nil(t, err)
	assert.False(t, IsUniqueProof(p, e), "p is no longer unique")
	assert.True(t, IsUniqueProof(p1, e), "p is unique")
}

func TestAllEvidence_AddGetEvidence(t *testing.T) {
	appPubKey := getRandomPubKey().RawString()
	servicerPubKey := getRandomPubKey().RawString()
	clientPubKey := getRandomPubKey().RawString()
	ethereum := hex.EncodeToString([]byte{0001})
	header := SessionHeader{
		ApplicationPubKey:  appPubKey,
		Chain:              ethereum,
		SessionBlockHeight: 1,
	}
	proof := RelayProof{
		Entropy:            0,
		RequestHash:        header.HashString(), // fake
		SessionBlockHeight: 1,
		ServicerPubKey:     servicerPubKey,
		Blockchain:         ethereum,
		Token: AAT{
			Version:              "0.0.1",
			ApplicationPublicKey: appPubKey,
			ClientPublicKey:      clientPubKey,
			ApplicationSignature: "",
		},
		Signature: "",
	}
	SetProof(header, RelayEvidence, proof, sdk.NewInt(100000), GlobalEvidenceCacheLegacy)
	assert.True(t, reflect.DeepEqual(GetProof(header, RelayEvidence, 0, GlobalEvidenceCacheLegacy), proof))
}

func TestAllEvidence_DeleteEvidence(t *testing.T) {
	appPubKey := getRandomPubKey().RawString()
	servicerPubKey := getRandomPubKey().RawString()
	clientPubKey := getRandomPubKey().RawString()
	ethereum := hex.EncodeToString([]byte{0001})
	header := SessionHeader{
		ApplicationPubKey:  appPubKey,
		Chain:              ethereum,
		SessionBlockHeight: 1,
	}
	proof := RelayProof{
		Entropy:            0,
		SessionBlockHeight: 1,
		ServicerPubKey:     servicerPubKey,
		RequestHash:        header.HashString(), // fake
		Blockchain:         ethereum,
		Token: AAT{
			Version:              "0.0.1",
			ApplicationPublicKey: appPubKey,
			ClientPublicKey:      clientPubKey,
			ApplicationSignature: "",
		},
		Signature: "",
	}
	SetProof(header, RelayEvidence, proof, sdk.NewInt(100000), GlobalEvidenceCacheLegacy)
	assert.True(t, reflect.DeepEqual(GetProof(header, RelayEvidence, 0, GlobalEvidenceCacheLegacy), proof))
	GetProof(header, RelayEvidence, 0, GlobalEvidenceCacheLegacy)
	_ = DeleteEvidence(header, RelayEvidence, GlobalEvidenceCacheLegacy)
	assert.Empty(t, GetProof(header, RelayEvidence, 0, GlobalEvidenceCacheLegacy))
}

func TestAllEvidence_GetTotalProofs(t *testing.T) {
	appPubKey := getRandomPubKey().RawString()
	servicerPubKey := getRandomPubKey().RawString()
	clientPubKey := getRandomPubKey().RawString()
	ethereum := hex.EncodeToString([]byte{0001})
	header := SessionHeader{
		ApplicationPubKey:  appPubKey,
		Chain:              ethereum,
		SessionBlockHeight: 1,
	}
	header2 := SessionHeader{
		ApplicationPubKey:  appPubKey,
		Chain:              ethereum,
		SessionBlockHeight: 101,
	}
	proof := RelayProof{
		Entropy:            0,
		SessionBlockHeight: 1,
		ServicerPubKey:     servicerPubKey,
		RequestHash:        header.HashString(), // fake
		Blockchain:         ethereum,
		Token: AAT{
			Version:              "0.0.1",
			ApplicationPublicKey: appPubKey,
			ClientPublicKey:      clientPubKey,
			ApplicationSignature: "",
		},
		Signature: "",
	}
	proof2 := RelayProof{
		Entropy:            0,
		SessionBlockHeight: 1,
		ServicerPubKey:     servicerPubKey,
		RequestHash:        header.HashString(), // fake
		Blockchain:         ethereum,
		Token: AAT{
			Version:              "0.0.1",
			ApplicationPublicKey: appPubKey,
			ClientPublicKey:      clientPubKey,
			ApplicationSignature: "",
		},
		Signature: "",
	}
	SetProof(header, RelayEvidence, proof, sdk.NewInt(100000), GlobalEvidenceCacheLegacy)
	SetProof(header, RelayEvidence, proof2, sdk.NewInt(100000), GlobalEvidenceCacheLegacy)
	SetProof(header2, RelayEvidence, proof2, sdk.NewInt(100000), GlobalEvidenceCacheLegacy) // different header so shouldn't be counted
	_, totalRelays := GetTotalProofs(header, RelayEvidence, sdk.NewInt(100000), GlobalEvidenceCacheLegacy)
	assert.Equal(t, totalRelays, int64(2))
}

func TestSetGetSession(t *testing.T) {
	session := NewTestSession(t, hex.EncodeToString(Hash([]byte("foo"))))
	session2 := NewTestSession(t, hex.EncodeToString(Hash([]byte("bar"))))
	SetSession(session, GlobalSessionCacheLegacy)
	s, found := GetSession(session.SessionHeader, GlobalSessionCacheLegacy)
	assert.True(t, found)
	assert.Equal(t, s, session)
	_, found = GetSession(session2.SessionHeader, GlobalSessionCacheLegacy)
	assert.False(t, found)
	SetSession(session2, GlobalSessionCacheLegacy)
	s, found = GetSession(session2.SessionHeader, GlobalSessionCacheLegacy)
	assert.True(t, found)
	assert.Equal(t, s, session2)
}

func TestDeleteSession(t *testing.T) {
	session := NewTestSession(t, hex.EncodeToString(Hash([]byte("foo"))))
	SetSession(session, GlobalSessionCacheLegacy)
	DeleteSession(session.SessionHeader, GlobalSessionCacheLegacy)
	_, found := GetSession(session.SessionHeader, GlobalSessionCacheLegacy)
	assert.False(t, found)
}

func TestClearCache(t *testing.T) {
	session := NewTestSession(t, hex.EncodeToString(Hash([]byte("foo"))))
	SetSession(session, GlobalSessionCacheLegacy)
	ClearSessionCache(GlobalSessionCacheLegacy)
	iter := SessionIterator(GlobalSessionCacheLegacy)
	var count = 0
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		count++
	}
	assert.Zero(t, count)
}

func NewTestSession(t *testing.T, chain string) Session {
	appPubKey := getRandomPubKey()
	var vals []sdk.Address
	for i := 0; i < 5; i++ {
		nodePubKey := getRandomPubKey()
		vals = append(vals, sdk.Address(nodePubKey.Address()))
	}
	return Session{
		SessionHeader: SessionHeader{
			ApplicationPubKey:  appPubKey.RawString(),
			Chain:              chain,
			SessionBlockHeight: 1,
		},
		SessionKey:   appPubKey.RawBytes(), // fake
		SessionNodes: vals,
	}
}
