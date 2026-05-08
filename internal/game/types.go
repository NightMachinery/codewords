// Package game contains the pure Codewords game engine.
package game

import (
	"encoding/json"
	"errors"
)

const (
	// BoardSize is the number of cards in a Codewords match.
	BoardSize = 25
	// DefaultTotalCards is the classic Codewords board size.
	DefaultTotalCards = 25
	// MinTotalCards is the minimum supported configurable board size.
	MinTotalCards = 9
	// MaxTotalCards is the maximum supported configurable board size.
	MaxTotalCards = 100
)

var (
	// ErrForbidden means the actor is not allowed to perform the command.
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidCommand means a command payload is malformed or invalid for state.
	ErrInvalidCommand = errors.New("invalid command")
	// ErrInvalidSettings means room or match settings are outside supported bounds.
	ErrInvalidSettings = errors.New("invalid settings")
	// ErrNotEnoughWords means a wordpack cannot provide enough unique words.
	ErrNotEnoughWords = errors.New("not enough unique words")
	// ErrNotEnoughImages means the local picture catalog cannot provide enough unique images.
	ErrNotEnoughImages = errors.New("not enough unique images")
	// ErrCannotStart means lobby requirements are not satisfied.
	ErrCannotStart = errors.New("cannot start match")
	// ErrInvalidClueNumber means a clue number is not valid for current settings.
	ErrInvalidClueNumber = errors.New("invalid clue number")
	// ErrClueRequired means enforced clue mode requires a submitted clue first.
	ErrClueRequired = errors.New("clue required")
	// ErrGuessLimitReached means the current clue's guess cap has been reached.
	ErrGuessLimitReached = errors.New("guess limit reached")
)

// Team identifies one of the two playable teams, or the neutral observers team.
type Team string

const (
	TeamBlue      Team = "blue"
	TeamRed       Team = "red"
	TeamObservers Team = "observers"
)

// Opponent returns the opposing team. Not valid for observers.
func (t Team) Opponent() Team {
	if t == TeamBlue {
		return TeamRed
	}
	if t == TeamRed {
		return TeamBlue
	}
	return ""
}

// Color returns the hidden card color corresponding to a team.
func (t Team) Color() Color {
	if t == TeamBlue {
		return ColorBlue
	}
	if t == TeamRed {
		return ColorRed
	}
	return ""
}

// Color identifies a card's hidden allegiance.
type Color string

const (
	ColorBlue     Color = "blue"
	ColorRed      Color = "red"
	ColorBlack    Color = "black"
	ColorCivilian Color = "civilian"
)

// Phase identifies the lifecycle phase of a game state.
type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhaseActive   Phase = "active"
	PhaseGameOver Phase = "game_over"
)

// ContentType identifies the kind of card content.
type ContentType string

const (
	ContentWord  ContentType = "word"
	ContentImage ContentType = "image"
)

// CardContent is the player-visible content of a card.
type CardContent struct {
	Type    ContentType `json:"type"`
	Text    string      `json:"text,omitempty"`
	ImageID string      `json:"imageId,omitempty"`
}

// Card is an authoritative board card.
type Card struct {
	Content  CardContent `json:"content"`
	Color    Color       `json:"color"`
	Revealed bool        `json:"revealed"`
}

// Settings are match/lobby options owned by the engine.
type Settings struct {
	Seed                    int64  `json:"seed"`
	BlackCards              int    `json:"blackCards"`
	TotalCards              int    `json:"totalCards"`
	AutoColorCounts         bool   `json:"autoColorCounts"`
	BlueCards               int    `json:"blueCards"`
	RedCards                int    `json:"redCards"`
	NeutralCards            int    `json:"neutralCards"`
	StartingTeamHandicap    int    `json:"startingTeamHandicap"`
	StartingTeamHandicapSet bool   `json:"-"`
	WordpackID              string `json:"wordpackId"`
	EnforceClueGuessLimit   bool   `json:"enforceClueGuessLimit"`
	AllowInfinityClue       bool   `json:"allowInfinityClue"`
	ImageCardCount          int    `json:"imageCardCount"`
	RandomizeTeams          bool   `json:"randomizeTeams"`
	CustomColorBlue         string `json:"customColorBlue,omitempty"`
	CustomColorRed          string `json:"customColorRed,omitempty"`
	ObserverChatEnabled     bool   `json:"observerChatEnabled"`
	MixedImageOrderFirst    bool   `json:"mixedImageOrderFirst"`
	MemoryRoastsDisabled    bool   `json:"memoryRoastsDisabled,omitempty"`
	TeamNameBlue            string `json:"teamNameBlue,omitempty"`
	TeamNameRed             string `json:"teamNameRed,omitempty"`
}

