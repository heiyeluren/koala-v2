// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

// Package matcher provides pattern matching for Koala rate limiter rules.
package matcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Exact Matcher Tests
// =============================================================================

func TestExact_Match_Equal(t *testing.T) {
	m := &ExactMatcher{}
	assert.True(t, m.Match("hello", "hello"))
}

func TestExact_Match_NotEqual(t *testing.T) {
	m := &ExactMatcher{}
	assert.False(t, m.Match("hello", "world"))
}

func TestExact_Match_EmptyPattern(t *testing.T) {
	m := &ExactMatcher{}
	assert.True(t, m.Match("", ""))
	assert.False(t, m.Match("", "value"))
}

func TestExact_Match_CaseSensitive(t *testing.T) {
	m := &ExactMatcher{}
	assert.False(t, m.Match("Hello", "hello"))
}

func TestExact_Type(t *testing.T) {
	m := &ExactMatcher{}
	assert.Equal(t, "exact", m.Type())
}

// =============================================================================
// Any Matcher Tests
// =============================================================================

func TestAny_Match_NonEmpty(t *testing.T) {
	m := &AnyMatcher{}
	assert.True(t, m.Match("+", "anything"))
	assert.True(t, m.Match("+", "123"))
	assert.True(t, m.Match("+", " "))
}

func TestAny_Match_Empty(t *testing.T) {
	m := &AnyMatcher{}
	assert.False(t, m.Match("+", ""))
}

func TestAny_Type(t *testing.T) {
	m := &AnyMatcher{}
	assert.Equal(t, "any", m.Type())
}

// =============================================================================
// Not Matcher Tests
// =============================================================================

func TestNot_Match_NotEqual(t *testing.T) {
	m := &NotMatcher{}
	assert.True(t, m.Match("!admin", "user"))
	assert.True(t, m.Match("!admin", "guest"))
}

func TestNot_Match_Equal(t *testing.T) {
	m := &NotMatcher{}
	assert.False(t, m.Match("!admin", "admin"))
}

func TestNot_Match_EmptyValue(t *testing.T) {
	m := &NotMatcher{}
	assert.True(t, m.Match("!admin", ""))
}

func TestNot_Match_EmptyPattern(t *testing.T) {
	m := &NotMatcher{}
	// Pattern "!" means not empty string
	assert.True(t, m.Match("!", "anything"))
	assert.False(t, m.Match("!", ""))
}

func TestNot_Type(t *testing.T) {
	m := &NotMatcher{}
	assert.Equal(t, "not", m.Type())
}

// =============================================================================
// Multi Matcher Tests
// =============================================================================

func TestMulti_Match_InList(t *testing.T) {
	m := &MultiMatcher{}
	assert.True(t, m.Match("a,b,c", "a"))
	assert.True(t, m.Match("a,b,c", "b"))
	assert.True(t, m.Match("a,b,c", "c"))
}

func TestMulti_Match_NotInList(t *testing.T) {
	m := &MultiMatcher{}
	assert.False(t, m.Match("a,b,c", "d"))
	assert.False(t, m.Match("a,b,c", ""))
}

func TestMulti_Match_WithSpaces(t *testing.T) {
	m := &MultiMatcher{}
	// Spaces should be trimmed
	assert.True(t, m.Match("a, b, c", "b"))
	assert.True(t, m.Match(" a , b , c ", "a"))
}

func TestMulti_Match_SingleValue(t *testing.T) {
	m := &MultiMatcher{}
	// Single value without comma should still work
	assert.True(t, m.Match("only", "only"))
}

func TestMulti_Type(t *testing.T) {
	m := &MultiMatcher{}
	assert.Equal(t, "multi", m.Type())
}

// =============================================================================
// Range Matcher Tests
// =============================================================================

func TestRange_Match_InRange(t *testing.T) {
	m := &RangeMatcher{}
	assert.True(t, m.Match("1-100", "50"))
	assert.True(t, m.Match("1-100", "1"))
	assert.True(t, m.Match("1-100", "100"))
}

func TestRange_Match_OutOfRange(t *testing.T) {
	m := &RangeMatcher{}
	assert.False(t, m.Match("1-100", "0"))
	assert.False(t, m.Match("1-100", "101"))
	assert.False(t, m.Match("1-100", "-5"))
}

