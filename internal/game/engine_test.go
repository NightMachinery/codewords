package game

import (
	"encoding/json"
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

func TestGenerateBoardRejectsInvalidCountsAndSmallWordpacks(t *testing.T) {
	if _, err := GenerateWordBoard(Settings{Seed: 1, BlackCards: -5}, makeWords(25)); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected ErrInvalidSettings for negative black cards, got %v", err)
	}
	if _, err := GenerateWordBoard(Settings{Seed: 1, BlackCards: 9}, makeWords(40)); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected ErrInvalidSettings when assassins exceed neutral cards, got %v", err)
	}
	_, err := GenerateWordBoard(Settings{Seed: 1}, makeWords(24))
	if !errors.Is(err, ErrNotEnoughWords) {
		t.Fatalf("expected ErrNotEnoughWords, got %v", err)
	}
}

func TestSettingsDefaultsStartingTeamHandicapOnlyWhenOmitted(t *testing.T) {
	var omitted Settings
	if err := json.Unmarshal([]byte(`{"autoColorCounts":true,"totalCards":25}`), &omitted); err != nil {
		t.Fatalf("unmarshal omitted handicap: %v", err)
	}
	if got := SettingsWithDefaults(omitted).StartingTeamHandicap; got != 1 {
		t.Fatalf("missing automatic handicap should default to 1, got %d", got)
	}

	var explicitZero Settings
	if err := json.Unmarshal([]byte(`{"autoColorCounts":true,"totalCards":25,"startingTeamHandicap":0}`), &explicitZero); err != nil {
		t.Fatalf("unmarshal explicit zero handicap: %v", err)
	}
	if got := SettingsWithDefaults(explicitZero).StartingTeamHandicap; got != 0 {
		t.Fatalf("explicit zero handicap should be preserved, got %d", got)
	}

	var legacyManual Settings
	if err := json.Unmarshal([]byte(`{"autoColorCounts":false,"totalCards":18,"blueCards":4,"redCards":6,"neutralCards":8}`), &legacyManual); err != nil {
		t.Fatalf("unmarshal legacy manual counts: %v", err)
	}
	if err := ValidateSettings(legacyManual); err != nil {
		t.Fatalf("legacy manual counts without handicap should remain valid: %v", err)
	}
}

func TestGenerateBoardSupportsAutomaticDynamicTotals(t *testing.T) {
	board, err := GenerateWordBoard(Settings{Seed: 4, TotalCards: 30, AutoColorCounts: true, BlackCards: 2}, makeWords(40))
	if err != nil {
		t.Fatalf("generate dynamic board: %v", err)
	}
	if len(board.Cards) != 30 {
		t.Fatalf("expected 30 cards, got %d", len(board.Cards))
	}
	counts := countColors(board.Cards)
	if counts[ColorBlack] != 2 {
		t.Fatalf("expected 2 assassins, got %#v", counts)
	}
	if counts[ColorCivilian] != 9 {
		t.Fatalf("expected 9 civilians after assassins consume neutral cards, got %#v", counts)
	}
	if counts[board.StartingTeam.Color()] != 10 {
		t.Fatalf("starting team should have one extra card, got %#v", counts)
	}
	if counts[board.StartingTeam.Opponent().Color()] != 9 {
		t.Fatalf("opposing team should have the smaller team count, got %#v", counts)
	}
}

func TestGenerateBoardRandomizesAutomaticStartingTeamBySeed(t *testing.T) {
	seen := map[Team]bool{}
	for seed := int64(1); seed <= 20; seed++ {
		board, err := GenerateWordBoard(Settings{Seed: seed, TotalCards: 30, AutoColorCounts: true, BlackCards: 1}, makeWords(40))
		if err != nil {
			t.Fatalf("generate board for seed %d: %v", seed, err)
		}
		seen[board.StartingTeam] = true
		counts := countColors(board.Cards)
		if counts[board.StartingTeam.Color()] != counts[board.StartingTeam.Opponent().Color()]+1 {
			t.Fatalf("seed %d starting team %s should have exactly one extra card, got %#v", seed, board.StartingTeam, counts)
		}
	}
	if !seen[TeamBlue] || !seen[TeamRed] {
		t.Fatalf("expected different seeds to produce both starting teams, got %#v", seen)
	}
}

