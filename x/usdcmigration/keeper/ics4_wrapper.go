package keeper

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

var _ porttypes.ICS4Wrapper = (*Keeper)(nil)

// SendPacket rejects ICS20 packets carrying the Noble voucher, whoever the
// sender is, then forwards to the next ICS4 wrapper. The check runs before
// forwarding so a rejected packet never gets a sequence or commitment.
// Data that is not ICS20 is not checked. MsgReturnToNoble sends directly to
// the next wrapper and is not affected.
func (k *Keeper) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	var packet transfertypes.FungibleTokenPacketData
	if sourcePort == types.TransferPortID && json.Unmarshal(data, &packet) == nil {
		if err := k.CheckSend(ctx, packet.Denom); err != nil {
			return 0, err
		}
	}
	return k.ics4Wrapper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement forwards to the next ICS4 wrapper.
func (k *Keeper) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.PacketI, ack exported.Acknowledgement) error {
	return k.ics4Wrapper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion forwards to the next ICS4 wrapper.
func (k *Keeper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// CheckSend returns ErrSendBlocked if denomPath, the full ICS20 denom path of
// an outgoing packet, is the Noble voucher.
func (k *Keeper) CheckSend(ctx sdk.Context, denomPath string) error {
	if k.GetParams(ctx).ShouldBlockSend(denomPath) {
		return errorsmod.Wrapf(types.ErrSendBlocked, "%s can only be returned to Noble with MsgReturnToNoble", denomPath)
	}
	return nil
}
