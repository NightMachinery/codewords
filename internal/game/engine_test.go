package game

import (
	"errors"
	"strings"
	"testing"
)

func TestParseWordpackSkipsEmptyAndCommentLines(t *testing.T) {
	words := ParseWordpack(" Alpha \n# comment\n\nBeta\n  Gamma  ")

	want := []string{"Alpha", "Beta", "Gamma"}
	if len(words) != len(want) {
		t.Fatalf("expected %d words, got %d: %#v", len(want), len(words), words)
	}
	for i := range want {
		if words[i] != want[i] {
			t.Fatalf("word %d: expected %q, got %q", i, want[i], words[i])
		}
	}
}

func TestGenerateBoardIsDeterministicAndHasExpectedCounts(t *testing.T) {
	words := makeWords(40)
	settings := Settings{Seed: 42, BlackCards: 1, WordpackID: "test"}

	first, err := GenerateWordBoard(settings, words)
	if err != nil {
		t.Fatalf("generate first board: %v", err)
	}
	second, err := GenerateWordBoard(settings, words)
	if err != nil {
		t.Fatalf("generate second board: %v", err)
	}

	if first.StartingTeam != second.StartingTeam {
		t.Fatalf("starting team not deterministic: %s vs %s", first.StartingTeam, second.StartingTeam)
	}
	if len(first.Cards) != BoardSize {
		t.Fatalf("expected %d cards, got %d", BoardSize, len(first.Cards))
	}
	seen := map[string]bool{}
	for i, card := range first.Cards {
		if card.Content.Type != ContentWord {
			t.Fatalf("card %d expected word content, got %s", i, card.Content.Type)
		}
		if card.Content.Text == "" {
			t.Fatalf("card %d has empty word", i)
		}
		if seen[card.Content.Text] {
			t.Fatalf("duplicate selected word %q", card.Content.Text)
		}
		seen[card.Content.Text] = true
		if card != second.Cards[i] {
			t.Fatalf("card %d not deterministic: %#v vs %#v", i, card, second.Cards[i])
		}
	}

	counts := countColors(first.Cards)
	if counts[ColorBlack] != 1 {
		t.Fatalf("expected 1 assassin, got %#v", counts)
	}
	if counts[ColorBlue]+counts[ColorRed] != 17 {
		t.Fatalf("expected 17 team cards, got %#v", counts)
	}
	if counts[first.StartingTeam.Color()] != 9 {
		t.Fatalf("starting team %s should have 9 cards, got %#v", first.StartingTeam, counts)
	}
	if counts[first.StartingTeam.Opponent().Color()] != 8 {
		t.Fatalf("other team should have 8 cards, got %#v", counts)
	}
	if counts[ColorCivilian] != 7 {
		t.Fatalf("expected 7 civilians, got %#v", counts)
	}
}

func TestGenerateBoardClampsBlackCardsAndRejectsSmallWordpacks(t *testing.T) {
	board, err := GenerateWordBoard(Settings{Seed: 1, BlackCards: -5}, makeWords(25))
	if err != nil {
		t.Fatalf("generate board with negative black cards: %v", err)
	}
	if got := countColors(board.Cards)[ColorBlack]; got != 0 {
		t.Fatalf("expected negative black cards to clamp to 0, got %d", got)
	}

	board, err = GenerateWordBoard(Settings{Seed: 1, BlackCards: 99}, makeWords(40))
	if err != nil {
		t.Fatalf("generate board with high black cards: %v", err)
	}
	if got := countColors(board.Cards)[ColorBlack]; got != MaxBlackCards {
		t.Fatalf("expected high black cards to clamp to %d, got %d", MaxBlackCards, got)
	}

	_, err = GenerateWordBoard(Settings{Seed: 1}, makeWords(24))
	if !errors.Is(err, ErrNotEnoughWords) {
		t.Fatalf("expected ErrNotEnoughWords, got %v", err)
	}
}

