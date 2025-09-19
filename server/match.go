package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

const (
	OpMove = 1
)

type MatchState struct {
    Board      [9]rune // X, O, or 0
    Turn       rune    // 'X' or 'O'
    Presences  map[string]runtime.Presence
    StartedAt  time.Time
    Winner     rune   // 'X', 'O', or 0 ('.' draw)
    Mode       string
    MatchDbID  string
    UserSymbol map[string]rune
    Finalized  bool
}

type MovePayload struct {
	Index int `json:"index"`
}

func registerMatch(initializer runtime.Initializer) error {
	return initializer.RegisterMatch("tictactoe", NewMatch)
}

func NewMatch(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (runtime.Match, error) {
    mode := "casual"
    if v, ok := params["mode"].(string); ok && v != "" { mode = v }
    state := &MatchState{Turn: 'X', Presences: map[string]runtime.Presence{}, StartedAt: time.Now(), Mode: mode, UserSymbol: map[string]rune{}}
    return &TicTacToeMatch{state: state, logger: logger}, nil
}

type TicTacToeMatch struct {
	state  *MatchState
	logger runtime.Logger
}

func (m *TicTacToeMatch) MatchInit(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (interface{}, int, string) {
	return m.state, 1, ""
}

func (m *TicTacToeMatch) MatchJoinAttempt(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presence runtime.Presence, metadata map[string]string) (interface{}, bool, string) {
	st := state.(*MatchState)
	if len(st.Presences) >= 2 {
		return st, false, "match full"
	}
	return st, true, ""
}

func (m *TicTacToeMatch) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	st := state.(*MatchState)
    for _, p := range presences {
        st.Presences[p.GetUserId()] = p
        if _, ok := st.UserSymbol[p.GetUserId()]; !ok {
            if len(st.UserSymbol) == 0 { st.UserSymbol[p.GetUserId()] = 'X' } else { st.UserSymbol[p.GetUserId()] = 'O' }
        }
    }
    if len(st.Presences) == 2 && st.MatchDbID == "" {
        if id, err := createMatchRow(ctx, db, st); err == nil { st.MatchDbID = id } else { logger.Error("createMatchRow failed: %v", err) }
    }
    return st
}

func (m *TicTacToeMatch) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
	st := state.(*MatchState)
	for _, p := range presences {
		delete(st.Presences, p.GetUserId())
	}
	return st
}

func (m *TicTacToeMatch) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
	st := state.(*MatchState)
	// Turn timeout: 30s
	if st.Winner == 0 && time.Since(st.StartedAt) > 30*time.Second {
		st.Winner = other(st.Turn) // opponent wins on timeout
	}
	for _, msg := range messages {
		if msg.GetOpCode() == OpMove && st.Winner == 0 {
			var mv MovePayload
			if err := json.Unmarshal(msg.GetData(), &mv); err != nil {
				continue
			}
			if mv.Index < 0 || mv.Index >= 9 {
				continue
			}
			mark := st.Turn
			if st.Board[mv.Index] != 0 {
				continue
			}
			st.Board[mv.Index] = mark
			if checkWin(st.Board, mark) {
				st.Winner = mark
			}
			// draw detection
			if st.Winner == 0 {
				full := true
				for _, v := range st.Board {
					if v == 0 {
						full = false
						break
					}
				}
				if full {
					st.Winner = '.'
				} // '.' denotes draw
			}
			if st.Winner == 0 {
				st.Turn = other(mark)
			}
			st.StartedAt = time.Now()
			payload, _ := json.Marshal(map[string]interface{}{
				"board":  stringFromBoard(st.Board),
				"turn":   string([]rune{st.Turn}),
				"winner": string([]rune{st.Winner}),
			})
			dispatcher.BroadcastMessage(OpMove, payload, nil, nil, true)
		}
	}
    if st.Winner != 0 && !st.Finalized {
        if err := persistResults(ctx, db, st); err != nil { logger.Error("persistResults failed: %v", err) } else { st.Finalized = true }
    }
    return st
}

func (m *TicTacToeMatch) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
	return state
}

func other(r rune) rune {
	if r == 'X' {
		return 'O'
	}
	return 'X'
}

func checkWin(b [9]rune, r rune) bool {
	lines := [8][3]int{{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, {0, 3, 6}, {1, 4, 7}, {2, 5, 8}, {0, 4, 8}, {2, 4, 6}}
	for _, ln := range lines {
		if b[ln[0]] == r && b[ln[1]] == r && b[ln[2]] == r {
			return true
		}
	}
	return false
}

func stringFromBoard(b [9]rune) string {
	bytes := make([]rune, 9)
	for i, v := range b {
		if v == 0 {
			bytes[i] = '.'
		} else {
			bytes[i] = v
		}
	}
	return string(bytes)
}

// RPC to create a new match and return match ID for testing
func registerCreateMatchRPC(initializer runtime.Initializer) error {
	return initializer.RegisterRpc("create_match", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		matchID, err := nk.MatchCreate(ctx, "tictactoe", map[string]interface{}{})
		if err != nil {
			return "", runtime.NewError(fmt.Sprintf("create failed: %v", err), 13)
		}
		out, _ := json.Marshal(map[string]string{"match_id": matchID})
		return string(out), nil
	})
}

