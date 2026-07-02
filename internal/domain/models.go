package domain

type URL struct {
	ID    int64  `json:"id"`
	URL   string `json:"url"`
	Alias string `json:"alias"`
}
