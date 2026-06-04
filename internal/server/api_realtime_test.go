package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/codewords/internal/game"
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
	wordpacks := packs["wordpacks"].([]any)
	if len(wordpacks) == 0 {
		t.Fatalf("expected bundled wordpacks, got %#v", packs)
	}
	if wordpacks[0].(map[string]any)["id"] != "english" || wordpacks[1].(map[string]any)["id"] != "english-alternative" {
		t.Fatalf("expected mined legacy wordpack order first, got %#v", wordpacks[:2])
	}
	pictures := getJSON(t, h, "/api/pictures/catalog", http.StatusOK)
	if pictures["available"].(bool) {
		t.Fatalf("test catalog has no local pictures, got %#v", pictures)
	}
}

func TestGameOverSnapshotIncludesFinishedAt(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-finished", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-finished", "settings": map[string]any{"wordpackId": "english", "seed": 12, "totalCards": 9, "blackCards": 1}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-spy-finished", "displayName": "Red Spy"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "blue-guess-finished", "displayName": "Blue Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-guess-finished", "displayName": "Red Guess"}, http.StatusOK)
	makeRoomStartable(t, h, roomID, map[string]string{"host-finished": "blueSpy", "red-spy-finished": "redSpy", "blue-guess-finished": "blueGuess", "red-guess-finished": "redGuess"})

	postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "host-finished"}, http.StatusOK)
	server := httptest.NewServer(h)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-finished"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}
	cards := msg["snapshot"].(map[string]any)["cards"].([]any)
	blackIndex := -1
	for i, raw := range cards {
		card := raw.(map[string]any)
		if card["color"] == "black" {
			blackIndex = i
			break
		}
	}
	if blackIndex < 0 {
		t.Fatalf("expected a black card in snapshot: %#v", cards)
	}
	currentTeam := msg["snapshot"].(map[string]any)["currentTeam"].(string)
	guesserToken := "blue-guess-finished"
	if currentTeam == "red" {
		guesserToken = "red-guess-finished"
	}
	guesserURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=" + guesserToken
	guesserConn, _, err := websocket.DefaultDialer.Dial(guesserURL, nil)
	if err != nil {
		t.Fatalf("dial guesser websocket: %v", err)
	}
	defer guesserConn.Close()
	if err := guesserConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set guesser read deadline: %v", err)
	}
	if err := guesserConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read guesser initial snapshot: %v", err)
	}
	if err := guesserConn.WriteJSON(map[string]any{"type": "guessCard", "index": blackIndex}); err != nil {
		t.Fatalf("write assassin guess: %v", err)
	}
	if err := guesserConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read game over snapshot: %v", err)
	}
	snapshot := msg["snapshot"].(map[string]any)
	if snapshot["phase"] != "game_over" {
		t.Fatalf("expected game over snapshot, got %#v", snapshot)
	}
	finishedAt, ok := snapshot["finishedAt"].(string)
	if !ok || finishedAt == "" {
		t.Fatalf("expected finishedAt in game over snapshot, got %#v", snapshot)
	}
	if _, err := time.Parse(time.RFC3339Nano, finishedAt); err != nil {
		t.Fatalf("finishedAt should be RFC3339Nano, got %q: %v", finishedAt, err)
	}
}

func TestRoomStartPersistsSnapshotAndRestoresOverWebSocket(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host", "settings": map[string]any{"wordpackId": "english", "seed": 11}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest", "displayName": "Guest"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "blue-guess", "displayName": "Blue Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-guess", "displayName": "Red Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/settings", map[string]any{"authToken": "host", "settings": map[string]any{"wordpackId": "english", "seed": 11}}, http.StatusOK)
	makeRoomStartable(t, h, roomID, map[string]string{"host": "blueSpy", "guest": "redSpy", "blue-guess": "blueGuess", "red-guess": "redGuess"})

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

func TestUnityStartSnapshotIncludesSafeActiveAndOwnBoards(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "unity-host", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "unity-host", "settings": map[string]any{"mode": "unity", "wordpackId": "english", "seed": 301, "totalCards": 25, "unityTurnLimit": 3}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "unity-p2", "displayName": "P2"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "unity-p3", "displayName": "P3"}, http.StatusOK)
	makeUnityRoomStartable(t, h, roomID, []string{"unity-host", "unity-p2", "unity-p3"})

	start := postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "unity-host"}, http.StatusOK)
	snapshot := start["snapshot"].(map[string]any)
	if snapshot["currentTeam"] != "unity" {
		t.Fatalf("expected unity current team, got %#v", snapshot)
	}
	if snapshot["winner"] != "" {
		t.Fatalf("expected active unity match, got winner %#v", snapshot["winner"])
	}
	progress := snapshot["unityProgress"].(map[string]any)
	if progress["totalUnityCards"].(float64) != 30 || progress["sharedTurnsRemaining"].(float64) != 9 {
		t.Fatalf("unexpected unity progress: %#v", progress)
	}
	activeBoard := snapshot["activeBoard"].(map[string]any)
	activeCards := activeBoard["cards"].([]any)
	if len(activeCards) != 25 {
		t.Fatalf("expected active board cards, got %#v", activeBoard)
	}
	activeOwner := activeBoard["ownerId"].(string)
	viewer := snapshot["viewer"].(map[string]any)["userId"].(string)
	firstCard := activeCards[0].(map[string]any)
	if activeOwner != viewer {
		if _, ok := firstCard["color"]; ok {
			t.Fatalf("non-owner start response should not leak active hidden color: %#v", firstCard)
		}
	}
	ownBoard := snapshot["ownBoard"].(map[string]any)
	ownCards := ownBoard["cards"].([]any)
	if _, ok := ownCards[0].(map[string]any)["color"]; !ok {
		t.Fatalf("own board should include hidden color for owner: %#v", ownCards[0])
	}

	handler := h.(*Handler)
	rt, err := handler.app.loadRuntime(context.Background(), roomID)
	if err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	rt.mu.Lock()
	player := rt.state.Players[activeOwner]
	player.Representative = true
	rt.state.Players[activeOwner] = player
	tempSnapshot := snapshotDTO(rt.state, activeOwner)
	rt.mu.Unlock()
	tempProgress := tempSnapshot["unityProgress"].(map[string]any)
	tempRep, ok := tempProgress["temporaryRepresentativeId"].(string)
	if !ok || tempRep == "" || tempRep == activeOwner {
		t.Fatalf("expected temporary representative id in unity progress, got %#v", tempProgress)
	}
}

