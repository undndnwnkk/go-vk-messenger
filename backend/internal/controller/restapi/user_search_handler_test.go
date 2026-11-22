package restapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

func TestSearchUsersHandler(t *testing.T) {
	repo := newFakeUserRepo()
	handler, jwtService := newTestHandler(t, repo)
	for id, name := range map[string]string{"u1": "alice", "u2": "alina", "u3": "bob"} {
		repo.usersByID[id] = model.User{ID: id, Username: name}
	}
	token, err := jwtService.GenerateToken("u1")
	if err != nil {
		t.Fatal(err)
	}
	get := func(target string, authorized bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		if authorized {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	rec := get("/api/v1/users/search?q=al", true)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var users []model.UserSummary
	if err := json.NewDecoder(rec.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0] != (model.UserSummary{ID: "u2", Username: "alina"}) {
		t.Fatalf("users = %+v, want only alina", users)
	}
	if call := repo.searchCalls[0]; call.userID != "u1" || call.limit != 20 {
		t.Fatalf("search call = %+v, want caller u1 and default limit 20", call)
	}

	rec = get("/api/v1/users/search", true)
	if rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Fatalf("contacts response = %d %q, want 200 []", rec.Code, rec.Body.String())
	}

	if rec = get("/api/v1/users/search?q=a&limit=7", true); rec.Code != http.StatusOK || repo.searchCalls[1].limit != 7 {
		t.Fatalf("limit=7: status %d, call %+v", rec.Code, repo.searchCalls[1])
	}

	for _, limit := range []string{"abc", "0"} {
		rec = get("/api/v1/users/search?q=a&limit="+limit, true)
		if rec.Code != http.StatusBadRequest || !jsonHasCode(rec, "invalid_limit") {
			t.Fatalf("limit=%s: %d %s", limit, rec.Code, rec.Body.String())
		}
	}

	if rec = get("/api/v1/users/search?q=al", false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want 401", rec.Code)
	}
}

func jsonHasCode(rec *httptest.ResponseRecorder, code string) bool {
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	return json.Unmarshal(rec.Body.Bytes(), &body) == nil && body.Error.Code == code
}
