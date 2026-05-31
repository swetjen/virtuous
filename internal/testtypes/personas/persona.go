package personas

type Persona struct {
	ID string `json:"id"`
}

type PersonaInsight struct {
	Reason string `json:"reason"`
}

type MatchCriteriaRowInDb struct {
	Criteria string `json:"criteria"`
}

type Query struct {
	Search string `json:"search"`
}