func TestGenerateBoardAppliesAutomaticStartingTeamHandicap(t *testing.T) {
	board, err := GenerateWordBoard(Settings{Seed: 4, TotalCards: 30, AutoColorCounts: true, StartingTeamHandicap: 2, BlackCards: 1}, makeWords(40))
	if err != nil {
		t.Fatalf("generate automatic handicap board: %v", err)
	}
	counts := countColors(board.Cards)
	if counts[board.StartingTeam.Color()] != counts[board.StartingTeam.Opponent().Color()]+2 {
		t.Fatalf("starting team should carry configured handicap, got %#v", counts)
	}
	if counts[ColorCivilian] != 9 {
		t.Fatalf("expected automatic neutral share to account for handicap, got %#v", counts)
	}
}

func TestGenerateBoardSupportsManualDynamicTotals(t *testing.T) {
	settings := Settings{
		Seed:            5,
		TotalCards:      18,
		AutoColorCounts: false,
		BlueCards:       4,
		RedCards:        6,
		NeutralCards:    8,
		BlackCards:      3,
	}
	board, err := GenerateWordBoard(settings, makeWords(30))
	if err != nil {
		t.Fatalf("generate manual board: %v", err)
	}
	counts := countColors(board.Cards)
	if counts[ColorBlue] != 4 || counts[ColorRed] != 6 || counts[ColorBlack] != 3 || counts[ColorCivilian] != 5 {
		t.Fatalf("manual counts not honored: %#v", counts)
	}

	settings.NeutralCards = 7
	if _, err := GenerateWordBoard(settings, makeWords(30)); !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected invalid settings for manual counts that do not sum to total, got %v", err)
	}
}

