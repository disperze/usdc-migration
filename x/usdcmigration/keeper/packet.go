package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

// OnReturnRefunded records a return packet that came back as an error ack or
// timed out. The caller must have checked the packet with IsReturnPacket. The
// return burned nothing, so nothing is minted back and no state changes: the
// voucher stays backed by Noble's escrow and can be returned again.
func (k *Keeper) OnReturnRefunded(ctx sdk.Context, sourceChannel string, sequence uint64, amount, reason string) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeReturnRefunded,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyChannelID, sourceChannel),
		sdk.NewAttribute(types.AttributeKeySequence, strconv.FormatUint(sequence, 10)),
		sdk.NewAttribute(types.AttributeKeyAmount, amount),
		sdk.NewAttribute(types.AttributeKeyReason, reason),
	))
}
