package keeper

import (
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

// ShouldBlockRecv reports whether an ICS20 packet of packetDenom arriving on
// destChannel must be answered with an error acknowledgement.
func (k *Keeper) ShouldBlockRecv(ctx sdk.Context, destChannel, packetDenom string) bool {
	return k.GetParams(ctx).ShouldBlockRecv(destChannel, packetDenom)
}

// OnReturnRefunded releases the returned total of a return packet that came
// back as an error ack or timed out. The return burned nothing, so nothing is
// minted back: the voucher stays backed by Noble's escrow and can be returned
// again. Packets that are not return packets are ignored.
func (k *Keeper) OnReturnRefunded(ctx sdk.Context, sourceChannel, denomPath, sender, amount, reason string) error {
	if !k.IsReturnPacket(ctx, sourceChannel, denomPath, sender) {
		return nil
	}
	amt, ok := math.NewIntFromString(amount)
	if !ok {
		return nil
	}
	if err := k.subReturnedTotal(ctx, sourceChannel, amt); err != nil {
		return err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReturnRefunded,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyChannelID, sourceChannel),
		sdk.NewAttribute(types.AttributeKeyAmount, amt.String()),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))
	return nil
}
