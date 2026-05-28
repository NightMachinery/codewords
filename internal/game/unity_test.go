package game

import (
	"errors"
	"slices"
	"testing"
	"time"
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

func TestUnityRestartClearsRuntimeBoardState(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 515, UnityTurnLimit: 4})

	mustApply(t, &state, StartCommand{GameID: "game-515", Words: makeWords(80)}, "host")
	if state.ActiveBoardOwner == "" || len(state.UnityBoards) == 0 {
		t.Fatalf("expected active unity runtime state before restart")
	}

	mustApply(t, &state, RestartMatchCommand{}, "host")

	if state.Phase != PhaseLobby {
		t.Fatalf("expected restart to return to lobby, got %s", state.Phase)
	}
	if state.Mode != ModeUnity || state.Settings.Mode != ModeUnity {
		t.Fatalf("restart should preserve selected unity mode, got mode=%s settings=%s", state.Mode, state.Settings.Mode)
	}
	if state.ActiveBoardOwner != "" {
		t.Fatalf("expected restart to clear active board owner, got %q", state.ActiveBoardOwner)
	}
	if len(state.UnityBoards) != 0 || len(state.UnityBoardOrder) != 0 {
		t.Fatalf("expected restart to clear unity boards, got boards=%d order=%d", len(state.UnityBoards), len(state.UnityBoardOrder))
	}
	if state.UnitySharedTurnsRemaining != 0 || state.UnityWaitingForGuessers {
		t.Fatalf("expected restart to clear unity progress, remaining=%d waiting=%v", state.UnitySharedTurnsRemaining, state.UnityWaitingForGuessers)
	}
	if len(state.UnityWords) != 0 || len(state.UnityImageIDs) != 0 || state.UnityEndStats != nil || state.GameID != "" {
		t.Fatalf("expected restart to clear unity match payloads, gameID=%q words=%d images=%d stats=%#v", state.GameID, len(state.UnityWords), len(state.UnityImageIDs), state.UnityEndStats)
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

func TestUnityRequiresEnoughImagesForAllActiveBoards(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 516, TotalCards: 9, ImageCardCount: 2, UnityTurnLimit: 6})

	if _, err := Apply(&state, StartCommand{GameID: "game-images-short", Words: makeWords(40), ImageIDs: []string{"img-a", "img-b", "img-c", "img-d", "img-e"}}, "host"); !errors.Is(err, ErrNotEnoughImages) {
		t.Fatalf("expected not enough images for three Unity boards, got %v", err)
	}
}

func TestUnityMidGameJoinUsesUnusedImages(t *testing.T) {
	state := NewLobby("host", Settings{Mode: ModeUnity, Seed: 517, TotalCards: 9, ImageCardCount: 2, UnityTurnLimit: 6})
	for _, player := range []string{"host", "p2"} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: player, DisplayName: player}, player)
		mustApply(t, &state, AssignTeamCommand{PlayerID: player, Team: TeamUnity}, "host")
	}
	mustApply(t, &state, StartCommand{GameID: "game-mid-images", Words: makeWords(40), ImageIDs: []string{"img-a", "img-b", "img-c", "img-d", "img-e", "img-f"}}, "host")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "late", DisplayName: "Late"}, "late")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "late", Team: TeamObservers}, "host")

	mustApply(t, &state, AssignTeamCommand{PlayerID: "late", Team: TeamUnity}, "host")

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
		t.Fatalf("expected six unique images after late join, got %#v", seen)
	}
}