func TestGenerateBoardAppliesManualStartingTeamHandicapToRandomStartingTeam(t *testing.T) {
	seen := map[Team]bool{}
	for seed := int64(1); seed <= 20; seed++ {
		board, err := GenerateWordBoard(Settings{
			Seed:                 seed,
			TotalCards:           24,
			AutoColorCounts:      false,
			BlueCards:            8,
			RedCards:             8,
			NeutralCards:         7,
			StartingTeamHandicap: 1,
			BlackCards:           1,
		}, makeWords(30))
		if err != nil {
			t.Fatalf("generate manual handicap board for seed %d: %v", seed, err)
		}
		seen[board.StartingTeam] = true
		counts := countColors(board.Cards)
		if counts[board.StartingTeam.Color()] != 9 {
			t.Fatalf("seed %d starting team %s should receive manual handicap, got %#v", seed, board.StartingTeam, counts)
		}
		if counts[board.StartingTeam.Opponent().Color()] != 8 {
			t.Fatalf("seed %d opposing team should stay at base count, got %#v", seed, counts)
		}
		if counts[ColorBlack] != 1 || counts[ColorCivilian] != 6 {
			t.Fatalf("seed %d neutral counts changed unexpectedly: %#v", seed, counts)
		}
	}
	if !seen[TeamBlue] || !seen[TeamRed] {
		t.Fatalf("manual boards should still randomize starting team by seed, got %#v", seen)
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

	_, err = GenerateBoard(Settings{Seed: 1, ImageCardCount: 30}, makeWords(40), makeImageIDs(30))
	if !errors.Is(err, ErrInvalidSettings) {
		t.Fatalf("expected high image count to be invalid, got %v", err)
	}
}

func TestShuffledImageIDsIsSeededAndDeterministic(t *testing.T) {
	ids := []string{"img-4", "img-1", "img-3", "img-2", "img-1", ""}

	one := ShuffledImageIDs(Settings{Seed: 123}, ids)
	again := ShuffledImageIDs(Settings{Seed: 123}, ids)
	other := ShuffledImageIDs(Settings{Seed: 456}, ids)

	if strings.Join(one, ",") != strings.Join(again, ",") {
		t.Fatalf("expected same seed to produce same order, got %#v and %#v", one, again)
	}
	if strings.Join(one, ",") == strings.Join(other, ",") {
		t.Fatalf("expected different seed to change image order, got %#v", one)
	}
	if strings.Join(one, ",") == "img-1,img-2,img-3,img-4" {
		t.Fatalf("expected shuffled order, got sorted order %#v", one)
	}
	if len(one) != 4 {
		t.Fatalf("expected unique non-empty ids, got %#v", one)
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
	if state.canStart() {
		t.Fatalf("two-player all-spymaster lobby should not satisfy current start role requirements")
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

func TestRandomizeTeamsBalancesEligiblePlayersAndResetsRoles(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 77})
	for _, player := range []struct {
		id    string
		team  Team
		spy   bool
		rep   bool
		isMod bool
	}{
		{id: "host", team: TeamBlue, spy: true, isMod: true},
		{id: "blueRep", team: TeamBlue, rep: true},
		{id: "redSpy", team: TeamRed, spy: true},
		{id: "floater"},
		{id: "observer", team: TeamObservers, spy: true, rep: true},
	} {
		state.Players[player.id] = Player{ID: player.id, DisplayName: player.id, Team: player.team, Spymaster: player.spy, Representative: player.rep, Mod: player.isMod}
	}

	event, err := Apply(&state, RandomizeTeamsCommand{}, "host")
	if err != nil {
		t.Fatalf("randomize teams should be accepted: %v", err)
	}
	if event.Type != EventTeamsRandomized {
		t.Fatalf("expected teams randomized event, got %#v", event)
	}

	counts := map[Team]int{}
	spies := map[Team]int{}
	for _, player := range state.Players {
		if player.ID == "observer" {
			if player.Team != TeamObservers || player.Spymaster || player.Representative {
				t.Fatalf("observer should stay observer with roles cleared, got %#v", player)
			}
			continue
		}
		if player.Team != TeamBlue && player.Team != TeamRed {
			t.Fatalf("eligible player should be assigned to a playable team: %#v", player)
		}
		if player.Representative {
			t.Fatalf("representative roles should be cleared: %#v", player)
		}
		counts[player.Team]++
		if player.Spymaster {
			spies[player.Team]++
		}
	}
	if counts[TeamBlue]+counts[TeamRed] != 4 || abs(counts[TeamBlue]-counts[TeamRed]) > 1 {
		t.Fatalf("expected balanced teams, counts=%#v players=%#v", counts, state.Players)
	}
	if spies[TeamBlue] != 1 || spies[TeamRed] != 1 {
		t.Fatalf("expected one spy per team, spies=%#v players=%#v", spies, state.Players)
	}
}

func TestRandomizeTeamsRequiresManagerLobbyAndEnoughEligiblePlayers(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 5})
	state.Players["host"] = Player{ID: "host", Team: TeamBlue, Mod: true}
	state.Players["guest"] = Player{ID: "guest", Team: TeamRed}

	if _, err := Apply(&state, RandomizeTeamsCommand{}, "guest"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-manager randomize should be forbidden, got %v", err)
	}
	state.Phase = PhaseActive
	if _, err := Apply(&state, RandomizeTeamsCommand{}, "host"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("active-phase randomize should be invalid, got %v", err)
	}
	state.Phase = PhaseLobby
	state.Players["guest"] = Player{ID: "guest", Team: TeamObservers}
	if _, err := Apply(&state, RandomizeTeamsCommand{}, "host"); !errors.Is(err, ErrCannotStart) {
		t.Fatalf("single eligible player randomize should be rejected, got %v", err)
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

func TestObserverRejoinRestoresPreviousPlayableTeamAndRole(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 7})
	state.Players["host"] = Player{ID: "host", Team: TeamBlue, Mod: true}
	state.Players["rep"] = Player{ID: "rep", Team: TeamBlue, Representative: true}
	state.Players["spy"] = Player{ID: "spy", Team: TeamRed, Spymaster: true}

	mustApply(t, &state, AssignTeamCommand{PlayerID: "rep", Team: TeamObservers}, "rep")
	if got := state.Players["rep"]; got.Team != TeamObservers || got.Representative || got.PreviousTeam != TeamBlue || !got.PreviousRepresentative {
		t.Fatalf("representative observer transition should remember prior role, got %#v", got)
	}
	mustApply(t, &state, RejoinTeamCommand{PlayerID: "rep"}, "rep")
	if got := state.Players["rep"]; got.Team != TeamBlue || !got.Representative || got.Spymaster {
		t.Fatalf("representative rejoin should restore team and role, got %#v", got)
	}

	mustApply(t, &state, AssignTeamCommand{PlayerID: "spy", Team: TeamObservers}, "host")
	mustApply(t, &state, RejoinTeamCommand{PlayerID: "spy"}, "host")
	if got := state.Players["spy"]; got.Team != TeamRed || !got.Spymaster || got.Representative {
		t.Fatalf("spymaster rejoin should restore team and role, got %#v", got)
	}
}