func TestUnityActiveRoomLinkOpenAddsObserverAndModCanAssignUnityBoard(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "unity-host-observer", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "unity-host-observer", "settings": map[string]any{"mode": "unity", "wordpackId": "english", "seed": 302, "unityTurnLimit": 3}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "unity-o-p2", "displayName": "P2"}, http.StatusOK)
	makeUnityRoomStartable(t, h, roomID, []string{"unity-host-observer", "unity-o-p2"})
	postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "unity-host-observer"}, http.StatusOK)

	late := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "unity-late", "displayName": "Late"}, http.StatusOK)
	lateID := late["userId"].(string)
	getJSON(t, h, "/api/rooms/"+roomID+"?authToken=unity-late", http.StatusOK)

	handler := h.(*Handler)
	rt, err := handler.app.loadRuntime(context.Background(), roomID)
	if err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	rt.mu.Lock()
	if got := rt.state.Players[lateID].Team; got != game.TeamObservers {
		t.Fatalf("late active-room opener should become observer, got %q", got)
	}
	if _, ok := rt.state.UnityBoards[lateID]; ok {
		t.Fatalf("observer should not get unity board until assigned")
	}
	_, err = game.Apply(&rt.state, game.AssignTeamCommand{PlayerID: lateID, Team: game.TeamUnity}, rt.state.HostID)
	rt.mu.Unlock()
	if err != nil {
		t.Fatalf("assign late player to unity: %v", err)
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if _, ok := rt.state.UnityBoards[lateID]; !ok {
		t.Fatalf("assigned late unity player should receive board")
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
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "blue-guess-ws", "displayName": "Blue Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-guess-ws", "displayName": "Red Guess"}, http.StatusOK)
	makeRoomStartable(t, h, roomID, map[string]string{"host-ws": "blueSpy", "guest-ws": "redSpy", "blue-guess-ws": "blueGuess", "red-guess-ws": "redGuess"})

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

	if err := conn.WriteJSON(map[string]any{"type": "updateSettings", "settings": map[string]any{"wordpackId": "english", "seed": 12, "blackCards": 1, "imageCardCount": 5, "observerChatEnabled": true, "mixedImageOrderFirst": true}}); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read settings snapshot: %v", err)
	}
	settings := msg["snapshot"].(map[string]any)["settings"].(map[string]any)
	if settings["imageCardCount"] != float64(5) || settings["mixedImageOrderFirst"] != true {
		t.Fatalf("expected updated websocket settings, got %#v", settings)
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

	if err := conn.WriteJSON(map[string]any{"type": "restartMatch"}); err != nil {
		t.Fatalf("write restart: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read restart snapshot: %v", err)
	}
	if msg["type"] != "snapshot" || msg["snapshot"].(map[string]any)["phase"] != "lobby" {
		t.Fatalf("expected lobby snapshot after restart, got %#v", msg)
	}
	handler := h.(*Handler)
	room, err := handler.app.store.RoomByID(context.Background(), roomID)
	if err != nil {
		t.Fatalf("room after restart: %v", err)
	}
	if room.Status != storage.RoomStatusLobby || room.CurrentMatchID != "" {
		t.Fatalf("restart should clear persisted active match, got %#v", room)
	}
}

func TestLobbyWebSocketReceivesJoinAndHostViewerSnapshot(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-lobby", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-lobby", "settings": map[string]any{"wordpackId": "english", "seed": 21}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)

	server := httptest.NewServer(h)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-lobby"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}
	viewer := msg["snapshot"].(map[string]any)["viewer"].(map[string]any)
	if viewer["isHost"] != true {
		t.Fatalf("expected host viewer snapshot, got %#v", viewer)
	}

	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest-lobby", "displayName": "Guest"}, http.StatusOK)
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read join broadcast: %v", err)
	}
	players := msg["snapshot"].(map[string]any)["players"].([]any)
	if len(players) != 2 {
		t.Fatalf("expected join broadcast with both players, got %#v", msg)
	}
}