// UnmarshalJSON gives API/DB payloads the product default for randomized team
// assignment while still allowing clients to explicitly save false.
func (s *Settings) UnmarshalJSON(data []byte) error {
	type settingsAlias Settings
	next := settingsAlias{RandomizeTeams: true, ObserverChatEnabled: true, TotalCards: DefaultTotalCards, AutoColorCounts: true}
	if err := json.Unmarshal(data, &next); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		_, next.StartingTeamHandicapSet = raw["startingTeamHandicap"]
	}
	*s = Settings(next)
	return nil
}

// Player is the authoritative per-room player state.
type Player struct {
	ID                     string `json:"id"`
	DisplayName            string `json:"displayName"`
	Team                   Team   `json:"team"`
	Spymaster              bool   `json:"spymaster"`
	Representative         bool   `json:"representative"`
	Mod                    bool   `json:"mod"`
	PreviousTeam           Team   `json:"previousTeam,omitempty"`
	PreviousSpymaster      bool   `json:"previousSpymaster,omitempty"`
	PreviousRepresentative bool   `json:"previousRepresentative,omitempty"`
}

// LastSelected records the most recent accepted guess.
type LastSelected struct {
	Index int  `json:"index"`
	Team  Team `json:"team"`
}

// ClueNumberKind identifies clue number variants.
type ClueNumberKind string

const (
	ClueNumberBlank    ClueNumberKind = "blank"
	ClueNumberNumeric  ClueNumberKind = "numeric"
	ClueNumberInfinity ClueNumberKind = "infinity"
)

// ClueNumber is a blank, numeric, or infinity clue count.
type ClueNumber struct {
	Kind  ClueNumberKind `json:"kind"`
	Value int            `json:"value,omitempty"`
}

// ClueStatus identifies whether a clue row is active, finalized, or absent.
type ClueStatus string

const (
	ClueActive ClueStatus = "active"
	ClueFinal  ClueStatus = "final"
	ClueNA     ClueStatus = "na"
)

// ClueEntry is one row in the per-round clue log.
type ClueEntry struct {
	Round       int        `json:"round"`
	Team        Team       `json:"team"`
	Text        string     `json:"text"`
	Number      ClueNumber `json:"number"`
	Status      ClueStatus `json:"status"`
	SubmittedBy string     `json:"submittedBy,omitempty"`
	UpdatedBy   string     `json:"updatedBy,omitempty"`
	Guesses     int        `json:"guesses"`
}

// State is the authoritative game engine state.
type State struct {
	HostID       string            `json:"hostId"`
	Settings     Settings          `json:"settings"`
	Phase        Phase             `json:"phase"`
	Players      map[string]Player `json:"players"`
	Cards        []Card            `json:"cards"`
	CurrentTeam  Team              `json:"currentTeam"`
	Winner       Team              `json:"winner"`
	FinishedAt   string            `json:"finishedAt,omitempty"`
	ActionID     int               `json:"actionId"`
	LastSelected *LastSelected     `json:"lastSelected"`
	ClueLog      []ClueEntry       `json:"clueLog"`
	Round        int               `json:"round"`
}

// Board is a generated board plus starting team.
type Board struct {
	Cards        []Card
	StartingTeam Team
}

// EventType identifies accepted engine events.
type EventType string

const (
	EventPlayerAdded     EventType = "player_added"
	EventTeamAssigned    EventType = "team_assigned"
	EventRoleChanged     EventType = "role_changed"
	EventSettingsUpdated EventType = "settings_updated"
	EventModChanged      EventType = "mod_changed"
	EventMatchStarted    EventType = "match_started"
	EventGuessAccepted   EventType = "guess_accepted"
	EventPassAccepted    EventType = "pass_accepted"
	EventClueSubmitted   EventType = "clue_submitted"
	EventClueFinalized   EventType = "clue_finalized"
	EventTeamsRandomized EventType = "teams_randomized"
	EventRolesShuffled   EventType = "roles_shuffled"
	EventClueReset       EventType = "clue_reset"
	EventMatchRestarted  EventType = "match_restarted"
)

// Event is returned for an accepted command.
type Event struct {
	Type EventType
}

// Viewer identifies a snapshot recipient.
type Viewer struct {
	PlayerID string
}

// SnapshotCard is a card in a viewer-safe snapshot.
type SnapshotCard struct {
	Content  CardContent
	Color    Color
	Revealed bool
}

// Snapshot is a viewer-safe game state.
type Snapshot struct {
	Phase        Phase
	CurrentTeam  Team
	Winner       Team
	FinishedAt   string
	ActionID     int
	Cards        []SnapshotCard
	LastSelected *LastSelected
	ClueLog      []ClueEntry
}
