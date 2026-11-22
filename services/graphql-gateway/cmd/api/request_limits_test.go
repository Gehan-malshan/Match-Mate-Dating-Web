package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func graphRequest(query string) *http.Request {
	body, _ := json.Marshal(map[string]string{"query": query})
	return httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(string(body)))
}

func TestValidateGraphQLRequestRejectsDeepSelections(t *testing.T) {
	request := graphRequest(`query { me { profile { interests } } }`)
	if err := validateGraphQLRequest(request, 1024, 2); err != errQueryTooDeep {
		t.Fatalf("expected depth rejection, got %v", err)
	}
}

func TestValidateGraphQLRequestAllowsFragmentsWithinDepth(t *testing.T) {
	request := graphRequest(`query { me { ...Member } } fragment Member on Me { account { id } }`)
	if err := validateGraphQLRequest(request, 1024, 3); err != nil {
		t.Fatalf("expected fragment query to pass, got %v", err)
	}
}

func TestValidateGraphQLRequestRejectsOversizedPayload(t *testing.T) {
	request := graphRequest(`query { events { items { description } } }`)
	if err := validateGraphQLRequest(request, 10, 12); err != errRequestTooLarge {
		t.Fatalf("expected size rejection, got %v", err)
	}
}
