package main

import (
	"context"
	"database/sql"
	"github.com/heroiclabs/nakama-common/runtime"
)

// InitModule is called by Nakama on startup to initialize runtime handlers.
func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	logger.Info("tictactoe module loaded")

	// Register RPCs and handlers
	if err := registerAuthRPC(initializer); err != nil {
		return err
	}

	return nil
}