func TestLobbyWebSocketRandomizeTeamsPersistsBalancedRoles(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-random", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-random", "settings": map[string]any{"wordpackId": "english", "seed": 31, "randomizeTeams": false}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	for _, token := range []string{"a-random", "b-random", "c-random"} {
		postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": token, "displayName": token}, http.StatusOK)
	}
	makeRoomStartable(t, h, roomID, map[string]string{"host-random": "blueSpy", "a-random": "blueGuess", "b-random": "redSpy", "c-random": "redGuess"})

	server := httptest.NewServer(h)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-random"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}

	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "randomizeTeams"}); err != nil {
		t.Fatalf("write randomize command: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read randomize snapshot: %v", err)
	}
	players := msg["snapshot"].(map[string]any)["players"].([]any)
	assertBalancedRandomizedPlayers(t, players)

	persisted := getJSON(t, h, "/api/rooms/"+roomID+"?authToken=host-random", http.StatusOK)
	assertBalancedRandomizedPlayers(t, persisted["players"].([]any))
}

func TestWebSocketForceBoardLayoutRequiresModAndBroadcastsSanitizedPreferences(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-layout", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-layout", "settings": map[string]any{"wordpackId": "english", "seed": 41}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest-layout", "displayName": "Guest"}, http.StatusOK)

	server := httptest.NewServer(h)
	defer server.Close()
	hostURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-layout"
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatalf("dial host websocket: %v", err)
	}
	defer hostConn.Close()
	guestURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=guest-layout"
	guestConn, _, err := websocket.DefaultDialer.Dial(guestURL, nil)
	if err != nil {
		t.Fatalf("dial guest websocket: %v", err)
	}
	defer guestConn.Close()
	if err := hostConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set host read deadline: %v", err)
	}
	if err := guestConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set guest read deadline: %v", err)
	}

	var msg map[string]any
	if err := hostConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read host initial snapshot: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read guest initial snapshot: %v", err)
	}

	if err := guestConn.WriteJSON(map[string]any{"type": "forceBoardLayout", "preferences": map[string]any{"boardColumnsMobile": 4}}); err != nil {
		t.Fatalf("write guest force layout: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read guest rejection: %v", err)
	}
	if msg["type"] != "error" || msg["code"] != "command_rejected" {
		t.Fatalf("expected non-mod rejection, got %#v", msg)
	}

	if err := hostConn.WriteJSON(map[string]any{"type": "forceBoardLayout", "preferences": map[string]any{
		"boardColumnsMobile":     0,
		"boardColumnsDesktop":    99,
		"imageCardScale":         8,
		"strictCardAspectRatios": true,
	}}); err != nil {
		t.Fatalf("write host force layout: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read layout broadcast: %v", err)
	}
	if msg["type"] != "boardLayoutForced" {
		t.Fatalf("expected boardLayoutForced broadcast, got %#v", msg)
	}
	preferences := msg["preferences"].(map[string]any)
	if preferences["boardColumnsMobile"] != float64(1) ||
		preferences["boardColumnsDesktop"] != float64(13) ||
		preferences["imageCardScale"] != float64(8) ||
		preferences["strictCardAspectRatios"] != true {
		t.Fatalf("expected sanitized layout preferences, got %#v", preferences)
	}
	if _, ok := preferences["cardGridMode"]; ok {
		t.Fatalf("grid mode should not be broadcast anymore, got %#v", preferences)
	}
	if msg["by"] == "" {
		t.Fatalf("expected broadcaster id, got %#v", msg)
	}
}

func TestWebSocketForceThemeRequiresModAndBroadcastsSanitizedTheme(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-theme", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-theme", "settings": map[string]any{"wordpackId": "english", "seed": 41}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest-theme", "displayName": "Guest"}, http.StatusOK)

	server := httptest.NewServer(h)
	defer server.Close()
	hostURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-theme"
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatalf("dial host websocket: %v", err)
	}
	defer hostConn.Close()
	guestURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=guest-theme"
	guestConn, _, err := websocket.DefaultDialer.Dial(guestURL, nil)
	if err != nil {
		t.Fatalf("dial guest websocket: %v", err)
	}
	defer guestConn.Close()
	if err := hostConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set host read deadline: %v", err)
	}
	if err := guestConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set guest read deadline: %v", err)
	}

	var msg map[string]any
	if err := hostConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read host initial snapshot: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read guest initial snapshot: %v", err)
	}

	if err := guestConn.WriteJSON(map[string]any{"type": "forceTheme", "theme": "matrix"}); err != nil {
		t.Fatalf("write guest force theme: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read guest rejection: %v", err)
	}
	if msg["type"] != "error" || msg["code"] != "command_rejected" {
		t.Fatalf("expected non-mod rejection, got %#v", msg)
	}

	if err := hostConn.WriteJSON(map[string]any{"type": "forceTheme", "theme": "neon"}); err != nil {
		t.Fatalf("write host force theme: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read theme broadcast: %v", err)
	}
	if msg["type"] != "themeForced" {
		t.Fatalf("expected themeForced broadcast, got %#v", msg)
	}
	if msg["theme"] != "dark" {
		t.Fatalf("expected unknown theme to be sanitized to dark, got %#v", msg)
	}
	if msg["by"] == "" {
		t.Fatalf("expected broadcaster id, got %#v", msg)
	}

	if err := hostConn.WriteJSON(map[string]any{"type": "forceTheme", "theme": "matrix"}); err != nil {
		t.Fatalf("write host force valid theme: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read second theme broadcast: %v", err)
	}
	if msg["type"] != "themeForced" || msg["theme"] != "matrix" {
		t.Fatalf("expected matrix theme broadcast, got %#v", msg)
	}

	if err := hostConn.WriteJSON(map[string]any{"type": "forceTheme", "theme": "dracula"}); err != nil {
		t.Fatalf("write host force dracula theme: %v", err)
	}
	if err := guestConn.ReadJSON(&msg); err != nil {
		t.Fatalf("read dracula theme broadcast: %v", err)
	}
	if msg["type"] != "themeForced" || msg["theme"] != "dracula" {
		t.Fatalf("expected dracula theme broadcast, got %#v", msg)
	}
}

