// Copyright © by Jeff Foley 2017-2026. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.
// SPDX-License-Identifier: Apache-2.0

package brute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdditions(t *testing.T) {
	assert.Equal(t, []string{
		"xabc", "yabc", "axbc", "aybc", "abxc", "abyc", "abcx", "abcy",
	}, additions([]string{"abc"}, "xy"))
}

func TestDeletions(t *testing.T) {
	assert.Equal(t, []string{"bc", "ac", "ab"}, deletions([]string{"abc"}))
	assert.Equal(t, []string{"b", "a", "d", "c"}, deletions([]string{"ab", "cd"}))
	assert.Empty(t, deletions([]string{"a"}))
}

func TestSubstitutions(t *testing.T) {
	assert.Equal(t, []string{"xb", "yb", "ax", "ay"}, substitutions([]string{"ab"}, "xy"))
}

func TestFuzzyLabelSearches(t *testing.T) {
	assert.Equal(t, []string{
		"ab.example.com",
		"xab.example.com", "axb.example.com", "abx.example.com",
		"b.example.com", "a.example.com",
		"xb.example.com", "ax.example.com",
	}, fuzzyLabelSearches("ab.example.com", 1, "x"))
}
