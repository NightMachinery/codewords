package game

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
)

// Command is implemented by every engine command.
type Command interface {
	apply(state *State, actorID string) (Event, error)
}

// AddPlayerCommand seats or refreshes a player in the lobby state.
type AddPlayerCommand struct {
	PlayerID    string
	DisplayName string
}

// AssignTeamCommand assigns a player to a team.
type AssignTeamCommand struct {
	PlayerID string
	Team     Team
}

// ToggleSpymasterCommand toggles spymaster status for a player.
type ToggleSpymasterCommand struct {
	PlayerID string
}

// ToggleRepresentativeCommand toggles representative status for a player.
type ToggleRepresentativeCommand struct {
	PlayerID string
}

// ToggleModCommand promotes or demotes a lobby moderator.
type ToggleModCommand struct {
	PlayerID string
}

// UpdateSettingsCommand updates lobby/game clue and board settings.
type UpdateSettingsCommand struct {
	Settings Settings
}

// StartCommand starts a match using supplied parsed wordpack words and local image ids.
type StartCommand struct {
	Words    []string
	ImageIDs []string
}

// SubmitClueCommand submits or updates the current team's clue for this round.
type SubmitClueCommand struct {
	Text   string
	Number ClueNumber
}

// GuessCommand reveals a card for the current team.
type GuessCommand struct {
	Index int
}

// PassCommand ends the current team's round voluntarily.
type PassCommand struct{}

// ShuffleRolesCommand randomizes unrevealed red/blue/civilian/assassin cards.
type ShuffleRolesCommand struct{}

// ResetClueCommand clears the current clue but leaves it in history.
type ResetClueCommand struct{}

// RestartMatchCommand returns the game back to the lobby phase.
type RestartMatchCommand struct{}

// NewLobby creates a lobby state.
func NewLobby(hostID string, settings Settings) State {
	return State{HostID: hostID, Settings: settings, Phase: PhaseLobby, Players: map[string]Player{}}
}

// NewTwoPlayerLobby creates a legacy-compatible two-player lobby.
func NewTwoPlayerLobby(firstPlayerID string, secondPlayerID string, settings Settings) State {
	state := NewLobby(firstPlayerID, settings)
	state.Players[firstPlayerID] = Player{ID: firstPlayerID, Team: TeamBlue, Spymaster: true, Mod: true}
	state.Players[secondPlayerID] = Player{ID: secondPlayerID, Team: TeamRed, Spymaster: true}
	return state
}

// Apply applies a validated command to state.
func Apply(state *State, command Command, actorID string) (Event, error) {
	if state == nil {
		return Event{}, fmt.Errorf("%w: nil state", ErrInvalidCommand)
	}
	return command.apply(state, actorID)
}

// GenerateWordBoard generates a deterministic 25-card word board.
func GenerateWordBoard(settings Settings, words []string) (Board, error) {
	settings.ImageCardCount = 0
	return GenerateBoard(settings, words, nil)
}

// GenerateBoard generates a deterministic board with words, images, or a mix of both.
func GenerateBoard(settings Settings, words []string, imageIDs []string) (Board, error) {
	imageCount := clamp(settings.ImageCardCount, 0, BoardSize)
	wordCount := BoardSize - imageCount
	uniqueWords := uniqueWords(words)
	uniqueImages := uniqueImageIDs(imageIDs)
	if len(uniqueWords) < wordCount {
		return Board{}, ErrNotEnoughWords
	}
	if len(uniqueImages) < imageCount {
		return Board{}, ErrNotEnoughImages
	}
	rng := rand.New(rand.NewSource(settings.Seed))
	cards := make([]Card, 0, BoardSize)
	wordPerm := rng.Perm(len(uniqueWords))
	for i := 0; i < wordCount; i++ {
		cards = append(cards, Card{Content: CardContent{Type: ContentWord, Text: uniqueWords[wordPerm[i]]}})
	}
	imagePerm := rng.Perm(len(uniqueImages))
	for i := 0; i < imageCount; i++ {
		cards = append(cards, Card{Content: CardContent{Type: ContentImage, ImageID: uniqueImages[imagePerm[i]]}})
	}
	contentPerm := rng.Perm(BoardSize)
	mixed := make([]Card, BoardSize)
	for i, contentIndex := range contentPerm {
		mixed[i] = cards[contentIndex]
	}
	startingTeam := TeamBlue
	if rng.Intn(2) == 1 {
		startingTeam = TeamRed
	}
	colors := colorsFor(settings, startingTeam)
	colorPerm := rng.Perm(BoardSize)
	for i, colorIndex := range colorPerm {
		mixed[i].Color = colors[colorIndex]
	}
	return Board{Cards: mixed, StartingTeam: startingTeam}, nil
}

