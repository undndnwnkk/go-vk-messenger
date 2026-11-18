package restapi

import (
	"bufio"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/undndnwnkk/go-vk-messenger/internal/realtime"
	"github.com/undndnwnkk/go-vk-messenger/internal/service"
)

// TestOpenAPICoversRouter keeps backend/api/openapi.yaml in sync with the router:
// every registered route must be documented and every documented route must exist.
func TestOpenAPICoversRouter(t *testing.T) {
	handler := NewHandler(service.UserService{}, nil, fakeHealthChecker{}, service.ChatService{}, service.MessageService{}, realtime.NewHub())
	routes, ok := handler.(chi.Routes)
	if !ok {
		t.Fatalf("NewHandler returned %T, want chi.Routes", handler)
	}

	fromRouter := map[string]bool{}
	err := chi.Walk(routes, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// chi registers "/chats" and "/chats/" (and "/members/") as separate routes;
		// the spec documents the canonical form without the trailing slash.
		if len(route) > 1 {
			route = strings.TrimSuffix(route, "/")
		}
		fromRouter[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatalf("walk router: %v", err)
	}

	fromSpec := readOpenAPIOperations(t, filepath.Join("..", "..", "..", "api", "openapi.yaml"))

	for op := range fromRouter {
		if !fromSpec[op] {
			t.Errorf("route %q is not documented in openapi.yaml", op)
		}
	}
	for op := range fromSpec {
		if !fromRouter[op] {
			t.Errorf("openapi.yaml documents %q, which the router does not serve", op)
		}
	}
	if t.Failed() {
		t.Logf("router operations: %v", sortedKeys(fromRouter))
	}
}

var (
	specPathLine   = regexp.MustCompile(`^  (/[^:]*):\s*$`)
	specMethodLine = regexp.MustCompile(`^    (get|put|post|delete|patch|head|options):\s*$`)
)

// readOpenAPIOperations returns "METHOD /path" for every operation under `paths:`.
// It relies on the file's two-space indentation instead of a YAML dependency.
func readOpenAPIOperations(t *testing.T, path string) map[string]bool {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open spec: %v", err)
	}
	defer f.Close()

	ops := map[string]bool{}
	inPaths, current := false, ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		switch {
		case line == "paths:":
			inPaths = true
		case inPaths && line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "#"):
			inPaths = false
		case inPaths:
			if m := specPathLine.FindStringSubmatch(line); m != nil {
				current = m[1]
			} else if m := specMethodLine.FindStringSubmatch(line); m != nil && current != "" {
				ops[strings.ToUpper(m[1])+" "+current] = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read spec: %v", err)
	}
	if len(ops) == 0 {
		t.Fatal("no operations found in spec")
	}
	return ops
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
