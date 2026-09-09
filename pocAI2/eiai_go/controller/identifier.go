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

		// Direct operations: never hijack update_asset, delete_asset, add_asset, or no_tools with get_assets
		if fnName == "update_asset" || fnName == "delete_asset" || fnName == "add_asset" || fnName == "no_tools" || fnName == "no_tool" {
			if fnName == "update_asset" || fnName == "delete_asset" {
				idVal := strings.TrimSpace(fmt.Sprintf("%v", args["id"]))
				if idVal == "" || idVal == "<nil>" {
					idVal = strings.TrimSpace(fmt.Sprintf("%v", args["asset_id"]))
				}
				// If id is not formatted as AST- ID, try resolving from accumulated records
				if !strings.HasPrefix(strings.ToUpper(idVal), "AST-") {
					records := findResolverRecords("get_assets", accumulatedResults)
					if len(records) > 0 {
						idField := inferIdField(records, "id")
						searchTarget := idVal
						if searchTarget == "" || searchTarget == "<nil>" {
							if n, ok := args["name"].(string); ok && n != "" {
								searchTarget = n
							} else {
								searchTarget = query
							}
						}
						for _, r := range records {
							for _, fv := range r {
								fvStr := strings.TrimSpace(fmt.Sprintf("%v", fv))
								if fvStr != "" && (strings.EqualFold(fvStr, searchTarget) || (len(searchTarget) >= 3 && strings.Contains(strings.ToLower(fvStr), strings.ToLower(searchTarget)))) {
									if realID, ok := r[idField]; ok {
										args["id"] = realID
										log.Printf("[resolveIdentifiers] Auto-resolved %s id to %v", fnName, realID)
										break
									}
								}
							}
						}
					}
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
					if valStr != "" && (strings.EqualFold(valStr, supplied) || (len(supplied) >= 4 && strings.Contains(strings.ToLower(valStr), strings.ToLower(supplied)))) {
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
						if len(valStr) >= 4 && strings.Contains(strings.ToLower(query), strings.ToLower(valStr)) {
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
