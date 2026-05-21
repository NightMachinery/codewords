package game

import (
	"errors"
	"testing"
)

func TestUnityStartGeneratesPersonalBoardsAndSafeSnapshots(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 51, TotalCards: 25, BlackCards: 0, UnityTurnLimit: 5})

	mustApply(t, &state, StartCommand{GameID: "game-1", Words: makeWords(80)}, "host")

	if state.Mode != ModeUnity || state.CurrentTeam != TeamUnity {
		t.Fatalf("expected active unity mode, got mode=%s team=%s", state.Mode, state.CurrentTeam)
	}
	if state.ActiveBoardOwner == "" {
		t.Fatalf("expected active unity board owner")
	}
	if len(state.UnityBoards) != 3 {
		t.Fatalf("expected one board per unity player, got %d", len(state.UnityBoards))
	}
	for ownerID, board := range state.UnityBoards {
		if board.OwnerID != ownerID {
			t.Fatalf("board owner mismatch: key=%s board=%#v", ownerID, board)
		}
		counts := countColors(board.Cards)
		if counts[ColorUnity] != 10 {
			t.Fatalf("expected round(25/2.5)=10 unity cards, got %#v", counts)
		}
		if counts[ColorBlack] != 4 {
			t.Fatalf("expected unity assassin default 4, got %#v", counts)
		}
		if len(board.Cards) != 25 {
			t.Fatalf("expected 25 cards, got %d", len(board.Cards))
		}
	}

	activeOwner := state.ActiveBoardOwner
	ownerView := state.SnapshotFor(Viewer{PlayerID: activeOwner})
	if ownerView.ActiveBoard.OwnerID != activeOwner {
		t.Fatalf("expected active board in snapshot, got %#v", ownerView.ActiveBoard)
	}
	if ownerView.ActiveBoard.Cards[0].Color == "" {
		t.Fatalf("active board owner should see hidden colors on own board")
	}
	other := "host"
	if other == activeOwner {
		other = "p2"
	}
	otherView := state.SnapshotFor(Viewer{PlayerID: other})
	if otherView.ActiveBoard.Cards[0].Color != "" {
		t.Fatalf("non-owner should not see active hidden color, got %q", otherView.ActiveBoard.Cards[0].Color)
	}
	if otherView.OwnBoard == nil || otherView.OwnBoard.OwnerID != other {
		t.Fatalf("unity player should receive own board snapshot, got %#v", otherView.OwnBoard)
	}
	if otherView.OwnBoard.Cards[0].Color == "" {
		t.Fatalf("own board should expose hidden colors to owner")
	}
}

func TestUnityDefaultsToSixTurnsPerBoard(t *testing.T) {
	settings := SettingsWithDefaults(Settings{Mode: ModeUnity, Seed: 510})

	if settings.UnityTurnLimit != 6 {
		t.Fatalf("expected Unity finite default to be 6 turns per board, got %d", settings.UnityTurnLimit)
	}
}

func TestUnityLobbyRejoinReturnsObserverToUnityWithoutLosingPolarityMetadata(t *testing.T) {
	state := NewLobby("host", Settings{Mode: ModeUnity, Seed: 511})
	mustApply(t, &state, AddPlayerCommand{PlayerID: "host", DisplayName: "Host"}, "host")
	state.Players["host"] = Player{
		ID:                     "host",
		DisplayName:            "Host",
		Team:                   TeamObservers,
		PreviousTeam:           TeamRed,
		PreviousSpymaster:      true,
		PreviousRepresentative: false,
		Mod:                    true,
	}

	mustApply(t, &state, RejoinTeamCommand{PlayerID: "host"}, "host")

	player := state.Players["host"]
	if player.Team != TeamUnity {
		t.Fatalf("expected Unity-mode rejoin to assign Unity, got %#v", player)
	}
	if player.PreviousTeam != TeamRed || !player.PreviousSpymaster {
		t.Fatalf("expected retained Polarity metadata, got %#v", player)
	}
	if player.Spymaster || player.Representative {
		t.Fatalf("Unity rejoin should not restore Polarity roles onto Unity, got %#v", player)
	}
}

func TestUnityLobbyAddsNewPlayersToUnityTeam(t *testing.T) {
	state := NewLobby("host", Settings{Mode: ModeUnity, Seed: 514, RandomizeTeams: true})

	mustApply(t, &state, AddPlayerCommand{PlayerID: "guest", DisplayName: "Guest"}, "guest")

	if state.Players["guest"].Team != TeamUnity {
		t.Fatalf("expected new Unity lobby player to join Unity, got %#v", state.Players["guest"])
	}
}

