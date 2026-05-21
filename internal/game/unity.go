package game

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strings"
)

// UnityCardCount returns the automatic target count for one Unity board.
func UnityCardCount(totalCards int) int {
	return int(math.Round(float64(totalCards) / 2.5))
}

func (c StartCommand) startUnity(state *State, actorID string) (Event, error) {
	state.GameID = strings.TrimSpace(c.GameID)
	state.Mode = ModeUnity
	state.CurrentTeam = TeamUnity
	state.Phase = PhaseActive
	state.Winner = ""
	state.FinishedAt = ""
	state.ActionID = 0
	state.LastSelected = nil
	state.ClueLog = nil
	state.Round = 0
	state.RoundGuesses = 0
	state.UnityWaitingForGuessers = false
	state.UnityEndStats = nil
	state.UnityWords = uniqueWords(c.Words)
	state.UnityImageIDs = uniqueImageIDs(c.ImageIDs)
	state.UnityBoards = map[string]UnityBoardState{}
	state.UnityBoardOrder = state.unityActivePlayerIDs()

	for _, ownerID := range state.UnityBoardOrder {
		board, err := GenerateUnityBoard(state.Settings, state.GameID, ownerID, state.UnityWords, state.UnityImageIDs)
		if err != nil {
			return Event{}, err
		}
		state.UnityBoards[ownerID] = UnityBoardState{OwnerID: ownerID, Cards: board.Cards}
	}
	if !state.Settings.UnityUnlimitedTurns && !state.Settings.UnityStrictPerBoardTurns {
		state.UnitySharedTurnsRemaining = state.Settings.UnityTurnLimit * len(state.UnityBoardOrder)
	} else {
		state.UnitySharedTurnsRemaining = 0
	}
	if !state.advanceUnityTurnFrom(-1) {
		state.UnityWaitingForGuessers = true
	}
	return Event{Type: EventMatchStarted}, nil
}

// GenerateUnityBoard generates a deterministic personal Unity board.
func GenerateUnityBoard(settings Settings, gameID string, ownerID string, words []string, imageIDs []string) (Board, error) {
	settings = SettingsWithDefaults(settings)
	if err := ValidateSettings(settings); err != nil {
		return Board{}, err
	}
	totalCards := settings.TotalCards
	imageCount := settings.ImageCardCount
	wordCount := totalCards - imageCount
	uniqueWords := uniqueWords(words)
	uniqueImages := uniqueImageIDs(imageIDs)
	if len(uniqueWords) < wordCount {
		return Board{}, ErrNotEnoughWords
	}
	if len(uniqueImages) < imageCount {
		return Board{}, ErrNotEnoughImages
	}
	rng := rand.New(rand.NewSource(unityBoardSeed(settings.Seed, gameID, ownerID)))
	cards := make([]Card, 0, totalCards)
	wordPerm := rng.Perm(len(uniqueWords))
	for i := 0; i < wordCount; i++ {
		cards = append(cards, Card{Content: CardContent{Type: ContentWord, Text: uniqueWords[wordPerm[i]]}})
	}
	imagePerm := rng.Perm(len(uniqueImages))
	for i := 0; i < imageCount; i++ {
		cards = append(cards, Card{Content: CardContent{Type: ContentImage, ImageID: uniqueImages[imagePerm[i]]}})
	}
	contentPerm := rng.Perm(totalCards)
	mixed := make([]Card, totalCards)
	for i, contentIndex := range contentPerm {
		mixed[i] = cards[contentIndex]
	}
	colors := make([]Color, 0, totalCards)
	for i := 0; i < settings.BlackCards; i++ {
		colors = append(colors, ColorBlack)
	}
	for i := 0; i < UnityCardCount(settings.TotalCards); i++ {
		colors = append(colors, ColorUnity)
	}
	for len(colors) < totalCards {
		colors = append(colors, ColorCivilian)
	}
	colorPerm := rng.Perm(totalCards)
	for i, colorIndex := range colorPerm {
		mixed[i].Color = colors[colorIndex]
	}
	return Board{Cards: mixed, StartingTeam: TeamUnity}, nil
}

func unityBoardSeed(seed int64, gameID string, ownerID string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s:%s", seed, gameID, ownerID)))
	return int64(h.Sum64())
}

