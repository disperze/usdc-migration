package usdcmigration

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/keeper"
	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/types"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.UpgradableModule      = (*IBCMiddleware)(nil)
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
func (im IBCMiddleware) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, version string) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

// OnChanOpenTry implements porttypes.IBCModule.
func (im IBCMiddleware) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, chanCap *capabilitytypes.Capability, counterparty channeltypes.Counterparty, counterpartyVersion string) (string, error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
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
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err == nil &&
		im.keeper.ShouldBlockRecv(ctx, packet.DestinationChannel, data.Denom) {
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrapf(types.ErrRecvBlocked, "%s on %s", data.Denom, packet.DestinationChannel),
		)
	}
	return im.app.OnRecvPacket(ctx, packet, relayer)
}

// OnAcknowledgementPacket releases the returned total when a return to Noble
// is answered with an error acknowledgement.
func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if err := im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer); err != nil {
		return err
	}

	var ack channeltypes.Acknowledgement
	if err := transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil || ack.Success() {
		return nil
	}
	var data transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		return nil
	}
	return im.keeper.OnReturnRefunded(ctx, packet.SourceChannel, data.Denom, data.Sender, data.Amount, "error_ack")
}

// OnTimeoutPacket releases the returned total when a return to Noble times out.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if err := im.app.OnTimeoutPacket(ctx, packet, relayer); err != nil {
		return err
	}

	var data transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		return nil
	}
	return im.keeper.OnReturnRefunded(ctx, packet.SourceChannel, data.Denom, data.Sender, data.Amount, "timeout")
}

// SendPacket implements porttypes.ICS4Wrapper.
func (im IBCMiddleware) SendPacket(ctx sdk.Context, chanCap *capabilitytypes.Capability, sourcePort, sourceChannel string, timeoutHeight clienttypes.Height, timeoutTimestamp uint64, data []byte) (uint64, error) {
	return im.keeper.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement implements porttypes.ICS4Wrapper.
func (im IBCMiddleware) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet exported.PacketI, ack exported.Acknowledgement) error {
	return im.keeper.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

// GetAppVersion implements porttypes.ICS4Wrapper.
func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.keeper.GetAppVersion(ctx, portID, channelID)
}

// UnmarshalPacketData delegates to the wrapped app so middleware stacked on
// top of this one (e.g. callbacks) keeps working.
func (im IBCMiddleware) UnmarshalPacketData(bz []byte) (interface{}, error) {
	unmarshaler, ok := im.app.(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrInvalidParams, "%T does not implement PacketDataUnmarshaler", im.app)
	}
	return unmarshaler.UnmarshalPacketData(bz)
}

// The channel upgrade callbacks delegate to the wrapped app, so wrapping the
// transfer stack keeps ICS-04 channel upgradability.

func (im IBCMiddleware) upgradable() (porttypes.UpgradableModule, error) {
	cbs, ok := im.app.(porttypes.UpgradableModule)
	if !ok {
		return nil, errorsmod.Wrap(porttypes.ErrInvalidRoute, "upgrade route not found to module in application callstack")
	}
	return cbs, nil
}

// OnChanUpgradeInit implements porttypes.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeInit(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) (string, error) {
	cbs, err := im.upgradable()
	if err != nil {
		return "", err
	}
	return cbs.OnChanUpgradeInit(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
}

// OnChanUpgradeTry implements porttypes.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeTry(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, counterpartyVersion string) (string, error) {
	cbs, err := im.upgradable()
	if err != nil {
		return "", err
	}
	return cbs.OnChanUpgradeTry(ctx, portID, channelID, proposedOrder, proposedConnectionHops, counterpartyVersion)
}

// OnChanUpgradeAck implements porttypes.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeAck(ctx sdk.Context, portID, channelID, counterpartyVersion string) error {
	cbs, err := im.upgradable()
	if err != nil {
		return err
	}
	return cbs.OnChanUpgradeAck(ctx, portID, channelID, counterpartyVersion)
}

// OnChanUpgradeOpen implements porttypes.UpgradableModule.
func (im IBCMiddleware) OnChanUpgradeOpen(ctx sdk.Context, portID, channelID string, proposedOrder channeltypes.Order, proposedConnectionHops []string, proposedVersion string) {
	cbs, err := im.upgradable()
	if err != nil {
		panic(err)
	}
	cbs.OnChanUpgradeOpen(ctx, portID, channelID, proposedOrder, proposedConnectionHops, proposedVersion)
}
