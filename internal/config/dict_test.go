// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadDict tests loading a valid dictionary file.
func TestLoadDict(t *testing.T) {
	tmpDir := t.TempDir()
	dictPath := filepath.Join(tmpDir, "whitelist.txt")

	dictContent := `# IP Whitelist
# Comment lines start with #

127.0.0.1
192.168.1.1
10.0.0.1

# Another comment
192.168.0.0/24
`

	err := os.WriteFile(dictPath, []byte(dictContent), 0644)
	require.NoError(t, err)

	dict, err := LoadDict(dictPath)
	require.NoError(t, err)
	require.NotNil(t, dict)

	// Verify entries are loaded correctly
	assert.Len(t, dict.Entries, 4)
	assert.Contains(t, dict.Entries, "127.0.0.1")
	assert.Contains(t, dict.Entries, "192.168.1.1")
	assert.Contains(t, dict.Entries, "10.0.0.1")
	assert.Contains(t, dict.Entries, "192.168.0.0/24")

	// Comments and empty lines should be ignored
	assert.NotContains(t, dict.Entries, "# IP Whitelist")
	assert.NotContains(t, dict.Entries, "")
}

// TestLoadDictNotFound tests loading a non-existent dictionary file.
func TestLoadDictNotFound(t *testing.T) {
	dict, err := LoadDict("/non/existent/path/dict.txt")
	assert.Error(t, err)
	assert.Nil(t, dict)
}

// TestLoadDictEmpty tests loading an empty dictionary file.
func TestLoadDictEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	dictPath := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(dictPath, []byte(""), 0644)
	require.NoError(t, err)

	dict, err := LoadDict(dictPath)
	require.NoError(t, err)
	require.NotNil(t, dict)
	assert.Len(t, dict.Entries, 0)
}

// TestLoadDictOnlyComments tests loading a dictionary with only comments.
func TestLoadDictOnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	dictPath := filepath.Join(tmpDir, "comments.txt")

	dictContent := `# This is a comment
# Another comment

# Yet another comment
`

	err := os.WriteFile(dictPath, []byte(dictContent), 0644)
	require.NoError(t, err)

	dict, err := LoadDict(dictPath)
	require.NoError(t, err)
	require.NotNil(t, dict)
	assert.Len(t, dict.Entries, 0)
}

// TestDictContains tests the Contains method.
func TestDictContains(t *testing.T) {
	dict := &Dict{
		Entries: map[string]struct{}{
			"127.0.0.1":   {},
			"192.168.1.1": {},
			"user123":     {},
		},
	}

	assert.True(t, dict.Contains("127.0.0.1"))
	assert.True(t, dict.Contains("192.168.1.1"))
	assert.True(t, dict.Contains("user123"))
	assert.False(t, dict.Contains("10.0.0.1"))
	assert.False(t, dict.Contains("unknown"))
}

// TestDictAdd tests the Add method.
func TestDictAdd(t *testing.T) {
	dict := NewDict()

	dict.Add("127.0.0.1")
	dict.Add("192.168.1.1")
	dict.Add("127.0.0.1") // Duplicate should be ignored

	assert.Len(t, dict.Entries, 2)
	assert.True(t, dict.Contains("127.0.0.1"))
	assert.True(t, dict.Contains("192.168.1.1"))
}

// TestDictRemove tests the Remove method.
func TestDictRemove(t *testing.T) {
	dict := &Dict{
		Entries: map[string]struct{}{
			"127.0.0.1":   {},
			"192.168.1.1": {},
		},
	}

	dict.Remove("127.0.0.1")
	assert.False(t, dict.Contains("127.0.0.1"))
	assert.True(t, dict.Contains("192.168.1.1"))
	assert.Len(t, dict.Entries, 1)

	// Remove non-existent entry should not panic
	dict.Remove("10.0.0.1")
	assert.Len(t, dict.Entries, 1)
}

// TestDictList tests the List method.
func TestDictList(t *testing.T) {
	dict := &Dict{
		Entries: map[string]struct{}{
			"127.0.0.1":   {},
			"192.168.1.1": {},
			"10.0.0.1":    {},
		},
	}

	list := dict.List()
	assert.Len(t, list, 3)
	assert.Contains(t, list, "127.0.0.1")
	assert.Contains(t, list, "192.168.1.1")
	assert.Contains(t, list, "10.0.0.1")
}

// TestDictSize tests the Size method.
func TestDictSize(t *testing.T) {
	dict := NewDict()
	assert.Equal(t, 0, dict.Size())

	dict.Add("127.0.0.1")
	assert.Equal(t, 1, dict.Size())

	dict.Add("192.168.1.1")
	assert.Equal(t, 2, dict.Size())

	dict.Remove("127.0.0.1")
	assert.Equal(t, 1, dict.Size())
}

// TestDictClear tests the Clear method.
func TestDictClear(t *testing.T) {
	dict := &Dict{
		Entries: map[string]struct{}{
			"127.0.0.1":   {},
			"192.168.1.1": {},
		},
	}

	dict.Clear()
	assert.Len(t, dict.Entries, 0)
	assert.False(t, dict.Contains("127.0.0.1"))
}

