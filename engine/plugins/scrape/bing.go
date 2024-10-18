// Copyright © by Jeff Foley 2017-2024. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package scrape

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/caffix/stringset"
	"github.com/owasp-amass/amass/v4/engine/plugins/support"
	et "github.com/owasp-amass/amass/v4/engine/types"
	"github.com/owasp-amass/amass/v4/utils/net/http"
	dbt "github.com/owasp-amass/asset-db/types"
	oam "github.com/owasp-amass/open-asset-model"
	"github.com/owasp-amass/open-asset-model/domain"
	"github.com/owasp-amass/open-asset-model/source"
	"go.uber.org/ratelimit"
)

type bing struct {
	name   string
	fmtstr string
	log    *slog.Logger
	rlimit ratelimit.Limiter
	source *source.Source
}

func NewBing() et.Plugin {
	return &bing{
		name:   "Bing",
		fmtstr: "https://www.ask.com/web?o=0&l=dir&qo=pagination&page=%d&q=site:%s -www.%s",
		rlimit: ratelimit.New(2, ratelimit.WithoutSlack),
		source: &source.Source{
			Name:       "Bing",
			Confidence: 60,
		},
	}
}

func (b *bing) Name() string {
	return b.name
}

func (b *bing) Start(r et.Registry) error {
	b.log = r.Log().WithGroup("plugin").With("name", b.name)

	if err := r.RegisterHandler(&et.Handler{
		Plugin:       b,
		Name:         b.name + "-Handler",
		Priority:     7,
		MaxInstances: 10,
		Transforms:   []string{string(oam.FQDN)},
		EventType:    oam.FQDN,
		Callback:     b.check,
	}); err != nil {
		return err
	}

	b.log.Info("Plugin started")
	return nil
}

func (b *bing) Stop() {
	b.log.Info("Plugin stopped")
}

func (b *bing) check(e *et.Event) error {
	fqdn, ok := e.Asset.Asset.(*domain.FQDN)
	if !ok {
		return errors.New("failed to extract the FQDN asset")
	}

	if a, conf := e.Session.Scope().IsAssetInScope(fqdn, 0); conf == 0 || a == nil {
		return nil
	} else if f, ok := a.(*domain.FQDN); !ok || f == nil || !strings.EqualFold(fqdn.Name, f.Name) {
		return nil
	}

	src := support.GetSource(e.Session, b.source)
	if src == nil {
		return errors.New("failed to obtain the plugin source information")
	}

	since, err := support.TTLStartTime(e.Session.Config(), string(oam.FQDN), string(oam.FQDN), b.name)
	if err != nil {
		return err
	}

	var names []*dbt.Asset
	if support.AssetMonitoredWithinTTL(e.Session, e.Asset, src, since) {
		names = append(names, b.lookup(e, fqdn.Name, src, since)...)
	} else {
		names = append(names, b.query(e, fqdn.Name, src)...)
		support.MarkAssetMonitored(e.Session, e.Asset, src)
	}

	if len(names) > 0 {
		b.process(e, names, src)
	}
	return nil
}

func (b *bing) lookup(e *et.Event, name string, src *dbt.Asset, since time.Time) []*dbt.Asset {
	return support.SourceToAssetsWithinTTL(e.Session, name, string(oam.FQDN), src, since)
}

func (b *bing) query(e *et.Event, name string, src *dbt.Asset) []*dbt.Asset {
	subs := stringset.New()
	defer subs.Close()

	for i := 1; i < 10; i++ {
		b.rlimit.Take()
		resp, err := http.RequestWebPage(context.TODO(), &http.Request{URL: fmt.Sprintf(b.fmtstr, i, name, name)})
		if err != nil || resp.Body == "" {
			break
		}

		for _, n := range support.ScrapeSubdomainNames(resp.Body) {
			nstr := strings.ToLower(strings.TrimSpace(n))
			// if the subdomain is not in scope, skip it
			if _, conf := e.Session.Scope().IsAssetInScope(&domain.FQDN{Name: nstr}, 0); conf > 0 {
				subs.Insert(nstr)
			}
		}
	}

	return b.store(e, subs.Slice(), src)
}

func (b *bing) store(e *et.Event, names []string, src *dbt.Asset) []*dbt.Asset {
	return support.StoreFQDNsWithSource(e.Session, names, src, b.name, b.name+"-Handler")
}

func (b *bing) process(e *et.Event, assets []*dbt.Asset, src *dbt.Asset) {
	support.ProcessFQDNsWithSource(e, assets, src)
}