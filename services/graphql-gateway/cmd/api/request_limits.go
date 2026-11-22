package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

var (
	errRequestTooLarge = errors.New("GRAPHQL_REQUEST_TOO_LARGE")
	errQueryTooDeep    = errors.New("GRAPHQL_QUERY_TOO_DEEP")
	errInvalidQuery    = errors.New("GRAPHQL_QUERY_INVALID")
)

// validateGraphQLRequest protects the gateway before gqlgen executes or
// resolves anything. Complexity is separately bounded by gqlgen's fixed limit.
func validateGraphQLRequest(r *http.Request, maxBytes int64, maxDepth int) error {
	if r.Method != http.MethodPost {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
	if err != nil {
		return errInvalidQuery
	}
	if int64(len(body)) > maxBytes {
		return errRequestTooLarge
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	var request struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(body, &request); err != nil || request.Query == "" {
		return nil // gqlgen returns its normal, standards-compliant validation error.
	}
	document, err := parser.ParseQuery(&ast.Source{Input: request.Query})
	if err != nil {
		return errInvalidQuery
	}
	fragments := document.Fragments
	for _, operation := range document.Operations {
		if selectionDepth(operation.SelectionSet, fragments, map[string]bool{}) > maxDepth {
			return errQueryTooDeep
		}
	}
	return nil
}

func selectionDepth(selections ast.SelectionSet, fragments ast.FragmentDefinitionList, visiting map[string]bool) int {
	maximum := 0
	for _, selection := range selections {
		depth := 0
		switch item := selection.(type) {
		case *ast.Field:
			depth = 1 + selectionDepth(item.SelectionSet, fragments, visiting)
		case *ast.InlineFragment:
			depth = selectionDepth(item.SelectionSet, fragments, visiting)
		case *ast.FragmentSpread:
			if !visiting[item.Name] {
				if fragment := fragments.ForName(item.Name); fragment != nil {
					visiting[item.Name] = true
					depth = selectionDepth(fragment.SelectionSet, fragments, visiting)
					delete(visiting, item.Name)
				}
			}
		}
		if depth > maximum {
			maximum = depth
		}
	}
	return maximum
}

func writeGraphQLRequestError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := errInvalidQuery.Error()
	if errors.Is(err, errRequestTooLarge) {
		status, code = http.StatusRequestEntityTooLarge, errRequestTooLarge.Error()
	} else if errors.Is(err, errQueryTooDeep) {
		code = errQueryTooDeep.Error()
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"errors": []map[string]any{{"message": "GraphQL request was rejected", "extensions": map[string]string{"code": code}}}})
}
