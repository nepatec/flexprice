package service

import (
	"testing"
	"time"

	"github.com/flexprice/flexprice/internal/api/dto"
	"github.com/flexprice/flexprice/internal/domain/taxassociation"
	taxrate "github.com/flexprice/flexprice/internal/domain/tax"
	"github.com/flexprice/flexprice/internal/testutil"
	"github.com/flexprice/flexprice/internal/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type TaxRateMetadataSuite struct {
	testutil.BaseServiceTestSuite
}

func TestTaxRateMetadataSuite(t *testing.T) {
	suite.Run(t, new(TaxRateMetadataSuite))
}

func (s *TaxRateMetadataSuite) TestMetadataOnlyUpdateOfTaxRateInUse() {
	ctx := s.GetContext()
	stores := s.GetStores()
	svc := NewTaxService(ServiceParams{
		Logger:             s.GetLogger(),
		TaxRateRepo:        stores.TaxRateRepo,
		TaxAssociationRepo: stores.TaxAssociationRepo,
		TaxAppliedRepo:     stores.TaxAppliedRepo,
	})
	pct := decimal.NewFromInt(19)
	s.Require().NoError(stores.TaxRateRepo.Create(ctx, &taxrate.TaxRate{
		ID:              "taxrate-1",
		Name:            "VAT",
		Code:            "vat",
		TaxRateStatus:   types.TaxRateStatusActive,
		TaxRateType:     types.TaxRateTypePercentage,
		PercentageValue: &pct,
		Metadata:        map[string]string{"old": "1"},
		EnvironmentID:   types.GetEnvironmentID(ctx),
		BaseModel:       types.GetDefaultBaseModel(ctx),
	}))
	s.Require().NoError(stores.TaxAssociationRepo.Create(ctx, &taxassociation.TaxAssociation{
		ID:            "taxassoc-1",
		TaxRateID:     "taxrate-1",
		EntityType:    types.TaxRateEntityTypeCustomer,
		EntityID:      "cust-1",
		StartDate:     time.Now().UTC(),
		EnvironmentID: types.GetEnvironmentID(ctx),
		BaseModel:     types.GetDefaultBaseModel(ctx),
	}))

	resp, err := svc.UpdateTaxRate(ctx, "taxrate-1", dto.UpdateTaxRateRequest{Metadata: map[string]string{"sales_customer_default": "inclusive"}})
	s.Require().NoError(err)
	s.Equal(map[string]string{"sales_customer_default": "inclusive"}, resp.Metadata)

	// an empty map removes all metadata
	_, err = svc.UpdateTaxRate(ctx, "taxrate-1", dto.UpdateTaxRateRequest{Metadata: map[string]string{}})
	s.Require().NoError(err)
	stored, err := stores.TaxRateRepo.Get(ctx, "taxrate-1")
	s.Require().NoError(err)
	s.Empty(stored.Metadata)

	// other fields stay locked while the tax rate is in use
	_, err = svc.UpdateTaxRate(ctx, "taxrate-1", dto.UpdateTaxRateRequest{Name: "VAT new"})
	s.ErrorContains(err, "tax rate is being used in tax assignments")
}
