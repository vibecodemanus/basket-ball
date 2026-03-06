package game

import (
	"sort"
	"sync"
)

// PlayerStats tracks a single player's tournament performance.
type PlayerStats struct {
	Nickname      string `json:"nickname"`
	Wins          int    `json:"wins"`
	Losses        int    `json:"losses"`
	Draws         int    `json:"draws"`
	PointsFor     int    `json:"pointsFor"`
	PointsAgainst int    `json:"pointsAgainst"`
	GamesPlayed   int    `json:"gamesPlayed"`
}

// LeaderboardEntry is the JSON-serializable leaderboard row.
type LeaderboardEntry struct {
	Nickname      string `json:"nickname"`
	Wins          int    `json:"wins"`
	Losses        int    `json:"losses"`
	Draws         int    `json:"draws"`
	PointsFor     int    `json:"pointsFor"`
	PointsAgainst int    `json:"pointsAgainst"`
	GamesPlayed   int    `json:"gamesPlayed"`
}

// Tournament holds all in-memory tournament state.
type Tournament struct {
	mu       sync.RWMutex
	stats    map[string]*PlayerStats
	pairings map[string]map[string]int // pairings[a][b] = times played
}

func NewTournament() *Tournament {
	return &Tournament{
		stats:    make(map[string]*PlayerStats),
		pairings: make(map[string]map[string]int),
	}
}

// getOrCreate returns stats for a nickname, creating if needed. Caller must hold lock.
func (t *Tournament) getOrCreate(nickname string) *PlayerStats {
	s, ok := t.stats[nickname]
	if !ok {
		s = &PlayerStats{Nickname: nickname}
		t.stats[nickname] = s
	}
	return s
}

// RecordResult updates tournament stats after a game.
func (t *Tournament) RecordResult(nick1, nick2 string, score1, score2 uint8) {
	t.mu.Lock()
	defer t.mu.Unlock()

	s1 := t.getOrCreate(nick1)
	s2 := t.getOrCreate(nick2)

	s1.GamesPlayed++
	s2.GamesPlayed++
	s1.PointsFor += int(score1)
	s1.PointsAgainst += int(score2)
	s2.PointsFor += int(score2)
	s2.PointsAgainst += int(score1)

	if score1 > score2 {
		s1.Wins++
		s2.Losses++
	} else if score2 > score1 {
		s2.Wins++
		s1.Losses++
	} else {
		s1.Draws++
		s2.Draws++
	}

	// Update pairings
	if t.pairings[nick1] == nil {
		t.pairings[nick1] = make(map[string]int)
	}
	if t.pairings[nick2] == nil {
		t.pairings[nick2] = make(map[string]int)
	}
	t.pairings[nick1][nick2]++
	t.pairings[nick2][nick1]++
}

// GetStats returns a copy of stats for a nickname.
func (t *Tournament) GetStats(nickname string) PlayerStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	s, ok := t.stats[nickname]
	if !ok {
		return PlayerStats{Nickname: nickname}
	}
	return *s
}

// HavePlayedBefore returns true if these two nicknames have been matched before.
func (t *Tournament) HavePlayedBefore(nick1, nick2 string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if m, ok := t.pairings[nick1]; ok {
		return m[nick2] > 0
	}
	return false
}

// TimesPlayed returns how many times two nicknames have played each other.
func (t *Tournament) TimesPlayed(nick1, nick2 string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if m, ok := t.pairings[nick1]; ok {
		return m[nick2]
	}
	return 0
}

// LeaderboardByWins returns top players sorted by wins desc, then points desc.
func (t *Tournament) LeaderboardByWins(limit int) []LeaderboardEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entries := t.allEntries()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Wins != entries[j].Wins {
			return entries[i].Wins > entries[j].Wins
		}
		return entries[i].PointsFor > entries[j].PointsFor
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}

// LeaderboardByPoints returns top players sorted by points desc, then wins desc.
func (t *Tournament) LeaderboardByPoints(limit int) []LeaderboardEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entries := t.allEntries()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].PointsFor != entries[j].PointsFor {
			return entries[i].PointsFor > entries[j].PointsFor
		}
		return entries[i].Wins > entries[j].Wins
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries
}

// allEntries returns all stats as LeaderboardEntry slice. Caller must hold RLock.
func (t *Tournament) allEntries() []LeaderboardEntry {
	entries := make([]LeaderboardEntry, 0, len(t.stats))
	for _, s := range t.stats {
		entries = append(entries, LeaderboardEntry{
			Nickname:      s.Nickname,
			Wins:          s.Wins,
			Losses:        s.Losses,
			Draws:         s.Draws,
			PointsFor:     s.PointsFor,
			PointsAgainst: s.PointsAgainst,
			GamesPlayed:   s.GamesPlayed,
		})
	}
	return entries
}
