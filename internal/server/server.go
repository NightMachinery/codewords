package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/NightMachinery/codewords/internal/game"
	"github.com/NightMachinery/codewords/internal/identity"
	"github.com/NightMachinery/codewords/internal/storage"
	"github.com/gorilla/websocket"
)

// Options wires the HTTP/API handler to persistence, identity, and local assets.
type Options struct {
	Store         *storage.DB
	Identity      *identity.Service
	WordpacksDir  string
	ImageDir      string
	ImageCacheDir string
	AVIFProcess   bool
	BaseURL       string
	LogPictures   bool
}

type app struct {
	store    *storage.DB
	identity *identity.Service
	packs    map[string]game.Wordpack
	baseURL  string
	pictures *pictureCatalog
	rooms    map[string]*roomRuntime
	mu       sync.Mutex
}

type Handler struct {
	mux *http.ServeMux
	app *app
}

type roomRuntime struct {
	mu      sync.Mutex
	state   game.State
	clients map[*websocket.Conn]string
}

// NewHandler returns the HTTP handler for API, WebSocket, and health routes.
func NewHandler(options ...Options) (http.Handler, error) {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}
	packs := map[string]game.Wordpack{}
	if strings.TrimSpace(opts.WordpacksDir) != "" {
		loaded, err := game.LoadWordpacks(opts.WordpacksDir)
		if err != nil {
			return nil, err
		}
		packs = loaded
	}
	pictureOptions := pictureCatalogOptions{ImageDir: opts.ImageDir, ImageCacheDir: opts.ImageCacheDir, ProcessAVIF: opts.AVIFProcess}
	if opts.LogPictures {
		pictureOptions.Logf = log.Printf
	}
	pictures, err := loadPictureCatalog(pictureOptions)
	if err != nil {
		return nil, err
	}
	a := &app{store: opts.Store, identity: opts.Identity, packs: packs, baseURL: strings.TrimRight(opts.BaseURL, "/"), pictures: pictures, rooms: map[string]*roomRuntime{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("POST /api/identity/bootstrap", a.handleBootstrap)
	mux.HandleFunc("POST /api/identity/display-name", a.handleDisplayName)
	mux.HandleFunc("POST /api/rooms", a.handleCreateRoom)
	mux.HandleFunc("GET /api/rooms/{roomId}", a.handleGetRoom)
	mux.HandleFunc("POST /api/rooms/{roomId}/join", a.handleJoinRoom)
	mux.HandleFunc("POST /api/rooms/{roomId}/settings", a.handleSettings)
	mux.HandleFunc("POST /api/rooms/{roomId}/start", a.handleStart)
	mux.HandleFunc("POST /api/rooms/{roomId}/migrate-link", a.handleMigrateLink)
	mux.HandleFunc("POST /api/rooms/{roomId}/migrate-bootstrap", a.handleMigrateBootstrap)
	mux.HandleFunc("GET /api/wordpacks", a.handleWordpacks)
	mux.HandleFunc("GET /api/pictures/catalog", a.handlePictureCatalog)
	mux.HandleFunc("GET /api/pictures/{imageId}", a.handlePicture)
	mux.HandleFunc("GET /ws/rooms/{roomId}", a.handleWS)
	return &Handler{mux: mux, app: a}, nil
}

// MustNewHandler is a convenience for tests or tools that only need health routes.
func MustNewHandler(options ...Options) http.Handler {
	h, err := NewHandler(options...)
	if err != nil {
		panic(err)
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) PictureDiagnostics() string {
	return h.app.pictures.Diagnostics().StartupLogLine()
}

// PictureDiagnostics returns startup diagnostics for local image mode.
func PictureDiagnostics(handler http.Handler) (string, bool) {
	appHandler, ok := handler.(interface{ PictureDiagnostics() string })
	if !ok {
		return "", false
	}
	return appHandler.PictureDiagnostics(), true
}

func healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) requireReady() error {
	if a.store == nil || a.identity == nil {
		return fmt.Errorf("server persistence is not configured")
	}
	return nil
}

func (a *app) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct{ AuthToken, DisplayName string }
	if !decodeRequest(w, r, &req) {
		return
	}
	res, err := a.identity.Bootstrap(r.Context(), req.AuthToken, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_identity", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"userId": res.UserID, "displayName": res.DisplayName})
}

func (a *app) handleDisplayName(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct{ AuthToken, DisplayName string }
	if !decodeRequest(w, r, &req) {
		return
	}
	user, err := a.authUser(r.Context(), req.AuthToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	if err := a.identity.SaveDisplayName(r.Context(), user.ID, req.DisplayName); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_display_name", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"userId": user.ID, "displayName": strings.TrimSpace(req.DisplayName)})
}

func (a *app) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct {
		AuthToken string
		Settings  game.Settings
	}
	if !decodeRequest(w, r, &req) {
		return
	}
	user, err := a.authUser(r.Context(), req.AuthToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	settings := normalizeSettings(req.Settings)
	settingsJSON, _ := json.Marshal(settings)
	room, err := a.store.CreateRoom(r.Context(), storage.CreateRoomParams{HostUserID: user.ID, SettingsJSON: string(settingsJSON)})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create_room_failed", err.Error())
		return
	}
	state := game.NewLobby(user.ID, settings)
	_, _ = game.Apply(&state, game.AddPlayerCommand{PlayerID: user.ID, DisplayName: user.DisplayName}, user.ID)
	rt := a.runtime(room.ID)
	rt.mu.Lock()
	rt.state = state
	_ = a.syncRoomPlayers(r.Context(), room.ID, rt.state)
	rt.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{"room": roomDTO(room), "settings": settings, "roomLink": a.roomLink(r, room.ID), "viewer": viewerDTO(user.ID, true, true)})
}

