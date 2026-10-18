package keeper

import (
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/types"
)

// ShouldBlockRecv reports whether an ICS20 packet of packetDenom arriving on
// destChannel must be answered with an error acknowledgement.
func (k *Keeper) ShouldBlockRecv(ctx sdk.Context, destChannel, packetDenom string) bool {
	return k.GetParams(ctx).ShouldBlockRecv(destChannel, packetDenom)
}

// OnReturnRefunded releases the returned total of a voucher packet the
// authority sent to Noble that came back as an error ack or timed out. The
// transfer module refunds the voucher to the authority, so it can be returned
// again. Packets not sent by the authority, or not carrying the voucher, are ignored.
func (k *Keeper) OnReturnRefunded(ctx sdk.Context, sourceChannel, denomPath, sender, amount, reason string) error {
	params := k.GetParams(ctx)
	if sender != k.authority || !params.IsNobleChannel(sourceChannel) ||
		denomPath != types.VoucherPath(sourceChannel, params.BaseDenom) {
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
