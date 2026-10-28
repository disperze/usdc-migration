package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
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
	if err := m.checkAuthority(msg.Authority); err != nil {
		return nil, err
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
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
	if err := m.checkAuthority(msg.Authority); err != nil {
		return nil, err
	}
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if m.ics4Wrapper == nil || m.scopedTransferKeeper == nil {
		return nil, errorsmod.Wrap(types.ErrInvalidState, "ics4 wrapper or scoped transfer keeper not set")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := m.GetParams(ctx)
	if !params.IsNobleChannel(msg.ChannelId) {
		return nil, errorsmod.Wrapf(types.ErrUnknownChannel, "%s not in %v", msg.ChannelId, params.NobleChannels)
	}

	// Noble enforces its escrow balance for the channel: a larger amount gets
	// an error ack, so nothing is tracked here.
	sequence, err := m.sendReturnPacket(ctx, msg, params.BaseDenom)
	if err != nil {
		return nil, err
	}
	denom := types.VoucherDenom(msg.ChannelId, params.BaseDenom)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReturnToNoble,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyChannelID, msg.ChannelId),
		sdk.NewAttribute(types.AttributeKeyDenom, denom),
		sdk.NewAttribute(types.AttributeKeyAmount, msg.Amount.String()),
		sdk.NewAttribute(types.AttributeKeyReceiver, msg.Receiver),
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
	))

	return &types.MsgReturnToNobleResponse{
		Sequence: sequence,
		Denom:    denom,
		Amount:   msg.Amount,
	}, nil
}
