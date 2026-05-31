package client

type Persona struct {
	ID string `json:"id"`
}

type PersonaInsight struct {
	Score float64 `json:"score"`
}

type Query struct {
	Limit int `json:"limit"`
}

type WorkbenchConfig struct {
	Theme string `json:"theme"`
}

type MatchCriteriaRowInDb struct {
	Value string `json:"value"`
}
