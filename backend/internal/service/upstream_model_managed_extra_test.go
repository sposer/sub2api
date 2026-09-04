package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func testManagedModelExtra() map[string]any {
	return map[string]any{
		UpstreamModelCatalogExtraKey: UpstreamModelCatalogSnapshot{
			Source: "upstream",
			Models: []string{"gemini-live"},
		},
		UpstreamModelMetadataExtraKey: UpstreamModelMetadataSnapshot{
			Source: "upstream",
			Models: map[string]UpstreamModelMetadata{
				"gemini-live": {ID: "gemini-live", ContextWindow: 128000},
			},
		},
	}
}

func TestBuildAccountForCreateStripsUserSuppliedModelSnapshots(t *testing.T) {
	extra := testManagedModelExtra()
	extra["custom"] = "kept"

	account, err := buildAccountForCreate(&CreateAccountInput{
		Name:     "antigravity",
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
	}, extra)

	require.NoError(t, err)
	require.Equal(t, "kept", account.Extra["custom"])
	require.NotContains(t, account.Extra, UpstreamModelCatalogExtraKey)
	require.NotContains(t, account.Extra, UpstreamModelMetadataExtraKey)
}

func TestAdminUpdateAccountPreservesManagedModelSnapshots(t *testing.T) {
	const accountID = int64(401)
	managed := testManagedModelExtra()
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
		accountID: {
			ID:       accountID,
			Name:     "antigravity",
			Platform: PlatformAntigravity,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    mergeMap(nil, managed),
		},
	}}
	forged := testManagedModelExtra()
	forged[UpstreamModelCatalogExtraKey] = UpstreamModelCatalogSnapshot{Source: "forged", Models: []string{"forged-model"}}
	forged[UpstreamModelMetadataExtraKey] = UpstreamModelMetadataSnapshot{Source: "forged", Models: map[string]UpstreamModelMetadata{"forged-model": {ID: "forged-model"}}}
	forged["custom"] = "updated"

	updated, err := (&adminServiceImpl{accountRepo: repo}).UpdateAccount(context.Background(), accountID, &UpdateAccountInput{Extra: forged})

	require.NoError(t, err)
	require.Equal(t, "updated", updated.Extra["custom"])
	require.Equal(t, managed[UpstreamModelCatalogExtraKey], updated.Extra[UpstreamModelCatalogExtraKey])
	require.Equal(t, managed[UpstreamModelMetadataExtraKey], updated.Extra[UpstreamModelMetadataExtraKey])
}

func TestAdminExtraMutationRoutesStripManagedModelSnapshots(t *testing.T) {
	const accountID = int64(402)
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
		accountID: {
			ID:       accountID,
			Platform: PlatformAntigravity,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    testManagedModelExtra(),
		},
	}}
	svc := &adminServiceImpl{accountRepo: repo}

	updates := testManagedModelExtra()
	updates["custom"] = "updated"
	require.NoError(t, svc.UpdateAccountExtra(context.Background(), accountID, updates))
	require.Len(t, repo.updates[accountID], 1)
	require.Equal(t, "updated", repo.updates[accountID][0]["custom"])
	require.NotContains(t, repo.updates[accountID][0], UpstreamModelCatalogExtraKey)
	require.NotContains(t, repo.updates[accountID][0], UpstreamModelMetadataExtraKey)

	bulkExtra := testManagedModelExtra()
	bulkExtra["bulk_custom"] = true
	result, err := svc.BulkUpdateAccounts(context.Background(), &BulkUpdateAccountsInput{
		AccountIDs: []int64{accountID},
		Extra:      bulkExtra,
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.Success)
	require.Len(t, repo.bulkUpdates, 1)
	require.Equal(t, true, repo.bulkUpdates[0].Extra["bulk_custom"])
	require.NotContains(t, repo.bulkUpdates[0].Extra, UpstreamModelCatalogExtraKey)
	require.NotContains(t, repo.bulkUpdates[0].Extra, UpstreamModelMetadataExtraKey)
}

func TestAccountServiceCreateStripsUserSuppliedModelSnapshots(t *testing.T) {
	repo := &upstreamBillingProbeAccountRepo{accounts: make(map[int64]*Account)}
	extra := testManagedModelExtra()
	extra["custom"] = "kept"

	created, err := NewAccountService(repo, nil).Create(context.Background(), CreateAccountRequest{
		Name:     "antigravity",
		Platform: PlatformAntigravity,
		Type:     AccountTypeAPIKey,
		Extra:    extra,
	})

	require.NoError(t, err)
	require.Equal(t, "kept", created.Extra["custom"])
	require.NotContains(t, created.Extra, UpstreamModelCatalogExtraKey)
	require.NotContains(t, created.Extra, UpstreamModelMetadataExtraKey)
}

func TestAccountServiceUpdatePreservesManagedModelSnapshots(t *testing.T) {
	const accountID = int64(403)
	managed := testManagedModelExtra()
	repo := &upstreamBillingProbeAccountRepo{accounts: map[int64]*Account{
		accountID: {
			ID:       accountID,
			Name:     "antigravity",
			Platform: PlatformAntigravity,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Extra:    mergeMap(nil, managed),
		},
	}}
	incoming := testManagedModelExtra()
	incoming[UpstreamModelCatalogExtraKey] = UpstreamModelCatalogSnapshot{Source: "forged", Models: []string{"forged-model"}}
	incoming["custom"] = "updated"

	updated, err := NewAccountService(repo, nil).Update(context.Background(), accountID, UpdateAccountRequest{Extra: &incoming})

	require.NoError(t, err)
	require.Equal(t, "updated", updated.Extra["custom"])
	require.Equal(t, managed[UpstreamModelCatalogExtraKey], updated.Extra[UpstreamModelCatalogExtraKey])
	require.Equal(t, managed[UpstreamModelMetadataExtraKey], updated.Extra[UpstreamModelMetadataExtraKey])
}
