package keeper

import (
	"encoding/hex"
	"fmt"
	sdk "github.com/pokt-network/pocket-core/types"
	"github.com/pokt-network/pocket-core/x/nodes/exported"
	"github.com/pokt-network/pocket-core/x/pocketcore/types"
)

// "HandleDispatch" - Handles a client request for their session information
func (k Keeper) HandleDispatch(ctx sdk.Ctx, header types.SessionHeader) (*types.DispatchResponse, sdk.Error) {
	// retrieve the latest session block height
	var sessionBlockHeight int64
	if types.GlobalPocketConfig.ClientSessionSyncAllowance > 0 && header.SessionBlockHeight != 0 {
		sessionBlockHeight = header.SessionBlockHeight
	} else {
		sessionBlockHeight = k.GetLatestSessionBlockHeight(ctx)
		header.SessionBlockHeight = sessionBlockHeight
	}

	// validate the header
	err := header.ValidateHeader()
	if err != nil {
		return nil, err
	}
	// get the session context
	sessionCtx, er := ctx.PrevCtx(sessionBlockHeight)
	if er != nil {
		return nil, sdk.ErrInternal(er.Error())
	}
	// check cache
	session, found := types.GetSession(header, types.GlobalSessionCache)
	// if not found generate the session
	if !found {
		var err sdk.Error
		blockHashBz, er := sessionCtx.BlockHash(k.Cdc, sessionCtx.BlockHeight())
		if er != nil {
			return nil, sdk.ErrInternal(er.Error())
		}
		session, err = types.NewSession(sessionCtx, ctx, k.posKeeper, header, hex.EncodeToString(blockHashBz), int(k.SessionNodeCount(sessionCtx)))
		if err != nil {
			return nil, err
		}
		// add to cache
		types.SetSession(session, types.GlobalSessionCache)
	}
	actualNodes := make([]exported.ValidatorI, len(session.SessionNodes))
	for i, addr := range session.SessionNodes {
		actualNodes[i], _ = k.GetNode(sessionCtx, addr)
	}
	return &types.DispatchResponse{Session: types.DispatchSession{
		SessionHeader: session.SessionHeader,
		SessionKey:    session.SessionKey,
		SessionNodes:  actualNodes,
	}, BlockHeight: ctx.BlockHeight()}, nil
}

// "IsSessionBlock" - Returns true if current block, is a session block (beginning of a session)
func (k Keeper) IsSessionBlock(ctx sdk.Ctx) bool {
	return ctx.BlockHeight()%k.posKeeper.BlocksPerSession(ctx) == 1
}

// IsLatestSessionBlockHeightWithinTolerance checks if the latest session block height is within the configurable session sync allowance.
func (k Keeper) IsLatestSessionHeightWithinTolerance(ctx sdk.Ctx, relaySessionBlockHeight int64) bool {
	// Session block height can never be zero.
	if relaySessionBlockHeight <= 0 {
		return false
	}
	latestSessionHeight := k.GetLatestSessionBlockHeight(ctx)
	tolerance := types.GlobalPocketConfig.ClientSessionSyncAllowance * k.posKeeper.BlocksPerSession(ctx)
	minHeight := latestSessionHeight - tolerance
	fmt.Println(minHeight, tolerance, relaySessionBlockHeight, latestSessionHeight)
	return sdk.IsBetween(relaySessionBlockHeight, minHeight, latestSessionHeight)
}

// "GetLatestSessionBlockHeight" - Returns the latest session block height (first block of the session, (see blocksPerSession))
func (k Keeper) GetLatestSessionBlockHeight(ctx sdk.Ctx) (sessionBlockHeight int64) {
	// get the latest block height
	blockHeight := ctx.BlockHeight()
	// get the blocks per session
	blocksPerSession := k.posKeeper.BlocksPerSession(ctx)
	// if block height / blocks per session remainder is zero, just subtract blocks per session and add 1
	if blockHeight%blocksPerSession == 0 {
		sessionBlockHeight = blockHeight - k.posKeeper.BlocksPerSession(ctx) + 1
	} else {
		// calculate the latest session block height by diving the current block height by the blocksPerSession
		sessionBlockHeight = (blockHeight/blocksPerSession)*blocksPerSession + 1
	}
	return
}

// "IsPocketSupportedBlockchain" - Returns true if network identifier param is supported by pocket
func (k Keeper) IsPocketSupportedBlockchain(ctx sdk.Ctx, chain string) bool {
	// loop through supported blockchains (network identifiers)
	for _, c := range k.SupportedBlockchains(ctx) {
		// if contains chain return true
		if c == chain {
			return true
		}
	}
	// else return false
	return false
}

func (Keeper) ClearSessionCache() {
	types.ClearSessionCache(types.GlobalSessionCache)
}
