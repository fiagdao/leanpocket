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

var sessionCache, evidenceCache *CacheStorage

func InitCacheTest() {
	logger := log.NewNopLogger()
	testingConfig := sdk.DefaultTestingPocketConfig()
	AddPocketNode(GetRandomPrivateKey(), log.NewNopLogger())
	InitConfig(&HostedBlockchains{
		M: make(map[string]HostedBlockchain),
	}, logger, testingConfig)
	testNode := GetPocketNode()
	sessionCache = testNode.SessionStore
	evidenceCache = testNode.EvidenceStore
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
	e, _ := GetEvidence(h, RelayEvidence, sdk.NewInt(100000), evidenceCache)
	p := RelayProof{
		Entropy: 1,
	}
	p1 := RelayProof{
		Entropy: 2,
	}
	assert.True(t, IsUniqueProof(p, e), "p is unique")
	e.AddProof(p)
	SetEvidence(e, evidenceCache)
	e, err := GetEvidence(h, RelayEvidence, sdk.ZeroInt(), evidenceCache)
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
	SetProof(header, RelayEvidence, proof, sdk.NewInt(100000), evidenceCache)
	assert.True(t, reflect.DeepEqual(GetProof(header, RelayEvidence, 0, evidenceCache), proof))
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
	SetProof(header, RelayEvidence, proof, sdk.NewInt(100000), evidenceCache)
	assert.True(t, reflect.DeepEqual(GetProof(header, RelayEvidence, 0, evidenceCache), proof))
	GetProof(header, RelayEvidence, 0, evidenceCache)
	_ = DeleteEvidence(header, RelayEvidence, evidenceCache)
	assert.Empty(t, GetProof(header, RelayEvidence, 0, evidenceCache))
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
	SetProof(header, RelayEvidence, proof, sdk.NewInt(100000), evidenceCache)
	SetProof(header, RelayEvidence, proof2, sdk.NewInt(100000), evidenceCache)
	SetProof(header2, RelayEvidence, proof2, sdk.NewInt(100000), evidenceCache) // different header so shouldn't be counted
	_, totalRelays := GetTotalProofs(header, RelayEvidence, sdk.NewInt(100000), evidenceCache)
	assert.Equal(t, totalRelays, int64(2))
}

func TestSetGetSession(t *testing.T) {
	session := NewTestSession(t, hex.EncodeToString(Hash([]byte("foo"))))
	session2 := NewTestSession(t, hex.EncodeToString(Hash([]byte("bar"))))
	SetSession(session, sessionCache)
	s, found := GetSession(session.SessionHeader, sessionCache)
	assert.True(t, found)
	assert.Equal(t, s, session)
	_, found = GetSession(session2.SessionHeader, sessionCache)
	assert.False(t, found)
	SetSession(session2, sessionCache)
	s, found = GetSession(session2.SessionHeader, sessionCache)
	assert.True(t, found)
	assert.Equal(t, s, session2)
}



func TestDeleteSession(t *testing.T) {
	session := NewTestSession(t, hex.EncodeToString(Hash([]byte("foo"))))
	SetSession(session, sessionCache)
	DeleteSession(session.SessionHeader, sessionCache)
	_, found := GetSession(session.SessionHeader, sessionCache)
	assert.False(t, found)
}


func TestClearCache(t *testing.T) {
	session := NewTestSession(t, hex.EncodeToString(Hash([]byte("foo"))))
	SetSession(session, sessionCache)
	ClearSessionCache(sessionCache)
	iter := SessionIterator(sessionCache)
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