func (s State) unityActivePlayerIDs() []string {
	ids := make([]string, 0, len(s.Players))
	for id, player := range s.Players {
		if player.Team == TeamUnity {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (s *State) applyLobbyModeSwitch(oldMode Mode, nextMode Mode) {
	if nextMode == ModeUnity {
		for id, player := range s.Players {
			if player.Team == TeamBlue || player.Team == TeamRed {
				player.PreviousTeam = player.Team
				player.PreviousSpymaster = player.Spymaster
				player.PreviousRepresentative = player.Representative
				player.Team = TeamUnity
				player.Spymaster = false
				s.Players[id] = player
			}
		}
		return
	}
	if oldMode == ModeUnity && nextMode == ModePolarity {
		for id, player := range s.Players {
			if player.Team == TeamUnity {
				if player.PreviousTeam == TeamBlue || player.PreviousTeam == TeamRed {
					player.Team = player.PreviousTeam
					player.Spymaster = player.PreviousSpymaster
					player.Representative = player.PreviousRepresentative && !player.Spymaster
				} else {
					player.Team = s.nextBalancedTeam(id)
					player.Spymaster = false
					player.Representative = false
				}
				s.Players[id] = player
			}
		}
	}
}

func (s State) activeUnityBoardIDs() []string {
	ids := make([]string, 0, len(s.UnityBoards))
	for _, id := range s.UnityBoardOrder {
		if s.Players[id].Team == TeamUnity && s.unityRemainingFor(id) > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}

func (s State) unityRemainingFor(ownerID string) int {
	board, ok := s.UnityBoards[ownerID]
	if !ok {
		return 0
	}
	left := 0
	for _, card := range board.Cards {
		if card.Color == ColorUnity && !card.Revealed {
			left++
		}
	}
	return left
}

func (s *State) advanceUnityTurnFrom(startIndex int) bool {
	active := s.activeUnityBoardIDs()
	if len(active) == 0 {
		s.unityWin("all_unity_found")
		return true
	}
	for offset := 1; offset <= len(s.UnityBoardOrder); offset++ {
		idx := (startIndex + offset) % len(s.UnityBoardOrder)
		ownerID := s.UnityBoardOrder[idx]
		if s.Players[ownerID].Team != TeamUnity || s.unityRemainingFor(ownerID) == 0 {
			continue
		}
		s.ActiveBoardOwner = ownerID
		if !s.hasEligibleUnityGuessers() {
			s.UnityWaitingForGuessers = true
			return false
		}
		if !s.chargeUnityTurn(ownerID) {
			return true
		}
		s.UnityWaitingForGuessers = false
		s.Round = s.startRound(TeamUnity)
		s.RoundGuesses = 0
		return true
	}
	s.UnityWaitingForGuessers = true
	return false
}

func (s *State) chargeUnityTurn(ownerID string) bool {
	if s.Settings.UnityUnlimitedTurns {
		return true
	}
	board := s.UnityBoards[ownerID]
	board.TurnsUsed++
	s.UnityBoards[ownerID] = board
	if s.Settings.UnityStrictPerBoardTurns {
		return true
	}
	if s.UnitySharedTurnsRemaining <= 0 {
		s.unityLose("turn_pool_empty")
		return false
	}
	s.UnitySharedTurnsRemaining--
	return true
}

func (s State) unityBoardIndex(ownerID string) int {
	for i, id := range s.UnityBoardOrder {
		if id == ownerID {
			return i
		}
	}
	return -1
}

func (s State) hasEligibleUnityGuessers() bool {
	for id := range s.Players {
		if s.isUnityActiveGuesser(id) {
			return true
		}
	}
	return false
}

func (s State) isUnityActiveGuesser(playerID string) bool {
	player, ok := s.Players[playerID]
	if !ok || player.Team != TeamUnity || playerID == s.ActiveBoardOwner {
		return false
	}
	reps := make([]Player, 0)
	for _, p := range s.Players {
		if p.Team == TeamUnity && p.Representative {
			reps = append(reps, p)
		}
	}
	if len(reps) == 0 {
		return true
	}
	if len(reps) == 1 && reps[0].ID == s.ActiveBoardOwner {
		return true
	}
	return player.Representative
}

func (c GuessCommand) guessUnity(state *State, actorID string) (Event, error) {
	board := state.UnityBoards[state.ActiveBoardOwner]
	if c.Index < 0 || c.Index >= len(board.Cards) {
		return Event{}, fmt.Errorf("%w: card index", ErrInvalidCommand)
	}
	if board.Cards[c.Index].Revealed {
		return Event{}, fmt.Errorf("%w: card revealed", ErrInvalidCommand)
	}
	if !state.IsActiveGuesser(actorID, TeamUnity) {
		return Event{}, ErrForbidden
	}
	if err := state.validateUnityGuessAgainstClue(&board); err != nil {
		return Event{}, err
	}
	board.Cards[c.Index].Revealed = true
	state.ActionID++
	state.LastSelected = &LastSelected{Index: c.Index, Team: TeamUnity}
	state.incrementUnityClueGuesses(&board)
	selectedColor := board.Cards[c.Index].Color
	state.UnityBoards[board.OwnerID] = board
	if selectedColor == ColorBlack {
		state.unityLose("assassin")
		return Event{Type: EventGuessAccepted}, nil
	}
	if state.allActiveUnityBoardsSolved() {
		state.unityWin("all_unity_found")
		return Event{Type: EventGuessAccepted}, nil
	}
	if selectedColor != ColorUnity {
		state.finalizeUnityRound(&board)
		state.UnityBoards[board.OwnerID] = board
		state.endUnityTurn(board.OwnerID)
	} else if state.Settings.EnforceClueGuessLimit {
		clue := currentUnityClue(board)
		if clue != nil && clue.Number.Kind == ClueNumberNumeric && clue.Guesses >= clue.Number.Value {
			state.finalizeUnityRound(&board)
			state.UnityBoards[board.OwnerID] = board
			state.endUnityTurn(board.OwnerID)
		}
	}
	return Event{Type: EventGuessAccepted}, nil
}

func (c PassCommand) passUnity(state *State, actorID string) (Event, error) {
	if !state.IsActiveGuesser(actorID, TeamUnity) {
		return Event{}, ErrForbidden
	}
	board := state.UnityBoards[state.ActiveBoardOwner]
	state.ActionID++
	state.finalizeUnityRound(&board)
	state.UnityBoards[board.OwnerID] = board
	state.endUnityTurn(board.OwnerID)
	return Event{Type: EventPassAccepted}, nil
}

func (s *State) endUnityTurn(ownerID string) {
	if s.Phase == PhaseGameOver {
		return
	}
	if s.Settings.UnityStrictPerBoardTurns && !s.Settings.UnityUnlimitedTurns {
		board := s.UnityBoards[ownerID]
		if board.TurnsUsed >= s.Settings.UnityTurnLimit && s.unityRemainingFor(ownerID) > 0 {
			s.unityLose("turn_limit")
			return
		}
	}
	if !s.Settings.UnityStrictPerBoardTurns && !s.Settings.UnityUnlimitedTurns && s.UnitySharedTurnsRemaining <= 0 && !s.allActiveUnityBoardsSolved() {
		if s.hasEligibleUnityGuessers() {
			s.unityLose("turn_pool_empty")
		} else {
			s.UnityWaitingForGuessers = true
		}
		return
	}
	s.advanceUnityTurnFrom(s.unityBoardIndex(ownerID))
}

func (c SubmitClueCommand) submitUnityClue(state *State, actorID string) (Event, error) {
	if actorID != state.ActiveBoardOwner {
		return Event{}, ErrForbidden
	}
	board := state.UnityBoards[state.ActiveBoardOwner]
	text := strings.TrimSpace(c.Text)
	if text == "" {
		return Event{}, fmt.Errorf("%w: empty clue text", ErrInvalidCommand)
	}
	if err := validateClueNumber(state.Settings, c.Number, currentUnityRoundGuesses(board, state.RoundGuesses)); err != nil {
		return Event{}, err
	}
	idx := ensureUnityClue(&board, state.Round)
	entry := board.ClueLog[idx]
	entry.Team = TeamUnity
	entry.Text = text
	entry.Number = c.Number
	entry.Status = ClueActive
	entry.Guesses = currentUnityRoundGuesses(board, state.RoundGuesses)
	if entry.SubmittedBy == "" {
		entry.SubmittedBy = actorID
	}
	entry.UpdatedBy = actorID
	board.ClueLog[idx] = entry
	state.UnityBoards[board.OwnerID] = board
	return Event{Type: EventClueSubmitted}, nil
}

func (s *State) validateUnityGuessAgainstClue(board *UnityBoardState) error {
	if !s.Settings.EnforceClueGuessLimit {
		return nil
	}
	clue := currentUnityClue(*board)
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

func ensureUnityClue(board *UnityBoardState, round int) int {
	if len(board.ClueLog) > 0 {
		idx := len(board.ClueLog) - 1
		if board.ClueLog[idx].Round == round && board.ClueLog[idx].Status == ClueActive {
			return idx
		}
	}
	board.ClueLog = append(board.ClueLog, ClueEntry{Round: round, Team: TeamUnity, Status: ClueActive})
	return len(board.ClueLog) - 1
}

func currentUnityClue(board UnityBoardState) *ClueEntry {
	if len(board.ClueLog) == 0 {
		return nil
	}
	entry := board.ClueLog[len(board.ClueLog)-1]
	if entry.Status == ClueActive {
		return &entry
	}
	return nil
}

func currentUnityRoundGuesses(board UnityBoardState, roundGuesses int) int {
	guesses := roundGuesses
	if len(board.ClueLog) > 0 {
		entry := board.ClueLog[len(board.ClueLog)-1]
		if entry.Status == ClueActive && entry.Guesses > guesses {
			guesses = entry.Guesses
		}
	}
	return guesses
}

func (s *State) incrementUnityClueGuesses(board *UnityBoardState) {
	s.RoundGuesses = currentUnityRoundGuesses(*board, s.RoundGuesses) + 1
	if len(board.ClueLog) == 0 {
		return
	}
	idx := len(board.ClueLog) - 1
	if board.ClueLog[idx].Status == ClueActive && board.ClueLog[idx].Text != "" {
		board.ClueLog[idx].Guesses = s.RoundGuesses
	}
}

func (s *State) finalizeUnityRound(board *UnityBoardState) {
	guesses := currentUnityRoundGuesses(*board, s.RoundGuesses)
	if len(board.ClueLog) > 0 {
		idx := len(board.ClueLog) - 1
		entry := board.ClueLog[idx]
		if entry.Status == ClueActive && entry.Text != "" {
			entry.Guesses = guesses
			entry.Status = ClueFinal
			board.ClueLog[idx] = entry
			return
		}
	}
	board.ClueLog = append(board.ClueLog, ClueEntry{Round: s.Round, Team: TeamUnity, Text: "NA", Number: ClueNumber{Kind: ClueNumberBlank}, Status: ClueNA, Guesses: guesses})
}

func (s *State) reconcileUnityRoster(playerID string, nextTeam Team) {
	if s.UnityBoards == nil {
		return
	}
	if nextTeam == TeamUnity {
		if _, ok := s.UnityBoards[playerID]; !ok {
			board, err := GenerateUnityBoard(s.Settings, s.GameID, playerID, s.UnityWords, s.UnityImageIDs)
			if err == nil {
				s.UnityBoards[playerID] = UnityBoardState{OwnerID: playerID, Cards: board.Cards}
				s.UnityBoardOrder = append(s.UnityBoardOrder, playerID)
				sort.Strings(s.UnityBoardOrder)
				if !s.Settings.UnityUnlimitedTurns && !s.Settings.UnityStrictPerBoardTurns {
					s.UnitySharedTurnsRemaining += s.Settings.UnityTurnLimit
				}
			}
		} else if !s.Settings.UnityUnlimitedTurns && !s.Settings.UnityStrictPerBoardTurns {
			board := s.UnityBoards[playerID]
			s.UnitySharedTurnsRemaining += board.WithdrawnSharedTurns
			board.WithdrawnSharedTurns = 0
			s.UnityBoards[playerID] = board
		}
	} else {
		board, boardExists := s.UnityBoards[playerID]
		if !boardExists {
			s.ResolveUnityAfterRosterChange()
			return
		}
		if !s.Settings.UnityUnlimitedTurns && !s.Settings.UnityStrictPerBoardTurns {
			remainingContribution := s.Settings.UnityTurnLimit - board.TurnsUsed
			if remainingContribution < 0 {
				remainingContribution = 0
			}
			board.WithdrawnSharedTurns = remainingContribution
			if remainingContribution > s.UnitySharedTurnsRemaining {
				s.UnitySharedTurnsRemaining = 0
			} else {
				s.UnitySharedTurnsRemaining -= remainingContribution
			}
			s.UnityBoards[playerID] = board
		}
		if s.ActiveBoardOwner == playerID {
			s.advanceUnityTurnFrom(s.unityBoardIndex(playerID))
		}
	}
	s.ResolveUnityAfterRosterChange()
}

// ResolveUnityAfterRosterChange applies immediate Unity win/loss/wait checks.
func (s *State) ResolveUnityAfterRosterChange() {
	if s.Mode != ModeUnity || s.Phase != PhaseActive {
		return
	}
	if s.allActiveUnityBoardsSolved() {
		s.unityWin("all_unity_found")
		return
	}
	if s.Players[s.ActiveBoardOwner].Team != TeamUnity || s.unityRemainingFor(s.ActiveBoardOwner) == 0 {
		s.advanceUnityTurnFrom(s.unityBoardIndex(s.ActiveBoardOwner))
	}
	if !s.hasEligibleUnityGuessers() {
		s.UnityWaitingForGuessers = true
		return
	}
	if !s.Settings.UnityUnlimitedTurns && !s.Settings.UnityStrictPerBoardTurns && s.UnitySharedTurnsRemaining <= 0 {
		s.unityLose("turn_pool_empty")
	}
}

func (s State) allActiveUnityBoardsSolved() bool {
	for _, board := range s.UnityBoards {
		if s.Players[board.OwnerID].Team == TeamUnity && s.unityRemainingFor(board.OwnerID) > 0 {
			return false
		}
	}
	return true
}

func (s *State) unityWin(reason string) {
	s.Winner = TeamUnity
	s.Phase = PhaseGameOver
	s.UnityEndStats = s.buildUnityEndStats(reason)
}

func (s *State) unityLose(reason string) {
	s.Winner = TeamObservers
	s.Phase = PhaseGameOver
	s.UnityEndStats = s.buildUnityEndStats(reason)
}

func (s State) buildUnityEndStats(reason string) *UnityEndStats {
	found, total, assassins, turns := 0, 0, 0, 0
	for _, board := range s.UnityBoards {
		turns += board.TurnsUsed
		for _, card := range board.Cards {
			if card.Color == ColorUnity {
				total++
				if card.Revealed {
					found++
				}
			}
			if card.Color == ColorBlack {
				assassins++
			}
		}
	}
	score := 0.0
	if turns > 0 {
		score = float64(found) / float64(turns)
	}
	return &UnityEndStats{UnityCardsFound: found, TotalUnityCards: total, TotalTurns: turns, AssassinCount: assassins, Score: score, Reason: reason}
}

func (s State) unitySnapshotFor(viewer Viewer) Snapshot {
	activeBoard := s.safeUnityBoardSnapshot(s.ActiveBoardOwner, viewer.PlayerID)
	var ownBoard *SnapshotBoard
	if viewer.PlayerID != "" {
		if _, ok := s.UnityBoards[viewer.PlayerID]; ok {
			board := s.safeUnityBoardSnapshot(viewer.PlayerID, viewer.PlayerID)
			ownBoard = &board
		}
	}
	summaries := make([]UnityBoardSummary, 0, len(s.UnityBoardOrder))
	for _, id := range s.UnityBoardOrder {
		board := s.UnityBoards[id]
		summaries = append(summaries, UnityBoardSummary{OwnerID: id, UnityRemaining: s.unityRemainingFor(id), TurnsUsed: board.TurnsUsed, Active: id == s.ActiveBoardOwner, Observer: s.Players[id].Team == TeamObservers})
	}
	found, total := 0, 0
	for _, board := range s.UnityBoards {
		for _, card := range board.Cards {
			if card.Color == ColorUnity {
				total++
				if card.Revealed {
					found++
				}
			}
		}
	}
	return Snapshot{
		Phase:         s.Phase,
		CurrentTeam:   TeamUnity,
		Winner:        s.Winner,
		FinishedAt:    s.FinishedAt,
		ActionID:      s.ActionID,
		Cards:         activeBoard.Cards,
		LastSelected:  s.LastSelected,
		ClueLog:       activeBoard.ClueLog,
		ActiveBoard:   activeBoard,
		OwnBoard:      ownBoard,
		UnityBoards:   summaries,
		UnityProgress: UnityProgress{UnityCardsFound: found, TotalUnityCards: total, SharedTurnsRemaining: s.UnitySharedTurnsRemaining, UnlimitedTurns: s.Settings.UnityUnlimitedTurns, StrictPerBoardTurns: s.Settings.UnityStrictPerBoardTurns, WaitingForGuessers: s.UnityWaitingForGuessers},
		UnityEndStats: s.UnityEndStats,
	}
}

func (s State) safeUnityBoardSnapshot(ownerID string, viewerID string) SnapshotBoard {
	board := s.UnityBoards[ownerID]
	showAll := s.Phase == PhaseGameOver || ownerID == viewerID
	cards := make([]SnapshotCard, len(board.Cards))
	counts := CardCounts{}
	for i, card := range board.Cards {
		cards[i] = SnapshotCard{Content: card.Content, Revealed: card.Revealed}
		if showAll || card.Revealed {
			cards[i].Color = card.Color
		}
		if !card.Revealed {
			switch card.Color {
			case ColorUnity:
				counts.Blue++
			case ColorCivilian:
				counts.Civilian++
			case ColorBlack:
				counts.Black++
			}
		}
	}
	log := make([]ClueEntry, len(board.ClueLog))
	copy(log, board.ClueLog)
	return SnapshotBoard{OwnerID: ownerID, Cards: cards, ClueLog: log, TurnsUsed: board.TurnsUsed, RemainingCounts: counts}
}
