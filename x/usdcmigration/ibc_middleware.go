package usdcmigration

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/keeper"
	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
)

// IBCMiddleware must be the outermost layer of the ICS20 transfer stack so it
// sees packets before PFM, ibc-hooks or other middleware act on them.
type IBCMiddleware struct {
	app    porttypes.IBCModule
	keeper *keeper.Keeper
}

// NewIBCMiddleware wraps app (the existing transfer stack).
func NewIBCMiddleware(k *keeper.Keeper, app porttypes.IBCModule) IBCMiddleware {
	return IBCMiddleware{app: app, keeper: k}
}

// OnChanOpenInit implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

// OnChanOpenTry implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanOpenAck(ctx sdk.Context, portID, channelID, counterpartyChannelID, counterpartyVersion string) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket rejects the deprecated Noble denom arriving on a Noble channel
// with an error acknowledgement, so Noble refunds the sender.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err == nil {
		block, err := im.keeper.ShouldBlockRecv(ctx, packet.DestinationChannel, data.Denom)
		if err != nil {
			// Fail closed: an error ack makes Noble refund the sender.
			return channeltypes.NewErrorAcknowledgement(err)
		}
		if block {
			return channeltypes.NewErrorAcknowledgement(
				errorsmod.Wrapf(types.ErrRecvBlocked, "%s on %s", data.Denom, packet.DestinationChannel),
			)
		}
	}
	return im.app.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

// OnAcknowledgementPacket releases the returned total when a return to Noble
// is answered with an error acknowledgement. The error ack of a return packet
// never reaches the transfer app: the return burned nothing, and the app's
// refund would mint the voucher to the authority.
func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	data, isReturn, err := im.returnPacketData(ctx, packet)
	if err != nil {
		return err
	}
	if isReturn {
		var ack channeltypes.Acknowledgement
		if err := transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err == nil && !ack.Success() {
			return im.keeper.OnReturnRefunded(ctx, packet.SourceChannel, packet.Sequence, data.Denom, data.Sender, data.Amount, types.RefundReasonErrorAck)
		}
	}
	return im.app.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// OnTimeoutPacket releases the returned total when a return to Noble times
// out. Like an error ack, the timeout of a return packet never reaches the
// transfer app, so no voucher is minted back.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	data, isReturn, err := im.returnPacketData(ctx, packet)
	if err != nil {
		return err
	}
	if isReturn {
		return im.keeper.OnReturnRefunded(ctx, packet.SourceChannel, packet.Sequence, data.Denom, data.Sender, data.Amount, types.RefundReasonTimeout)
	}
	return im.app.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

// returnPacketData decodes the ICS20 data of packet and reports whether it is
// a return packet sent by MsgReturnToNoble.
func (im IBCMiddleware) returnPacketData(ctx sdk.Context, packet channeltypes.Packet) (transfertypes.FungibleTokenPacketData, bool, error) {
	var data transfertypes.FungibleTokenPacketData
	if packet.SourcePort != types.TransferPortID {
		return data, false, nil
	}
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		return data, false, nil
	}
	isReturn, err := im.keeper.IsReturnPacket(ctx, packet.SourceChannel, data.Denom, data.Sender)
	return data, isReturn, err
}

// SendPacket implements porttypes.ICS4Wrapper.
func (im IBCMiddleware) SendPacket(ctx sdk.Context, sourcePort, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	return im.keeper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements porttypes.ICS4Wrapper.
func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, packet exported.PacketI, ack exported.Acknowledgement) error {
	return im.keeper.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion implements porttypes.ICS4Wrapper.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData delegates to the wrapped app so middleware stacked on
// top of this one (e.g. callbacks) keeps working.
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (any, string, error) {
	unmarshaler, ok := im.app.(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, "", errorsmod.Wrapf(types.ErrUnsupportedApp, "%T does not implement PacketDataUnmarshaler", im.app)
	}
	return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
}