func colorsFor(settings Settings, startingTeam Team) []Color {
	blackCards := clamp(settings.BlackCards, 0, MaxBlackCards)
	colors := make([]Color, 0, BoardSize)
	for i := 0; i < blackCards; i++ {
		colors = append(colors, ColorBlack)
	}
	blueCount := 8
	redCount := 8
	if startingTeam == TeamBlue {
		blueCount++
	} else {
		redCount++
	}
	for i := 0; i < blueCount; i++ {
		colors = append(colors, ColorBlue)
	}
	for i := 0; i < redCount; i++ {
		colors = append(colors, ColorRed)
	}
	for len(colors) < BoardSize {
		colors = append(colors, ColorCivilian)
	}
	return colors
}

func (c AddPlayerCommand) apply(state *State, actorID string) (Event, error) {
	if c.PlayerID == "" {
		return Event{}, fmt.Errorf("%w: empty player id", ErrInvalidCommand)
	}
	if state.Players == nil {
		state.Players = map[string]Player{}
	}
	player := state.Players[c.PlayerID]
	player.ID = c.PlayerID
	player.DisplayName = c.DisplayName
	if c.PlayerID == state.HostID {
		player.Mod = true
	}
	if player.Team == "" && state.Settings.RandomizeTeams {
		player.Team = state.nextBalancedTeam(c.PlayerID)
	}
	state.Players[c.PlayerID] = player
	return Event{Type: EventPlayerAdded}, nil
}

func (c AssignTeamCommand) apply(state *State, actorID string) (Event, error) {
	if !state.CanManage(actorID) && actorID != c.PlayerID {
		return Event{}, ErrForbidden
	}
	if c.Team != TeamBlue && c.Team != TeamRed && c.Team != TeamObservers && c.Team != "" {
		return Event{}, fmt.Errorf("%w: invalid team", ErrInvalidCommand)
	}
	player, ok := state.Players[c.PlayerID]
	if !ok {
		return Event{}, fmt.Errorf("%w: unknown player", ErrInvalidCommand)
	}
	if player.Team != c.Team {
		player.Spymaster = false
		player.Representative = false
	}
	player.Team = c.Team
	state.Players[c.PlayerID] = player
	return Event{Type: EventTeamAssigned}, nil
}

func (c ToggleSpymasterCommand) apply(state *State, actorID string) (Event, error) {
	if !state.CanManage(actorID) {
		return Event{}, ErrForbidden
	}
	player, ok := state.Players[c.PlayerID]
	if !ok || player.Team == "" {
		return Event{}, fmt.Errorf("%w: unknown or unassigned player", ErrInvalidCommand)
	}
	player.Spymaster = !player.Spymaster
	if player.Spymaster {
		player.Representative = false
	}
	state.Players[c.PlayerID] = player
	return Event{Type: EventRoleChanged}, nil
}

func (c ToggleRepresentativeCommand) apply(state *State, actorID string) (Event, error) {
	if !state.CanManage(actorID) {
		return Event{}, ErrForbidden
	}
	player, ok := state.Players[c.PlayerID]
	if !ok || player.Team == "" {
		return Event{}, fmt.Errorf("%w: unknown or unassigned player", ErrInvalidCommand)
	}
	player.Representative = !player.Representative
	if player.Representative {
		player.Spymaster = false
	}
	state.Players[c.PlayerID] = player
	return Event{Type: EventRoleChanged}, nil
}

func (c ToggleModCommand) apply(state *State, actorID string) (Event, error) {
	if !state.CanManage(actorID) {
		return Event{}, ErrForbidden
	}
	if c.PlayerID == state.HostID {
		return Event{}, fmt.Errorf("%w: host mod status cannot change", ErrInvalidCommand)
	}
	player, ok := state.Players[c.PlayerID]
	if !ok {
		return Event{}, fmt.Errorf("%w: unknown player", ErrInvalidCommand)
	}
	player.Mod = !player.Mod
	state.Players[c.PlayerID] = player
	return Event{Type: EventModChanged}, nil
}