func (a *app) handleGetRoom(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	roomID := r.PathValue("roomId")
	room, err := a.store.RoomByID(r.Context(), roomID)
	if err != nil {
		writeStorageErr(w, err, "room_not_found")
		return
	}
	players, err := a.store.RoomPlayers(r.Context(), roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "players_failed", err.Error())
		return
	}
	viewerID := ""
	if migrateID := r.URL.Query().Get("migrateId"); migrateID != "" {
		if link, err := a.identity.ResolveMigrate(r.Context(), roomID, migrateID); err == nil {
			viewerID = link.UserID
		}
	} else if token := tokenFromRequest(r); token != "" {
		if user, err := a.authUser(r.Context(), token); err == nil {
			viewerID = user.ID
		}
	}
	chats, _ := a.store.ChatMessages(r.Context(), roomID, 50)
	writeJSON(w, http.StatusOK, map[string]any{"room": roomDTO(room), "players": playerDTOs(players), "settings": mustSettings(room.SettingsJSON), "viewer": viewerDTO(viewerID, viewerID == room.HostUserID, viewerIsMod(players, room.HostUserID, viewerID)), "chatMessages": chatDTOs(chats)})
}

func (a *app) handleJoinRoom(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct{ AuthToken, DisplayName string }
	if !decodeRequest(w, r, &req) {
		return
	}
	roomID := r.PathValue("roomId")
	room, err := a.store.RoomByID(r.Context(), roomID)
	if err != nil {
		writeStorageErr(w, err, "room_not_found")
		return
	}
	boot, err := a.identity.Bootstrap(r.Context(), req.AuthToken, req.DisplayName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	if err := a.store.UpsertRoomPlayer(r.Context(), storage.RoomPlayer{RoomID: roomID, UserID: boot.UserID}); err != nil {
		writeError(w, http.StatusInternalServerError, "join_failed", err.Error())
		return
	}
	rt, err := a.loadRuntime(r.Context(), roomID)
	if err == nil {
		rt.mu.Lock()
		_, _ = game.Apply(&rt.state, game.AddPlayerCommand{PlayerID: boot.UserID, DisplayName: boot.DisplayName}, boot.UserID)
		_ = a.syncRoomPlayers(r.Context(), roomID, rt.state)
		rt.broadcastLocked(snapshotMessage(rt.state, ""))
		rt.mu.Unlock()
	}
	players, _ := a.store.RoomPlayers(r.Context(), roomID)
	writeJSON(w, http.StatusOK, map[string]any{"room": roomDTO(room), "viewer": viewerDTO(boot.UserID, boot.UserID == room.HostUserID, viewerIsMod(players, room.HostUserID, boot.UserID))})
}

func (a *app) handleSettings(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct {
		AuthToken string
		Settings  game.Settings
	}
	if !decodeRequest(w, r, &req) {
		return
	}
	roomID := r.PathValue("roomId")
	room, user, ok := a.requireMod(w, r.Context(), roomID, req.AuthToken)
	if !ok {
		return
	}
	settings := normalizeSettings(req.Settings)
	settingsJSON, _ := json.Marshal(settings)
	if err := a.store.UpdateRoomSettings(r.Context(), room.ID, string(settingsJSON)); err != nil {
		writeError(w, http.StatusInternalServerError, "settings_failed", err.Error())
		return
	}
	rt, _ := a.loadRuntime(r.Context(), roomID)
	rt.mu.Lock()
	_, _ = game.Apply(&rt.state, game.UpdateSettingsCommand{Settings: settings}, user.ID)
	_ = a.syncRoomPlayers(r.Context(), roomID, rt.state)
	rt.broadcastLocked(snapshotMessage(rt.state, ""))
	rt.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (a *app) handleStart(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct{ AuthToken string }
	if !decodeRequest(w, r, &req) {
		return
	}
	roomID := r.PathValue("roomId")
	room, user, ok := a.requireMod(w, r.Context(), roomID, req.AuthToken)
	if !ok {
		return
	}
	rt, err := a.loadRuntime(r.Context(), roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "room_state_failed", err.Error())
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	match, err := a.startMatchLocked(r.Context(), room, rt, user.ID)
	if err != nil {
		writeEngineErr(w, err)
		return
	}
	rt.broadcastLocked(snapshotMessage(rt.state, ""))
	writeJSON(w, http.StatusOK, map[string]any{"matchId": match.ID, "snapshot": snapshotDTO(rt.state, user.ID)})
}

func (a *app) handleMigrateLink(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct{ AuthToken string }
	if !decodeRequest(w, r, &req) {
		return
	}
	roomID := r.PathValue("roomId")
	_, user, ok := a.requireMember(w, r.Context(), roomID, req.AuthToken)
	if !ok {
		return
	}
	link, err := a.identity.CreateMigrateLink(r.Context(), roomID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "migrate_link_failed", err.Error())
		return
	}
	url := a.roomLink(r, roomID) + "?migrateId=" + link.MigrateID
	writeJSON(w, http.StatusOK, map[string]any{"roomId": roomID, "userId": user.ID, "migrateId": link.MigrateID, "migrateUrl": url})
}

func (a *app) handleMigrateBootstrap(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	var req struct{ MigrateID string }
	if !decodeRequest(w, r, &req) {
		return
	}
	roomID := r.PathValue("roomId")
	link, err := a.identity.ResolveMigrate(r.Context(), roomID, req.MigrateID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_migrate_id", err.Error())
		return
	}
	user, err := a.store.UserByID(r.Context(), link.UserID)
	if err != nil {
		writeStorageErr(w, err, "user_not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roomId": roomID, "userId": user.ID, "displayName": user.DisplayName})
}

func (a *app) handleWordpacks(w http.ResponseWriter, r *http.Request) {
	packs := make([]map[string]any, 0, len(a.packs))
	for _, p := range a.packs {
		packs = append(packs, map[string]any{"id": p.ID, "label": p.Label, "wordCount": len(p.Words)})
	}
	sort.Slice(packs, func(i, j int) bool {
		return wordpackSortKey(packs[i]["id"].(string)) < wordpackSortKey(packs[j]["id"].(string))
	})
	writeJSON(w, http.StatusOK, map[string]any{"wordpacks": packs})
}

func wordpackSortKey(id string) string {
	preferred := []string{"english", "english-alternative", "dutch", "czech", "german", "persian-1", "harry-potter-1", "harry-potter-1-fa"}
	for i, candidate := range preferred {
		if id == candidate {
			return fmt.Sprintf("%02d", i)
		}
	}
	return "99-" + strings.ToLower(id)
}

func (a *app) handlePictureCatalog(w http.ResponseWriter, r *http.Request) {
	images := a.pictures.listDTO(r)
	available := len(images) > 0 || (a.pictures != nil && len(a.pictures.sourcePaths) > 0)
	writeJSON(w, http.StatusOK, map[string]any{"available": available, "images": images})
}
func (a *app) handlePicture(w http.ResponseWriter, r *http.Request) {
	imageID := r.PathValue("imageId")
	asset, ok := a.pictures.images[imageID]
	cachePath := ""
	contentType := "image/avif"
	if ok {
		cachePath = asset.CachePath
		contentType = asset.ContentType
	} else if a.pictures != nil && !a.pictures.diag.ProcessAVIF && validAVIFCacheBasename.MatchString(imageID+".avif") {
		cachePath = filepath.Join(a.pictures.cacheDir, imageID+".avif")
	} else {
		writeError(w, http.StatusNotFound, "picture_not_found", "local picture not found")
		return
	}
	if info, err := os.Stat(cachePath); err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "picture_not_found", "local picture not found")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, cachePath)
}

