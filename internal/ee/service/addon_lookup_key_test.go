package service

import (
	"testing"

	"github.com/flexprice/flexprice/internal/api/dto"
	"github.com/flexprice/flexprice/internal/domain/addon"
	"github.com/flexprice/flexprice/internal/testutil"
	"github.com/flexprice/flexprice/internal/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"
)

type AddonLookupKeySuite struct {
	testutil.BaseServiceTestSuite
}

func TestAddonLookupKeySuite(t *testing.T) {
	suite.Run(t, new(AddonLookupKeySuite))
}

func (s *AddonLookupKeySuite) TestUpdateAddonLookupKey() {
	svc := &addonService{ServiceParams: ServiceParams{Logger: s.GetLogger(), AddonRepo: s.GetStores().AddonRepo}}
	s.NoError(s.GetStores().AddonRepo.Create(s.GetContext(), &addon.Addon{
		ID:        "addon-1",
		Name:      "Addon",
		LookupKey: "addon-old",
		BaseModel: types.GetDefaultBaseModel(s.GetContext()),
	}))

	resp, err := svc.UpdateAddon(s.GetContext(), "addon-1", dto.UpdateAddonRequest{LookupKey: lo.ToPtr("addon-new")})
	s.Require().NoError(err)
	s.Equal("addon-new", resp.LookupKey)
	stored, err := s.GetStores().AddonRepo.GetByID(s.GetContext(), "addon-1")
	s.Require().NoError(err)
	s.Equal("addon-new", stored.LookupKey)

	_, err = svc.UpdateAddon(s.GetContext(), "addon-1", dto.UpdateAddonRequest{LookupKey: lo.ToPtr("")})
	s.ErrorContains(err, "lookup_key must not be empty")
}
