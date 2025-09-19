package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "github.com/heroiclabs/nakama-common/runtime"
    "time"
)

const (
    OpMove = 1
)

type MatchState struct {
    Board      [9]rune  // X, O, or 0
    Turn       rune     // 'X' or 'O'
    Presences  map[string]runtime.Presence
    StartedAt  time.Time
    Winner     rune     // 'X', 'O', or 0
}

type MovePayload struct { Index int `json:"index"` }

func registerMatch(initializer runtime.Initializer) error {
    return initializer.RegisterMatch("tictactoe", NewMatch)
}

func NewMatch(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, params map[string]interface{}) (runtime.Match, error) {
    state := &MatchState{Turn: 'X', Presences: map[string]runtime.Presence{}, StartedAt: time.Now()}
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
    if len(st.Presences) >= 2 { return st, false, "match full" }
    return st, true, ""
}

func (m *TicTacToeMatch) MatchJoin(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
    st := state.(*MatchState)
    for _, p := range presences { st.Presences[p.GetUserId()] = p }
    return st
}

func (m *TicTacToeMatch) MatchLeave(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, presences []runtime.Presence) interface{} {
    st := state.(*MatchState)
    for _, p := range presences { delete(st.Presences, p.GetUserId()) }
    return st
}

func (m *TicTacToeMatch) MatchLoop(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, messages []runtime.MatchData) interface{} {
    st := state.(*MatchState)
    for _, msg := range messages {
        if msg.GetOpCode() == OpMove && st.Winner == 0 {
            var mv MovePayload
            if err := json.Unmarshal(msg.GetData(), &mv); err != nil { continue }
            if mv.Index < 0 || mv.Index >= 9 { continue }
            mark := st.Turn
            if st.Board[mv.Index] != 0 { continue }
            st.Board[mv.Index] = mark
            if checkWin(st.Board, mark) { st.Winner = mark }
            if st.Winner == 0 { st.Turn = other(mark) }
            payload, _ := json.Marshal(map[string]interface{}{
                "board": stringFromBoard(st.Board),
                "turn": string([]rune{st.Turn}),
                "winner": string([]rune{st.Winner}),
            })
            dispatcher.BroadcastMessage(OpMove, payload, nil, nil, true)
        }
    }
    return st
}

func (m *TicTacToeMatch) MatchTerminate(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, dispatcher runtime.MatchDispatcher, tick int64, state interface{}, graceSeconds int) interface{} {
    return state
}

func other(r rune) rune { if r=='X' { return 'O' }; return 'X' }

func checkWin(b [9]rune, r rune) bool {
    lines := [8][3]int{{0,1,2},{3,4,5},{6,7,8},{0,3,6},{1,4,7},{2,5,8},{0,4,8},{2,4,6}}
    for _, ln := range lines {
        if b[ln[0]]==r && b[ln[1]]==r && b[ln[2]]==r { return true }
    }
    return false
}

func stringFromBoard(b [9]rune) string {
    bytes := make([]rune, 9)
    for i,v := range b { if v==0 { bytes[i]='.' } else { bytes[i]=v } }
    return string(bytes)
}

// RPC to create a new match and return match ID for testing
func registerCreateMatchRPC(initializer runtime.Initializer) error {
    return initializer.RegisterRpc("create_match", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
        matchID, err := nk.MatchCreate(ctx, "tictactoe", map[string]interface{}{})
        if err != nil { return "", runtime.NewError(fmt.Sprintf("create failed: %v", err), 13) }
        out, _ := json.Marshal(map[string]string{"match_id": matchID})
        return string(out), nil
    })
}

