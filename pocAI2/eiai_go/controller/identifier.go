package controller

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// verifikasi dan benerin parameter
func resolveIdentifiers(toolCalls []ToolCall, mcpTools []mcp.Tool, accumulatedResults []ToolExecutionResult, resolverMap map[string]string, query string) []ToolCall {
	if len(resolverMap) == 0 {
		resolverMap = map[string]string{
			"id":       "get_assets",
			"asset_id": "get_assets",
		}
	}

	var resolvedCalls []ToolCall
	injectedResolvers := make(map[string]bool)

	for _, call := range toolCalls {
		fnName := strings.Trim(call.Name, "_")
		args := make(map[string]any)
		for k, v := range call.Arguments {
			args[k] = v
		}

		// Direct operations: never hijack update_asset, delete_asset, add_asset, create_borrow_transaction, or no_tools with get_assets
		if fnName == "update_asset" || fnName == "delete_asset" || fnName == "add_asset" || fnName == "create_borrow_transaction" || fnName == "create_transaction" || fnName == "get_borrow_transactions" || fnName == "create_schedule" || fnName == "get_schedules" || fnName == "no_tools" || fnName == "no_tool" {
		if fnName == "update_asset" || fnName == "delete_asset" || fnName == "create_borrow_transaction" || fnName == "create_transaction" {
			idVal := strings.TrimSpace(fmt.Sprintf("%v", args["asset_id"]))
			if idVal == "" || idVal == "<nil>" {
				idVal = strings.TrimSpace(fmt.Sprintf("%v", args["id"]))
			}
			records := findResolverRecords("get_assets", accumulatedResults)
			// Resolve comma/semicolon-separated asset_id list + single asset_id
			if len(records) > 0 && idVal != "" && idVal != "<nil>" {
				parts := strings.FieldsFunc(idVal, func(r rune) bool { return r == ',' || r == ';' })
				resolvedParts := make([]string, 0, len(parts))
				changed := false
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					if strings.HasPrefix(strings.ToUpper(p), "AST-") {
						resolvedParts = append(resolvedParts, p)
						continue
					}
					if realID := matchAssetID(p, args, query, records); realID != "" {
						resolvedParts = append(resolvedParts, realID)
						changed = true
						log.Printf("[resolveIdentifiers] Auto-resolved %s asset_id part %q to %v", fnName, p, realID)
					} else {
						resolvedParts = append(resolvedParts, p)
					}
				}
				if changed && len(resolvedParts) > 0 {
					joined := strings.Join(resolvedParts, ",")
					args["asset_id"] = joined
					args["id"] = joined
				} else if !strings.HasPrefix(strings.ToUpper(idVal), "AST-") && !strings.ContainsAny(idVal, ",;") {
					// Single non-ID value: try resolving from accumulated records
					if realID := matchAssetID(idVal, args, query, records); realID != "" {
						args["asset_id"] = realID
						args["id"] = realID
						log.Printf("[resolveIdentifiers] Auto-resolved %s asset_id to %v", fnName, realID)
					}
				}
			}
			// Resolve items[].asset_id / items[].asset_name for multi-asset borrow (strict: keep distinct IDs)
			if (fnName == "create_borrow_transaction" || fnName == "create_transaction") && len(records) > 0 {
				args["items"] = resolveBorrowItems(args["items"], args, query, records)
			}
		}
			resolvedCalls = append(resolvedCalls, ToolCall{Name: fnName, Arguments: args})
			continue
		}

		needsResolver := ""
		dropped := false

		for paramName, val := range args {
			if !strings.HasSuffix(paramName, "_id") {
				continue
			}
			resolverTool, ok := resolverMap[paramName]
			if !ok {
				continue
			}
			supplied := strings.TrimSpace(fmt.Sprintf("%v", val))
			if supplied == "" {
				continue
			}

			if !resolverToolAvailable(resolverTool, mcpTools) {
				log.Printf("[resolveIdentifiers] Cannot verify %s=%q: resolver '%s' not permitted. Dropping call '%s'.", paramName, supplied, resolverTool, fnName)
				dropped = true
				break
			}

			records := findResolverRecords(resolverTool, accumulatedResults)
			if len(records) == 0 {
				needsResolver = resolverTool
				break
			}

			idField := inferIdField(records, paramName)
			matched := false

			for _, r := range records {
				if fmt.Sprintf("%v", r[idField]) == supplied {
					matched = true
					break
				}
			}

			if matched {
				continue // Already a valid ID
			}

			// Try matching against any property
			var matchedRecord map[string]any
			for _, r := range records {
				for _, fieldVal := range r {
					valStr := strings.TrimSpace(fmt.Sprintf("%v", fieldVal))
					if valStr != "" && (strings.EqualFold(valStr, supplied) || (len(supplied) >= 4 && strings.Contains(strings.ToLower(valStr), strings.ToLower(supplied))) || isFuzzyMatch(supplied, valStr)) {
						matchedRecord = r
						break
					}
				}
				if matchedRecord != nil {
					break
				}
			}

			// Fallback: match query string against record values
			if matchedRecord == nil && query != "" {
				for _, r := range records {
					for _, fieldVal := range r {
						valStr := strings.TrimSpace(fmt.Sprintf("%v", fieldVal))
						if len(valStr) >= 4 && (strings.Contains(strings.ToLower(query), strings.ToLower(valStr)) || isFuzzyMatch(valStr, query)) {
							matchedRecord = r
							break
						}
					}
					if matchedRecord != nil {
						break
					}
				}
			}

			if matchedRecord != nil {
				if realID, ok := matchedRecord[idField]; ok {
					args[paramName] = realID
					log.Printf("[resolveIdentifiers] Auto-corrected %s: %q -> %v via '%s'", paramName, supplied, realID, resolverTool)
				}
			} else {
				needsResolver = resolverTool
				break
			}
		}

		if dropped {
			continue
		}

		if needsResolver != "" {
			if injectedResolvers[needsResolver] {
				continue
			}
			if !resolverNeedsNoArgs(needsResolver, mcpTools) {
				resolvedCalls = append(resolvedCalls, ToolCall{Name: fnName, Arguments: args})
				continue
			}
			injectedResolvers[needsResolver] = true
			resolvedCalls = append(resolvedCalls, ToolCall{Name: needsResolver, Arguments: map[string]any{}})
			continue
		}

		resolvedCalls = append(resolvedCalls, ToolCall{Name: fnName, Arguments: args})
	}

	return resolvedCalls
}