func TestRange_Match_InvalidValue(t *testing.T) {
	m := &RangeMatcher{}
	assert.False(t, m.Match("1-100", "abc"))
	assert.False(t, m.Match("1-100", ""))
}

func TestRange_Match_NegativeRange(t *testing.T) {
	m := &RangeMatcher{}
	// Pattern "-10-10" means from -10 to 10
	assert.True(t, m.Match("-10-10", "-5"))
	assert.True(t, m.Match("-10-10", "0"))
	assert.True(t, m.Match("-10-10", "10"))
	assert.False(t, m.Match("-10-10", "-11"))
}

func TestRange_Match_InvalidPattern(t *testing.T) {
	m := &RangeMatcher{}
	assert.False(t, m.Match("abc-def", "5"))
	assert.False(t, m.Match("1-", "5"))
	assert.False(t, m.Match("-100", "50"))
}

func TestRange_Type(t *testing.T) {
	m := &RangeMatcher{}
	assert.Equal(t, "range", m.Type())
}

// =============================================================================
// Greater Matcher Tests
// =============================================================================

func TestGreater_Match_Greater(t *testing.T) {
	m := &GreaterMatcher{}
	assert.True(t, m.Match(">10", "11"))
	assert.True(t, m.Match(">10", "100"))
}

func TestGreater_Match_Equal(t *testing.T) {
	m := &GreaterMatcher{}
	assert.False(t, m.Match(">10", "10"))
}

func TestGreater_Match_Less(t *testing.T) {
	m := &GreaterMatcher{}
	assert.False(t, m.Match(">10", "9"))
	assert.False(t, m.Match(">10", "-5"))
}

func TestGreater_Match_InvalidValue(t *testing.T) {
	m := &GreaterMatcher{}
	assert.False(t, m.Match(">10", "abc"))
	assert.False(t, m.Match(">10", ""))
}

func TestGreater_Match_NegativeThreshold(t *testing.T) {
	m := &GreaterMatcher{}
	assert.True(t, m.Match(">-5", "0"))
	assert.True(t, m.Match(">-5", "-4"))
	assert.False(t, m.Match(">-5", "-5"))
	assert.False(t, m.Match(">-5", "-10"))
}

func TestGreater_Match_InvalidPattern(t *testing.T) {
	m := &GreaterMatcher{}
	assert.False(t, m.Match(">abc", "10"))
	assert.False(t, m.Match(">", "10"))
}

func TestGreater_Type(t *testing.T) {
	m := &GreaterMatcher{}
	assert.Equal(t, "greater", m.Type())
}

// =============================================================================
// Less Matcher Tests
// =============================================================================

func TestLess_Match_Less(t *testing.T) {
	m := &LessMatcher{}
	assert.True(t, m.Match("<10", "9"))
	assert.True(t, m.Match("<10", "0"))
	assert.True(t, m.Match("<10", "-5"))
}

func TestLess_Match_Equal(t *testing.T) {
	m := &LessMatcher{}
	assert.False(t, m.Match("<10", "10"))
}

func TestLess_Match_Greater(t *testing.T) {
	m := &LessMatcher{}
	assert.False(t, m.Match("<10", "11"))
	assert.False(t, m.Match("<10", "100"))
}

func TestLess_Match_InvalidValue(t *testing.T) {
	m := &LessMatcher{}
	assert.False(t, m.Match("<10", "abc"))
	assert.False(t, m.Match("<10", ""))
}

func TestLess_Match_NegativeThreshold(t *testing.T) {
	m := &LessMatcher{}
	assert.True(t, m.Match("<-5", "-10"))
	assert.True(t, m.Match("<-5", "-6"))
	assert.False(t, m.Match("<-5", "-5"))
	assert.False(t, m.Match("<-5", "0"))
}

func TestLess_Match_InvalidPattern(t *testing.T) {
	m := &LessMatcher{}
	assert.False(t, m.Match("<abc", "10"))
	assert.False(t, m.Match("<", "10"))
}