func (c UpdateSettingsCommand) apply(state *State, actorID string) (Event, error) {
	if !state.CanManage(actorID) {
		return Event{}, ErrForbidden
	}
	c.Settings.BlackCards = clamp(c.Settings.BlackCards, 0, MaxBlackCards)
	state.Settings = c.Settings
	if state.Settings.RandomizeTeams {
		for id, player := range state.Players {
			if player.Team == "" {
				player.Team = state.nextBalancedTeam(id)
				state.Players[id] = player
			}
		}
	}
	return Event{Type: EventSettingsUpdated}, nil
}

func (c StartCommand) apply(state *State, actorID string) (Event, error) {
	if !state.CanManage(actorID) {
		return Event{}, ErrForbidden
	}
	if !state.canStart() {
		return Event{}, ErrCannotStart
	}
	board, err := GenerateBoard(state.Settings, c.Words, c.ImageIDs)
	if err != nil {
		return Event{}, err
	}
	state.Cards = board.Cards
	state.CurrentTeam = board.StartingTeam
	state.Phase = PhaseActive
	state.Winner = ""
	state.ActionID = 0
	state.LastSelected = nil
	state.ClueLog = nil
	state.Round = state.startRound(board.StartingTeam)
	return Event{Type: EventMatchStarted}, nil
}

// CanManage reports whether playerID can administer the lobby.
func (s State) CanManage(playerID string) bool {
	if playerID == "" {
		return false
	}
	if playerID == s.HostID {
		return true
	}
	return s.Players[playerID].Mod
}

func (c SubmitClueCommand) apply(state *State, actorID string) (Event, error) {
	if state.Phase != PhaseActive {
		return Event{}, fmt.Errorf("%w: clue outside active phase", ErrInvalidCommand)
	}
	player, ok := state.Players[actorID]
	if !ok || player.Team != state.CurrentTeam || !player.Spymaster {
		return Event{}, ErrForbidden
	}
	text := strings.TrimSpace(c.Text)
	if text == "" {
		return Event{}, fmt.Errorf("%w: empty clue text", ErrInvalidCommand)
	}
	if err := validateClueNumber(state.Settings, c.Number, state.currentRoundGuesses()); err != nil {
		return Event{}, err
	}
	idx := state.ensureCurrentClue()
	entry := state.ClueLog[idx]
	entry.Text = text
	entry.Number = c.Number
	entry.Status = ClueActive
	if entry.SubmittedBy == "" {
		entry.SubmittedBy = actorID
	}
	entry.UpdatedBy = actorID
	state.ClueLog[idx] = entry
	return Event{Type: EventClueSubmitted}, nil
}

func (c GuessCommand) apply(state *State, actorID string) (Event, error) {
	if state.Phase != PhaseActive {
		return Event{}, fmt.Errorf("%w: guess outside active phase", ErrInvalidCommand)
	}
	if c.Index < 0 || c.Index >= len(state.Cards) {
		return Event{}, fmt.Errorf("%w: card index", ErrInvalidCommand)
	}
	if state.Cards[c.Index].Revealed {
		return Event{}, fmt.Errorf("%w: card revealed", ErrInvalidCommand)
	}
	if !state.IsActiveGuesser(actorID, state.CurrentTeam) {
		return Event{}, ErrForbidden
	}
	if err := state.validateGuessAgainstClue(); err != nil {
		return Event{}, err
	}
	state.Cards[c.Index].Revealed = true
	state.ActionID++
	state.LastSelected = &LastSelected{Index: c.Index, Team: state.CurrentTeam}
	state.incrementCurrentClueGuesses()
	selectedColor := state.Cards[c.Index].Color
	if state.resolveWin() {
		state.finalizeRound()
		return Event{Type: EventGuessAccepted}, nil
	}
	if selectedColor != state.CurrentTeam.Color() {
		state.endRoundAndSwitch()
	} else if state.Settings.EnforceClueGuessLimit {
		clue := state.CurrentClue()
		if clue != nil && clue.Number.Kind == ClueNumberNumeric && clue.Guesses >= clue.Number.Value+1 {
			state.endRoundAndSwitch()
		}
	}
	return Event{Type: EventGuessAccepted}, nil
}

