package httpapi

import "testing"

func TestMetadataHelpers(t *testing.T) {
	path := PathParam("id", int64(0))
	if path.Name != "id" || path.In != ParamInPath || !path.Required {
		t.Fatalf("unexpected path helper output: %#v", path)
	}

	query := QueryParam("limit", int(0))
	if query.Name != "limit" || query.In != ParamInQuery || query.Required {
		t.Fatalf("unexpected query helper output: %#v", query)
	}

	header := HeaderParam("X-Trace", "")
	if header.In != ParamInHeader {
		t.Fatalf("unexpected header helper output: %#v", header)
	}

	cookie := CookieParam("sid", "")
	if cookie.In != ParamInCookie {
		t.Fatalf("unexpected cookie helper output: %#v", cookie)
	}

	jsonBody := JSONBody(struct{ Name string }{})
	if jsonBody == nil || !jsonBody.Required || len(jsonBody.Content) != 1 || jsonBody.Content[0].MediaType != MediaTypeJSON {
		t.Fatalf("unexpected JSON body helper output: %#v", jsonBody)
	}

	formBody := FormBody(struct{ Name string }{})
	if formBody == nil || !formBody.Required || len(formBody.Content) != 1 || formBody.Content[0].MediaType != MediaTypeFormURLEncoded {
		t.Fatalf("unexpected form body helper output: %#v", formBody)
	}
}

func TestSecurityHelpers(t *testing.T) {
	apiKey := GuardSpec{Name: "ApiKeyAuth", In: "header", Param: "X-API-Key"}
	token := GuardSpec{Name: "TokenAuth", In: "header", Param: "Authorization"}

	any := SecurityAny(apiKey, GuardSpec{}, token)
	if len(any.Alternatives) != 2 {
		t.Fatalf("SecurityAny alternatives = %d, want 2", len(any.Alternatives))
	}
	if len(any.Alternatives[0].Guards) != 1 || any.Alternatives[0].Guards[0].Name != "ApiKeyAuth" {
		t.Fatalf("unexpected first SecurityAny alternative: %#v", any.Alternatives[0])
	}

	all := SecurityAll(apiKey, GuardSpec{}, token)
	if len(all.Alternatives) != 1 || len(all.Alternatives[0].Guards) != 2 {
		t.Fatalf("unexpected SecurityAll output: %#v", all)
	}

	empty := SecurityAll(GuardSpec{})
	if len(empty.Alternatives) != 0 {
		t.Fatalf("empty SecurityAll should have no alternatives: %#v", empty)
	}
}

func TestSecuritySpecFromGuardsCombinesANDWithOR(t *testing.T) {
	apiKey := testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"}
	token := testGuard{name: "TokenAuth", in: "header", param: "Authorization"}
	session := testGuard{name: "SessionAuth", in: "cookie", param: "sid"}

	spec := securitySpecFromGuards([]Guard{session, AuthAny(apiKey, token)})
	if len(spec.Alternatives) != 2 {
		t.Fatalf("alternatives = %d, want 2", len(spec.Alternatives))
	}
	for _, alt := range spec.Alternatives {
		if len(alt.Guards) != 2 {
			t.Fatalf("combined alternative should include session plus one OR guard: %#v", alt)
		}
		if alt.Guards[0].Name != "SessionAuth" {
			t.Fatalf("first guard = %q, want SessionAuth", alt.Guards[0].Name)
		}
	}
}

func TestFlattenSecuritySpecDedupesGuards(t *testing.T) {
	apiKey := GuardSpec{Name: "ApiKeyAuth", In: "header", Param: "X-API-Key"}
	spec := SecuritySpec{Alternatives: []SecurityRequirement{
		{Guards: []GuardSpec{apiKey}},
		{Guards: []GuardSpec{apiKey}},
	}}

	flat := flattenSecuritySpec(spec)
	if len(flat) != 1 {
		t.Fatalf("flat guards = %d, want 1", len(flat))
	}
	if flat[0].Name != "ApiKeyAuth" {
		t.Fatalf("unexpected flat guard: %#v", flat[0])
	}
}