func TestUnityMidGameJoinRejectsWhenUnusedImagesAreUnavailable(t *testing.T) {
	state := NewLobby("host", Settings{Mode: ModeUnity, Seed: 518, TotalCards: 9, ImageCardCount: 2, UnityTurnLimit: 6})
	for _, player := range []string{"host", "p2"} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: player, DisplayName: player}, player)
		mustApply(t, &state, AssignTeamCommand{PlayerID: player, Team: TeamUnity}, "host")
	}
	mustApply(t, &state, StartCommand{GameID: "game-mid-images-short", Words: makeWords(40), ImageIDs: []string{"img-a", "img-b", "img-c", "img-d"}}, "host")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "late", DisplayName: "Late"}, "late")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "late", Team: TeamObservers}, "host")

	if _, err := Apply(&state, AssignTeamCommand{PlayerID: "late", Team: TeamUnity}, "host"); !errors.Is(err, ErrNotEnoughImages) {
		t.Fatalf("expected not enough unused images, got %v", err)
	}
	if state.Players["late"].Team != TeamObservers {
		t.Fatalf("failed Unity assignment should leave late player as observer, got %#v", state.Players["late"])
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
	tempRep := state.UnityTemporaryRepresentativeID()
	if tempRep != "p2" && tempRep != "p3" {
		t.Fatalf("expected one non-owner temp rep, got %q", tempRep)
	}
	for _, id := range []string{"p2", "p3"} {
		if state.IsActiveGuesser(id, TeamUnity) != (id == tempRep) {
			t.Fatalf("if only rep is active owner, only selected temp rep should guess; id=%s temp=%s", id, tempRep)
		}
	}
	if state.SnapshotFor(Viewer{PlayerID: "p2"}).UnityProgress.TemporaryRepresentativeID != tempRep {
		t.Fatalf("snapshot should expose temp rep %q", tempRep)
	}

	mustApply(t, &state, ToggleRepresentativeCommand{PlayerID: "p2"}, "host")
	if !state.IsActiveGuesser("p2", TeamUnity) || state.IsActiveGuesser("p3", TeamUnity) || state.IsActiveGuesser("host", TeamUnity) {
		t.Fatalf("with two reps, only non-owner reps should guess")
	}
}

func TestUnitySharedPoolObserverLedgerAndRejoin(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 53, UnityTurnLimit: 2})
	mustApply(t, &state, StartCommand{GameID: "game-3", Words: makeWords(80)}, "host")
	if state.UnitySharedTurnsRemaining != 6 {
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

func TestUnityTurnsChargeOnlyWhenTurnFinalizesWithActivity(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 530, UnityTurnLimit: 2})
	mustApply(t, &state, StartCommand{GameID: "game-charge-finalize", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	state.UnitySharedTurnsRemaining = 6
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorCivilian})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity})

	mustApply(t, &state, SwitchUnitySpymasterCommand{Now: time.Unix(100, 0)}, "host")
	if state.UnityBoards["host"].TurnsUsed != 0 || state.UnitySharedTurnsRemaining != 6 {
		t.Fatalf("empty spy switch should not spend a turn, turns=%d shared=%d", state.UnityBoards["host"].TurnsUsed, state.UnitySharedTurnsRemaining)
	}

	state.UnityTransitionUntil = ""
	state.ActiveBoardOwner = "host"
	mustApply(t, &state, SubmitClueCommand{Text: "bridge", Number: ClueNumber{Kind: ClueNumberBlank}}, "host")
	mustApply(t, &state, SwitchUnitySpymasterCommand{Now: time.Unix(101, 0)}, "host")
	if state.UnityBoards["host"].TurnsUsed != 1 || state.UnitySharedTurnsRemaining != 5 {
		t.Fatalf("spy switch after clue should spend current board turn, turns=%d shared=%d", state.UnityBoards["host"].TurnsUsed, state.UnitySharedTurnsRemaining)
	}
}

func TestUnitySwitchSpymasterIsModOnlyAndCreatesTransition(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 531, UnityTurnLimit: 3})
	mustApply(t, &state, StartCommand{GameID: "game-switch-spy", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	setUnityBoardColors(&state, "host", []Color{ColorUnity})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity})
	setUnityBoardColors(&state, "p3", []Color{ColorUnity})

	if _, err := Apply(&state, SwitchUnitySpymasterCommand{Now: time.Unix(200, 0)}, "p2"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected non-mod switch to be forbidden, got %v", err)
	}
	mustApply(t, &state, SwitchUnitySpymasterCommand{Now: time.Unix(200, 0)}, "host")

	if state.PreviousBoardOwner != "host" || state.ActiveBoardOwner != "p2" {
		t.Fatalf("expected switch from host to p2, previous=%s active=%s", state.PreviousBoardOwner, state.ActiveBoardOwner)
	}
	if state.UnityTransitionUntil == "" {
		t.Fatalf("expected transition deadline to be set")
	}
	if state.Round == 0 {
		t.Fatalf("expected next board round to be active")
	}
}

