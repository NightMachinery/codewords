package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NightMachinery/codewords/internal/identity"
	"github.com/NightMachinery/codewords/internal/storage"
	"github.com/gorilla/websocket"
)

func TestIdentityRoomLifecycleREST(t *testing.T) {
	h := newTestHandler(t)

	boot := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "token-host", "displayName": "Host"}, http.StatusOK)
	hostID := boot["userId"].(string)
	if hostID == "" || boot["displayName"].(string) != "Host" {
		t.Fatalf("unexpected bootstrap response: %#v", boot)
	}

	saved := postJSON(t, h, "/api/identity/display-name", map[string]any{"authToken": "token-host", "displayName": "Captain"}, http.StatusOK)
	if saved["displayName"].(string) != "Captain" {
		t.Fatalf("expected saved display name, got %#v", saved)
	}

	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "token-host", "settings": map[string]any{"wordpackId": "english", "seed": 7, "blackCards": 1}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	if roomID == "" || !strings.Contains(roomResp["roomLink"].(string), roomID) {
		t.Fatalf("unexpected room response: %#v", roomResp)
	}

	room := getJSON(t, h, "/api/rooms/"+roomID+"?authToken=token-host", http.StatusOK)
	if room["viewer"].(map[string]any)["userId"].(string) != hostID {
		t.Fatalf("expected host viewer context, got %#v", room["viewer"])
	}

	join := postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "token-guest", "displayName": "Guest"}, http.StatusOK)
	if join["viewer"].(map[string]any)["userId"].(string) == hostID {
		t.Fatalf("guest should be distinct from host: %#v", join)
	}

	settings := postJSON(t, h, "/api/rooms/"+roomID+"/settings", map[string]any{"authToken": "token-host", "settings": map[string]any{"wordpackId": "english", "seed": 9, "blackCards": 2, "enforceClueGuessLimit": true}}, http.StatusOK)
	if settings["settings"].(map[string]any)["blackCards"].(float64) != 2 {
		t.Fatalf("expected updated settings, got %#v", settings)
	}

	link := postJSON(t, h, "/api/rooms/"+roomID+"/migrate-link", map[string]any{"authToken": "token-host"}, http.StatusOK)
	migrateID := link["migrateId"].(string)
	if migrateID == "" || !strings.Contains(link["migrateUrl"].(string), migrateID) || strings.Contains(link["migrateUrl"].(string), "token-host") {
		t.Fatalf("unexpected migrate response: %#v", link)
	}
	resolved := postJSON(t, h, "/api/rooms/"+roomID+"/migrate-bootstrap", map[string]any{"migrateId": migrateID}, http.StatusOK)
	if resolved["userId"].(string) != hostID {
		t.Fatalf("expected migrate to resolve host, got %#v", resolved)
	}

	packs := getJSON(t, h, "/api/wordpacks", http.StatusOK)
	if len(packs["wordpacks"].([]any)) == 0 {
		t.Fatalf("expected bundled wordpacks, got %#v", packs)
	}
	pictures := getJSON(t, h, "/api/pictures/catalog", http.StatusOK)
	if pictures["available"].(bool) {
		t.Fatalf("test catalog has no local pictures, got %#v", pictures)
	}
}

func TestRoomStartPersistsSnapshotAndRestoresOverWebSocket(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host", "settings": map[string]any{"wordpackId": "english", "seed": 11}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest", "displayName": "Guest"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/settings", map[string]any{"authToken": "host", "settings": map[string]any{"wordpackId": "english", "seed": 11}}, http.StatusOK)

	// Lobby commands over HTTP through start endpoint auto-seat the minimal two-team setup from persisted players.
	start := postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "host"}, http.StatusOK)
	if start["snapshot"].(map[string]any)["phase"].(string) != "active" {
		t.Fatalf("expected active snapshot, got %#v", start)
	}

	server := httptest.NewServer(h)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}
	if msg["type"] != "snapshot" || msg["snapshot"].(map[string]any)["phase"] != "active" {
		t.Fatalf("unexpected initial ws message: %#v", msg)
	}

	if err := conn.WriteJSON(map[string]any{"type": "ping"}); err != nil {
		t.Fatalf("write ping: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if msg["type"] != "pong" {
		t.Fatalf("expected pong, got %#v", msg)
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "codewords.sqlite"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ids := identity.NewService(db, identity.Options{HashKey: []byte("test-key"), MigrateIDBytes: 8})
	h, err := NewHandler(Options{Store: db, Identity: ids, WordpacksDir: filepath.Join("..", "..", "assets", "wordpacks"), BaseURL: "http://example.test"})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h
}

func postJSON(t *testing.T, h http.Handler, path string, body map[string]any, want int) map[string]any {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != want {
		t.Fatalf("POST %s expected %d, got %d: %s", path, want, res.Code, res.Body.String())
	}
	return decodeMap(t, res)
}

func getJSON(t *testing.T, h http.Handler, path string, want int) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != want {
		t.Fatalf("GET %s expected %d, got %d: %s", path, want, res.Code, res.Body.String())
	}
	return decodeMap(t, res)
}

func decodeMap(t *testing.T, res *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %v", res.Body.String(), err)
	}
	return out
}

func TestWebSocketStartGameAndChatMessage(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-ws", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-ws", "settings": map[string]any{"wordpackId": "english", "seed": 12}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest-ws", "displayName": "Guest"}, http.StatusOK)

	server := httptest.NewServer(h)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}

	if err := conn.WriteJSON(map[string]any{"type": "startGame"}); err != nil {
		t.Fatalf("write startGame: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read startGame snapshot: %v", err)
	}
	if msg["type"] != "snapshot" || msg["snapshot"].(map[string]any)["phase"] != "active" {
		t.Fatalf("expected active snapshot after ws start, got %#v", msg)
	}

	if err := conn.WriteJSON(map[string]any{"type": "sendChat", "body": "hello team"}); err != nil {
		t.Fatalf("write chat: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read chat message: %v", err)
	}
	if msg["type"] != "chatMessage" || msg["message"].(map[string]any)["body"] != "hello team" {
		t.Fatalf("expected chat message broadcast, got %#v", msg)
	}
}
