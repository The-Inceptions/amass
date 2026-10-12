// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package scope

import (
	"strings"

	"github.com/caffix/stringset"
	oam "github.com/owasp-amass/open-asset-model"
	oamorg "github.com/owasp-amass/open-asset-model/org"
)

func (s *Scope) AddOrganization(o *oamorg.Organization) bool {
	var names []string

	for _, n := range []string{o.Name, o.LegalName} {
		if name := strings.ToLower(n); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return false
	}

	s.orgLock.Lock()
	defer s.orgLock.Unlock()

	var added bool
	for _, name := range names {
		if _, found := s.orgs[name]; !found {
			s.orgs[name] = o
			added = true
		}
	}
	return added
}

func (s *Scope) AddOrgByName(o string) bool {
	return s.AddOrganization(&oamorg.Organization{ID: o, Name: o})
}

func (s *Scope) Organizations() []*oamorg.Organization {
	set := stringset.New()
	defer set.Close()

	s.orgLock.Lock()
	defer s.orgLock.Unlock()

	var results []*oamorg.Organization
	for _, v := range s.orgs {
		if o, valid := v.(*oamorg.Organization); valid {
			if !set.Has(o.ID) {
				set.Insert(o.ID)
				results = append(results, o)
			}
		}
	}
	return results
}

// matchesOrg returns the in-scope organization with the same name, ignoring case.
// Similar names are deliberately not matched: the similarity metric treats a name
// contained in a longer one as identical, which would pull unrelated organizations
// into scope. An exact match always meets the requested confidence.
func (s *Scope) matchesOrg(o *oamorg.Organization, _ int) (oam.Asset, int) {
	var names []string

	for _, n := range []string{o.Name, o.LegalName} {
		if name := strings.ToLower(n); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return nil, 0
	}

	s.orgLock.Lock()
	defer s.orgLock.Unlock()

	for n, val := range s.orgs {
		for _, name := range names {
			if strings.EqualFold(name, n) {
				return val, 100
			}
		}
	}
	return nil, 0
}