func TestDuplicateDisplayNamesReceiveStableRoomScopedNumbers(t *testing.T) {
	players := []map[string]any{
		{"id": "b", "displayName": "Alex"},
		{"id": "a", "displayName": "Alex"},
		{"id": "c", "displayName": "Bea"},
	}

	applyStableDuplicateDisplayNames(players)

	names := map[string]string{}
	for _, player := range players {
		names[player["id"].(string)] = player["displayName"].(string)
	}
	if names["a"] != "Alex 1" || names["b"] != "Alex 2" || names["c"] != "Bea" {
		t.Fatalf("unexpected duplicate display names: %#v", names)
	}
}

func TestUnitySwitchSpymasterWebSocketBroadcastsPreviousBoardTransition(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-switch", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-switch", "settings": map[string]any{"mode": "unity", "wordpackId": "english", "seed": 431, "totalCards": 9, "unityTurnLimit": 3}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "switch-p2", "displayName": "P2"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "switch-p3", "displayName": "P3"}, http.StatusOK)
	makeUnityRoomStartable(t, h, roomID, []string{"host-switch", "switch-p2", "switch-p3"})
	postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "host-switch"}, http.StatusOK)

	server := httptest.NewServer(h)
	defer server.Close()
	hostURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-switch"
	hostConn, _, err := websocket.DefaultDialer.Dial(hostURL, nil)
	if err != nil {
		t.Fatalf("dial host websocket: %v", err)
	}
	defer hostConn.Close()
	guestURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=switch-p2"
	guestConn, _, err := websocket.DefaultDialer.Dial(guestURL, nil)
	if err != nil {
		t.Fatalf("dial guest websocket: %v", err)
	}
	defer guestConn.Close()
	if err := hostConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set host read deadline: %v", err)
	}
	if err := guestConn.SetReadDeadline(testDeadline()); err != nil {
		t.Fatalf("set guest read deadline: %v", err)
	}

	var hostMsg map[string]any
	if err := hostConn.ReadJSON(&hostMsg); err != nil {
		t.Fatalf("read host initial snapshot: %v", err)
	}
	var guestMsg map[string]any
	if err := guestConn.ReadJSON(&guestMsg); err != nil {
		t.Fatalf("read guest initial snapshot: %v", err)
	}
	initialOwner := hostMsg["snapshot"].(map[string]any)["activeBoard"].(map[string]any)["ownerId"].(string)

	if err := guestConn.WriteJSON(map[string]any{"type": "switchUnitySpymaster"}); err != nil {
		t.Fatalf("write guest switch: %v", err)
	}
	if err := guestConn.ReadJSON(&guestMsg); err != nil {
		t.Fatalf("read guest rejection: %v", err)
	}
	if guestMsg["type"] != "error" || guestMsg["code"] != "command_rejected" {
		t.Fatalf("expected non-mod rejection, got %#v", guestMsg)
	}

	if err := hostConn.WriteJSON(map[string]any{"type": "switchUnitySpymaster"}); err != nil {
		t.Fatalf("write host switch: %v", err)
	}
	if err := hostConn.ReadJSON(&hostMsg); err != nil {
		t.Fatalf("read host switch snapshot: %v", err)
	}
	snapshot := hostMsg["snapshot"].(map[string]any)
	nextOwner := snapshot["activeBoard"].(map[string]any)["ownerId"].(string)
	previous := snapshot["previousBoard"].(map[string]any)
	if previous["ownerId"] != initialOwner || nextOwner == initialOwner {
		t.Fatalf("expected previous=%s and new active owner, previous=%#v active=%s", initialOwner, previous, nextOwner)
	}
	if snapshot["unityTransitionUntil"] == "" {
		t.Fatalf("expected transition deadline in snapshot: %#v", snapshot)
	}
}

