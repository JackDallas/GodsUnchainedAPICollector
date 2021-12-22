package guapi

type Order string

const (
	OrderAsc  Order = "asc"
	OrderDesc Order = "desc"
)

//API Responses
type PropertiesResponse struct {
	Total   int          `json:"total"`
	Page    int          `json:"page"`
	PerPage int          `json:"perPage"`
	Records []UserRecord `json:"records"`
}
type MatchResponse struct {
	Total   int           `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"perPage"`
	Records []MatchRecord `json:"records"`
}

//
type UserRecord struct {
	UserID      int64  `json:"user_id"`
	XpLevel     int    `json:"xp_level"`
	TotalXp     int    `json:"total_xp"`
	XpToNext    int    `json:"xp_to_next"`
	WonMatches  int    `json:"won_matches"`
	LostMatches int    `json:"lost_matches"`
	Username    string `json:"username"`
}
type PlayerInfo struct {
	God    string `json:"god"`
	Cards  []int  `json:"cards"`
	Global bool   `json:"global"`
	Health int    `json:"health"`
	Status string `json:"status"`
	UserID int    `json:"user_id"`
}
type MatchRecord struct {
	PlayerWon   int          `json:"player_won"`
	PlayerLost  int          `json:"player_lost"`
	GameMode    int          `json:"game_mode"`
	GameID      string       `json:"game_id"`
	StartTime   int          `json:"start_time"`
	EndTime     int          `json:"end_time"`
	PlayerInfo  []PlayerInfo `json:"player_info"`
	TotalTurns  int          `json:"total_turns"`
	TotalRounds int          `json:"total_rounds"`
}
