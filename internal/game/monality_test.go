package game

import (
	"math"
	"testing"
	"time"
)

func TestMonalityStartCreatesSharedRoundAndSafeAttemptSnapshots(t *testing.T) {
	state := monalityLobby(t, Settings{Mode: ModeMonality, Seed: 901, TotalCards: 25, BlackCards: 0})

	mustApply(t, &state, StartCommand{GameID: "monality-safe", Words: makeWords(80)}, "host")

	if state.Mode != ModeMonality || state.CurrentTeam != TeamMonality || state.Phase != PhaseActive {
		t.Fatalf("expected active monality mode, got mode=%s team=%s phase=%s", state.Mode, state.CurrentTeam, state.Phase)
	}
	if state.MonalitySpymasterID == "" {
		t.Fatalf("expected round spymaster")
	}
	if len(state.MonalityAttempts) != 2 {
		t.Fatalf("expected attempts for non-spymasters only, got %d", len(state.MonalityAttempts))
	}
	if _, ok := state.MonalityAttempts[state.MonalitySpymasterID]; ok {
		t.Fatalf("spymaster should not get a guessing attempt")
	}
	counts := countColors(state.MonalityBoard.Cards)
	if counts[ColorUnity] != 10 || counts[ColorBlack] != 4 || len(state.MonalityBoard.Cards) != 25 {
		t.Fatalf("unexpected monality board composition: %#v len=%d", counts, len(state.MonalityBoard.Cards))
	}

	spyView := state.SnapshotFor(Viewer{PlayerID: state.MonalitySpymasterID})
	if spyView.Monality == nil || spyView.Monality.SpymasterID != state.MonalitySpymasterID {
		t.Fatalf("expected monality snapshot for spy, got %#v", spyView.Monality)
	}
	if spyView.Monality.Board == nil || spyView.Monality.Board.Cards[0].Color == "" {
		t.Fatalf("spymaster should see hidden board colors")
	}
	if spyView.Monality.OwnAttempt != nil {
		t.Fatalf("spymaster should not receive own attempt")
	}

	guesserID := firstMonalityGuesser(&state)
	guesserView := state.SnapshotFor(Viewer{PlayerID: guesserID})
	if guesserView.Monality == nil || guesserView.Monality.Board != nil {
		t.Fatalf("guesser should not receive spymaster board, got %#v", guesserView.Monality)
	}
	if guesserView.Monality.OwnAttempt == nil || guesserView.Monality.OwnAttempt.OwnerID != guesserID {
		t.Fatalf("guesser should receive own attempt, got %#v", guesserView.Monality.OwnAttempt)
	}
	if guesserView.Monality.OwnAttempt.Cards[0].Color != "" {
		t.Fatalf("guesser should not see hidden unrevealed color, got %q", guesserView.Monality.OwnAttempt.Cards[0].Color)
	}
}

func TestMonalityRoundScoringAveragesGuessersAndBombResets(t *testing.T) {
	state := monalityLobby(t, Settings{Mode: ModeMonality, Seed: 902, TotalCards: 9, BlackCards: 1})
	mustApply(t, &state, StartCommand{GameID: "monality-score", Words: makeWords(40)}, "host")
	setMonalityRoundForTest(&state, "host", []Color{ColorUnity, ColorUnity, ColorBlack, ColorCivilian})
	mustApply(t, &state, SubmitClueCommand{Text: "two", Number: ClueNumber{Kind: ClueNumberBlank}}, "host")

	mustApply(t, &state, GuessCommand{Index: 0}, "p2")
	mustApply(t, &state, GuessCommand{Index: 1}, "p2")
	mustApply(t, &state, GuessCommand{Index: 0}, "p3")
	mustApply(t, &state, GuessCommand{Index: 2}, "p3")

	if got := state.MonalityTotalScores["p2"]; got != 2 {
		t.Fatalf("p2 score = %v, want 2", got)
	}
	if got := state.MonalityTotalScores["p3"]; got != MonalityBombScore {
		t.Fatalf("p3 score = %v, want bomb score", got)
	}
	if got := state.MonalityTotalScores["host"]; math.Abs(got-0.5) > 0.0001 {
		t.Fatalf("host spymaster average = %v, want 0.5", got)
	}
	if state.MonalitySpymasterCounts["host"] != 1 {
		t.Fatalf("host spymaster count = %d, want 1", state.MonalitySpymasterCounts["host"])
	}
	if state.MonalitySpymasterID == "host" || state.Round != 2 || state.Phase != PhaseActive {
		t.Fatalf("expected next active round with new spymaster, spy=%q round=%d phase=%s", state.MonalitySpymasterID, state.Round, state.Phase)
	}
}