func (c PassCommand) apply(state *State, actorID string) (Event, error) {
	if state.Phase != PhaseActive {
		return Event{}, fmt.Errorf("%w: pass outside active phase", ErrInvalidCommand)
	}
	if !state.IsActiveGuesser(actorID, state.CurrentTeam) {
		return Event{}, ErrForbidden
	}
	state.ActionID++
	state.endRoundAndSwitch()
	return Event{Type: EventPassAccepted}, nil
}

func (c ShuffleRolesCommand) apply(state *State, actorID string) (Event, error) {
	if state.Phase != PhaseActive {
		return Event{}, fmt.Errorf("%w: cannot shuffle outside active phase", ErrInvalidCommand)
	}
	player, ok := state.Players[actorID]
	if !ok || !player.Mod {
		return Event{}, ErrForbidden
	}

	var unrevealedIndices []int
	var unrevealedColors []Color
	for i, card := range state.Cards {
		if !card.Revealed {
			unrevealedIndices = append(unrevealedIndices, i)
			unrevealedColors = append(unrevealedColors, card.Color)
		}
	}

	rng := rand.New(rand.NewSource(state.Settings.Seed + int64(state.ActionID)))
	perm := rng.Perm(len(unrevealedColors))
	for i, permIndex := range perm {
		state.Cards[unrevealedIndices[i]].Color = unrevealedColors[permIndex]
	}

	state.ActionID++
	return Event{Type: EventRolesShuffled}, nil
}

func (c ResetClueCommand) apply(state *State, actorID string) (Event, error) {
	if state.Phase != PhaseActive {
		return Event{}, fmt.Errorf("%w: cannot reset clue outside active phase", ErrInvalidCommand)
	}
	player, ok := state.Players[actorID]
	if !ok || !player.Mod {
		return Event{}, ErrForbidden
	}

	clue := state.CurrentClue()
	if clue == nil || clue.Status != ClueActive {
		return Event{}, fmt.Errorf("%w: no active clue to reset", ErrInvalidCommand)
	}

	clue.Status = ClueFinal
	state.ClueLog[len(state.ClueLog)-1] = *clue

	state.ActionID++
	return Event{Type: EventClueReset}, nil
}

func (c RestartMatchCommand) apply(state *State, actorID string) (Event, error) {
	player, ok := state.Players[actorID]
	if !ok || !player.Mod {
		return Event{}, ErrForbidden
	}

	state.Phase = PhaseLobby
	state.Cards = nil
	state.ClueLog = nil
	state.CurrentTeam = ""
	state.Winner = ""
	state.LastSelected = nil
	state.Round = 0
	state.ActionID++
	return Event{Type: EventMatchRestarted}, nil
}

// IsActiveGuesser reports whether playerID may guess/pass for team.
func (s State) IsActiveGuesser(playerID string, team Team) bool {
	player, ok := s.Players[playerID]
	if !ok || player.Team != team {
		return false
	}
	reps := 0
	teamSize := 0
	nonSpies := 0
	for _, p := range s.Players {
		if p.Team != team {
			continue
		}
		teamSize++
		if p.Representative {
			reps++
		}
		if !p.Spymaster {
			nonSpies++
		}
	}
	if teamSize == 0 {
		return false
	}
	if reps > 0 {
		return player.Representative
	}
	if nonSpies == 0 {
		return true
	}
	return !player.Spymaster
}

// CurrentClue returns the active clue row, if present.
func (s State) CurrentClue() *ClueEntry {
	if len(s.ClueLog) == 0 {
		return nil
	}
	entry := s.ClueLog[len(s.ClueLog)-1]
	if entry.Round == s.Round && entry.Status == ClueActive {
		return &entry
	}
	return nil
}

// SnapshotFor returns a viewer-safe snapshot.
func (s State) SnapshotFor(viewer Viewer) Snapshot {
	showAll := s.Phase == PhaseGameOver
	if player, ok := s.Players[viewer.PlayerID]; ok && player.Spymaster {
		showAll = true
	}
	cards := make([]SnapshotCard, len(s.Cards))
	for i, card := range s.Cards {
		cards[i] = SnapshotCard{Content: card.Content, Revealed: card.Revealed}
		if showAll || card.Revealed {
			cards[i].Color = card.Color
		}
	}
	log := make([]ClueEntry, len(s.ClueLog))
	copy(log, s.ClueLog)
	return Snapshot{Phase: s.Phase, CurrentTeam: s.CurrentTeam, Winner: s.Winner, ActionID: s.ActionID, Cards: cards, LastSelected: s.LastSelected, ClueLog: log}
}