func TestGenerateMixedBoardUsesRequestedImageCountAndWords(t *testing.T) {
	settings := Settings{Seed: 99, BlackCards: 1, WordpackID: "test", ImageCardCount: 7}
	board, err := GenerateBoard(settings, makeWords(40), makeImageIDs(10))
	if err != nil {
		t.Fatalf("generate mixed board: %v", err)
	}
	counts := map[ContentType]int{}
	seenWords := map[string]bool{}
	seenImages := map[string]bool{}
	for _, card := range board.Cards {
		counts[card.Content.Type]++
		switch card.Content.Type {
		case ContentWord:
			if card.Content.Text == "" {
				t.Fatalf("word card has empty text: %#v", card)
			}
			if seenWords[card.Content.Text] {
				t.Fatalf("duplicate word %q", card.Content.Text)
			}
			seenWords[card.Content.Text] = true
		case ContentImage:
			if card.Content.ImageID == "" {
				t.Fatalf("image card has empty id: %#v", card)
			}
			if seenImages[card.Content.ImageID] {
				t.Fatalf("duplicate image %q", card.Content.ImageID)
			}
			seenImages[card.Content.ImageID] = true
		default:
			t.Fatalf("unexpected content type %q", card.Content.Type)
		}
	}
	if counts[ContentImage] != 7 || counts[ContentWord] != 18 {
		t.Fatalf("expected 7 images and 18 words, got %#v", counts)
	}

	again, err := GenerateBoard(settings, makeWords(40), makeImageIDs(10))
	if err != nil {
		t.Fatalf("generate second mixed board: %v", err)
	}
	for i := range board.Cards {
		if board.Cards[i] != again.Cards[i] {
			t.Fatalf("card %d not deterministic: %#v vs %#v", i, board.Cards[i], again.Cards[i])
		}
	}
}

func TestGenerateBoardValidatesImageAndWordCounts(t *testing.T) {
	_, err := GenerateBoard(Settings{Seed: 1, ImageCardCount: 25}, makeWords(0), makeImageIDs(24))
	if !errors.Is(err, ErrNotEnoughImages) {
		t.Fatalf("expected ErrNotEnoughImages for image-only board, got %v", err)
	}

	_, err = GenerateBoard(Settings{Seed: 1, ImageCardCount: 24}, makeWords(0), makeImageIDs(24))
	if !errors.Is(err, ErrNotEnoughWords) {
		t.Fatalf("expected ErrNotEnoughWords for missing remaining word, got %v", err)
	}

	board, err := GenerateBoard(Settings{Seed: 1, ImageCardCount: 30}, makeWords(40), makeImageIDs(30))
	if err != nil {
		t.Fatalf("expected high image count to clamp to image-only: %v", err)
	}
	for _, card := range board.Cards {
		if card.Content.Type != ContentImage {
			t.Fatalf("expected image-only card, got %#v", card)
		}
	}
}

func TestNewTwoPlayerLobbySeatsBothPlayersAsSpymasters(t *testing.T) {
	state := NewTwoPlayerLobby("p0", "p1", Settings{Seed: 3})

	if state.Players["p0"].Team != TeamBlue || !state.Players["p0"].Spymaster {
		t.Fatalf("p0 should be blue spymaster, got %#v", state.Players["p0"])
	}
	if state.Players["p1"].Team != TeamRed || !state.Players["p1"].Spymaster {
		t.Fatalf("p1 should be red spymaster, got %#v", state.Players["p1"])
	}
	if !state.canStart() {
		t.Fatalf("two-player lobby should satisfy start role requirements")
	}
}

func TestHostIsModAndModsCanManageLobby(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 7})
	mustApply(t, &state, AddPlayerCommand{PlayerID: "host", DisplayName: "Host"}, "host")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "guest", DisplayName: "Guest"}, "guest")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "third", DisplayName: "Third"}, "third")

	if !state.Players["host"].Mod {
		t.Fatalf("room creator should be a mod by default: %#v", state.Players["host"])
	}
	if _, err := Apply(&state, ToggleModCommand{PlayerID: "third"}, "guest"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-mod mod toggle should be forbidden, got %v", err)
	}
	mustApply(t, &state, ToggleModCommand{PlayerID: "guest"}, "host")
	if !state.Players["guest"].Mod {
		t.Fatalf("host should be able to promote guests: %#v", state.Players["guest"])
	}
	mustApply(t, &state, AssignTeamCommand{PlayerID: "third", Team: TeamRed}, "guest")
	if state.Players["third"].Team != TeamRed {
		t.Fatalf("promoted mod should assign teams manually: %#v", state.Players["third"])
	}
	if _, err := Apply(&state, ToggleModCommand{PlayerID: "host"}, "guest"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("host mod status should not be mutable, got %v", err)
	}
}

