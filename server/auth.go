package main

import (
    "context"
    "database/sql"
    "encoding/json"

    "github.com/heroiclabs/nakama-common/runtime"
)

type DeviceAuthRequest struct {
	DeviceID string `json:"device_id"`
	Nickname string `json:"nickname"`
}

type DeviceAuthResponse struct {
	UserID   string `json:"user_id"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

func registerAuthRPC(initializer runtime.Initializer) error {
	return initializer.RegisterRpc("auth_device", rpcAuthDevice)
}

func rpcAuthDevice(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
	var req DeviceAuthRequest
	if err := json.Unmarshal([]byte(payload), &req); err != nil {
		return "", runtime.NewError("invalid payload", 3)
	}
	if req.DeviceID == "" {
		return "", runtime.NewError("missing device_id", 3)
	}

    userID, err := nk.AuthenticateDevice(ctx, req.DeviceID, req.Nickname, true)
	if err != nil {
		logger.Error("device auth failed: %v", err)
		return "", runtime.NewError("auth failed", 13)
	}

	session, err := nk.AuthenticateTokenGenerate(ctx, userID, nil, 60*60*24*7, nil)
	if err != nil {
		logger.Error("token gen failed: %v", err)
		return "", runtime.NewError("token failed", 13)
	}

    user, err := nk.UsersGetId(ctx, []string{userID})
	if err != nil || len(user) == 0 {
		return "", runtime.NewError("user fetch failed", 13)
	}

    // Upsert to players table on login
    _, _ = db.ExecContext(ctx, `INSERT INTO players(nakama_user_id, device_id, nickname)
        VALUES($1,$2,$3)
        ON CONFLICT (device_id) DO UPDATE SET nakama_user_id=EXCLUDED.nakama_user_id, nickname=COALESCE(EXCLUDED.nickname, players.nickname)`, userID, req.DeviceID, req.Nickname)

    resp := DeviceAuthResponse{UserID: userID, Token: session.Token(), Username: user[0].Username}
	out, _ := json.Marshal(resp)
	return string(out), nil
}