func TestMonalityMatchEndsAfterConfiguredSpymasterRoundsAndRanksScores(t *testing.T) {
	state := monalityLobby(t, Settings{Mode: ModeMonality, Seed: 903, TotalCards: 9, BlackCards: 1, MonalitySpymasterRounds: 1})
	mustApply(t, &state, StartCommand{GameID: "monality-end", Words: makeWords(40)}, "host")

	for state.Phase == PhaseActive {
		spy := state.MonalitySpymasterID
		setMonalityRoundForTest(&state, spy, []Color{ColorUnity, ColorCivilian, ColorBlack})
		mustApply(t, &state, SubmitClueCommand{Text: "one", Number: ClueNumber{Kind: ClueNumberBlank}}, spy)
		for id, attempt := range state.MonalityAttempts {
			if attempt.Completed {
				continue
			}
			mustApply(t, &state, GuessCommand{Index: 0}, id)
		}
	}

	if state.Winner != TeamMonality || state.MonalityEndStats == nil {
		t.Fatalf("expected monality game over stats, winner=%s stats=%#v", state.Winner, state.MonalityEndStats)
	}
	if len(state.MonalityEndStats.Rankings) != 3 {
		t.Fatalf("expected rankings for three players, got %#v", state.MonalityEndStats.Rankings)
	}
	for _, id := range []string{"host", "p2", "p3"} {
		if state.MonalitySpymasterCounts[id] != 1 {
			t.Fatalf("%s spymaster count = %d, want 1", id, state.MonalitySpymasterCounts[id])
		}
	}
	for i := 1; i < len(state.MonalityEndStats.Rankings); i++ {
		if state.MonalityEndStats.Rankings[i-1].TotalScore < state.MonalityEndStats.Rankings[i].TotalScore {
			t.Fatalf("rankings not sorted descending: %#v", state.MonalityEndStats.Rankings)
		}
	}
}

func TestMonalityFinishAssignsSameRankToEqualScores(t *testing.T) {
	state := monalityLobby(t, Settings{Mode: ModeMonality, Seed: 905, TotalCards: 9, BlackCards: 1})
	state.Mode = ModeMonality
	state.Phase = PhaseActive
	state.MonalityTotalScores = map[string]float64{"host": 3, "p2": 3, "p3": 1}

	state.finishMonality()

	if state.MonalityEndStats == nil || len(state.MonalityEndStats.Rankings) != 3 {
		t.Fatalf("expected three rankings, got %#v", state.MonalityEndStats)
	}
	got := map[string]int{}
	for _, ranking := range state.MonalityEndStats.Rankings {
		got[ranking.PlayerID] = ranking.Rank
	}
	if got["host"] != 1 || got["p2"] != 1 || got["p3"] != 3 {
		t.Fatalf("expected tied leaders to share rank and next player to rank third, got %#v", state.MonalityEndStats.Rankings)
	}
}

func TestMonalityCloseRoundCountsCompletedDisconnectedAndAbandonsUnfinishedDisconnected(t *testing.T) {
	state := monalityLobby(t, Settings{Mode: ModeMonality, Seed: 904, TotalCards: 9, BlackCards: 1})
	mustApply(t, &state, StartCommand{GameID: "monality-close", Words: makeWords(40)}, "host")
	setMonalityRoundForTest(&state, "host", []Color{ColorUnity, ColorUnity, ColorCivilian})
	mustApply(t, &state, SubmitClueCommand{Text: "two", Number: ClueNumber{Kind: ClueNumberBlank}}, "host")

	mustApply(t, &state, GuessCommand{Index: 0}, "p2")
	mustApply(t, &state, GuessCommand{Index: 0}, "p3")

	mustApply(t, &state, CloseMonalityRoundCommand{ConnectedPlayerIDs: []string{"host", "p2"}, Now: time.Unix(10, 0)}, "host")

	if got := state.MonalityTotalScores["p2"]; got != 1 {
		t.Fatalf("completed disconnected-capable p2 should count with score 1, got %v", got)
	}
	lastRound := state.MonalityRoundScores[len(state.MonalityRoundScores)-1]
	if _, ok := lastRound.Scores["p3"]; ok {
		t.Fatalf("unfinished disconnected p3 should be abandoned from round scores, got %#v", lastRound.Scores)
	}
	if got := state.MonalityTotalScores["host"]; got != 1 {
		t.Fatalf("spymaster average should use counted attempts only, got %v", got)
	}
}

func monalityLobby(t *testing.T, settings Settings) State {
	t.Helper()
	state := NewLobby("host", settings)
	for _, player := range []string{"host", "p2", "p3"} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: player, DisplayName: player}, player)
		mustApply(t, &state, AssignTeamCommand{PlayerID: player, Team: TeamMonality}, "host")
	}
	return state
}

func firstMonalityGuesser(state *State) string {
	for id := range state.MonalityAttempts {
		return id
	}
	return ""
}

func setMonalityRoundForTest(state *State, spymasterID string, colors []Color) {
	state.MonalitySpymasterID = spymasterID
	state.ActiveBoardOwner = spymasterID
	board := state.MonalityBoard
	board.OwnerID = spymasterID
	for i, color := range colors {
		board.Cards[i].Color = color
		board.Cards[i].Revealed = false
	}
	for i := len(colors); i < len(board.Cards); i++ {
		board.Cards[i].Color = ColorCivilian
		board.Cards[i].Revealed = false
	}
	state.MonalityBoard = board
	attempts := map[string]MonalityAttemptState{}
	for id, player := range state.Players {
		if player.Team != TeamMonality || id == spymasterID {
			continue
		}
		attempts[id] = MonalityAttemptState{OwnerID: id, Cards: cloneCards(board.Cards)}
	}
	state.MonalityAttempts = attempts
	state.RoundGuessesByPlayer = map[string]int{}
}