func TestRandomizedTeamAssignmentBalancesNewPlayers(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 7, RandomizeTeams: true})
	for _, id := range []string{"host", "p1", "p2", "p3", "p4"} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: id, DisplayName: id}, id)
	}

	blue := 0
	red := 0
	for _, player := range state.Players {
		if player.Team == TeamBlue {
			blue++
		}
		if player.Team == TeamRed {
			red++
		}
	}
	if blue+red != 5 || blue-red > 1 || red-blue > 1 {
		t.Fatalf("expected balanced randomized assignments, blue=%d red=%d players=%#v", blue, red, state.Players)
	}

	mustApply(t, &state, AssignTeamCommand{PlayerID: "p4", Team: TeamBlue}, "host")
	if state.Players["p4"].Team != TeamBlue {
		t.Fatalf("manual assignment should still be allowed with randomize enabled")
	}
}

func TestLobbyRoleValidationAndActiveGuessers(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 7, BlackCards: 1})
	mustApply(t, &state, AddPlayerCommand{PlayerID: "host", DisplayName: "Host"}, "host")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "blueSpy", DisplayName: "Blue Spy"}, "blueSpy")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "blueRep", DisplayName: "Blue Rep"}, "blueRep")
	mustApply(t, &state, AddPlayerCommand{PlayerID: "redSpy", DisplayName: "Red Spy"}, "redSpy")

	if _, err := Apply(&state, AssignTeamCommand{PlayerID: "redSpy", Team: TeamRed}, "blueSpy"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-host assign should be forbidden, got %v", err)
	}

	mustApply(t, &state, AssignTeamCommand{PlayerID: "host", Team: TeamBlue}, "host")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "blueSpy", Team: TeamBlue}, "host")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "blueRep", Team: TeamBlue}, "host")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "redSpy", Team: TeamRed}, "host")
	mustApply(t, &state, ToggleSpymasterCommand{PlayerID: "host"}, "host")
	mustApply(t, &state, ToggleSpymasterCommand{PlayerID: "redSpy"}, "host")
	mustApply(t, &state, ToggleRepresentativeCommand{PlayerID: "blueRep"}, "host")

	if !state.Players["blueRep"].Representative || state.Players["blueRep"].Spymaster {
		t.Fatalf("representative should be representative only: %#v", state.Players["blueRep"])
	}
	if !state.IsActiveGuesser("blueRep", TeamBlue) {
		t.Fatalf("representative should be active guesser")
	}
	if state.IsActiveGuesser("host", TeamBlue) {
		t.Fatalf("spymaster should not be active when representative exists")
	}

	mustApply(t, &state, AssignTeamCommand{PlayerID: "blueRep", Team: TeamRed}, "host")
	if state.Players["blueRep"].Representative || state.Players["blueRep"].Spymaster {
		t.Fatalf("team switch should clear roles: %#v", state.Players["blueRep"])
	}
}