func TestNormalizeSettingsDefaultsTeamNamesAndRejectsInvalidColors(t *testing.T) {
	settings, err := normalizeSettings(game.Settings{
		WordpackID:      "english",
		CustomColorBlue: "not-a-color",
		CustomColorRed:  "#123abc",
		TeamNameBlue:    "  ",
		TeamNameRed:     " Guild of a Very Long Name That Should Be Trimmed Past The Limit ",
	})
	if err != nil {
		t.Fatalf("normalize settings: %v", err)
	}

	if settings.TeamNameBlue != "Libertarians" {
		t.Fatalf("expected default blue team name, got %q", settings.TeamNameBlue)
	}
	if settings.TeamNameRed != "Guild of a Very Long Name That" {
		t.Fatalf("expected trimmed red team name, got %q", settings.TeamNameRed)
	}
	if settings.CustomColorBlue != "" {
		t.Fatalf("invalid blue color should be cleared, got %q", settings.CustomColorBlue)
	}
	if settings.CustomColorRed != "#123abc" {
		t.Fatalf("valid red color should be preserved, got %q", settings.CustomColorRed)
	}
	if settings.TotalCards != game.DefaultTotalCards || !settings.AutoColorCounts {
		t.Fatalf("expected dynamic board defaults, got %#v", settings)
	}
	if settings.MemoryRoastsDisabled {
		t.Fatalf("memory roasts should default to enabled")
	}

	roastsOff, err := normalizeSettings(game.Settings{WordpackID: "english", MemoryRoastsDisabled: true})
	if err != nil {
		t.Fatalf("normalize memory roast setting: %v", err)
	}
	if !roastsOff.MemoryRoastsDisabled {
		t.Fatalf("expected memory roast opt-out to persist")
	}
}

func TestNormalizeSettingsRejectsInvalidCardCounts(t *testing.T) {
	cases := []game.Settings{
		{WordpackID: "english", TotalCards: 8, AutoColorCounts: true},
		{WordpackID: "english", TotalCards: 101, AutoColorCounts: true},
		{WordpackID: "english", TotalCards: 25, AutoColorCounts: true, ImageCardCount: 26},
		{WordpackID: "english", TotalCards: 25, AutoColorCounts: true, BlackCards: 9},
		{WordpackID: "english", TotalCards: 20, AutoColorCounts: false, BlueCards: 8, RedCards: 8, NeutralCards: 3},
	}
	for _, tc := range cases {
		if _, err := normalizeSettings(tc); !errors.Is(err, game.ErrInvalidSettings) {
			t.Fatalf("expected ErrInvalidSettings for %#v, got %v", tc, err)
		}
	}
}

func TestMigrateIdProvidesRoomViewerContext(t *testing.T) {
	h := newTestHandler(t)
	boot := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-migrate", "displayName": "Host"}, http.StatusOK)
	hostID := boot["userId"].(string)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-migrate", "settings": map[string]any{"wordpackId": "english"}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	link := postJSON(t, h, "/api/rooms/"+roomID+"/migrate-link", map[string]any{"authToken": "host-migrate"}, http.StatusOK)

	room := getJSON(t, h, "/api/rooms/"+roomID+"?migrateId="+link["migrateId"].(string), http.StatusOK)
	viewer := room["viewer"].(map[string]any)
	if viewer["userId"] != hostID || viewer["isHost"] != true {
		t.Fatalf("expected migrate viewer to resolve host, got %#v", viewer)
	}
}

func TestModeratorCanCreateTargetPlayerMigrateLink(t *testing.T) {
	h := newTestHandler(t)
	host := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-target-migrate", "displayName": "Host"}, http.StatusOK)
	guest := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "guest-target-migrate", "displayName": "Guest"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-target-migrate", "settings": map[string]any{"wordpackId": "english"}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest-target-migrate", "displayName": "Guest"}, http.StatusOK)

	link := postJSON(t, h, "/api/rooms/"+roomID+"/migrate-link", map[string]any{"authToken": "host-target-migrate", "playerId": guest["userId"].(string)}, http.StatusOK)

	if link["userId"] != guest["userId"] || link["userId"] == host["userId"] {
		t.Fatalf("expected target player's migrate link, got %#v", link)
	}
	resolved := postJSON(t, h, "/api/rooms/"+roomID+"/migrate-bootstrap", map[string]any{"migrateId": link["migrateId"].(string)}, http.StatusOK)
	if resolved["userId"] != guest["userId"] {
		t.Fatalf("expected migrate id to resolve target player, got %#v", resolved)
	}
}

func TestTargetPlayerMigrateLinkRequiresModerator(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-target-migrate-denied", "displayName": "Host"}, http.StatusOK)
	guest := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "guest-target-migrate-denied", "displayName": "Guest"}, http.StatusOK)
	other := postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "other-target-migrate-denied", "displayName": "Other"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-target-migrate-denied", "settings": map[string]any{"wordpackId": "english"}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "guest-target-migrate-denied", "displayName": "Guest"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "other-target-migrate-denied", "displayName": "Other"}, http.StatusOK)

	postJSON(t, h, "/api/rooms/"+roomID+"/migrate-link", map[string]any{"authToken": "guest-target-migrate-denied", "playerId": other["userId"].(string)}, http.StatusForbidden)
	self := postJSON(t, h, "/api/rooms/"+roomID+"/migrate-link", map[string]any{"authToken": "guest-target-migrate-denied", "playerId": guest["userId"].(string)}, http.StatusOK)
	if self["userId"] != guest["userId"] {
		t.Fatalf("expected explicit self migrate link to remain allowed, got %#v", self)
	}
}

