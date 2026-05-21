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