// --- Persistence helpers ---
func createMatchRow(ctx context.Context, db *sql.DB, st *MatchState) (string, error) {
    var matchID string
    snap, _ := json.Marshal(map[string]interface{}{"board": stringFromBoard(st.Board), "turn": string([]rune{st.Turn})})
    if err := db.QueryRowContext(ctx, `INSERT INTO matches(mode, state_snapshot) VALUES($1,$2) RETURNING id`, st.Mode, string(snap)).Scan(&matchID); err != nil {
        return "", err
    }
    for uid, sym := range st.UserSymbol {
        var pid string
        var elo int
        if err := db.QueryRowContext(ctx, `SELECT id, elo FROM players WHERE nakama_user_id=$1`, uid).Scan(&pid, &elo); err != nil {
            return "", err
        }
        if _, err := db.ExecContext(ctx, `INSERT INTO match_players(match_id, player_id, symbol, rating_before) VALUES($1,$2,$3,$4)`, matchID, pid, string([]rune{sym}), elo); err != nil {
            return "", err
        }
    }
    return matchID, nil
}

func persistResults(ctx context.Context, db *sql.DB, st *MatchState) error {
    if st.MatchDbID == "" { return nil }
    var winnerPID *string
    if st.Winner == 'X' || st.Winner == 'O' {
        var uid string
        for u, sym := range st.UserSymbol { if sym == st.Winner { uid = u; break } }
        var pid string
        if err := db.QueryRowContext(ctx, `SELECT id FROM players WHERE nakama_user_id=$1`, uid).Scan(&pid); err == nil { winnerPID = &pid }
    }
    type rec struct{ id string; elo int }
    rows, err := db.QueryContext(ctx, `SELECT mp.player_id, p.elo FROM match_players mp JOIN players p ON p.id=mp.player_id WHERE mp.match_id=$1`, st.MatchDbID)
    if err != nil { return err }
    players := make([]rec, 0, 2)
    for rows.Next() { var r rec; if err := rows.Scan(&r.id, &r.elo); err != nil { return err }; players = append(players, r) }
    rows.Close()
    if len(players) == 2 && st.Winner != '.' {
        exp := func(a,b int) float64 { return 1.0 / (1.0 + math.Pow(10, float64(b-a)/400.0)) }
        symbolOf := func(pid string) rune { var uid string; _ = db.QueryRowContext(ctx, `SELECT nakama_user_id FROM players WHERE id=$1`, pid).Scan(&uid); return st.UserSymbol[uid] }
        eA, eB := exp(players[0].elo, players[1].elo), exp(players[1].elo, players[0].elo)
        var sA, sB float64
        if st.Winner == symbolOf(players[0].id) { sA, sB = 1, 0 } else { sA, sB = 0, 1 }
        k := 20.0
        newA := int(float64(players[0].elo) + k*(sA-eA))
        newB := int(float64(players[1].elo) + k*(sB-eB))
        _, _ = db.ExecContext(ctx, `UPDATE players SET elo=$1 WHERE id=$2`, newA, players[0].id)
        _, _ = db.ExecContext(ctx, `UPDATE players SET elo=$1 WHERE id=$2`, newB, players[1].id)
        _, _ = db.ExecContext(ctx, `UPDATE match_players SET rating_after=CASE WHEN player_id=$1 THEN $2 WHEN player_id=$3 THEN $4 END WHERE match_id=$5`, players[0].id, newA, players[1].id, newB, st.MatchDbID)
    }
    snap, _ := json.Marshal(map[string]interface{}{"board": stringFromBoard(st.Board), "turn": string([]rune{st.Turn}), "winner": string([]rune{st.Winner})})
    if winnerPID != nil {
        _, _ = db.ExecContext(ctx, `UPDATE matches SET winner_player_id=$1, ended_at=now(), state_snapshot=$2 WHERE id=$3`, *winnerPID, string(snap), st.MatchDbID)
        _, _ = db.ExecContext(ctx, `INSERT INTO leaderboard_alltime(player_id, wins, losses, elo) VALUES($1,1,0,(SELECT elo FROM players WHERE id=$1)) ON CONFLICT(player_id) DO UPDATE SET wins=leaderboard_alltime.wins+1, elo=EXCLUDED.elo`, *winnerPID)
        _, _ = db.ExecContext(ctx, `INSERT INTO leaderboard_alltime(player_id, wins, losses, elo) SELECT player_id, 0, 1, p.elo FROM match_players mp JOIN players p ON p.id=mp.player_id WHERE mp.match_id=$1 AND mp.player_id<>$2 ON CONFLICT(player_id) DO UPDATE SET losses=leaderboard_alltime.losses+1, elo=EXCLUDED.elo`, st.MatchDbID, *winnerPID)
    } else {
        _, _ = db.ExecContext(ctx, `UPDATE matches SET ended_at=now(), state_snapshot=$1 WHERE id=$2`, string(snap), st.MatchDbID)
    }
    _, _ = db.ExecContext(ctx, `INSERT INTO leaderboard_daily(player_id, period, wins, losses, elo) SELECT player_id, CURRENT_DATE, 0, 0, p.elo FROM match_players mp JOIN players p ON p.id=mp.player_id WHERE mp.match_id=$1 ON CONFLICT(player_id, period) DO UPDATE SET elo=EXCLUDED.elo`, st.MatchDbID)
    return nil
}