func TestRestartMatchReseedsNextBoard(t *testing.T) {
	state := startedState(t, Settings{Seed: 42, BlackCards: 1, WordpackID: "test"})
	originalSeed := state.Settings.Seed
	originalCards := append([]Card(nil), state.Cards...)

	mustApply(t, &state, RestartMatchCommand{}, "host")
	if state.Phase != PhaseLobby || len(state.Cards) != 0 {
		t.Fatalf("restart should return to empty lobby, got phase=%s cards=%d", state.Phase, len(state.Cards))
	}
	if state.Settings.Seed == originalSeed {
		t.Fatalf("restart should change seed from %d", originalSeed)
	}
	mustApply(t, &state, StartCommand{Words: makeWords(40), ImageIDs: makeImageIDs(40)}, "host")
	if cardsEqual(originalCards, state.Cards) {
		t.Fatalf("next board should be reshuffled after restart")
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
	if state.CurrentTeam != TeamRed {
		t.Fatalf("expected clue cap to auto-pass to red, got %s", state.CurrentTeam)
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

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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

func cardsEqual(a, b []Card) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCannotStartWithAllSpymasterTeams(t *testing.T) {
	state := NewLobby("host", Settings{Seed: 31, BlackCards: 1})
	for _, player := range []struct {
		id   string
		team Team
	}{
		{"blueSpy", TeamBlue},
		{"redSpy", TeamRed},
	} {
		mustApply(t, &state, AddPlayerCommand{PlayerID: player.id, DisplayName: player.id}, player.id)
		mustApply(t, &state, AssignTeamCommand{PlayerID: player.id, Team: player.team}, "host")
		mustApply(t, &state, ToggleSpymasterCommand{PlayerID: player.id}, "host")
	}

	if _, err := Apply(&state, StartCommand{Words: makeWords(40)}, "host"); !errors.Is(err, ErrCannotStart) {
		t.Fatalf("expected all-spymaster teams to be rejected, got %v", err)
	}
}

func TestSpymastersAndObserversCannotRevealOrReceivePlayRoles(t *testing.T) {
	state := startedState(t, Settings{Seed: 32, BlackCards: 0, WordpackID: "test"})
	state.CurrentTeam = TeamBlue
	setCardColors(&state, []Color{ColorBlue, ColorBlue})

	if _, err := Apply(&state, GuessCommand{Index: 0}, "blueSpy"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected spymaster guess to be forbidden, got %v", err)
	}
	mustApply(t, &state, AddPlayerCommand{PlayerID: "observer", DisplayName: "Observer"}, "observer")
	mustApply(t, &state, AssignTeamCommand{PlayerID: "observer", Team: TeamObservers}, "host")
	if _, err := Apply(&state, ToggleSpymasterCommand{PlayerID: "observer"}, "host"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected observer spymaster toggle to be invalid, got %v", err)
	}
	if _, err := Apply(&state, ToggleRepresentativeCommand{PlayerID: "observer"}, "host"); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("expected observer representative toggle to be invalid, got %v", err)
	}
}

func TestEnforcedClueLimitAutoPassesAtExactGuessCount(t *testing.T) {
	state := startedState(t, Settings{Seed: 33, BlackCards: 0, WordpackID: "test", EnforceClueGuessLimit: true})
	state.CurrentTeam = TeamBlue
	setCardColors(&state, []Color{ColorBlue, ColorBlue, ColorBlue})
	mustApply(t, &state, SubmitClueCommand{Text: "Sea", Number: ClueNumber{Kind: ClueNumberNumeric, Value: 2}}, "blueSpy")

	mustApply(t, &state, GuessCommand{Index: 0}, "blueGuess")
	if state.CurrentTeam != TeamBlue {
		t.Fatalf("first correct guess should keep blue turn, got %s", state.CurrentTeam)
	}
	mustApply(t, &state, GuessCommand{Index: 1}, "blueGuess")
	if state.CurrentTeam != TeamRed {
		t.Fatalf("second correct guess should auto-pass to red, got %s", state.CurrentTeam)
	}
	if state.ClueLog[0].Status != ClueFinal {
		t.Fatalf("auto-pass should finalize clue, got %#v", state.ClueLog[0])
	}
}