// matchAssetID fuzzy-matches a user-supplied name/keyword to a real AST- ID from get_assets records.
func matchAssetID(supplied string, args map[string]any, query string, records []map[string]any) string {
	supplied = strings.TrimSpace(supplied)
	if supplied == "" || supplied == "<nil>" {
		if n, ok := args["asset_name"].(string); ok && strings.TrimSpace(n) != "" {
			supplied = strings.TrimSpace(n)
		} else if n, ok := args["name"].(string); ok && strings.TrimSpace(n) != "" {
			supplied = strings.TrimSpace(n)
		} else {
			supplied = query
		}
	}
	if supplied == "" {
		return ""
	}
	for _, r := range records {
		for _, fv := range r {
			fvStr := strings.TrimSpace(fmt.Sprintf("%v", fv))
			if fvStr == "" {
				continue
			}
			if strings.EqualFold(fvStr, supplied) || (len(supplied) >= 3 && strings.Contains(strings.ToLower(fvStr), strings.ToLower(supplied))) || isFuzzyMatch(supplied, fvStr) {
				if realID, ok := r["asset_id"].(string); ok && strings.TrimSpace(realID) != "" {
					return strings.TrimSpace(realID)
				}
				if realID, ok := r["asset_id"]; ok && realID != nil {
					if s := strings.TrimSpace(fmt.Sprintf("%v", realID)); s != "" {
						return s
					}
				}
				if realID, ok := r["id"].(string); ok && strings.TrimSpace(realID) != "" {
					return strings.TrimSpace(realID)
				}
				if realID, ok := r["id"]; ok && realID != nil {
					if s := strings.TrimSpace(fmt.Sprintf("%v", realID)); s != "" {
						return s
					}
				}
			}
		}
	}
	return ""
}

// resolveBorrowItems resolves items[].asset_id / asset_name entries against get_assets records.
// Keeps insertion order, preserves per-item duration_days/quantity fields, and never duplicates IDs here
// (strict quantity enforcement happens in mcp_server).
func resolveBorrowItems(rawItems any, args map[string]any, query string, records []map[string]any) any {
	list, ok := rawItems.([]any)
	if !ok || len(list) == 0 {
		return rawItems
	}
	out := make([]any, 0, len(list))
	for _, elem := range list {
		m, ok := elem.(map[string]any)
		if !ok {
			if s, ok := elem.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" && !strings.HasPrefix(strings.ToUpper(s), "AST-") {
					if realID := matchAssetID(s, args, query, records); realID != "" {
						s = realID
					}
				}
				out = append(out, s)
				continue
			}
			out = append(out, elem)
			continue
		}
		cp := make(map[string]any, len(m))
		for k, v := range m {
			cp[k] = v
		}
		candidate := ""
		for _, k := range []string{"asset_id", "id", "asset", "code"} {
			if v, ok := cp[k].(string); ok && strings.TrimSpace(v) != "" {
				candidate = strings.TrimSpace(v)
				break
			}
		}
		if candidate == "" {
			for _, k := range []string{"asset_name", "name", "nama", "title"} {
				if v, ok := cp[k].(string); ok && strings.TrimSpace(v) != "" {
					candidate = strings.TrimSpace(v)
					break
				}
			}
		}
		if candidate != "" && !strings.HasPrefix(strings.ToUpper(candidate), "AST-") {
			if realID := matchAssetID(candidate, cp, query, records); realID != "" {
				cp["asset_id"] = realID
				cp["id"] = realID
				log.Printf("[resolveIdentifiers] Auto-resolved borrow items entry %q to %v", candidate, realID)
			}
		}
		out = append(out, cp)
	}
	return out
}

