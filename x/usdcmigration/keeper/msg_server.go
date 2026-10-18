package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"

	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/types"
)

type msgServer struct {
	*Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns the module's MsgServer implementation.
func NewMsgServerImpl(k *Keeper) types.MsgServer {
	return msgServer{Keeper: k}
}

// UpdateParams implements types.MsgServer.
func (m msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", m.authority, msg.Authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, errorsmod.Wrap(types.ErrInvalidParams, err.Error())
	}
	if err := m.Params.Set(goCtx, msg.Params); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeParamsUpdated,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgUpdateParamsResponse{}, nil
}

// ReturnToNoble implements types.MsgServer. It mints the voucher the authority
// is missing to cover the requested amount and IBC-transfers it back to Noble,
// where the matching USDC is released from the channel escrow to the receiver.
// User balances are not touched.
func (m msgServer) ReturnToNoble(goCtx context.Context, msg *types.MsgReturnToNoble) (*types.MsgReturnToNobleResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", m.authority, msg.Authority)
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if m.transferKeeper == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidParams, "transfer keeper not set")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)
	if !params.IsNobleChannel(msg.ChannelId) {
		return nil, errorsmod.Wrapf(types.ErrUnknownChannel, "%s not in %v", msg.ChannelId, params.NobleChannels)
	}

	authority := sdk.MustAccAddressFromBech32(m.authority)
	// ibc-go refuses to send from, or refund to, a blocked address. A refund
	// failing would leave the packet stuck, so fail early with a clear error.
	if m.bankKeeper.BlockedAddr(authority) {
		return nil, errorsmod.Wrapf(types.ErrAuthorityBlocked,
			"%s must be removed from the bank keeper blocked addresses to send and receive refunds", m.authority)
	}

	denom, _, authorityBalance, _, returnCap, err := m.ReturnCap(ctx, msg.ChannelId)
	if err != nil {
		return nil, err
	}

	amount := msg.Amount
	if amount.IsZero() {
		amount = returnCap
	}
	if !amount.IsPositive() {
		return nil, types.ErrNothingToReturn
	}
	if amount.GT(returnCap) {
		return nil, errorsmod.Wrapf(types.ErrExceedsReturnCap, "requested %s, cap %s", amount, returnCap)
	}

	// Reuse voucher the authority already holds (e.g. from a refunded attempt).
	minted := math.ZeroInt()
	if authorityBalance.LT(amount) {
		minted = amount.Sub(authorityBalance)
		coins := sdk.NewCoins(sdk.NewCoin(denom, minted))
		if err := m.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
			return nil, err
		}
		if err := m.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, authority, coins); err != nil {
			return nil, err
		}
	}

	timeoutSeconds := msg.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = types.DefaultTimeoutSeconds
	}
	transferMsg := &transfertypes.MsgTransfer{
		SourcePort:       types.TransferPortID,
		SourceChannel:    msg.ChannelId,
		Token:            sdk.NewCoin(denom, amount),
		Sender:           m.authority,
		Receiver:         msg.Receiver,
		TimeoutHeight:    clienttypes.ZeroHeight(),
		TimeoutTimestamp: uint64(ctx.BlockTime().UnixNano()) + timeoutSeconds*1_000_000_000,
		Memo:             msg.Memo,
	}
	if err := transferMsg.ValidateBasic(); err != nil {
		return nil, err
	}
	res, err := m.transferKeeper.Transfer(ctx, transferMsg)
	if err != nil {
		return nil, err
	}

	if err := m.addReturnedTotal(ctx, msg.ChannelId, amount); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReturnToNoble,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyChannelID, msg.ChannelId),
		sdk.NewAttribute(types.AttributeKeyDenom, denom),
		sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
		sdk.NewAttribute(types.AttributeKeyMinted, minted.String()),
		sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(res.Sequence, 10)),
	))

	return &types.MsgReturnToNobleResponse{
		Sequence: res.Sequence,
		Denom:    denom,
		Amount:   amount,
		Minted:   minted,
	}, nil
}
