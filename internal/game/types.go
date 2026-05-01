// Package game contains the pure Codewords game engine.
package game

import "errors"

const (
	// BoardSize is the number of cards in a Codewords match.
	BoardSize = 25
	// MaxBlackCards is the maximum supported assassin card count.
	MaxBlackCards = 8
)

var (
	// ErrForbidden means the actor is not allowed to perform the command.
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidCommand means a command payload is malformed or invalid for state.
	ErrInvalidCommand = errors.New("invalid command")
	// ErrNotEnoughWords means a wordpack cannot provide enough unique words.
	ErrNotEnoughWords = errors.New("not enough unique words")
	// ErrCannotStart means lobby requirements are not satisfied.
	ErrCannotStart = errors.New("cannot start match")
	// ErrInvalidClueNumber means a clue number is not valid for current settings.
	ErrInvalidClueNumber = errors.New("invalid clue number")
	// ErrClueRequired means enforced clue mode requires a submitted clue first.
	ErrClueRequired = errors.New("clue required")
	// ErrGuessLimitReached means the current clue's guess cap has been reached.
	ErrGuessLimitReached = errors.New("guess limit reached")
)

// Team identifies one of the two playable teams.
type Team string

const (
	TeamBlue Team = "blue"
	TeamRed  Team = "red"
)

// Opponent returns the opposing team.
func (t Team) Opponent() Team {
	if t == TeamBlue {
		return TeamRed
	}
	return TeamBlue
}

// Color returns the hidden card color corresponding to a team.
func (t Team) Color() Color {
	if t == TeamBlue {
		return ColorBlue
	}
	return ColorRed
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
	Type    ContentType
	Text    string
	ImageID string
}

// Card is an authoritative board card.
type Card struct {
	Content  CardContent
	Color    Color
	Revealed bool
}

// Settings are match/lobby options owned by the engine.
type Settings struct {
	Seed                  int64
	BlackCards            int
	WordpackID            string
	EnforceClueGuessLimit bool
	AllowInfinityClue     bool
}

// Player is the authoritative per-room player state.
type Player struct {
	ID             string
	DisplayName    string
	Team           Team
	Spymaster      bool
	Representative bool
}

// LastSelected records the most recent accepted guess.
type LastSelected struct {
	Index int
	Team  Team
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
	Kind  ClueNumberKind
	Value int
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
	Round       int
	Team        Team
	Text        string
	Number      ClueNumber
	Status      ClueStatus
	SubmittedBy string
	UpdatedBy   string
	Guesses     int
}

// State is the authoritative game engine state.
type State struct {
	HostID       string
	Settings     Settings
	Phase        Phase
	Players      map[string]Player
	Cards        []Card
	CurrentTeam  Team
	Winner       Team
	ActionID     int
	LastSelected *LastSelected
	ClueLog      []ClueEntry
	Round        int
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
	EventMatchStarted    EventType = "match_started"
	EventGuessAccepted   EventType = "guess_accepted"
	EventPassAccepted    EventType = "pass_accepted"
	EventClueSubmitted   EventType = "clue_submitted"
	EventClueFinalized   EventType = "clue_finalized"
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
	ActionID     int
	Cards        []SnapshotCard
	LastSelected *LastSelected
	ClueLog      []ClueEntry
}
