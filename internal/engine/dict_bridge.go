// Copyright 2026 heiyeluren. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
//
// Koala - High Performance Rate Limiting System
// Author:  heiyeluren
// Date:    2026-02-01

package engine

import (
	"koala/internal/config"
	"koala/internal/engine/matcher"
)

// SyncDictsToMatcher 将 DictManager 中的字典同步到 matcher 的 DictMatcher。
// 这个函数是连接 config.DictManager 和 matcher.DictMatcher 的桥梁。
func SyncDictsToMatcher(dicts *config.DictManager) {
	if dicts == nil {
		return
	}
	for _, name := range dicts.List() {
		dict, ok := dicts.Get(name)
		if ok {
			entries := dict.List()
			dictMap := make(map[string]bool, len(entries))
			for _, entry := range entries {
				dictMap[entry] = true
			}
			matcher.RegisterDict(name, dictMap)
		}
	}
}
