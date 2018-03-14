package lease_test

import (
	"fmt"
	"testing"

	dapp "github.com/ovrclk/akash/app/deployment"
	fapp "github.com/ovrclk/akash/app/fulfillment"
	app_ "github.com/ovrclk/akash/app/lease"
	oapp "github.com/ovrclk/akash/app/order"
	papp "github.com/ovrclk/akash/app/provider"
	state_ "github.com/ovrclk/akash/state"
	"github.com/ovrclk/akash/testutil"
	"github.com/ovrclk/akash/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/tendermint/abci/types"
)

func TestAcceptQuery(t *testing.T) {
	state := testutil.NewState(t, nil)

	account, _ := testutil.CreateAccount(t, state)
	address := account.Address

	app, err := app_.NewApp(state, testutil.Logger())
	require.NoError(t, err)

	{
		path := fmt.Sprintf("%v%X", state_.LeasePath, address)
		assert.True(t, app.AcceptQuery(tmtypes.RequestQuery{Path: path}))
	}

	{
		path := fmt.Sprintf("%v%X", "/foo/", address)
		assert.False(t, app.AcceptQuery(tmtypes.RequestQuery{Path: path}))
	}
}

func TestValidTx(t *testing.T) {

	state := testutil.NewState(t, nil)
	app, err := app_.NewApp(state, testutil.Logger())

	// create provider
	papp, err := papp.NewApp(state, testutil.Logger())
	require.NoError(t, err)
	paccount, pkey := testutil.CreateAccount(t, state)
	pnonce := uint64(0)
	provider := testutil.CreateProvider(t, papp, paccount, &pkey, pnonce)

	// create tenant
	taccount, tkey := testutil.CreateAccount(t, state)

	// create deployment
	dapp, err := dapp.NewApp(state, testutil.Logger())
	require.NoError(t, err)
	tnonce := uint64(1)
	deployment := testutil.CreateDeployment(t, dapp, taccount, &tkey, tnonce)
	groupSeq := deployment.Groups[0].Seq
	daddress := state_.DeploymentAddress(taccount.Address, tnonce)

	// create order
	oapp, err := oapp.NewApp(state, testutil.Logger())
	require.NoError(t, err)
	oSeq := uint64(0)
	testutil.CreateOrder(t, oapp, taccount, &tkey, deployment.Address, groupSeq, oSeq)
	price := uint32(0)

	// create fulfillment
	fapp, err := fapp.NewApp(state, testutil.Logger())
	testutil.CreateFulfillment(t, fapp, provider.Address, &pkey, daddress, groupSeq, oSeq, price)

	// create lease
	lease := testutil.CreateLease(t, app, provider.Address, &pkey, daddress, groupSeq, oSeq, price)

	{
		path := fmt.Sprintf("%v%X", state_.LeasePath, state.Lease().IDFor(lease))
		resp := app.Query(tmtypes.RequestQuery{Path: path})
		assert.Empty(t, resp.Log)
		require.True(t, resp.IsOK())
		lea := new(types.Lease)
		require.NoError(t, lea.Unmarshal(resp.Value))

		assert.Equal(t, lease.Deployment, lea.Deployment)
		assert.Equal(t, lease.Group, lea.Group)
		assert.Equal(t, lease.Order, lea.Order)
		assert.Equal(t, lease.Provider, lea.Provider)
		assert.Equal(t, lease.Price, lea.Price)
	}
}