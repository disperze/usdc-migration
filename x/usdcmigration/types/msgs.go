package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

var (
	_ sdk.Msg              = (*MsgUpdateParams)(nil)
	_ sdk.Msg              = (*MsgReturnToNoble)(nil)
	_ sdk.HasValidateBasic = (*MsgUpdateParams)(nil)
	_ sdk.HasValidateBasic = (*MsgReturnToNoble)(nil)
)

// ValidateBasic implements sdk.HasValidateBasic.
func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, err.Error())
	}
	if err := m.Params.Validate(); err != nil {
		return errorsmod.Wrap(ErrInvalidParams, err.Error())
	}
	return nil
}

// ValidateBasic implements sdk.HasValidateBasic.
func (m *MsgReturnToNoble) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errorsmod.Wrap(ErrInvalidAuthority, err.Error())
	}
	if err := validateChannelID(m.ChannelId); err != nil {
		return errorsmod.Wrap(ErrUnknownChannel, err.Error())
	}
	hrp, _, err := bech32.DecodeAndConvert(m.Receiver)
	if err != nil {
		return errorsmod.Wrap(ErrInvalidReceiver, err.Error())
	}
	if hrp != NobleBech32Prefix {
		return errorsmod.Wrapf(ErrInvalidReceiver, "expected %s prefix, got %s", NobleBech32Prefix, hrp)
	}
	if m.Amount.IsNil() || !m.Amount.IsPositive() {
		return errorsmod.Wrap(ErrInvalidAmount, "amount must be positive")
	}
	if m.TimeoutSeconds > MaxTimeoutSeconds {
		return errorsmod.Wrapf(ErrInvalidTimeout, "timeout_seconds must be <= %d", MaxTimeoutSeconds)
	}
	return nil
}
