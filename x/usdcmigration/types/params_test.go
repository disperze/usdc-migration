package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/types"
)

const (
	noble = "channel-224"
	other = "channel-1"
)

func activeParams() types.Params {
	p := types.DefaultParams()
	p.NobleChannels = []string{noble}
	return p
}

// Voucher denoms of Noble USDC as listed in cosmos/chain-registry assetlists.
func TestVoucherDenomMatchesRegistry(t *testing.T) {
	cases := map[string]string{
		"channel-224": "ibc/4A482FA914A4B9B05801ED81C33713899F322B24F76A06F4B8FE872485EA22FF", // juno
		"channel-253": "ibc/2C962DAB9F57FE0921435426AE75196009FAA1981BF86991203C8411F8980FDB", // terra2
		"channel-2":   "ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349", // xion
		"channel-29":  "ibc/43897B9739BD63E3A08A88191999C632E052724AB96BD4C74AE31375C991F48D", // archway
	}
	for ch, want := range cases {
		require.Equal(t, want, types.VoucherDenom(ch, "uusdc"), ch)
		// must agree with ibc-go's own derivation
		require.Equal(t, want, transfertypes.ParseDenomTrace(types.VoucherPath(ch, "uusdc")).IBCDenom(), ch)
	}
}

func TestParamsValidate(t *testing.T) {
	require.NoError(t, types.DefaultParams().Validate())
	require.NoError(t, activeParams().Validate())

	p := activeParams()
	p.NobleChannels = []string{noble, noble}
	require.Error(t, p.Validate(), "duplicate channel")

	for _, bad := range []string{"", "channel-", "channel-x", "connection-1", "07-tendermint-0"} {
		p = activeParams()
		p.NobleChannels = []string{bad}
		require.Error(t, p.Validate(), bad)
	}

	p = activeParams()
	p.BaseDenom = ""
	require.Error(t, p.Validate())
}

func TestShouldBlockRecv(t *testing.T) {
	cases := []struct {
		name    string
		params  types.Params
		channel string
		denom   string
		blocked bool
	}{
		{"noble usdc on noble channel", activeParams(), noble, "uusdc", true},
		{"other noble denom", activeParams(), noble, "uusdn", false},
		{"usdc on other channel", activeParams(), other, "uusdc", false},
		{"native token returning from noble", activeParams(), noble, "transfer/channel-3/ujuno", false},
		{"prefixed usdc multi-hop", activeParams(), noble, "transfer/channel-5/uusdc", false},
		{"recv block disabled", func() types.Params { p := activeParams(); p.RecvBlockEnabled = false; return p }(), noble, "uusdc", false},
		{"no noble channel configured", types.DefaultParams(), noble, "uusdc", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.blocked, tc.params.ShouldBlockRecv(tc.channel, tc.denom))
		})
	}
}

func TestShouldBlockSend(t *testing.T) {
	voucher := types.VoucherPath(noble, "uusdc")
	cases := []struct {
		name    string
		params  types.Params
		denom   string
		blocked bool
	}{
		{"voucher", activeParams(), voucher, true},
		{"native denom", activeParams(), "ujuno", false},
		{"other voucher", activeParams(), "transfer/channel-1/uatom", false},
		{"usdc that arrived via another chain", activeParams(), "transfer/channel-1/uusdc", false},
		{"send block disabled", func() types.Params { p := activeParams(); p.SendBlockEnabled = false; return p }(), voucher, false},
		{"no noble channel configured", types.DefaultParams(), voucher, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.blocked, tc.params.ShouldBlockSend(tc.denom))
		})
	}
}

func TestGenesisValidate(t *testing.T) {
	require.NoError(t, types.DefaultGenesis().Validate())
}