func TestLess_Type(t *testing.T) {
	m := &LessMatcher{}
	assert.Equal(t, "less", m.Type())
}

// =============================================================================
// IP Matcher Tests
// =============================================================================

func TestIP_Match_ExactIP(t *testing.T) {
	m := &IPMatcher{}
	assert.True(t, m.Match("192.168.1.1", "192.168.1.1"))
	assert.False(t, m.Match("192.168.1.1", "192.168.1.2"))
}

func TestIP_Match_Wildcard(t *testing.T) {
	m := &IPMatcher{}
	assert.True(t, m.Match("192.168.*.*", "192.168.1.1"))
	assert.True(t, m.Match("192.168.*.*", "192.168.255.255"))
	assert.False(t, m.Match("192.168.*.*", "192.169.1.1"))
}

func TestIP_Match_SingleWildcard(t *testing.T) {
	m := &IPMatcher{}
	assert.True(t, m.Match("192.168.1.*", "192.168.1.100"))
	assert.True(t, m.Match("192.168.1.*", "192.168.1.0"))
	assert.False(t, m.Match("192.168.1.*", "192.168.2.100"))
}

func TestIP_Match_LeadingWildcard(t *testing.T) {
	m := &IPMatcher{}
	assert.True(t, m.Match("*.168.1.1", "192.168.1.1"))
	assert.True(t, m.Match("*.168.1.1", "10.168.1.1"))
}

func TestIP_Match_InvalidIP(t *testing.T) {
	m := &IPMatcher{}
	assert.False(t, m.Match("192.168.*.*", "invalid"))
	assert.False(t, m.Match("192.168.*.*", "192.168.1"))
	assert.False(t, m.Match("192.168.*.*", ""))
}

func TestIP_Match_InvalidPattern(t *testing.T) {
	m := &IPMatcher{}
	// Pattern with fewer than 4 parts
	assert.False(t, m.Match("192.168.*", "192.168.1.1"))
}

func TestIP_Type(t *testing.T) {
	m := &IPMatcher{}
	assert.Equal(t, "ip", m.Type())
}

// =============================================================================
// Dict Matcher Tests
// =============================================================================

func TestDict_Match_InDict(t *testing.T) {
	m := NewDictMatcher()
	// Register a dictionary
	m.RegisterDict("blacklist", map[string]bool{
		"spam":    true,
		"malware": true,
		"phish":   true,
	})

	assert.True(t, m.Match("@blacklist", "spam"))
	assert.True(t, m.Match("@blacklist", "malware"))
}

func TestDict_Match_NotInDict(t *testing.T) {
	m := NewDictMatcher()
	m.RegisterDict("blacklist", map[string]bool{
		"spam": true,
	})

	assert.False(t, m.Match("@blacklist", "legitimate"))
	assert.False(t, m.Match("@blacklist", ""))
}

func TestDict_Match_UnknownDict(t *testing.T) {
	m := NewDictMatcher()
	assert.False(t, m.Match("@unknown", "value"))
}

func TestDict_Match_EmptyDictName(t *testing.T) {
	m := NewDictMatcher()
	assert.False(t, m.Match("@", "value"))
}

func TestDict_Type(t *testing.T) {
	m := NewDictMatcher()
	assert.Equal(t, "dict", m.Type())
}

func TestDict_RegisterDict(t *testing.T) {
	m := NewDictMatcher()
	m.RegisterDict("users", map[string]bool{
		"alice": true,
		"bob":   true,
	})

	assert.True(t, m.Match("@users", "alice"))
	assert.True(t, m.Match("@users", "bob"))
	assert.False(t, m.Match("@users", "charlie"))
}

func TestDict_RegisterDictSlice(t *testing.T) {
	m := NewDictMatcher()
	m.RegisterDictSlice("admins", []string{"root", "admin", "superuser"})

	assert.True(t, m.Match("@admins", "root"))
	assert.True(t, m.Match("@admins", "admin"))
	assert.False(t, m.Match("@admins", "user"))
}

// =============================================================================
// Parse Function Tests
// =============================================================================

func TestParse_Any(t *testing.T) {
	m := Parse("+")
	assert.Equal(t, "any", m.Type())
	assert.True(t, m.Match("+", "anything"))
}

