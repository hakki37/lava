package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/utils/slices"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestAddingTrackedCu(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 0) // 2 providers, 1 client, default providers-to-pair

	ts.AdvanceEpoch()

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, provider1Addr := ts.GetAccount(common.PROVIDER, 0)
	_, provider2Addr := ts.GetAccount(common.PROVIDER, 1)

	res, err := ts.QuerySubscriptionCurrent(client1Addr)
	require.Nil(t, err)
	sub := res.Sub

	// simulate relay payment for provider 1
	cuSum := uint64(100)
	relaySession := ts.newRelaySession(provider1Addr, 0, cuSum, ts.BlockHeight(), 0)

	sig, err := sigs.Sign(client1Acct.SK, *relaySession)
	require.Nil(t, err)
	relaySession.Sig = sig

	relayPaymentMessage := types.MsgRelayPayment{
		Creator: provider1Addr,
		Relays:  slices.Slice(relaySession),
	}

	ts.relayPaymentWithoutPay(relayPaymentMessage, true)

	// check trackedCU only updated on provider 1
	cu, _, _, found, _ := ts.Keepers.Subscription.GetTrackedCu(ts.Ctx, sub.Consumer, provider1Addr, ts.spec.Index)
	require.True(t, found)
	require.Equal(t, cuSum, cu)

	_, _, _, found, _ = ts.Keepers.Subscription.GetTrackedCu(ts.Ctx, sub.Consumer, provider2Addr, ts.spec.Index)
	require.False(t, found)
}