// TestLoadDictWithWhitespace tests loading a dictionary with whitespace.
func TestLoadDictWithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	dictPath := filepath.Join(tmpDir, "whitespace.txt")

	dictContent := `  127.0.0.1
	192.168.1.1
 10.0.0.1
`

	err := os.WriteFile(dictPath, []byte(dictContent), 0644)
	require.NoError(t, err)

	dict, err := LoadDict(dictPath)
	require.NoError(t, err)
	require.NotNil(t, dict)

	// Entries should be trimmed
	assert.Contains(t, dict.Entries, "127.0.0.1")
	assert.Contains(t, dict.Entries, "192.168.1.1")
	assert.Contains(t, dict.Entries, "10.0.0.1")
}

// TestLoadDictInlineComments tests handling of inline comments.
func TestLoadDictInlineComments(t *testing.T) {
	tmpDir := t.TempDir()
	dictPath := filepath.Join(tmpDir, "inline.txt")

	// Note: Inline comments are NOT supported - the entire line should be treated as content
	dictContent := `127.0.0.1
192.168.1.1 # This is a comment
`

	err := os.WriteFile(dictPath, []byte(dictContent), 0644)
	require.NoError(t, err)

	dict, err := LoadDict(dictPath)
	require.NoError(t, err)

	// With inline comment support
	assert.Contains(t, dict.Entries, "127.0.0.1")
	assert.Contains(t, dict.Entries, "192.168.1.1") // Inline comment should be stripped
}

// TestNewDict tests the NewDict constructor.
func TestNewDict(t *testing.T) {
	dict := NewDict()
	require.NotNil(t, dict)
	require.NotNil(t, dict.Entries)
	assert.Len(t, dict.Entries, 0)
}

// TestDictMerge tests merging two dictionaries.
func TestDictMerge(t *testing.T) {
	dict1 := &Dict{
		Entries: map[string]struct{}{
			"127.0.0.1":   {},
			"192.168.1.1": {},
		},
	}

	dict2 := &Dict{
		Entries: map[string]struct{}{
			"192.168.1.1": {}, // Duplicate
			"10.0.0.1":    {},
		},
	}

	dict1.Merge(dict2)

	assert.Len(t, dict1.Entries, 3)
	assert.Contains(t, dict1.Entries, "127.0.0.1")
	assert.Contains(t, dict1.Entries, "192.168.1.1")
	assert.Contains(t, dict1.Entries, "10.0.0.1")
}

// TestLoadDictManager tests the DictManager.
func TestLoadDictManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test dictionary files
	uidWhitelist := filepath.Join(tmpDir, "uid_whitelist.txt")
	ipBlacklist := filepath.Join(tmpDir, "ip_blacklist.txt")

	err := os.WriteFile(uidWhitelist, []byte("user1\nuser2\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(ipBlacklist, []byte("10.0.0.1\n10.0.0.2\n"), 0644)
	require.NoError(t, err)

	// Create dict configuration
	dicts := map[string]string{
		"uid_whitelist": uidWhitelist,
		"ip_blacklist":  ipBlacklist,
	}

	// Load dictionaries
	manager, err := LoadDicts(dicts)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Get uid whitelist
	dict, ok := manager.Get("uid_whitelist")
	assert.True(t, ok)
	assert.True(t, dict.Contains("user1"))
	assert.True(t, dict.Contains("user2"))

	// Get ip blacklist
	dict, ok = manager.Get("ip_blacklist")
	assert.True(t, ok)
	assert.True(t, dict.Contains("10.0.0.1"))
	assert.True(t, dict.Contains("10.0.0.2"))

	// Get non-existent dict
	_, ok = manager.Get("unknown")
	assert.False(t, ok)
}

// TestDictManagerReload tests reloading dictionaries.
func TestDictManagerReload(t *testing.T) {
	tmpDir := t.TempDir()
	dictPath := filepath.Join(tmpDir, "test.txt")

	// Initial content
	err := os.WriteFile(dictPath, []byte("user1\nuser2\n"), 0644)
	require.NoError(t, err)

	dicts := map[string]string{"test": dictPath}
	manager, err := LoadDicts(dicts)
	require.NoError(t, err)

	dict, _ := manager.Get("test")
	assert.True(t, dict.Contains("user1"))
	assert.True(t, dict.Contains("user2"))
	assert.False(t, dict.Contains("user3"))

	// Update file content
	err = os.WriteFile(dictPath, []byte("user3\nuser4\n"), 0644)
	require.NoError(t, err)

	// Reload
	err = manager.Reload()
	require.NoError(t, err)

	dict, _ = manager.Get("test")
	assert.False(t, dict.Contains("user1"))
	assert.False(t, dict.Contains("user2"))
	assert.True(t, dict.Contains("user3"))
	assert.True(t, dict.Contains("user4"))
}