func assertBalancedRandomizedPlayers(t *testing.T, players []any) {
	t.Helper()
	counts := map[string]int{}
	spies := map[string]int{}
	for _, raw := range players {
		player := raw.(map[string]any)
		team := player["team"].(string)
		if team != "blue" && team != "red" {
			t.Fatalf("expected only playable teams after randomize, got %#v", players)
		}
		counts[team]++
		if player["representative"].(bool) {
			t.Fatalf("representatives should be cleared after randomize: %#v", players)
		}
		if player["spymaster"].(bool) {
			spies[team]++
		}
	}
	if counts["blue"] != 2 || counts["red"] != 2 {
		t.Fatalf("expected two players per team, counts=%#v players=%#v", counts, players)
	}
	if spies["blue"] != 1 || spies["red"] != 1 {
		t.Fatalf("expected one spy per team, spies=%#v players=%#v", spies, players)
	}
}

func makeRoomStartable(t *testing.T, h http.Handler, roomID string, tokenRoles map[string]string) {
	t.Helper()
	handler := h.(*Handler)
	rt, err := handler.app.loadRuntime(context.Background(), roomID)
	if err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for token, role := range tokenRoles {
		user, err := handler.app.authUser(context.Background(), token)
		if err != nil {
			t.Fatalf("auth %s: %v", token, err)
		}
		team := game.TeamBlue
		if strings.HasPrefix(role, "red") {
			team = game.TeamRed
		}
		if _, err := game.Apply(&rt.state, game.AssignTeamCommand{PlayerID: user.ID, Team: team}, rt.state.HostID); err != nil {
			t.Fatalf("assign %s: %v", role, err)
		}
		if strings.HasSuffix(role, "Spy") {
			if _, err := game.Apply(&rt.state, game.ToggleSpymasterCommand{PlayerID: user.ID}, rt.state.HostID); err != nil {
				t.Fatalf("spy %s: %v", role, err)
			}
		}
	}
	if err := handler.app.syncRoomPlayers(context.Background(), roomID, rt.state); err != nil {
		t.Fatalf("sync room players: %v", err)
	}
}

func makeUnityRoomStartable(t *testing.T, h http.Handler, roomID string, tokens []string) {
	t.Helper()
	handler := h.(*Handler)
	rt, err := handler.app.loadRuntime(context.Background(), roomID)
	if err != nil {
		t.Fatalf("load runtime: %v", err)
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	for _, token := range tokens {
		user, err := handler.app.authUser(context.Background(), token)
		if err != nil {
			t.Fatalf("auth %s: %v", token, err)
		}
		if _, err := game.Apply(&rt.state, game.AssignTeamCommand{PlayerID: user.ID, Team: game.TeamUnity}, rt.state.HostID); err != nil {
			t.Fatalf("assign unity %s: %v", token, err)
		}
	}
	if err := handler.app.syncRoomPlayers(context.Background(), roomID, rt.state); err != nil {
		t.Fatalf("sync room players: %v", err)
	}
}

func testDeadline() time.Time {
	return time.Now().Add(2 * time.Second)
}

func TestPictureCatalogListsAndServesLocalImages(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	pngBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if err := os.WriteFile(filepath.Join(imageDir, "card one.png"), pngBytes, 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}
	expectedID := legacyImageID(pngBytes)
	avifBytes := []byte("cached avif")
	if err := os.WriteFile(filepath.Join(cacheDir, expectedID+".avif"), avifBytes, 0o644); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}
	h := newTestHandlerWithPictures(t, imageDir, cacheDir, false)

	catalog := getJSON(t, h, "/api/pictures/catalog", http.StatusOK)
	if catalog["available"] != true {
		t.Fatalf("expected picture catalog to be available, got %#v", catalog)
	}
	images := catalog["images"].([]any)
	if len(images) != 0 {
		t.Fatalf("expected disabled processing to defer image ids until start, got %#v", catalog)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/pictures/"+expectedID, nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "image/avif" || !bytes.Equal(res.Body.Bytes(), avifBytes) {
		t.Fatalf("unexpected image response code=%d type=%q body=%#v", res.Code, res.Header().Get("Content-Type"), res.Body.Bytes())
	}
}

func TestPictureCatalogDisabledWithoutCacheWhenProcessingOff(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(imageDir, "card one.png"), []byte("not-cached"), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}
	h := newTestHandlerWithPictures(t, imageDir, cacheDir, false)

	catalog := getJSON(t, h, "/api/pictures/catalog", http.StatusOK)
	if !catalog["available"].(bool) || len(catalog["images"].([]any)) != 0 {
		t.Fatalf("expected source candidates to enable image mode without exposing ids while AVIF processing is off, got %#v", catalog)
	}
}

