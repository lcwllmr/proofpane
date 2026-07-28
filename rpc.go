package main

import (
	"encoding/json"
	"html"
	"strings"
)

// FetchTacticState queries the Lean LSP for the plainGoal and plainTermGoal at the given cursor position,
// parses the JSON-RPC responses, cleans up Markdown backticks, and returns HTML-escaped blocks.
func FetchTacticState(px *Proxy, uri string, line, col int) (json.RawMessage, string, json.RawMessage, string) {
	params := map[string]interface{}{
		"textDocument": map[string]string{"uri": uri},
		"position": map[string]int{
			"line":      line,
			"character": col,
		},
	}

	rawGoal, htmlGoal := fetchGoal(px, "$/lean/plainGoal", params, "rendered")
	rawTerm, htmlTerm := fetchGoal(px, "$/lean/plainTermGoal", params, "expected")

	if htmlGoal == "" {
		htmlGoal = "No goals"
	} else {
		htmlGoal = "<pre>" + htmlGoal + "</pre>"
	}

	if htmlTerm == "" {
		htmlTerm = "No term goals"
	} else {
		htmlTerm = "<pre>" + htmlTerm + "</pre>"
	}

	return rawGoal, htmlGoal, rawTerm, htmlTerm
}

func fetchGoal(px *Proxy, method string, params interface{}, resultField string) (json.RawMessage, string) {
	res, err := px.Inject(method, params)
	if err != nil || res == nil {
		return nil, ""
	}
	if res.Error != nil {
		return res.Error, `<span style="color: #ef4444;">` + html.EscapeString(string(res.Error)) + `</span>`
	}
	if res.Result == nil || string(res.Result) == "null" {
		return nil, ""
	}

	// Dynamic parse based on result field (rendered vs expected)
	var raw map[string]interface{}
	if err := json.Unmarshal(res.Result, &raw); err != nil {
		return nil, ""
	}

	if val, ok := raw[resultField].(string); ok {
		clean := strings.TrimSpace(val)
		clean = strings.TrimPrefix(clean, "```lean\n")
		clean = strings.TrimPrefix(clean, "```lean")
		clean = strings.TrimSuffix(clean, "\n```")
		clean = strings.TrimSuffix(clean, "```")
		return res.Result, html.EscapeString(clean)
	}

	return res.Result, ""
}