func TestGameplayGuessesPassesWinsAndSnapshots(t *testing.T) {
	state := startedState(t, Settings{Seed: 11, BlackCards: 1, WordpackID: "test", EnforceClueGuessLimit: false})
	state.CurrentTeam = TeamBlue
	setCardColors(&state, []Color{ColorBlue, ColorRed, ColorBlack})

	mustApply(t, &state, SubmitClueCommand{Text: "Ocean", Number: ClueNumber{Kind: ClueNumberNumeric, Value: 1}}, "blueSpy")
	event := mustApply(t, &state, GuessCommand{Index: 0}, "blueGuess")
	if event.Type != EventGuessAccepted || state.CurrentTeam != TeamBlue || state.ActionID != 1 {
		t.Fatalf("own-team guess should keep turn and increment action: event=%#v state=%#v", event, state)
	}
	if state.LastSelected == nil || state.LastSelected.Index != 0 || state.LastSelected.Team != TeamBlue {
		t.Fatalf("last selected not recorded: %#v", state.LastSelected)
	}

	mustApply(t, &state, GuessCommand{Index: 1}, "blueGuess")
	if state.CurrentTeam != TeamRed {
		t.Fatalf("wrong-team guess should switch to red, got %s", state.CurrentTeam)
	}
	if got := state.ClueLog[0]; got.Status != ClueFinal || got.Text != "Ocean" {
		t.Fatalf("clue should finalize when round ends: %#v", got)
	}
	guessViewBeforeGameOver := state.SnapshotFor(Viewer{PlayerID: "blueGuess"})
	if guessViewBeforeGameOver.Cards[3].Color != "" {
		t.Fatalf("guesser should not see unrevealed color, got %q", guessViewBeforeGameOver.Cards[3].Color)
	}

	state.CurrentTeam = TeamBlue
	state.Phase = PhaseActive
	state.Winner = ""
	state.Round = state.startRound(TeamBlue)
	mustApply(t, &state, GuessCommand{Index: 2}, "blueGuess")
	if state.Phase != PhaseGameOver || state.Winner != TeamRed {
		t.Fatalf("assassin should award opponent: phase=%s winner=%s", state.Phase, state.Winner)
	}

	guessView := state.SnapshotFor(Viewer{PlayerID: "blueGuess"})
	if guessView.Cards[3].Color == "" {
		t.Fatalf("game-over guesser view should reveal unrevealed colors")
	}
	spyView := state.SnapshotFor(Viewer{PlayerID: "blueSpy"})
	if spyView.Cards[3].Color == "" {
		t.Fatalf("spymaster should see unrevealed color")
	}
	spectatorView := state.SnapshotFor(Viewer{})
	if spectatorView.ClueLog[0].Text != "Ocean" {
		t.Fatalf("spectator should see clue log, got %#v", spectatorView.ClueLog)
	}
}

func TestClueOptionalModeLogsNAAndAllowsUpdatesBeforeRoundEnds(t *testing.T) {
	state := startedState(t, Settings{Seed: 15, BlackCards: 0, WordpackID: "test"})
	state.CurrentTeam = TeamBlue
	setCardColors(&state, []Color{ColorRed})

	mustApply(t, &state, SubmitClueCommand{Text: "Ocen", Number: ClueNumber{Kind: ClueNumberBlank}}, "blueSpy")
	mustApply(t, &state, SubmitClueCommand{Text: "Ocean", Number: ClueNumber{Kind: ClueNumberNumeric, Value: 2}}, "blueSpy")
	if got := state.CurrentClue(); got == nil || got.Text != "Ocean" || got.Number.Value != 2 {
		t.Fatalf("expected updated current clue, got %#v", got)
	}
	mustApply(t, &state, GuessCommand{Index: 0}, "blueGuess")
	if got := state.ClueLog[0]; got.Text != "Ocean" || got.Status != ClueFinal {
		t.Fatalf("expected finalized updated clue, got %#v", got)
	}

	state.CurrentTeam = TeamBlue
	state.Phase = PhaseActive
	state.Round = state.startRound(TeamBlue)
	mustApply(t, &state, PassCommand{}, "blueGuess")
	last := state.ClueLog[len(state.ClueLog)-1]
	if last.Status != ClueNA || last.Text != "NA" || last.Number.Kind != ClueNumberBlank {
		t.Fatalf("turn without clue should log NA, got %#v", last)
	}
}

func TestEnforcedClueLimitRequiresClueAndCapsGuesses(t *testing.T) {
	state := startedState(t, Settings{Seed: 16, BlackCards: 0, WordpackID: "test", EnforceClueGuessLimit: true})
	state.CurrentTeam = TeamBlue
	setCardColors(&state, []Color{ColorBlue, ColorBlue, ColorBlue})

	if _, err := Apply(&state, GuessCommand{Index: 0}, "blueGuess"); !errors.Is(err, ErrClueRequired) {
		t.Fatalf("expected clue required, got %v", err)
	}
	if _, err := Apply(&state, SubmitClueCommand{Text: "Sea", Number: ClueNumber{Kind: ClueNumberBlank}}, "blueSpy"); !errors.Is(err, ErrInvalidClueNumber) {
		t.Fatalf("expected blank clue number rejected in enforced mode, got %v", err)
	}

	mustApply(t, &state, SubmitClueCommand{Text: "Sea", Number: ClueNumber{Kind: ClueNumberNumeric, Value: 2}}, "blueSpy")
	mustApply(t, &state, GuessCommand{Index: 0}, "blueGuess")
	if _, err := Apply(&state, SubmitClueCommand{Text: "Sea", Number: ClueNumber{Kind: ClueNumberNumeric, Value: 0}}, "blueSpy"); !errors.Is(err, ErrInvalidClueNumber) {
		t.Fatalf("expected lowering clue below accepted guesses to fail as invalid, got %v", err)
	}
	mustApply(t, &state, GuessCommand{Index: 1}, "blueGuess")
	if _, err := Apply(&state, GuessCommand{Index: 2}, "blueGuess"); !errors.Is(err, ErrGuessLimitReached) {
		t.Fatalf("expected clue cap reached, got %v", err)
	}
}

