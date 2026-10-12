// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package scope

import (
	"testing"

	oamcon "github.com/owasp-amass/open-asset-model/contact"
	oamorg "github.com/owasp-amass/open-asset-model/org"
	"github.com/stretchr/testify/assert"
)

func TestMatchesOrgIsExactOnly(t *testing.T) {
	s := New(nil).(*Scope)
	s.AddOrganization(&oamorg.Organization{ID: "apple", Name: "Apple"})

	// -1 is the confidence used when no transformation matches the asset type
	for _, conf := range []int{50, -1} {
		got, score := s.matchesOrg(&oamorg.Organization{Name: "APPLE"}, conf)
		assert.NotNil(t, got, "conf %d", conf)
		assert.Equal(t, 100, score, "conf %d", conf)

		for _, name := range []string{"Pineapple Inc", "Apple Holdings", "Appel"} {
			got, score = s.matchesOrg(&oamorg.Organization{Name: name}, conf)
			assert.Nil(t, got, "%q at conf %d", name, conf)
			assert.Equal(t, 0, score, "%q at conf %d", name, conf)
		}
	}
}

func TestMatchesLocationIsExactOnly(t *testing.T) {
	s := New(nil).(*Scope)
	s.AddLocation(&oamcon.Location{
		Address:        "1 Infinite Loop, Cupertino, CA",
		BuildingNumber: "1",
		StreetName:     "Infinite Loop",
		City:           "Cupertino",
		Province:       "CA",
	})

	for _, conf := range []int{50, -1} {
		got, score := s.matchesLocation(&oamcon.Location{
			BuildingNumber: "1",
			StreetName:     "INFINITE LOOP",
			City:           "cupertino",
			Province:       "ca",
		}, conf)
		assert.NotNil(t, got, "conf %d", conf)
		assert.Equal(t, 100, score, "conf %d", conf)

		got, score = s.matchesLocation(&oamcon.Location{
			BuildingNumber: "1",
			StreetName:     "Infinite Loop Rd",
			City:           "Cupertino",
			Province:       "CA",
		}, conf)
		assert.Nil(t, got, "conf %d", conf)
		assert.Equal(t, 0, score, "conf %d", conf)
	}
}
