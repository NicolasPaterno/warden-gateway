package http_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	auth "github.com/NicolasPaterno/warden-auth"
	"github.com/NicolasPaterno/warden-auth/authn"
	warden "github.com/NicolasPaterno/warden-gateway"
	httptransport "github.com/NicolasPaterno/warden-gateway/internal/http"
	"github.com/NicolasPaterno/warden-gateway/internal/hub"
	"github.com/NicolasPaterno/warden-gateway/internal/metrics"
	"github.com/NicolasPaterno/warden-gateway/internal/service"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

func TestMain(m *testing.M) {
	_ = metrics.Register()
	os.Exit(m.Run())
}

type mockReadingRepository struct{}

func (m *mockReadingRepository) Save(_ context.Context, _ warden.SensorReading) error {
	return nil
}

func (m *mockReadingRepository) GetByRoomAndType(_ context.Context, _, _ string, _ warden.SensorType, _, _ time.Time) ([]warden.SensorReading, error) {
	return nil, nil
}

type mockBroker struct{}

func (m *mockBroker) Publish(_ context.Context, _ warden.SensorReading) error { return nil }
func (m *mockBroker) Close() error                                            { return nil }
func (m *mockBroker) Ping() error                                             { return nil }

const (
	testKID      = "gateway-test-kid"
	testIssuer   = "warden-auth"
	testAudience = "warden-gateway"
)

func testVerifier(t *testing.T) (*authn.Verifier, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       &key.PublicKey,
		KeyID:     testKID,
		Algorithm: "RS256",
		Use:       "sig",
	}}}
	body, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return authn.New(srv.URL, testIssuer, testAudience), key
}

func signToken(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	now := time.Now()
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			Issuer:    testIssuer,
			Audience:  jwt.ClaimStrings{testAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
		Scope:  "access",
		Tenant: "tenant-a",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func newTestRouter(h *hub.Hub, verifier *authn.Verifier) http.Handler {
	wsHandler := httptransport.NewWsHandler(h, verifier)
	svc := service.NewReadingService(&mockReadingRepository{})
	readingsHandler := httptransport.NewReadingsHandler(svc)
	healthHandler := httptransport.NewHealthHandler(nil, &mockBroker{})
	return httptransport.NewRouter(wsHandler, readingsHandler, healthHandler, verifier)
}

func TestReadingsRequiresAuth(t *testing.T) {
	verifier, key := testVerifier(t)
	router := newTestRouter(hub.NewHub(), verifier)

	t.Run("no token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/readings?tenant=tenant-a&room=bedroom&type=temperature", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("valid token gets past auth to the handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/readings?tenant=tenant-a", nil)
		req.Header.Set("Authorization", "Bearer "+signToken(t, key))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusUnauthorized {
			t.Fatalf("status = 401, expected to pass auth (missing room should be 400)")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (room required)", rec.Code)
		}
	})
}

func TestNewRouter_WSRouteExists(t *testing.T) {
	verifier, key := testVerifier(t)
	router := newTestRouter(hub.NewHub(), verifier)

	request := httptest.NewRequest("GET", "/ws?tenant=tenant-a&token="+signToken(t, key), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code == http.StatusNotFound {
		t.Errorf("ws route not found")
	}
}

func TestWsHandler_RequiresToken(t *testing.T) {
	verifier, _ := testVerifier(t)
	router := newTestRouter(hub.NewHub(), verifier)

	request := httptest.NewRequest("GET", "/ws?tenant=tenant-a", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Errorf("ws without token should be 401, got %d", response.Code)
	}
}

func TestWsHandler_UpgradeFailsWithoutWSHeaders(t *testing.T) {
	verifier, key := testVerifier(t)
	router := newTestRouter(hub.NewHub(), verifier)

	request := httptest.NewRequest("GET", "/ws?tenant=tenant-a&token="+signToken(t, key), nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Errorf("ws handler should fail with 400, got %d", response.Code)
	}
}

func TestWsHandler_UpgradeSucceeds(t *testing.T) {
	verifier, key := testVerifier(t)
	h := hub.NewHub()
	router := newTestRouter(h, verifier)

	response := httptest.NewServer(router)
	defer response.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.Run(ctx)
	url := strings.Replace(response.URL, "http", "ws", 1) + "/ws?tenant=tenant-a&token=" + signToken(t, key)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
}
