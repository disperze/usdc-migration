package keeper

import (
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

// OnReturnRefunded releases the returned total of a return packet that came
// back as an error ack or timed out. The caller must have checked the packet
// with IsReturnPacket. The return burned nothing, so nothing is minted back:
// the voucher stays backed by Noble's escrow and can be returned again.
func (k *Keeper) OnReturnRefunded(ctx sdk.Context, sourceChannel string, sequence uint64, amount, reason string) error {
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
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
		sdk.NewAttribute(types.AttributeKeyAmount, amt.String()),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))
	return nil
}