func TestUnityTracksLastSelectedPerBoard(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 512, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-last-selected", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorUnity})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity, ColorUnity})

	mustApply(t, &state, GuessCommand{Index: 0}, "p2")

	hostBoard := state.SnapshotFor(Viewer{PlayerID: "p2"}).ActiveBoard
	if hostBoard.LastSelected == nil || hostBoard.LastSelected.Index != 0 || hostBoard.LastSelected.Team != TeamUnity {
		t.Fatalf("expected host board last-selected index 0, got %#v", hostBoard.LastSelected)
	}
	ownBoard := state.SnapshotFor(Viewer{PlayerID: "p2"}).OwnBoard
	if ownBoard == nil {
		t.Fatalf("expected p2 own board")
	}
	if ownBoard.LastSelected != nil {
		t.Fatalf("p2 board should not inherit host board last-selected, got %#v", ownBoard.LastSelected)
	}
}

func TestUnityUsesDistinctImagesAcrossBoardsWhenEnoughImagesExist(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 513, TotalCards: 9, ImageCardCount: 2, UnityTurnLimit: 6})
	mustApply(t, &state, StartCommand{GameID: "game-images", Words: makeWords(40), ImageIDs: []string{"img-a", "img-b", "img-c", "img-d", "img-e", "img-f"}}, "host")

	seen := map[string]string{}
	for ownerID, board := range state.UnityBoards {
		for _, card := range board.Cards {
			if card.Content.Type != ContentImage {
				continue
			}
			if previousOwner, ok := seen[card.Content.ImageID]; ok {
				t.Fatalf("image %s duplicated on owners %s and %s", card.Content.ImageID, previousOwner, ownerID)
			}
			seen[card.Content.ImageID] = ownerID
		}
	}
	if len(seen) != 6 {
		t.Fatalf("expected six distinct image cards across Unity boards, got %#v", seen)
	}
}

func TestUnityRepresentativeGuessingRules(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 52, UnityTurnLimit: 5})
	mustApply(t, &state, StartCommand{GameID: "game-2", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"

	if !state.IsActiveGuesser("p2", TeamUnity) || !state.IsActiveGuesser("p3", TeamUnity) {
		t.Fatalf("without reps, all non-owner players should guess")
	}
	if state.IsActiveGuesser("host", TeamUnity) {
		t.Fatalf("active board owner should not guess")
	}

	mustApply(t, &state, ToggleRepresentativeCommand{PlayerID: "p2"}, "host")
	if !state.IsActiveGuesser("p2", TeamUnity) {
		t.Fatalf("single non-owner rep should guess")
	}
	if state.IsActiveGuesser("p3", TeamUnity) {
		t.Fatalf("non-rep should not guess when one non-owner rep exists")
	}

	mustApply(t, &state, ToggleRepresentativeCommand{PlayerID: "p2"}, "host")
	mustApply(t, &state, ToggleRepresentativeCommand{PlayerID: "host"}, "host")
	if !state.IsActiveGuesser("p2", TeamUnity) || !state.IsActiveGuesser("p3", TeamUnity) {
		t.Fatalf("if the only rep is active owner, all other players should guess")
	}

	mustApply(t, &state, ToggleRepresentativeCommand{PlayerID: "p2"}, "host")
	if !state.IsActiveGuesser("p2", TeamUnity) || state.IsActiveGuesser("p3", TeamUnity) || state.IsActiveGuesser("host", TeamUnity) {
		t.Fatalf("with two reps, only non-owner reps should guess")
	}
}

func TestUnitySharedPoolObserverLedgerAndRejoin(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 53, UnityTurnLimit: 2})
	mustApply(t, &state, StartCommand{GameID: "game-3", Words: makeWords(80)}, "host")
	if state.UnitySharedTurnsRemaining != 5 {
		t.Fatalf("expected first active board to spend from 3 boards * 2 shared turns, got %d", state.UnitySharedTurnsRemaining)
	}

	state.ActiveBoardOwner = "host"
	state.UnitySharedTurnsRemaining = 1
	p2Board := state.UnityBoards["p2"]
	p2Board.TurnsUsed = 0
	state.UnityBoards["p2"] = p2Board
	mustApply(t, &state, AssignTeamCommand{PlayerID: "p2", Team: TeamObservers}, "host")

	if state.UnitySharedTurnsRemaining != 0 {
		t.Fatalf("observer withdrawal should clamp shared pool at zero, got %d", state.UnitySharedTurnsRemaining)
	}
	if state.Phase != PhaseGameOver || state.Winner != TeamObservers {
		t.Fatalf("bankrupt shared pool with unfinished active boards should lose, phase=%s winner=%s", state.Phase, state.Winner)
	}

	state = unityLobby(t, Settings{Mode: ModeUnity, Seed: 54, UnityTurnLimit: 2})
	mustApply(t, &state, StartCommand{GameID: "game-4", Words: makeWords(80)}, "host")
	state.UnitySharedTurnsRemaining = 3
	mustApply(t, &state, AssignTeamCommand{PlayerID: "p2", Team: TeamObservers}, "host")
	if state.UnityBoards["p2"].WithdrawnSharedTurns != 2 {
		t.Fatalf("expected p2 contribution to be tracked, got %#v", state.UnityBoards["p2"])
	}
	mustApply(t, &state, RejoinTeamCommand{PlayerID: "p2"}, "host")
	if state.Players["p2"].Team != TeamUnity {
		t.Fatalf("expected p2 to rejoin unity, got %#v", state.Players["p2"])
	}
	if state.UnitySharedTurnsRemaining != 3 {
		t.Fatalf("expected withdrawn contribution restored, got %d", state.UnitySharedTurnsRemaining)
	}
	if state.UnityBoards["p2"].WithdrawnSharedTurns != 0 {
		t.Fatalf("expected withdrawn contribution cleared, got %#v", state.UnityBoards["p2"])
	}
}

