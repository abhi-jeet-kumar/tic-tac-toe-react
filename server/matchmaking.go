package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/heroiclabs/nakama-common/runtime"
)

// registerMatchmaker wires Nakama's matchmaker matched callback to create an authoritative match.
func registerMatchmaker(initializer runtime.Initializer) error {
	return initializer.RegisterMatchmakerMatched(onMatched)
}

func onMatched(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, matches []runtime.MatchmakerResult) (string, error) {
	mode := "casual"
	if len(matches) > 0 {
		if v, ok := matches[0].StringProperties["mode"]; ok && v != "" {
			mode = v
		}
	}
	params := map[string]interface{}{ "mode": mode }
	matchID, err := nk.MatchCreate(ctx, "tictactoe", params)
	if err != nil {
		return "", err
	}
	// Optionally return metadata
	_, _ = json.Marshal(map[string]string{"mode": mode})
	return matchID, nil
}

