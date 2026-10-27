package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
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
	if err := m.SetParams(goCtx, msg.Params); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeParamsUpdated,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
	))
	return &types.MsgUpdateParamsResponse{}, nil
}

// ReturnToNoble implements types.MsgServer. It sends an ICS20 packet returning
// the voucher to Noble, where the matching USDC is released from the channel
// escrow to the receiver. Nothing is minted, burned or moved on this chain, so
// neither user balances nor the authority account are touched.
func (m msgServer) ReturnToNoble(goCtx context.Context, msg *types.MsgReturnToNoble) (*types.MsgReturnToNobleResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", m.authority, msg.Authority)
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if m.ics4Wrapper == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidState, "ics4 wrapper not set")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params, err := m.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if !params.IsNobleChannel(msg.ChannelId) {
		return nil, errorsmod.Wrapf(types.ErrUnknownChannel, "%s not in %v", msg.ChannelId, params.NobleChannels)
	}

	info, err := m.ReturnCap(ctx, msg.ChannelId)
	if err != nil {
		return nil, err
	}

	// Zero returns the remaining cap. An explicit amount is not limited by the
	// cap, so gov can release the whole Noble escrow when it holds more than
	// the voucher supply (e.g. USDC sent straight to the escrow address).
	// Noble enforces the escrow balance: a larger amount gets an error ack and
	// the returned total is released again.
	amount := msg.Amount
	if amount.IsZero() {
		amount = info.Cap
	}
	if !amount.IsPositive() {
		return nil, types.ErrNothingToReturn
	}

	timeoutSeconds := msg.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = types.DefaultTimeoutSeconds
	}
	timeoutTimestamp := uint64(ctx.BlockTime().UnixNano()) + timeoutSeconds*1_000_000_000
	sequence, err := m.sendReturnPacket(ctx, params.BaseDenom, msg.ChannelId, amount, msg.Receiver, msg.Memo, timeoutTimestamp)
	if err != nil {
		return nil, err
	}

	if err := m.addReturnedTotal(ctx, msg.ChannelId, amount); err != nil {
		return nil, err
	}
	m.Logger(ctx).Info("return to Noble sent", "channel", msg.ChannelId, "sequence", sequence, "amount", amount.String(), "receiver", msg.Receiver)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReturnToNoble,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyChannelID, msg.ChannelId),
		sdk.NewAttribute(types.AttributeKeyDenom, info.Denom),
		sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
		sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
	))

	return &types.MsgReturnToNobleResponse{
		Sequence: sequence,
		Denom:    info.Denom,
		Amount:   amount,
	}, nil
}