func TestUnityGuessPassRotateWinAndAssassinLoss(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 55, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-5", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorCivilian, ColorBlack})

	mustApply(t, &state, GuessCommand{Index: 0}, "p2")
	if state.ActiveBoardOwner != "host" || state.UnityBoards["host"].Cards[0].Revealed != true {
		t.Fatalf("correct unity guess should keep active board")
	}
	mustApply(t, &state, GuessCommand{Index: 1}, "p2")
	if state.ActiveBoardOwner == "host" {
		t.Fatalf("civilian should rotate away from host")
	}

	state.ActiveBoardOwner = "p2"
	setUnityBoardColors(&state, "p2", []Color{ColorBlack})
	mustApply(t, &state, GuessCommand{Index: 0}, "host")
	if state.Phase != PhaseGameOver || state.Winner != TeamObservers {
		t.Fatalf("assassin should be global unity loss, phase=%s winner=%s", state.Phase, state.Winner)
	}

	state = unityLobby(t, Settings{Mode: ModeUnity, Seed: 56, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-6", Words: makeWords(80)}, "host")
	for ownerID, board := range state.UnityBoards {
		for i := range board.Cards {
			if board.Cards[i].Color == ColorUnity {
				board.Cards[i].Revealed = true
			}
		}
		state.UnityBoards[ownerID] = board
	}
	state.ResolveUnityAfterRosterChange()
	if state.Phase != PhaseGameOver || state.Winner != TeamUnity {
		t.Fatalf("all unity cards revealed should win, phase=%s winner=%s", state.Phase, state.Winner)
	}
}

func TestUnityStrictPerBoardLimitLosesAfterKthTurnEnds(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 57, UnityTurnLimit: 1, UnityStrictPerBoardTurns: true})
	mustApply(t, &state, StartCommand{GameID: "game-7", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	setUnityBoardColors(&state, "host", []Color{ColorCivilian, ColorUnity})
	state.UnityBoards["host"] = UnityBoardState{OwnerID: "host", Cards: state.UnityBoards["host"].Cards, TurnsUsed: 1}

	mustApply(t, &state, GuessCommand{Index: 0}, "p2")
	if state.Phase != PhaseGameOver || state.Winner != TeamObservers {
		t.Fatalf("strict per-board limit should lose after Kth unfinished turn, phase=%s winner=%s", state.Phase, state.Winner)
	}
}

func TestUnityActiveOwnerCannotProceedWithoutEligibleGuessers(t *testing.T) {
	state := NewLobby("host", Settings{Mode: ModeUnity, Seed: 58, UnityTurnLimit: 3})
	mustApply(t, &state, AddPlayerCommand{PlayerID: "host", DisplayName: "Host"}, "host")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "host", Team: TeamUnity}, "host")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "p2", DisplayName: "P2"}, "p2")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "p2", Team: TeamUnity}, "host")
	mustApply(t, &state, StartCommand{GameID: "game-8", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	mustApply(t, &state, AssignTeamCommand{PlayerID: "p2", Team: TeamObservers}, "host")

	if _, err := Apply(&state, PassCommand{}, "host"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("active owner should not pass without eligible guessers, got %v", err)
	}
	if state.UnityWaitingForGuessers != true {
		t.Fatalf("expected unity waiting state")
	}
}

func unityLobby(t *testing.T, settings Settings) State {
	t.Helper()
	state := NewLobby("host", settings)
	for _, player := range []string{"host", "p2", "p3"} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: player, DisplayName: player}, player)
		mustApply(t, &state, AssignTeamCommand{PlayerID: player, Team: TeamUnity}, "host")
	}
	return state
}

func setUnityBoardColors(state *State, ownerID string, colors []Color) {
	board := state.UnityBoards[ownerID]
	for i, color := range colors {
		board.Cards[i].Color = color
		board.Cards[i].Revealed = false
	}
	for i := len(colors); i < len(board.Cards); i++ {
		board.Cards[i].Color = ColorCivilian
		board.Cards[i].Revealed = false
	}
	state.UnityBoards[ownerID] = board
}