// cek llm isi param dengan bener apa kagak
func resolverToolAvailable(resolverToolName string, mcpTools []mcp.Tool) bool {
	for _, t := range mcpTools {
		if t.Name == resolverToolName {
			return true
		}
	}
	return false
}

// narik hasil dari kumpulan result.
func findResolverRecords(resolverTool string, accumulatedResults []ToolExecutionResult) []map[string]any {
	var records []map[string]any
	for _, r := range accumulatedResults {
		if r.Tool == resolverTool {
			records = append(records, extractRecords(r.Result)...)
		}
	}
	return records
}

// pembuka wrapper status dan company karena yang bikin backend radak geblek
func extractRecords(result any) []map[string]any {
	var records []map[string]any

	switch v := result.(type) {
	case []map[string]any:
		return v
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				records = append(records, m)
			}
		}
		return records
	case map[string]any:
		for _, val := range v {
			if list, ok := val.([]any); ok && len(list) > 0 {
				if _, isMap := list[0].(map[string]any); isMap {
					for _, item := range list {
						if m, ok := item.(map[string]any); ok {
							records = append(records, m)
						}
					}
					return records
				}
			} else if list, ok := val.([]map[string]any); ok && len(list) > 0 {
				return list
			}
		}
	}

	return records
}

// cari kolom mana yang id
func inferIdField(records []map[string]any, paramName string) string {
	if len(records) == 0 {
		return "assetId"
	}

	sample := records[0]
	candidates := []string{
		paramName,
		"assetId",
		"asset_id",
		"assetid",
		"id",
		"_id",
		strings.ReplaceAll(paramName, "_id", "Id"),
	}

	for _, candidate := range candidates {
		if _, ok := sample[candidate]; ok {
			return candidate
		}
	}

	for k := range sample {
		if strings.HasSuffix(k, "_id") || strings.HasSuffix(k, "Id") {
			return k
		}
	}

	return "assetId"
}


func resolverNeedsNoArgs(resolverToolName string, mcpTools []mcp.Tool) bool {
	for _, t := range mcpTools {
		if t.Name == resolverToolName {
			var required []string
			for _, r := range t.InputSchema.Required {
				if !slices.Contains(INTERNAL_PARAM_NAMES, r) {
					required = append(required, r)
				}
			}
			return len(required) == 0
		}
	}
	return true
}

// cari kolom id
func inferResolverMap(mcpTools []mcp.Tool) map[string]string {
	resolverMap := map[string]string{
		"id":       "get_assets",
		"asset_id": "get_assets",
	}

	if len(mcpTools) == 0 {
		return resolverMap
	}

	for _, t := range mcpTools {
		for p := range t.InputSchema.Properties {
			if strings.HasSuffix(p, "_id") && !slices.Contains(INTERNAL_PARAM_NAMES, p) {
				entity := strings.TrimSuffix(p, "_id")
				for _, otherTool := range mcpTools {
					if strings.Contains(strings.ToLower(otherTool.Name), entity) {
						resolverMap[p] = otherTool.Name
						break
					}
				}
			}
		}
	}

	return resolverMap
}

// isFuzzyMatch checks if two strings match approximately with typo tolerance
func isFuzzyMatch(target, candidate string) bool {
	t := strings.TrimSpace(strings.ToLower(target))
	c := strings.TrimSpace(strings.ToLower(candidate))
	if t == "" || c == "" {
		return false
	}
	if t == c || strings.Contains(t, c) || strings.Contains(c, t) {
		return true
	}

	// Compare individual tokens / words
	tWords := strings.Fields(t)
	cWords := strings.Fields(c)
	for _, tw := range tWords {
		if len(tw) < 3 {
			continue
		}
		for _, cw := range cWords {
			if len(cw) < 3 {
				continue
			}
			if tw == cw {
				return true
			}
			dist := levenshteinDistance(tw, cw)
			if (len(tw) <= 4 && dist <= 1) || (len(tw) > 4 && dist <= 2) {
				return true
			}
		}
	}

	if len(t) >= 4 && len(c) >= 4 {
		dist := levenshteinDistance(t, c)
		if (len(t) <= 6 && dist <= 1) || (len(t) > 6 && dist <= 2) {
			return true
		}
	}
	return false
}

func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)
	if n1 == 0 {
		return n2
	}
	if n2 == 0 {
		return n1
	}
	dp := make([]int, n2+1)
	for j := 0; j <= n2; j++ {
		dp[j] = j
	}
	for i := 1; i <= n1; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= n2; j++ {
			temp := dp[j]
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[j] = min3(dp[j]+1, dp[j-1]+1, prev+cost)
			prev = temp
		}
	}
	return dp[n2]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