func (a *app) handleWS(w http.ResponseWriter, r *http.Request) {
	if !a.readyOrError(w) {
		return
	}
	roomID := r.PathValue("roomId")
	viewerID, err := a.viewerID(r.Context(), roomID, r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	rt, err := a.loadRuntime(r.Context(), roomID)
	if err != nil {
		writeError(w, http.StatusNotFound, "room_not_found", err.Error())
		return
	}
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	rt.mu.Lock()
	if rt.clients == nil {
		rt.clients = map[*websocket.Conn]string{}
	}
	rt.clients[conn] = viewerID
	initial := snapshotMessage(rt.state, viewerID)
	rt.mu.Unlock()
	defer func() { rt.mu.Lock(); delete(rt.clients, conn); rt.mu.Unlock() }()
	_ = conn.WriteJSON(initial)
	for {
		var msg map[string]any
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		a.handleWSMessage(r.Context(), roomID, rt, conn, viewerID, msg)
	}
}

func (a *app) handleWSMessage(ctx context.Context, roomID string, rt *roomRuntime, conn *websocket.Conn, viewerID string, msg map[string]any) {
	t, _ := msg["type"].(string)
	if t == "ping" {
		_ = conn.WriteJSON(map[string]any{"type": "pong"})
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if t == "startGame" {
		room, err := a.store.RoomByID(ctx, roomID)
		if err != nil {
			_ = conn.WriteJSON(errorMessage("room_not_found", err.Error()))
			return
		}
		if _, err := a.startMatchLocked(ctx, room, rt, viewerID); err != nil {
			_ = conn.WriteJSON(errorMessage("command_rejected", err.Error()))
			return
		}
		rt.broadcastLocked(snapshotMessage(rt.state, ""))
		return
	}
	if t == "sendChat" {
		body, _ := msg["body"].(string)
		if player, ok := rt.state.Players[viewerID]; ok && player.Team == game.TeamObservers && !rt.state.Settings.ObserverChatEnabled {
			_ = conn.WriteJSON(errorMessage("chat_rejected", "Observers cannot chat in this room"))
			return
		}
		chat, err := a.addChatMessage(ctx, roomID, viewerID, body)
		if err != nil {
			_ = conn.WriteJSON(errorMessage("chat_rejected", err.Error()))
			return
		}
		rt.broadcastLocked(map[string]any{"type": "chatMessage", "message": chatDTO(chat)})
		return
	}
	if t == "updateSettings" {
		settings, err := settingsFromMessage(msg)
		if err != nil {
			_ = conn.WriteJSON(errorMessage("invalid_command", err.Error()))
			return
		}
		settings = normalizeSettings(settings)
		event, err := game.Apply(&rt.state, game.UpdateSettingsCommand{Settings: settings}, viewerID)
		if err != nil {
			_ = conn.WriteJSON(errorMessage("command_rejected", err.Error()))
			return
		}
		room, err := a.store.RoomByID(ctx, roomID)
		if err != nil {
			_ = conn.WriteJSON(errorMessage("room_not_found", err.Error()))
			return
		}
		settingsJSON, _ := json.Marshal(settings)
		if err := a.store.UpdateRoomSettings(ctx, roomID, string(settingsJSON)); err != nil {
			_ = conn.WriteJSON(errorMessage("settings_failed", err.Error()))
			return
		}
		if err := a.syncRoomPlayers(ctx, roomID, rt.state); err != nil {
			_ = conn.WriteJSON(errorMessage("players_failed", err.Error()))
			return
		}
		if room.CurrentMatchID != "" {
			if err := a.persistState(ctx, room.CurrentMatchID, viewerID, string(event.Type), rt.state); err != nil {
				_ = conn.WriteJSON(errorMessage("persist_failed", err.Error()))
				return
			}
		}
		rt.broadcastLocked(snapshotMessage(rt.state, ""))
		return
	}
	cmd, err := commandFromMessage(t, msg)
	if err != nil {
		_ = conn.WriteJSON(errorMessage("invalid_command", err.Error()))
		return
	}
	event, err := game.Apply(&rt.state, cmd, viewerID)
	if err != nil {
		_ = conn.WriteJSON(errorMessage("command_rejected", err.Error()))
		return
	}
	room, err := a.store.RoomByID(ctx, roomID)
	if err == nil {
		if err := a.syncRoomPlayers(ctx, roomID, rt.state); err != nil {
			_ = conn.WriteJSON(errorMessage("players_failed", err.Error()))
			return
		}
		if room.CurrentMatchID != "" {
			if err := a.persistState(ctx, room.CurrentMatchID, viewerID, string(event.Type), rt.state); err != nil {
				_ = conn.WriteJSON(errorMessage("persist_failed", err.Error()))
				return
			}
			if event.Type == game.EventMatchRestarted {
				if err := a.store.ClearRoomCurrentMatch(ctx, roomID); err != nil {
					_ = conn.WriteJSON(errorMessage("restart_failed", err.Error()))
					return
				}
			}
		}
	}
	rt.broadcastLocked(snapshotMessage(rt.state, ""))
}

func (a *app) startMatchLocked(ctx context.Context, room storage.Room, rt *roomRuntime, actorID string) (storage.Match, error) {
	players, _ := a.store.RoomPlayers(ctx, room.ID)
	for _, p := range players {
		u, _ := a.store.UserByID(ctx, p.UserID)
		_, _ = game.Apply(&rt.state, game.AddPlayerCommand{PlayerID: p.UserID, DisplayName: u.DisplayName}, p.UserID)
	}
	pack := a.packs[rt.state.Settings.WordpackID]
	imageIDs, err := a.pictureIDsForStart(ctx, rt.state.Settings)
	if err != nil {
		return storage.Match{}, err
	}
	event, err := game.Apply(&rt.state, game.StartCommand{Words: pack.Words, ImageIDs: imageIDs}, actorID)
	if err != nil {
		return storage.Match{}, err
	}
	match, err := a.store.CreateMatch(ctx, storage.CreateMatchParams{RoomID: room.ID, Seed: rt.state.Settings.Seed, SettingsJSON: room.SettingsJSON})
	if err != nil {
		return storage.Match{}, err
	}
	if err := a.store.SetRoomCurrentMatch(ctx, room.ID, match.ID); err != nil {
		return storage.Match{}, err
	}
	if err := a.persistState(ctx, match.ID, actorID, string(event.Type), rt.state); err != nil {
		return storage.Match{}, err
	}
	return match, nil
}

func (a *app) addChatMessage(ctx context.Context, roomID, viewerID, body string) (storage.ChatMessage, error) {
	text := strings.TrimSpace(body)
	if text == "" {
		return storage.ChatMessage{}, fmt.Errorf("chat body required")
	}
	if len([]rune(text)) > 1000 {
		return storage.ChatMessage{}, fmt.Errorf("chat body too long")
	}
	user, err := a.store.UserByID(ctx, viewerID)
	if err != nil {
		return storage.ChatMessage{}, err
	}
	members, err := a.store.RoomPlayers(ctx, roomID)
	if err != nil {
		return storage.ChatMessage{}, err
	}
	isMember := false
	for _, member := range members {
		if member.UserID == viewerID {
			isMember = true
			break
		}
	}
	if !isMember {
		return storage.ChatMessage{}, fmt.Errorf("chat requires room membership")
	}
	room, err := a.store.RoomByID(ctx, roomID)
	if err != nil {
		return storage.ChatMessage{}, err
	}
	return a.store.AddChatMessage(ctx, storage.AddChatMessageParams{RoomID: roomID, MatchID: room.CurrentMatchID, SenderUserID: viewerID, DisplayName: user.DisplayName, Body: text})
}

func settingsFromMessage(msg map[string]any) (game.Settings, error) {
	raw, ok := msg["settings"]
	if !ok {
		return game.Settings{}, fmt.Errorf("missing settings")
	}
	payload, err := json.Marshal(raw)
	if err != nil {
		return game.Settings{}, err
	}
	var settings game.Settings
	if err := json.Unmarshal(payload, &settings); err != nil {
		return game.Settings{}, err
	}
	return settings, nil
}

func commandFromMessage(t string, msg map[string]any) (game.Command, error) {
	switch t {
	case "setTeam", "assignTeam":
		playerID, _ := msg["playerId"].(string)
		if playerID == "" {
			playerID, _ = msg["actorId"].(string)
		}
		team, _ := msg["team"].(string)
		return game.AssignTeamCommand{PlayerID: playerID, Team: game.Team(team)}, nil
	case "toggleSpymaster":
		playerID, _ := msg["playerId"].(string)
		return game.ToggleSpymasterCommand{PlayerID: playerID}, nil
	case "toggleRepresentative":
		playerID, _ := msg["playerId"].(string)
		return game.ToggleRepresentativeCommand{PlayerID: playerID}, nil
	case "toggleMod":
		playerID, _ := msg["playerId"].(string)
		return game.ToggleModCommand{PlayerID: playerID}, nil
	case "randomizeTeams":
		return game.RandomizeTeamsCommand{}, nil
	case "guessCard":
		idx := int(number(msg["index"]))
		return game.GuessCommand{Index: idx}, nil
	case "passTurn":
		return game.PassCommand{}, nil
	case "submitClue":
		text, _ := msg["text"].(string)
		return game.SubmitClueCommand{Text: text, Number: clueNumber(msg["number"])}, nil
	case "shuffleRoles":
		return game.ShuffleRolesCommand{}, nil
	case "resetClue":
		return game.ResetClueCommand{}, nil
	case "restartMatch":
		return game.RestartMatchCommand{}, nil
	default:
		return nil, fmt.Errorf("unknown command type %q", t)
	}
}

func (rt *roomRuntime) broadcastLocked(msg map[string]any) {
	for conn, viewerID := range rt.clients {
		if msg["type"] == "snapshot" {
			msg = snapshotMessage(rt.state, viewerID)
		}
		if err := conn.WriteJSON(msg); err != nil {
			_ = conn.Close()
			delete(rt.clients, conn)
		}
	}
}

func (a *app) runtime(roomID string) *roomRuntime {
	a.mu.Lock()
	defer a.mu.Unlock()
	rt := a.rooms[roomID]
	if rt == nil {
		rt = &roomRuntime{clients: map[*websocket.Conn]string{}}
		a.rooms[roomID] = rt
	}
	return rt
}

func (a *app) loadRuntime(ctx context.Context, roomID string) (*roomRuntime, error) {
	rt := a.runtime(roomID)
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.state.HostID != "" {
		return rt, nil
	}
	room, err := a.store.RoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room.CurrentMatchID != "" {
		if snap, err := a.store.LatestSnapshot(ctx, room.CurrentMatchID); err == nil {
			if err := json.Unmarshal([]byte(snap.StateJSON), &rt.state); err == nil {
				return rt, nil
			}
		}
	}
	state := game.NewLobby(room.HostUserID, mustSettings(room.SettingsJSON))
	players, _ := a.store.RoomPlayers(ctx, roomID)
	for _, p := range players {
		u, _ := a.store.UserByID(ctx, p.UserID)
		state.Players[p.UserID] = game.Player{ID: p.UserID, DisplayName: u.DisplayName, Team: game.Team(p.Team), Spymaster: p.Spymaster, Representative: p.Representative, Mod: p.Mod || p.UserID == room.HostUserID}
	}
	rt.state = state
	return rt, nil
}

func (a *app) persistState(ctx context.Context, matchID, actorID, eventType string, state game.State) error {
	payload, _ := json.Marshal(map[string]any{"at": time.Now().UTC().Format(time.RFC3339Nano)})
	event, err := a.store.AppendGameEvent(ctx, storage.AppendGameEventParams{MatchID: matchID, ActorUserID: actorID, EventType: eventType, PayloadJSON: string(payload)})
	if err != nil {
		return err
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return a.store.SaveSnapshot(ctx, storage.SaveSnapshotParams{MatchID: matchID, LatestSequence: event.Sequence, StateJSON: string(stateJSON)})
}

func (a *app) syncRoomPlayers(ctx context.Context, roomID string, state game.State) error {
	for _, player := range state.Players {
		if err := a.store.UpsertRoomPlayer(ctx, storage.RoomPlayer{
			RoomID:         roomID,
			UserID:         player.ID,
			Team:           string(player.Team),
			Spymaster:      player.Spymaster,
			Representative: player.Representative,
			Mod:            player.Mod || player.ID == state.HostID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) authUser(ctx context.Context, token string) (storage.User, error) {
	res, err := a.identity.Bootstrap(ctx, token, "")
	if err != nil {
		return storage.User{}, err
	}
	return a.store.UserByID(ctx, res.UserID)
}
func (a *app) requireHost(w http.ResponseWriter, ctx context.Context, roomID, token string) (storage.Room, storage.User, bool) {
	room, user, ok := a.requireMember(w, ctx, roomID, token)
	if !ok {
		return room, user, false
	}
	if room.HostUserID != user.ID {
		writeError(w, http.StatusForbidden, "forbidden", "host only")
		return room, user, false
	}
	return room, user, true
}
func (a *app) requireMod(w http.ResponseWriter, ctx context.Context, roomID, token string) (storage.Room, storage.User, bool) {
	room, user, ok := a.requireMember(w, ctx, roomID, token)
	if !ok {
		return room, user, false
	}
	players, err := a.store.RoomPlayers(ctx, roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "players_failed", err.Error())
		return room, user, false
	}
	if !viewerIsMod(players, room.HostUserID, user.ID) {
		writeError(w, http.StatusForbidden, "forbidden", "moderator only")
		return room, user, false
	}
	return room, user, true
}
func (a *app) requireMember(w http.ResponseWriter, ctx context.Context, roomID, token string) (storage.Room, storage.User, bool) {
	room, err := a.store.RoomByID(ctx, roomID)
	if err != nil {
		writeStorageErr(w, err, "room_not_found")
		return storage.Room{}, storage.User{}, false
	}
	user, err := a.authUser(ctx, token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return room, storage.User{}, false
	}
	return room, user, true
}
func (a *app) viewerID(ctx context.Context, roomID string, r *http.Request) (string, error) {
	if r.URL.Query().Get("spectator") == "1" {
		return "", nil
	}
	if m := r.URL.Query().Get("migrateId"); m != "" {
		link, err := a.identity.ResolveMigrate(ctx, roomID, m)
		if err != nil {
			return "", err
		}
		return link.UserID, nil
	}
	user, err := a.authUser(ctx, tokenFromRequest(r))
	if err != nil {
		return "", err
	}
	return user.ID, nil
}
func (a *app) readyOrError(w http.ResponseWriter) bool {
	if err := a.requireReady(); err != nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", err.Error())
		return false
	}
	return true
}

func normalizeSettings(s game.Settings) game.Settings {
	if s.WordpackID == "" {
		s.WordpackID = "english"
	}
	return s
}
func mustSettings(raw string) game.Settings {
	var s game.Settings
	_ = json.Unmarshal([]byte(raw), &s)
	return normalizeSettings(s)
}
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	if q := r.URL.Query().Get("authToken"); q != "" {
		return q
	}
	return r.URL.Query().Get("token")
}
func (a *app) roomLink(r *http.Request, roomID string) string {
	base := a.baseURL
	if base == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		base = scheme + "://" + r.Host
	}
	return base + "/rooms/" + roomID
}

func roomDTO(r storage.Room) map[string]any {
	return map[string]any{"id": r.ID, "hostUserId": r.HostUserID, "status": r.Status, "currentMatchId": r.CurrentMatchID}
}
func viewerDTO(userID string, host bool, mod bool) map[string]any {
	return map[string]any{"userId": userID, "isHost": host, "isMod": mod || host}
}
func playerDTOs(players []storage.RoomPlayer) []map[string]any {
	out := make([]map[string]any, len(players))
	for i, p := range players {
		out[i] = map[string]any{"userId": p.UserID, "id": p.UserID, "team": p.Team, "spymaster": p.Spymaster, "representative": p.Representative, "mod": p.Mod}
	}
	return out
}
func viewerIsMod(players []storage.RoomPlayer, hostID, viewerID string) bool {
	if viewerID == "" {
		return false
	}
	if viewerID == hostID {
		return true
	}
	for _, p := range players {
		if p.UserID == viewerID {
			return p.Mod
		}
	}
	return false
}
func chatDTO(m storage.ChatMessage) map[string]any {
	return map[string]any{"id": m.ID, "roomId": m.RoomID, "matchId": m.MatchID, "senderUserId": m.SenderUserID, "displayName": m.DisplayName, "body": m.Body, "createdAt": m.CreatedAt}
}
func chatDTOs(messages []storage.ChatMessage) []map[string]any {
	out := make([]map[string]any, len(messages))
	for i, message := range messages {
		out[i] = chatDTO(message)
	}
	return out
}
func (a *app) pictureIDs() []string {
	if a.pictures == nil {
		return nil
	}
	ids := make([]string, len(a.pictures.ids))
	copy(ids, a.pictures.ids)
	return ids
}

func (a *app) pictureIDsForStart(ctx context.Context, settings game.Settings) ([]string, error) {
	if a.pictures == nil {
		return nil, nil
	}
	imageCount := settings.ImageCardCount
	if imageCount < 0 {
		imageCount = 0
	}
	if imageCount > game.BoardSize {
		imageCount = game.BoardSize
	}
	if imageCount == 0 {
		return a.pictureIDs(), nil
	}
	if a.pictures.diag.ProcessAVIF {
		return game.ShuffledImageIDs(settings, a.pictureIDs()), nil
	}
	selected := a.pictures.cachedImageIDsForStart(ctx, settings, imageCount)
	if len(selected) < imageCount {
		return nil, game.ErrNotEnoughImages
	}
	return selected, nil
}

const cacheExistenceBatchSize = 32
const cacheExistenceParallelism = 16

func (c *pictureCatalog) cachedImageIDsForStart(ctx context.Context, settings game.Settings, needed int) []string {
	if c == nil || needed <= 0 {
		return nil
	}
	candidates := shuffledStrings(settings, c.sourcePaths)
	selected := make([]string, 0, needed)
	selectedSeen := map[string]bool{}
	for start := 0; start < len(candidates) && len(selected) < needed; start += cacheExistenceBatchSize {
		end := start + cacheExistenceBatchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		results := c.cachedSourceImageBatch(ctx, candidates[start:end])
		for _, path := range candidates[start:end] {
			if id := results[path]; id != "" {
				if selectedSeen[id] {
					continue
				}
				selectedSeen[id] = true
				selected = append(selected, id)
				if len(selected) == needed {
					break
				}
			}
		}
	}
	return selected
}

func (c *pictureCatalog) cachedSourceImageBatch(ctx context.Context, paths []string) map[string]string {
	results := make(map[string]string, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, cacheExistenceParallelism)
	seen := map[string]bool{}
	for _, path := range paths {
		path := path
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			bytes, err := os.ReadFile(path)
			if err != nil {
				return
			}
			id := legacyImageID(bytes)
			cachePath := filepath.Join(c.cacheDir, id+".avif")
			info, err := os.Stat(cachePath)
			if err != nil || info.IsDir() {
				return
			}
			mu.Lock()
			if !seen[id] {
				seen[id] = true
				results[path] = id
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}

func shuffledStrings(settings game.Settings, values []string) []string {
	return game.ShuffledImageIDs(settings, values)
}

func snapshotMessage(state game.State, viewerID string) map[string]any {
	return map[string]any{"type": "snapshot", "snapshot": snapshotDTO(state, viewerID)}
}
func errorMessage(code, message string) map[string]any {
	return map[string]any{"type": "error", "code": code, "message": message}
}

func snapshotDTO(state game.State, viewerID string) map[string]any {
	s := state.SnapshotFor(game.Viewer{PlayerID: viewerID})
	players := make([]map[string]any, 0, len(state.Players))
	for _, p := range state.Players {
		players = append(players, map[string]any{"id": p.ID, "displayName": p.DisplayName, "team": p.Team, "spymaster": p.Spymaster, "representative": p.Representative, "mod": p.Mod || p.ID == state.HostID})
	}
	sort.Slice(players, func(i, j int) bool { return players[i]["id"].(string) < players[j]["id"].(string) })
	cards := make([]map[string]any, len(s.Cards))
	remaining := map[string]int{"blue": 0, "red": 0, "civilian": 0, "black": 0}
	for i, c := range s.Cards {
		card := map[string]any{"contentType": c.Content.Type, "revealed": c.Revealed}
		if c.Content.Type == game.ContentWord {
			card["word"] = c.Content.Text
		} else {
			card["imageId"] = c.Content.ImageID
		}
		if c.Color != "" {
			card["color"] = c.Color
		}
		cards[i] = card
	}
	for _, c := range state.Cards {
		if !c.Revealed {
			remaining[string(c.Color)]++
		}
	}
	return map[string]any{"phase": s.Phase, "players": players, "settings": state.Settings, "currentTeam": s.CurrentTeam, "winner": s.Winner, "actionId": s.ActionID, "cards": cards, "lastSelected": s.LastSelected, "remainingCounts": remaining, "clueLog": s.ClueLog, "viewer": map[string]any{"playerId": viewerID, "userId": viewerID, "isHost": viewerID != "" && viewerID == state.HostID, "isMod": state.CanManage(viewerID)}}
}

func clueNumber(v any) game.ClueNumber {
	switch x := v.(type) {
	case string:
		if x == "∞" || x == "infinity" {
			return game.ClueNumber{Kind: game.ClueNumberInfinity}
		}
		n, _ := strconv.Atoi(x)
		if n > 0 {
			return game.ClueNumber{Kind: game.ClueNumberNumeric, Value: n}
		}
		return game.ClueNumber{Kind: game.ClueNumberBlank}
	case float64:
		return game.ClueNumber{Kind: game.ClueNumberNumeric, Value: int(x)}
	default:
		return game.ClueNumber{Kind: game.ClueNumberBlank}
	}
}
func number(v any) float64 {
	if n, ok := v.(float64); ok {
		return n
	}
	return 0
}

func decodeRequest(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
func writeStorageErr(w http.ResponseWriter, err error, code string) {
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, code, "not found")
		return
	}
	writeError(w, http.StatusInternalServerError, code, err.Error())
}
func writeEngineErr(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, game.ErrForbidden) {
		status = http.StatusForbidden
	}
	writeError(w, status, "command_rejected", err.Error())
}
