package dto

type DeckInfo struct {
	ID    uint32 `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Due   int    `json:"due"`
}

type DecksView struct {
	Active uint32     `json:"active"`
	Decks  []DeckInfo `json:"decks"`
}

type ExportReport struct {
	Count int    `json:"count"`
	Path  string `json:"path"`
}