func (s State) canStart() bool {
	blueSpy := false
	redSpy := false
	if len(s.Players) == 0 {
		return false
	}
	for _, player := range s.Players {
		if player.Team != TeamBlue && player.Team != TeamRed {
			return false
		}
		if player.Team == TeamBlue && player.Spymaster {
			blueSpy = true
		}
		if player.Team == TeamRed && player.Spymaster {
			redSpy = true
		}
	}
	return blueSpy && redSpy
}

func validateClueNumber(settings Settings, number ClueNumber, currentGuesses int) error {
	switch number.Kind {
	case ClueNumberBlank:
		if settings.EnforceClueGuessLimit {
			return ErrInvalidClueNumber
		}
		return nil
	case ClueNumberNumeric:
		if number.Value < 1 || number.Value > 9 {
			return ErrInvalidClueNumber
		}
		if settings.EnforceClueGuessLimit && number.Value < currentGuesses {
			return ErrInvalidClueNumber
		}
		return nil
	case ClueNumberInfinity:
		if !settings.AllowInfinityClue {
			return ErrInvalidClueNumber
		}
		return nil
	default:
		return ErrInvalidClueNumber
	}
}

func (s *State) validateGuessAgainstClue() error {
	if !s.Settings.EnforceClueGuessLimit {
		return nil
	}
	clue := s.CurrentClue()
	if clue == nil || clue.Number.Kind == ClueNumberBlank {
		return ErrClueRequired
	}
	if clue.Number.Kind == ClueNumberInfinity {
		return nil
	}
	if clue.Guesses >= clue.Number.Value {
		return ErrGuessLimitReached
	}
	return nil
}

func (s *State) startRound(team Team) int {
	return s.Round + 1
}

func (s *State) ensureCurrentClue() int {
	if len(s.ClueLog) > 0 {
		idx := len(s.ClueLog) - 1
		if s.ClueLog[idx].Round == s.Round && s.ClueLog[idx].Status == ClueActive {
			return idx
		}
	}
	s.ClueLog = append(s.ClueLog, ClueEntry{Round: s.Round, Team: s.CurrentTeam, Status: ClueActive})
	return len(s.ClueLog) - 1
}

func (s *State) currentRoundGuesses() int {
	if len(s.ClueLog) == 0 {
		return 0
	}
	entry := s.ClueLog[len(s.ClueLog)-1]
	if entry.Round != s.Round {
		return 0
	}
	return entry.Guesses
}

func (s *State) incrementCurrentClueGuesses() {
	idx := s.ensureCurrentClue()
	s.ClueLog[idx].Guesses++
}

func (s *State) finalizeRound() {
	idx := s.ensureCurrentClue()
	entry := s.ClueLog[idx]
	if entry.Text == "" {
		entry.Text = "NA"
		entry.Number = ClueNumber{Kind: ClueNumberBlank}
		entry.Status = ClueNA
	} else {
		entry.Status = ClueFinal
	}
	s.ClueLog[idx] = entry
}

func (s *State) endRoundAndSwitch() {
	s.finalizeRound()
	s.CurrentTeam = s.CurrentTeam.Opponent()
	s.Round = s.startRound(s.CurrentTeam)
}

func (s *State) resolveWin() bool {
	if s.LastSelected != nil && s.Cards[s.LastSelected.Index].Color == ColorBlack {
		s.Winner = s.LastSelected.Team.Opponent()
		s.Phase = PhaseGameOver
		return true
	}
	blueLeft := false
	redLeft := false
	for _, card := range s.Cards {
		if card.Color == ColorBlue && !card.Revealed {
			blueLeft = true
		}
		if card.Color == ColorRed && !card.Revealed {
			redLeft = true
		}
	}
	if !blueLeft {
		s.Winner = TeamBlue
		s.Phase = PhaseGameOver
		return true
	}
	if !redLeft {
		s.Winner = TeamRed
		s.Phase = PhaseGameOver
		return true
	}
	return false
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (s State) nextBalancedTeam(playerID string) Team {
	blue := 0
	red := 0
	for _, player := range s.Players {
		switch player.Team {
		case TeamBlue:
			blue++
		case TeamRed:
			red++
		}
	}
	if blue < red {
		return TeamBlue
	}
	if red < blue {
		return TeamRed
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s", s.Settings.Seed, playerID)))
	if h.Sum64()%2 == 0 {
		return TeamBlue
	}
	return TeamRed
}