func TestStartWithProcessingOffReplacesMissingSelectedImageCaches(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	ids := writePictureSources(t, imageDir, 8)
	settings := game.Settings{WordpackID: "english", Seed: 777, ImageCardCount: 3, ObserverChatEnabled: true}
	order := game.ShuffledImageIDs(settings, ids)
	missingSelected := order[0]
	wantSelected := order[1:4]
	for _, id := range wantSelected {
		if err := os.WriteFile(filepath.Join(cacheDir, id+".avif"), []byte("cached "+id), 0o644); err != nil {
			t.Fatalf("write cache %s: %v", id, err)
		}
	}
	h := newTestHandlerWithPictures(t, imageDir, cacheDir, false)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-images", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-images", "settings": map[string]any{"wordpackId": "english", "seed": 777, "imageCardCount": 3}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-spy-images", "displayName": "Red Spy"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "blue-guess-images", "displayName": "Blue Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-guess-images", "displayName": "Red Guess"}, http.StatusOK)
	makeRoomStartable(t, h, roomID, map[string]string{"host-images": "blueSpy", "red-spy-images": "redSpy", "blue-guess-images": "blueGuess", "red-guess-images": "redGuess"})

	start := postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "host-images"}, http.StatusOK)
	cards := start["snapshot"].(map[string]any)["cards"].([]any)
	gotImages := map[string]bool{}
	for _, raw := range cards {
		card := raw.(map[string]any)
		if card["contentType"] == "image" {
			gotImages[card["imageId"].(string)] = true
		}
	}

	if gotImages[missingSelected] {
		t.Fatalf("expected missing selected cache %s to be replaced, got images %#v", missingSelected, gotImages)
	}
	for _, id := range wantSelected {
		if !gotImages[id] {
			t.Fatalf("expected replacement-selected cached image %s in board, got %#v", id, gotImages)
		}
	}
	if len(gotImages) != 3 {
		t.Fatalf("expected exactly 3 image cards, got %#v", gotImages)
	}
}

func TestStartWithProcessingOffFailsWhenNotEnoughCachedImageCandidates(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	ids := writePictureSources(t, imageDir, 4)
	settings := game.Settings{WordpackID: "english", Seed: 778, ImageCardCount: 3, ObserverChatEnabled: true}
	order := game.ShuffledImageIDs(settings, ids)
	for _, id := range order[:2] {
		if err := os.WriteFile(filepath.Join(cacheDir, id+".avif"), []byte("cached "+id), 0o644); err != nil {
			t.Fatalf("write cache %s: %v", id, err)
		}
	}
	h := newTestHandlerWithPictures(t, imageDir, cacheDir, false)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-few-images", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-few-images", "settings": map[string]any{"wordpackId": "english", "seed": 778, "imageCardCount": 3}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-spy-few-images", "displayName": "Red Spy"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "blue-guess-few-images", "displayName": "Blue Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-guess-few-images", "displayName": "Red Guess"}, http.StatusOK)
	makeRoomStartable(t, h, roomID, map[string]string{"host-few-images": "blueSpy", "red-spy-few-images": "redSpy", "blue-guess-few-images": "blueGuess", "red-guess-few-images": "redGuess"})

	res := postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "host-few-images"}, http.StatusBadRequest)
	if !strings.Contains(res["error"].(map[string]any)["message"].(string), game.ErrNotEnoughImages.Error()) {
		t.Fatalf("expected not enough images error, got %#v", res)
	}
}

func TestPictureCatalogDiscoversImagesThroughSymlinkedDirectories(t *testing.T) {
	imageDir := t.TempDir()
	targetDir := t.TempDir()
	cacheDir := t.TempDir()
	nestedDir := filepath.Join(targetDir, "nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}
	sourceBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if err := os.WriteFile(filepath.Join(nestedDir, "linked.png"), sourceBytes, 0o644); err != nil {
		t.Fatalf("write linked image fixture: %v", err)
	}
	if err := os.Symlink(targetDir, filepath.Join(imageDir, "linked-pictures")); err != nil {
		t.Fatalf("create symlinked image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, legacyImageID(sourceBytes)+".avif"), []byte("cached avif"), 0o644); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}

	catalog, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir})
	if err != nil {
		t.Fatalf("load picture catalog: %v", err)
	}

	if len(catalog.sourcePaths) != 1 {
		t.Fatalf("expected symlinked nested image to be discovered, got sources=%#v diagnostics=%q", catalog.sourcePaths, catalog.Diagnostics().StartupLogLine())
	}
	if catalog.Diagnostics().SourceCount != 1 {
		t.Fatalf("expected diagnostics to count symlinked source image, got %#v", catalog.Diagnostics())
	}
}

func TestPictureCatalogDiagnosticsExplainDisabledCacheOnlyState(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cacheDir, "orphan.avif"), []byte("cached avif"), 0o644); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}

	catalog, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir, ProcessAVIF: true})
	if err != nil {
		t.Fatalf("load picture catalog: %v", err)
	}
	line := catalog.Diagnostics().StartupLogLine()

	for _, want := range []string{
		"image mode disabled",
		"no supported source images found",
		"cached AVIF files alone cannot be matched without source images",
		"source_images=0",
		"enabled_images=0",
		"cached_avif_images=1",
		"avif_processing=true",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected diagnostics to contain %q, got %q", want, line)
		}
	}
}