func TestUnityTransitionRejectsGameplayActions(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 532, UnityTurnLimit: 3})
	mustApply(t, &state, StartCommand{GameID: "game-transition-lock", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	setUnityBoardColors(&state, "host", []Color{ColorCivilian})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity})
	setUnityBoardColors(&state, "p3", []Color{ColorUnity})

	mustApply(t, &state, GuessCommand{Index: 0, Now: time.Unix(300, 0)}, "p2")
	if state.UnityTransitionUntil == "" || state.PreviousBoardOwner != "host" {
		t.Fatalf("expected civilian reveal to create transition, previous=%s until=%q", state.PreviousBoardOwner, state.UnityTransitionUntil)
	}
	if _, err := Apply(&state, GuessCommand{Index: 0, Now: time.Unix(305, 0)}, "host"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected transition to reject guesses, got %v", err)
	}
	if _, err := Apply(&state, PassCommand{Now: time.Unix(305, 0)}, "host"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected transition to reject passes, got %v", err)
	}
	if _, err := Apply(&state, SubmitClueCommand{Text: "late", Number: ClueNumber{Kind: ClueNumberBlank}, Now: time.Unix(305, 0)}, state.ActiveBoardOwner); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected transition to reject clues, got %v", err)
	}
	if _, err := Apply(&state, SwitchUnitySpymasterCommand{Now: time.Unix(305, 0)}, "host"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected transition to reject spy switches, got %v", err)
	}

	if _, err := Apply(&state, GuessCommand{Index: 0, Now: time.Unix(311, 0)}, "host"); err != nil {
		t.Fatalf("expected guess after transition to be accepted, got %v", err)
	}
}

func TestUnityGuessPassRotateWinAndAssassinLoss(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 55, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-5", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorCivilian, ColorBlack, ColorUnity})

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

func TestUnitySolvingOneBoardAutoRotatesWithoutExtraSharedTurn(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 560, UnityTurnLimit: 4})
	mustApply(t, &state, StartCommand{GameID: "game-auto-rotate", Words: makeWords(80)}, "host")
	state.ActiveBoardOwner = "host"
	state.UnitySharedTurnsRemaining = 6
	setUnityBoardColors(&state, "host", []Color{ColorUnity})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity})
	setUnityBoardColors(&state, "p3", []Color{})
	hostTurns := state.UnityBoards["host"].TurnsUsed

	mustApply(t, &state, GuessCommand{Index: 0}, "p2")

	if state.Phase != PhaseActive {
		t.Fatalf("expected match to stay active after solving only host board, phase=%s winner=%s", state.Phase, state.Winner)
	}
	if state.ActiveBoardOwner == "host" {
		t.Fatalf("expected solved host board to auto-rotate away")
	}
	if state.ActiveBoardOwner != "p2" {
		t.Fatalf("expected next unfinished board p2, got %s", state.ActiveBoardOwner)
	}
	if state.UnityBoards["host"].TurnsUsed != hostTurns+1 {
		t.Fatalf("solved board should spend exactly the completed turn, before=%d after=%d", hostTurns, state.UnityBoards["host"].TurnsUsed)
	}
	if state.UnitySharedTurnsRemaining != 5 {
		t.Fatalf("auto-rotation should spend exactly the completed turn, got %d", state.UnitySharedTurnsRemaining)
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

func TestUnityEndStatsIncludesPerBoardAverages(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 590, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-end-stats", Words: makeWords(80)}, "host")
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorUnity, ColorCivilian})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity, ColorUnity, ColorUnity})
	setUnityBoardColors(&state, "p3", []Color{ColorUnity})
	hostBoard := state.UnityBoards["host"]
	hostBoard.Cards[0].Revealed = true
	hostBoard.TurnsUsed = 2
	state.UnityBoards["host"] = hostBoard
	p2Board := state.UnityBoards["p2"]
	p2Board.Cards[0].Revealed = true
	p2Board.Cards[1].Revealed = true
	p2Board.Cards[2].Revealed = true
	p2Board.TurnsUsed = 3
	state.UnityBoards["p2"] = p2Board
	p3Board := state.UnityBoards["p3"]
	p3Board.TurnsUsed = 0
	state.UnityBoards["p3"] = p3Board

	stats := state.buildUnityEndStats("test")

	if len(stats.BoardStats) != 3 {
		t.Fatalf("expected one stat row per board, got %#v", stats.BoardStats)
	}
	byOwner := map[string]UnityBoardEndStats{}
	for _, board := range stats.BoardStats {
		byOwner[board.OwnerID] = board
	}
	if byOwner["host"].UnityCardsFound != 1 || byOwner["host"].TotalUnityCards != 2 || byOwner["host"].TurnsUsed != 2 {
		t.Fatalf("unexpected host stats: %#v", byOwner["host"])
	}
	if byOwner["host"].UnityCardsPerTurn == nil || *byOwner["host"].UnityCardsPerTurn != 0.5 {
		t.Fatalf("expected host average 0.5, got %#v", byOwner["host"].UnityCardsPerTurn)
	}
	if byOwner["p2"].UnityCardsPerTurn == nil || *byOwner["p2"].UnityCardsPerTurn != 1 {
		t.Fatalf("expected p2 average 1, got %#v", byOwner["p2"].UnityCardsPerTurn)
	}
	if byOwner["p3"].UnityCardsPerTurn != nil {
		t.Fatalf("expected unplayed board average to be nil, got %#v", byOwner["p3"].UnityCardsPerTurn)
	}
}

