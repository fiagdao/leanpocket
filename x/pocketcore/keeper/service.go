package keeper

import (
	"encoding/hex"
	"fmt"
	"github.com/pokt-network/pocket-core/crypto"
	"time"

	sdk "github.com/pokt-network/pocket-core/types"
	pc "github.com/pokt-network/pocket-core/x/pocketcore/types"
)

// "HandleRelay" - Handles an api (read/write) request to a non-native (external) blockchain
func (k Keeper) HandleRelay(ctx sdk.Ctx, relay pc.Relay) (*pc.RelayResponse, sdk.Error) {
	relayTimeStart := time.Now()
	// get the latest session block height because this relay will correspond with the latest session
	sessionBlockHeight := k.GetLatestSessionBlockHeight(ctx)

	var pk crypto.PrivateKey
	var err sdk.Error
	var evidenceStore *pc.CacheStorage

	if pc.GlobalPocketConfig.LeanPocket {
		// if lean pocket enabled, grab the targeted servicer through the relay proof and set the proper evidence/session caches
		servicerRelayPublicKey, err1 := crypto.NewPublicKey(relay.Proof.ServicerPubKey)
		if err1 != nil {
			return nil, sdk.ErrInternal("Could not convert servicer hex to public key")
		}
		selfAddr := sdk.GetAddress(servicerRelayPublicKey)
		node, err1 := pc.GetPocketNodeByAddress(&selfAddr)
		if err1 != nil {
			return nil, sdk.ErrInternal("Failed to find correct servicer PK")
		}
		pk = node.PrivateKey
		evidenceStore = node.EvidenceStore
	} else {
		// get self node (your validator) from the current state
		node := pc.GetPocketNode()
		evidenceStore = node.EvidenceStore
	}

	selfAddr := sdk.Address(pk.PublicKey().Address())
	// retrieve the nonNative blockchains your node is hosting
	hostedBlockchains := k.GetHostedBlockchains()
	// ensure the validity of the relay
	maxPossibleRelays, err := relay.Validate(ctx, k.posKeeper, k.appKeeper, k, selfAddr, hostedBlockchains, sessionBlockHeight, evidenceStore)
	if err != nil {
		if pc.GlobalPocketConfig.RelayErrors {
			ctx.Logger().Error(
				fmt.Sprintf("could not validate relay for app: %s for chainID: %v with error: %s",
					relay.Proof.ServicerPubKey,
					relay.Proof.Blockchain,
					err.Error(),
				),
			)
			ctx.Logger().Debug(
				fmt.Sprintf(
					"could not validate relay for app: %s, for chainID %v on node %s, at session height: %v, with error: %s",
					relay.Proof.ServicerPubKey,
					relay.Proof.Blockchain,
					selfAddr.String(),
					sessionBlockHeight,
					err.Error(),
				),
			)
		}
		return nil, err
	}
	// store the proof before execution, because the proof corresponds to the previous relay
	relay.Proof.Store(maxPossibleRelays, evidenceStore)
	// attempt to execute
	respPayload, err := relay.Execute(hostedBlockchains, &selfAddr)
	if err != nil {
		ctx.Logger().Error(fmt.Sprintf("could not send relay with error: %s", err.Error()))
		return nil, err
	}
	// generate response object
	resp := &pc.RelayResponse{
		Response: respPayload,
		Proof:    relay.Proof,
	}
	// sign the response
	sig, er := pk.Sign(resp.Hash())
	if er != nil {
		ctx.Logger().Error(
			fmt.Sprintf("could not sign response for address: %s with hash: %v, with error: %s",
				selfAddr.String(), resp.HashString(), er.Error()),
		)
		return nil, pc.NewKeybaseError(pc.ModuleName, er)
	}
	// attach the signature in hex to the response
	resp.Signature = hex.EncodeToString(sig)
	// track the relay time
	relayTime := time.Since(relayTimeStart)
	// add to metrics

	pc.GlobalServiceMetric().AddRelayTimingFor(relay.Proof.Blockchain, float64(relayTime.Milliseconds()), &selfAddr)
	pc.GlobalServiceMetric().AddRelayFor(relay.Proof.Blockchain, &selfAddr)

	return resp, nil
}

// "HandleChallenge" - Handles a client relay response challenge request
func (k Keeper) HandleChallenge(ctx sdk.Ctx, challenge pc.ChallengeProofInvalidData) (*pc.ChallengeResponse, sdk.Error) {

	var pk crypto.PrivateKey
	var err sdk.Error
	var evidenceStore *pc.CacheStorage
	var sessionStore *pc.CacheStorage

	// if lean pocket is enabled, grab a "random" node to handle the challenge request.
	node  := pc.GetPocketNode()

	pk = node.PrivateKey
	evidenceStore = node.EvidenceStore
	sessionStore = node.SessionStore

	// get self node (your validator) from the current state
	selfNode := sdk.GetAddress(pk.PublicKey())
	sessionBlkHeight := k.GetLatestSessionBlockHeight(ctx)
	// get the session context
	sessionCtx, er := ctx.PrevCtx(sessionBlkHeight)
	if er != nil {
		return nil, sdk.ErrInternal(er.Error())
	}
	// get the application that staked on behalf of the client
	app, found := k.GetAppFromPublicKey(sessionCtx, challenge.MinorityResponse.Proof.Token.ApplicationPublicKey)
	if !found {
		return nil, pc.NewAppNotFoundError(pc.ModuleName)
	}
	// generate header
	header := pc.SessionHeader{
		ApplicationPubKey:  challenge.MinorityResponse.Proof.Token.ApplicationPublicKey,
		Chain:              challenge.MinorityResponse.Proof.Blockchain,
		SessionBlockHeight: sessionCtx.BlockHeight(),
	}
	// check cache
	session, found := pc.GetSession(header, sessionStore)
	// if not found generate the session
	if !found {
		var err sdk.Error
		blockHashBz, er := sessionCtx.BlockHash(k.Cdc, sessionCtx.BlockHeight())
		if er != nil {
			return nil, sdk.ErrInternal(er.Error())
		}
		session, err = pc.NewSession(sessionCtx, ctx, k.posKeeper, header, hex.EncodeToString(blockHashBz), int(k.SessionNodeCount(sessionCtx)))
		if err != nil {
			return nil, err
		}
		// add to cache
		pc.SetSession(session, sessionStore)
	}
	// validate the challenge
	err = challenge.ValidateLocal(header, app.GetMaxRelays(), app.GetChains(), int(k.SessionNodeCount(sessionCtx)), session.SessionNodes, selfNode, evidenceStore)
	if err != nil {
		return nil, err
	}
	// store the challenge in memory
	challenge.Store(app.GetMaxRelays(), evidenceStore)
	// update metric
	pc.GlobalServiceMetric().AddChallengeFor(header.Chain, &selfNode)
	return &pc.ChallengeResponse{Response: fmt.Sprintf("successfully stored challenge proof for %s", challenge.MinorityResponse.Proof.ServicerPubKey)}, nil
}
