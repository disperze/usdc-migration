package keeper

import (
	"errors"
	"strconv"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
)

// ShouldBlockRecv reports whether an ICS20 packet of packetDenom arriving on
// destChannel must be answered with an error acknowledgement.
func (k *Keeper) ShouldBlockRecv(ctx sdk.Context, destChannel, packetDenom string) (bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return false, err
	}
	return params.ShouldBlockRecv(destChannel, packetDenom), nil
}

// TakePendingReturn removes the pending return sent over channelID with
// sequence and returns its amount. found is false if the packet is not a
// return sent by MsgReturnToNoble.
func (k *Keeper) TakePendingReturn(ctx sdk.Context, channelID string, sequence uint64) (amount math.Int, found bool, err error) {
	key := collections.Join(channelID, sequence)
	amount, err = k.PendingReturns.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return math.Int{}, false, nil
	}
	if err != nil {
		return math.Int{}, false, err
	}
	if err := k.PendingReturns.Remove(ctx, key); err != nil {
		return math.Int{}, false, err
	}
	return amount, true, nil
}

// RefundReturn releases amount from the returned total of channelID after a
// return packet came back as an error ack or timed out. The return burned
// nothing, so nothing is minted back: the voucher stays backed by Noble's
// escrow and can be returned again.
func (k *Keeper) RefundReturn(ctx sdk.Context, channelID string, sequence uint64, amount math.Int, reason string) error {
	if err := k.subReturnedTotal(ctx, channelID, amount); err != nil {
		return err
	}
	k.Logger(ctx).Info("return to Noble refunded", "channel", channelID, "sequence", sequence, "amount", amount.String(), "reason", reason)
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReturnRefunded,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
		sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))
	return nil
}
