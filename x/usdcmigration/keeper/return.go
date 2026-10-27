package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

// sendReturnPacket builds the ICS20 packet that returns amount of the Noble
// voucher over channelID and hands it to the next ICS4 wrapper. Unlike a
// transfer keeper send, no voucher is burned from, or escrowed for, any
// account: Noble releases the matching USDC from the channel escrow to
// receiver. The packet is sent below this keeper's own SendPacket, so the send
// block (which also covers the authority) does not apply to it.
func (k *Keeper) sendReturnPacket(ctx sdk.Context, baseDenom, channelID string, amount math.Int, receiver, memo string, timeoutTimestamp uint64) (uint64, error) {
	data := transfertypes.NewFungibleTokenPacketData(
		types.VoucherPath(channelID, baseDenom), amount.String(), k.authority, receiver, memo,
	)
	if err := data.ValidateBasic(); err != nil {
		return 0, err
	}
	return k.ics4Wrapper.SendPacket(ctx, types.TransferPortID, channelID, clienttypes.ZeroHeight(), timeoutTimestamp, data.GetBytes())
}

// IsReturnPacket reports whether an ICS20 packet sent over sourceChannel is a
// return to Noble built by MsgReturnToNoble. Since the send block rejects the
// voucher for every sender, the authority included, only the module can have
// sent such a packet. The match uses the current params, so they must not
// change while a return is in flight (see docs/integration.md).
func (k *Keeper) IsReturnPacket(ctx sdk.Context, sourceChannel, denomPath, sender string) bool {
	params := k.GetParams(ctx)
	return sender == k.authority && params.IsNobleChannel(sourceChannel) &&
		denomPath == types.VoucherPath(sourceChannel, params.BaseDenom)
}