func TestParse_Not(t *testing.T) {
	m := Parse("!admin")
	assert.Equal(t, "not", m.Type())
	assert.True(t, m.Match("!admin", "user"))
}

func TestParse_Dict(t *testing.T) {
	m := Parse("@blacklist")
	assert.Equal(t, "dict", m.Type())
}

func TestParse_Greater(t *testing.T) {
	m := Parse(">10")
	assert.Equal(t, "greater", m.Type())
	assert.True(t, m.Match(">10", "15"))
}

func TestParse_Less(t *testing.T) {
	m := Parse("<10")
	assert.Equal(t, "less", m.Type())
	assert.True(t, m.Match("<10", "5"))
}

func TestParse_Multi(t *testing.T) {
	m := Parse("a,b,c")
	assert.Equal(t, "multi", m.Type())
	assert.True(t, m.Match("a,b,c", "b"))
}

func TestParse_Range(t *testing.T) {
	m := Parse("1-100")
	assert.Equal(t, "range", m.Type())
	assert.True(t, m.Match("1-100", "50"))
}

func TestParse_IP(t *testing.T) {
	m := Parse("192.168.*.*")
	assert.Equal(t, "ip", m.Type())
	assert.True(t, m.Match("192.168.*.*", "192.168.1.1"))
}

func TestParse_Exact(t *testing.T) {
	m := Parse("hello")
	assert.Equal(t, "exact", m.Type())
	assert.True(t, m.Match("hello", "hello"))
}

func TestParse_ExactWithDash(t *testing.T) {
	// "abc-def" should be exact match, not range (since sides are not numbers)
	m := Parse("abc-def")
	assert.Equal(t, "exact", m.Type())
}

func TestParse_ExactNumber(t *testing.T) {
	// A single number without prefix should be exact match
	m := Parse("12345")
	assert.Equal(t, "exact", m.Type())
	assert.True(t, m.Match("12345", "12345"))
	assert.False(t, m.Match("12345", "12346"))
}

func TestParse_RangeWithNegativeStart(t *testing.T) {
	// "-10-10" should be range from -10 to 10
	m := Parse("-10-10")
	assert.Equal(t, "range", m.Type())
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestParse_AllMatchersWork(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		match   bool
	}{
		// Exact
		{"hello", "hello", true},
		{"hello", "world", false},
		// Any
		{"+", "anything", true},
		{"+", "", false},
		// Not
		{"!admin", "user", true},
		{"!admin", "admin", false},
		// Multi
		{"a,b,c", "b", true},
		{"a,b,c", "d", false},
		// Range
		{"1-100", "50", true},
		{"1-100", "150", false},
		// Greater
		{">10", "20", true},
		{">10", "5", false},
		// Less
		{"<10", "5", true},
		{"<10", "20", false},
		// IP
		{"192.168.*.*", "192.168.1.1", true},
		{"192.168.*.*", "10.0.0.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.value, func(t *testing.T) {
			m := Parse(tt.pattern)
			result := m.Match(tt.pattern, tt.value)
			assert.Equal(t, tt.match, result, "pattern=%s, value=%s", tt.pattern, tt.value)
		})
	}
}

// =============================================================================
// Edge Case Tests
// =============================================================================

func TestEdge_EmptyPattern(t *testing.T) {
	m := Parse("")
	assert.Equal(t, "exact", m.Type())
	assert.True(t, m.Match("", ""))
	assert.False(t, m.Match("", "something"))
}

func TestEdge_WhitespaceOnly(t *testing.T) {
	m := Parse("   ")
	assert.Equal(t, "exact", m.Type())
	assert.True(t, m.Match("   ", "   "))
}

func TestEdge_SpecialCharacters(t *testing.T) {
	m := Parse("user@example.com")
	// This has @ but not at the start, so should be exact
	assert.Equal(t, "exact", m.Type())
	assert.True(t, m.Match("user@example.com", "user@example.com"))
}

func TestEdge_IPLikeButNotWildcard(t *testing.T) {
	m := Parse("192.168.1.1")
	// No wildcard, should be exact match
	assert.Equal(t, "exact", m.Type())
}