func TestUnityEndStatsExcludeObserverBoards(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 592, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-end-stats-observers", Words: makeWords(80)}, "host")
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorUnity})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity})
	setUnityBoardColors(&state, "p3", []Color{ColorUnity, ColorUnity, ColorUnity})
	hostBoard := state.UnityBoards["host"]
	hostBoard.Cards[0].Revealed = true
	hostBoard.TurnsUsed = 2
	state.UnityBoards["host"] = hostBoard
	p2Board := state.UnityBoards["p2"]
	p2Board.Cards[0].Revealed = true
	p2Board.TurnsUsed = 1
	state.UnityBoards["p2"] = p2Board
	p3Board := state.UnityBoards["p3"]
	p3Board.Cards[0].Revealed = true
	p3Board.Cards[1].Revealed = true
	p3Board.Cards[2].Revealed = true
	p3Board.TurnsUsed = 3
	state.UnityBoards["p3"] = p3Board
	mustApply(t, &state, AssignTeamCommand{PlayerID: "p3", Team: TeamObservers}, "host")
	state.Phase = PhaseActive

	stats := state.buildUnityEndStats("test")

	if stats.UnityCardsFound != 2 || stats.TotalUnityCards != 3 || stats.TotalTurns != 3 {
		t.Fatalf("expected observer board excluded from totals, got %#v", stats)
	}
	gotOwners := []string{}
	for _, row := range stats.BoardStats {
		gotOwners = append(gotOwners, row.OwnerID)
	}
	if !slices.Equal(gotOwners, []string{"p2", "host"}) {
		t.Fatalf("expected only active Unity board rows, got %v", gotOwners)
	}
}

func TestUnityEndStatsRanksHigherAveragesFirst(t *testing.T) {
	state := unityLobby(t, Settings{Mode: ModeUnity, Seed: 591, UnityTurnLimit: 10})
	mustApply(t, &state, StartCommand{GameID: "game-end-stats-sort", Words: makeWords(80)}, "host")
	setUnityBoardColors(&state, "host", []Color{ColorUnity, ColorUnity})
	setUnityBoardColors(&state, "p2", []Color{ColorUnity, ColorUnity})
	setUnityBoardColors(&state, "p3", []Color{ColorUnity})
	hostBoard := state.UnityBoards["host"]
	hostBoard.Cards[0].Revealed = true
	hostBoard.TurnsUsed = 2
	state.UnityBoards["host"] = hostBoard
	p2Board := state.UnityBoards["p2"]
	p2Board.Cards[0].Revealed = true
	p2Board.Cards[1].Revealed = true
	p2Board.TurnsUsed = 1
	state.UnityBoards["p2"] = p2Board
	p3Board := state.UnityBoards["p3"]
	p3Board.TurnsUsed = 0
	state.UnityBoards["p3"] = p3Board

	stats := state.buildUnityEndStats("test")

	got := []string{}
	for _, board := range stats.BoardStats {
		got = append(got, board.OwnerID)
	}
	if !slices.Equal(got, []string{"p2", "host", "p3"}) {
		t.Fatalf("expected board stats ranked by average desc, got %v", got)
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