func TestInfinityClueRequiresSetting(t *testing.T) {
	state := startedState(t, Settings{Seed: 17, WordpackID: "test", EnforceClueGuessLimit: true, AllowInfinityClue: false})
	state.CurrentTeam = TeamBlue
	if _, err := Apply(&state, SubmitClueCommand{Text: "Everything", Number: ClueNumber{Kind: ClueNumberInfinity}}, "blueSpy"); !errors.Is(err, ErrInvalidClueNumber) {
		t.Fatalf("expected infinity rejected without setting, got %v", err)
	}

	state.Settings.AllowInfinityClue = true
	setCardColors(&state, []Color{ColorBlue, ColorBlue, ColorBlue})
	mustApply(t, &state, SubmitClueCommand{Text: "Everything", Number: ClueNumber{Kind: ClueNumberInfinity}}, "blueSpy")
	mustApply(t, &state, GuessCommand{Index: 0}, "blueGuess")
	mustApply(t, &state, GuessCommand{Index: 1}, "blueGuess")
	mustApply(t, &state, GuessCommand{Index: 2}, "blueGuess")
}

func TestLoadBundledWordpacksIncludesEnglish(t *testing.T) {
	packs, err := LoadWordpacks("../../assets/wordpacks")
	if err != nil {
		t.Fatalf("load wordpacks: %v", err)
	}
	pack, ok := packs["english"]
	if !ok {
		t.Fatalf("expected english wordpack, got keys %#v", mapsKeys(packs))
	}
	if pack.Label != "English" {
		t.Fatalf("expected English label, got %q", pack.Label)
	}
	if len(pack.Words) < BoardSize {
		t.Fatalf("expected at least %d English words, got %d", BoardSize, len(pack.Words))
	}
}

func makeWords(n int) []string {
	words := make([]string, n)
	for i := range words {
		words[i] = "word-" + strings.Repeat("x", i+1)
	}
	return words
}

func countColors(cards []Card) map[Color]int {
	counts := map[Color]int{}
	for _, card := range cards {
		counts[card.Color]++
	}
	return counts
}

func startedState(t *testing.T, settings Settings) State {
	t.Helper()
	state := NewLobby("host", settings)
	for _, player := range []struct {
		id   string
		name string
		team Team
		spy  bool
	}{
		{"host", "Host", TeamBlue, false},
		{"blueSpy", "Blue Spy", TeamBlue, true},
		{"blueGuess", "Blue Guess", TeamBlue, false},
		{"redSpy", "Red Spy", TeamRed, true},
		{"redGuess", "Red Guess", TeamRed, false},
	} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: player.id, DisplayName: player.name}, player.id)
		mustApply(t, &state, AssignTeamCommand{PlayerID: player.id, Team: player.team}, "host")
		if player.spy {
			mustApply(t, &state, ToggleSpymasterCommand{PlayerID: player.id}, "host")
		}
	}
	mustApply(t, &state, StartCommand{Words: makeWords(40)}, "host")
	return state
}

func setCardColors(state *State, colors []Color) {
	for i, color := range colors {
		state.Cards[i].Color = color
		state.Cards[i].Revealed = false
	}
	for i := len(colors); i < len(state.Cards); i++ {
		if i%2 == 0 {
			state.Cards[i].Color = ColorBlue
		} else {
			state.Cards[i].Color = ColorRed
		}
		state.Cards[i].Revealed = false
	}
}

func mustApply(t *testing.T, state *State, command Command, actor string) Event {
	t.Helper()
	event, err := Apply(state, command, actor)
	if err != nil {
		t.Fatalf("apply %T by %s: %v", command, actor, err)
	}
	return event
}

func mapsKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func makeImageIDs(n int) []string {
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = "image-" + strings.Repeat("x", i+1)
	}
	return ids
}
