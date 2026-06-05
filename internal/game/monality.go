package game

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const MonalityBombScore = -1

type CloseMonalityRoundCommand struct {
	ConnectedPlayerIDs []string
	Now                time.Time
}

func (c StartCommand) startMonality(state *State, actorID string) (Event, error) {
	state.GameID = strings.TrimSpace(c.GameID)
	state.Mode = ModeMonality
	state.CurrentTeam = TeamMonality
	state.Phase = PhaseActive
	state.Winner = ""
	state.FinishedAt = ""
	state.ActionID = 0
	state.LastSelected = nil
	state.ClueLog = nil
	state.Round = 0
	state.RoundGuesses = 0
	state.RoundGuessesByPlayer = map[string]int{}
	state.ActiveBoardOwner = ""
	state.PreviousBoardOwner = ""
	state.MonalitySpymasterCounts = map[string]int{}
	state.MonalityTotalScores = map[string]float64{}
	state.MonalityRoundScores = nil
	state.MonalityEndStats = nil
	state.UnityWords = uniqueWords(c.Words)
	state.UnityImageIDs = uniqueImageIDs(c.ImageIDs)
	for _, id := range state.monalityActivePlayerIDs() {
		state.MonalitySpymasterCounts[id] = 0
		state.MonalityTotalScores[id] = 0
	}
	if err := state.startNextMonalityRound(time.Time{}); err != nil {
		return Event{}, err
	}
	return Event{Type: EventMatchStarted}, nil
}

func (s *State) startNextMonalityRound(now time.Time) error {
	if s.monalityComplete() {
		s.finishMonality()
		return nil
	}
	spy := s.chooseMonalitySpymaster()
	if spy == "" {
		return ErrCannotStart
	}
	s.Round = s.startRound(TeamMonality)
	s.RoundGuesses = 0
	s.RoundGuessesByPlayer = map[string]int{}
	s.MonalitySpymasterID = spy
	s.ActiveBoardOwner = spy
	board, err := GenerateUnityBoard(s.Settings, s.GameID, fmt.Sprintf("monality-round-%d", s.Round), s.UnityWords, s.UnityImageIDs)
	if err != nil {
		return err
	}
	s.MonalityBoard = UnityBoardState{OwnerID: spy, Cards: board.Cards}
	s.MonalityAttempts = map[string]MonalityAttemptState{}
	for _, id := range s.monalityActivePlayerIDs() {
		if id == spy {
			continue
		}
		s.MonalityAttempts[id] = MonalityAttemptState{OwnerID: id, Cards: cloneCards(board.Cards)}
	}
	if s.Settings.MonalityRoundSeconds > 0 {
		base := nowOrCurrent(now)
		s.MonalityDeadline = base.Add(time.Duration(s.Settings.MonalityRoundSeconds) * time.Second).UTC().Format(time.RFC3339Nano)
	} else {
		s.MonalityDeadline = ""
	}
	return nil
}

func (s State) monalityComplete() bool {
	ids := s.monalityActivePlayerIDs()
	if len(ids) < 2 {
		return true
	}
	limit := SettingsWithDefaults(s.Settings).MonalitySpymasterRounds
	for _, id := range ids {
		if s.MonalitySpymasterCounts[id] < limit {
			return false
		}
	}
	return true
}

func (s State) chooseMonalitySpymaster() string {
	ids := s.monalityActivePlayerIDs()
	if len(ids) == 0 {
		return ""
	}
	limit := SettingsWithDefaults(s.Settings).MonalitySpymasterRounds
	minCount := int(^uint(0) >> 1)
	candidates := make([]string, 0, len(ids))
	for _, id := range ids {
		count := s.MonalitySpymasterCounts[id]
		if count >= limit {
			continue
		}
		if count < minCount {
			minCount = count
			candidates = candidates[:0]
		}
		if count == minCount {
			candidates = append(candidates, id)
		}
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.Strings(candidates)
	rng := rand.New(rand.NewSource(monalityRoundSeed(s.Settings.Seed, s.GameID, s.Round+1, minCount)))
	return candidates[rng.Intn(len(candidates))]
}

func monalityRoundSeed(seed int64, gameID string, round int, minCount int) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%d:%s:%d:%d:monality", seed, gameID, round, minCount)))
	return int64(h.Sum64())
}