func TestPictureCatalogDiagnosticsReportEnabledCachedImages(t *testing.T) {
	imageDir := t.TempDir()
	cacheDir := t.TempDir()
	sourceBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	if err := os.WriteFile(filepath.Join(imageDir, "card.png"), sourceBytes, 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, legacyImageID(sourceBytes)+".avif"), []byte("cached avif"), 0o644); err != nil {
		t.Fatalf("write cache fixture: %v", err)
	}

	catalog, err := loadPictureCatalog(pictureCatalogOptions{ImageDir: imageDir, ImageCacheDir: cacheDir})
	if err != nil {
		t.Fatalf("load picture catalog: %v", err)
	}
	line := catalog.Diagnostics().StartupLogLine()

	for _, want := range []string{
		"image mode enabled",
		"available",
		"source_images=1",
		"enabled_images=1",
		"cached_avif_images=1",
		"avif_processing=false",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected diagnostics to contain %q, got %q", want, line)
		}
	}
}

func TestLegacyImageIDVector(t *testing.T) {
	got := legacyImageID([]byte("legacy-cache-test"))
	const want = "93670c3199ed9a9f911da869573fe47af8ec93bfe02516f1cc9ad67ed5a284fe"
	if got != want {
		t.Fatalf("legacy id mismatch: got %s want %s", got, want)
	}
}

func TestSnapshotIncludesChatHistoryAndAnonymousObserverSocketIsRejected(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-chat", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-chat", "settings": map[string]any{"wordpackId": "english", "seed": 12}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)

	server := httptest.NewServer(h)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?authToken=host-chat"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()
	var msg map[string]any
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read initial: %v", err)
	}
	if err := conn.WriteJSON(map[string]any{"type": "sendChat", "body": "hello lobby"}); err != nil {
		t.Fatalf("write chat: %v", err)
	}
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read chat: %v", err)
	}

	snap := getJSON(t, h, "/api/rooms/"+roomID+"?authToken=host-chat", http.StatusOK)
	messages := snap["chatMessages"].([]any)
	if len(messages) != 1 || messages[0].(map[string]any)["body"] != "hello lobby" {
		t.Fatalf("expected chat history on room payload, got %#v", snap)
	}

	observerURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + roomID + "?observer=1"
	observerDialer := websocket.Dialer{HandshakeTimeout: time.Second}
	if observer, _, err := observerDialer.Dial(observerURL, nil); err == nil {
		_ = observer.Close()
		t.Fatalf("anonymous observer websocket should require authentication")
	}
}

func TestActiveRoomNonMemberWithDisplayNameBecomesObserver(t *testing.T) {
	h := newTestHandler(t)
	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "host-observe", "displayName": "Host"}, http.StatusOK)
	roomResp := postJSON(t, h, "/api/rooms", map[string]any{"authToken": "host-observe", "settings": map[string]any{"wordpackId": "english", "seed": 44, "randomizeTeams": true}}, http.StatusCreated)
	roomID := roomResp["room"].(map[string]any)["id"].(string)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-spy-observe", "displayName": "Red Spy"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "blue-guess-observe", "displayName": "Blue Guess"}, http.StatusOK)
	postJSON(t, h, "/api/rooms/"+roomID+"/join", map[string]any{"authToken": "red-guess-observe", "displayName": "Red Guess"}, http.StatusOK)
	makeRoomStartable(t, h, roomID, map[string]string{"host-observe": "blueSpy", "red-spy-observe": "redSpy", "blue-guess-observe": "blueGuess", "red-guess-observe": "redGuess"})
	postJSON(t, h, "/api/rooms/"+roomID+"/start", map[string]any{"authToken": "host-observe"}, http.StatusOK)

	postJSON(t, h, "/api/identity/bootstrap", map[string]any{"authToken": "observer-active", "displayName": "Observer Active"}, http.StatusOK)
	room := getJSON(t, h, "/api/rooms/"+roomID+"?authToken=observer-active", http.StatusOK)
	players := room["players"].([]any)
	found := false
	for _, raw := range players {
		player := raw.(map[string]any)
		if player["team"] == "observers" && player["id"] == room["viewer"].(map[string]any)["userId"] {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("active-room non-member with a display name should be added as an observer, got %#v", room)
	}
}

func newTestHandlerWithPictures(t *testing.T, picturesDir, cacheDir string, process bool) http.Handler {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "codewords.sqlite"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ids := identity.NewService(db, identity.Options{HashKey: []byte("test-key"), MigrateIDBytes: 8})
	h, err := NewHandler(Options{Store: db, Identity: ids, WordpacksDir: filepath.Join("..", "..", "assets", "wordpacks"), ImageDir: picturesDir, ImageCacheDir: cacheDir, AVIFProcess: process, BaseURL: "http://example.test"})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h
}

func writePictureSources(t *testing.T, imageDir string, count int) []string {
	t.Helper()
	ids := make([]string, 0, count)
	for i := 0; i < count; i++ {
		sourceBytes := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', byte(i), byte(i >> 8), 0, 0}
		if err := os.WriteFile(filepath.Join(imageDir, fmt.Sprintf("card-%02d.png", i)), sourceBytes, 0o644); err != nil {
			t.Fatalf("write source image %d: %v", i, err)
		}
		ids = append(ids, legacyImageID(sourceBytes))
	}
	return ids
}