func (s State) monalityActivePlayerIDs() []string {
	ids := make([]string, 0, len(s.Players))
	for id, player := range s.Players {
		if player.Team == TeamMonality {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (c SubmitClueCommand) submitMonalityClue(state *State, actorID string) (Event, error) {
	if actorID != state.MonalitySpymasterID {
		return Event{}, ErrForbidden
	}
	text := strings.TrimSpace(c.Text)
	if text == "" {
		return Event{}, fmt.Errorf("%w: empty clue text", ErrInvalidCommand)
	}
	if err := validateClueNumber(state.Settings, c.Number, state.maxMonalityRoundGuesses()); err != nil {
		return Event{}, err
	}
	idx := ensureUnityClue(&state.MonalityBoard, state.Round)
	entry := state.MonalityBoard.ClueLog[idx]
	entry.Team = TeamMonality
	entry.Text = text
	entry.Number = c.Number
	entry.Status = ClueActive
	entry.Guesses = state.maxMonalityRoundGuesses()
	if entry.SubmittedBy == "" {
		entry.SubmittedBy = actorID
	}
	entry.UpdatedBy = actorID
	state.MonalityBoard.ClueLog[idx] = entry
	for id, attempt := range state.MonalityAttempts {
		attempt.ClueLog = cloneClueLog(state.MonalityBoard.ClueLog)
		state.MonalityAttempts[id] = attempt
	}
	return Event{Type: EventClueSubmitted}, nil
}

func (c GuessCommand) guessMonality(state *State, actorID string) (Event, error) {
	attempt, ok := state.MonalityAttempts[actorID]
	if !ok || attempt.Completed || attempt.Abandoned {
		return Event{}, ErrForbidden
	}
	if c.Index < 0 || c.Index >= len(attempt.Cards) {
		return Event{}, fmt.Errorf("%w: card index", ErrInvalidCommand)
	}
	if attempt.Cards[c.Index].Revealed {
		return Event{}, fmt.Errorf("%w: card revealed", ErrInvalidCommand)
	}
	if err := state.validateMonalityGuessAgainstClue(attempt); err != nil {
		return Event{}, err
	}
	attempt.Cards[c.Index].Revealed = true
	attempt.LastSelected = &LastSelected{Index: c.Index, Team: TeamMonality}
	state.LastSelected = attempt.LastSelected
	state.RoundGuessesByPlayer[actorID]++
	state.incrementMonalityClueGuesses(actorID, &attempt)
	switch attempt.Cards[c.Index].Color {
	case ColorUnity:
		attempt.Score++
		if monalityRemaining(attempt.Cards) == 0 || state.monalityClueLimitReached(attempt) {
			attempt.Completed = true
		}
	case ColorBlack:
		attempt.Score = MonalityBombScore
		attempt.Bombed = true
		attempt.Completed = true
	default:
		attempt.Completed = true
	}
	state.MonalityAttempts[actorID] = attempt
	state.ActionID++
	state.resolveMonalityRoundIfComplete(c.Now)
	return Event{Type: EventGuessAccepted}, nil
}

func (c PassCommand) passMonality(state *State, actorID string) (Event, error) {
	attempt, ok := state.MonalityAttempts[actorID]
	if !ok || attempt.Completed || attempt.Abandoned {
		return Event{}, ErrForbidden
	}
	attempt.Completed = true
	state.MonalityAttempts[actorID] = attempt
	state.ActionID++
	state.resolveMonalityRoundIfComplete(c.Now)
	return Event{Type: EventPassAccepted}, nil
}

func (c CloseMonalityRoundCommand) apply(state *State, actorID string) (Event, error) {
	if state.Phase != PhaseActive || state.Mode != ModeMonality {
		return Event{}, fmt.Errorf("%w: close outside active monality phase", ErrInvalidCommand)
	}
	if !state.CanManage(actorID) {
		return Event{}, ErrForbidden
	}
	connected := map[string]bool{}
	for _, id := range c.ConnectedPlayerIDs {
		connected[id] = true
	}
	for id, attempt := range state.MonalityAttempts {
		if attempt.Completed {
			continue
		}
		player := state.Players[id]
		if player.Team == TeamObservers || !connected[id] {
			attempt.Abandoned = true
		} else {
			attempt.Completed = true
		}
		state.MonalityAttempts[id] = attempt
	}
	state.ActionID++
	state.scoreAndAdvanceMonalityRound(c.Now)
	return Event{Type: EventMonalityRoundClosed}, nil
}

func (s *State) resolveMonalityRoundIfComplete(now time.Time) {
	for _, attempt := range s.MonalityAttempts {
		if !attempt.Completed && !attempt.Abandoned {
			return
		}
	}
	s.scoreAndAdvanceMonalityRound(now)
}

func (s *State) scoreAndAdvanceMonalityRound(now time.Time) {
	if s.Phase != PhaseActive || s.Mode != ModeMonality {
		return
	}
	if len(s.MonalityBoard.ClueLog) > 0 {
		s.finalizeMonalityClue()
	}
	scores := map[string]float64{}
	total := 0.0
	count := 0
	for id, attempt := range s.MonalityAttempts {
		if attempt.Abandoned {
			continue
		}
		scores[id] = attempt.Score
		s.MonalityTotalScores[id] += attempt.Score
		total += attempt.Score
		count++
	}
	avg := 0.0
	if count > 0 {
		avg = total / float64(count)
	}
	s.MonalityTotalScores[s.MonalitySpymasterID] += avg
	s.MonalitySpymasterCounts[s.MonalitySpymasterID]++
	s.MonalityRoundScores = append(s.MonalityRoundScores, MonalityRoundStats{Round: s.Round, SpymasterID: s.MonalitySpymasterID, Average: avg, Scores: scores})
	if err := s.startNextMonalityRound(now); err != nil {
		s.finishMonality()
	}
}

func (s *State) finishMonality() {
	s.Phase = PhaseGameOver
	s.Winner = TeamMonality
	s.FinishedAt = nowOrCurrent(time.Time{}).Format(time.RFC3339Nano)
	s.MonalityDeadline = ""
	rankings := make([]MonalityRanking, 0, len(s.monalityActivePlayerIDs()))
	for _, id := range s.monalityActivePlayerIDs() {
		rankings = append(rankings, MonalityRanking{PlayerID: id, TotalScore: s.MonalityTotalScores[id]})
	}
	sort.SliceStable(rankings, func(i, j int) bool {
		if rankings[i].TotalScore == rankings[j].TotalScore {
			return rankings[i].PlayerID < rankings[j].PlayerID
		}
		return rankings[i].TotalScore > rankings[j].TotalScore
	})
	lastRank := 0
	var lastScore float64
	for i := range rankings {
		if i == 0 || rankings[i].TotalScore != lastScore {
			lastRank = i + 1
			lastScore = rankings[i].TotalScore
		}
		rankings[i].Rank = lastRank
	}
	s.MonalityEndStats = &MonalityEndStats{Rankings: rankings, Rounds: len(s.MonalityRoundScores)}
}

func (s *State) validateMonalityGuessAgainstClue(attempt MonalityAttemptState) error {
	if !s.Settings.EnforceClueGuessLimit {
		return nil
	}
	clue := currentUnityClue(s.MonalityBoard)
	if clue == nil || clue.Number.Kind == ClueNumberBlank {
		return ErrClueRequired
	}
	if clue.Number.Kind == ClueNumberInfinity {
		return nil
	}
	if s.RoundGuessesByPlayer[attempt.OwnerID] >= clue.Number.Value {
		return ErrGuessLimitReached
	}
	return nil
}

func (s State) monalityClueLimitReached(attempt MonalityAttemptState) bool {
	if !s.Settings.EnforceClueGuessLimit {
		return false
	}
	clue := currentUnityClue(s.MonalityBoard)
	return clue != nil && clue.Number.Kind == ClueNumberNumeric && s.RoundGuessesByPlayer[attempt.OwnerID] >= clue.Number.Value
}

func (s State) maxMonalityRoundGuesses() int {
	max := 0
	for _, n := range s.RoundGuessesByPlayer {
		if n > max {
			max = n
		}
	}
	return max
}

func (s *State) incrementMonalityClueGuesses(playerID string, attempt *MonalityAttemptState) {
	guesses := s.RoundGuessesByPlayer[playerID]
	if len(attempt.ClueLog) > 0 {
		idx := len(attempt.ClueLog) - 1
		if attempt.ClueLog[idx].Status == ClueActive && attempt.ClueLog[idx].Text != "" {
			attempt.ClueLog[idx].Guesses = guesses
		}
	}
	if len(s.MonalityBoard.ClueLog) > 0 {
		idx := len(s.MonalityBoard.ClueLog) - 1
		if s.MonalityBoard.ClueLog[idx].Status == ClueActive && s.MonalityBoard.ClueLog[idx].Text != "" {
			s.MonalityBoard.ClueLog[idx].Guesses = s.maxMonalityRoundGuesses()
		}
	}
}

func (s *State) finalizeMonalityClue() {
	if len(s.MonalityBoard.ClueLog) == 0 {
		return
	}
	idx := len(s.MonalityBoard.ClueLog) - 1
	if s.MonalityBoard.ClueLog[idx].Status == ClueActive {
		s.MonalityBoard.ClueLog[idx].Guesses = s.maxMonalityRoundGuesses()
		s.MonalityBoard.ClueLog[idx].Status = ClueFinal
	}
	for id, attempt := range s.MonalityAttempts {
		attempt.ClueLog = cloneClueLog(s.MonalityBoard.ClueLog)
		stateIdx := len(attempt.ClueLog) - 1
		if stateIdx >= 0 {
			attempt.ClueLog[stateIdx].Guesses = s.RoundGuessesByPlayer[id]
		}
		s.MonalityAttempts[id] = attempt
	}
}

func monalityRemaining(cards []Card) int {
	left := 0
	for _, card := range cards {
		if card.Color == ColorUnity && !card.Revealed {
			left++
		}
	}
	return left
}

func cloneCards(cards []Card) []Card {
	out := make([]Card, len(cards))
	copy(out, cards)
	return out
}

func cloneClueLog(log []ClueEntry) []ClueEntry {
	out := make([]ClueEntry, len(log))
	copy(out, log)
	return out
}

func (s State) monalitySnapshotFor(viewer Viewer) Snapshot {
	mon := &MonalitySnapshot{SpymasterID: s.MonalitySpymasterID, Scores: map[string]float64{}, SpymasterCounts: map[string]int{}, Deadline: s.MonalityDeadline, EndStats: s.MonalityEndStats, RoundScores: append([]MonalityRoundStats(nil), s.MonalityRoundScores...)}
	for id, score := range s.MonalityTotalScores {
		mon.Scores[id] = score
	}
	for id, count := range s.MonalitySpymasterCounts {
		mon.SpymasterCounts[id] = count
	}
	if viewer.PlayerID == s.MonalitySpymasterID || s.Phase == PhaseGameOver {
		board := s.safeMonalityBoardSnapshot(true)
		mon.Board = &board
	}
	if attempt, ok := s.MonalityAttempts[viewer.PlayerID]; ok {
		board := s.safeMonalityAttemptSnapshot(attempt, viewer.PlayerID == s.MonalitySpymasterID || s.Phase == PhaseGameOver)
		mon.OwnAttempt = &board
	}
	ids := make([]string, 0, len(s.MonalityAttempts))
	for id := range s.MonalityAttempts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		attempt := s.MonalityAttempts[id]
		mon.Attempts = append(mon.Attempts, MonalityAttemptSummary{OwnerID: id, Score: attempt.Score, Completed: attempt.Completed, Abandoned: attempt.Abandoned, Bombed: attempt.Bombed})
	}
	cards := []SnapshotCard{}
	clueLog := []ClueEntry{}
	if mon.OwnAttempt != nil {
		cards = mon.OwnAttempt.Cards
		clueLog = mon.OwnAttempt.ClueLog
	} else if mon.Board != nil {
		cards = mon.Board.Cards
		clueLog = mon.Board.ClueLog
	}
	return Snapshot{Phase: s.Phase, CurrentTeam: TeamMonality, Winner: s.Winner, FinishedAt: s.FinishedAt, ActionID: s.ActionID, Cards: cards, LastSelected: s.LastSelected, ClueLog: clueLog, Monality: mon}
}

func (s State) safeMonalityBoardSnapshot(showAll bool) SnapshotBoard {
	board := s.MonalityBoard
	cards := snapshotCards(board.Cards, showAll)
	return SnapshotBoard{OwnerID: board.OwnerID, Cards: cards, ClueLog: cloneClueLog(board.ClueLog), LastSelected: board.LastSelected, RemainingCounts: remainingCountsFor(board.Cards)}
}

func (s State) safeMonalityAttemptSnapshot(attempt MonalityAttemptState, showAll bool) SnapshotBoard {
	return SnapshotBoard{OwnerID: attempt.OwnerID, Cards: snapshotCards(attempt.Cards, showAll), ClueLog: cloneClueLog(attempt.ClueLog), LastSelected: attempt.LastSelected, RemainingCounts: remainingCountsFor(attempt.Cards)}
}

func snapshotCards(cards []Card, showAll bool) []SnapshotCard {
	out := make([]SnapshotCard, len(cards))
	for i, card := range cards {
		out[i] = SnapshotCard{Content: card.Content, Revealed: card.Revealed}
		if showAll || card.Revealed {
			out[i].Color = card.Color
		}
	}
	return out
}

func remainingCountsFor(cards []Card) CardCounts {
	counts := CardCounts{}
	for _, card := range cards {
		if card.Revealed {
			continue
		}
		switch card.Color {
		case ColorUnity:
			counts.Blue++
		case ColorBlack:
			counts.Black++
		case ColorCivilian:
			counts.Civilian++
		}
	}
	return counts
}

func roundFloat(v float64) float64 {
	return math.Round(v*1000) / 1000
}
