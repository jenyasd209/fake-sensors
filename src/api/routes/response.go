package routes

// swagger:model
type Groups struct {
	Groups []string `json:"groups"`
}

// swagger:model
type Average struct {
	Average string `json:"average"`
}

// swagger:model
type Species struct {
	Name  string `json:"name"`
	Count string `json:"count"`
}

// swagger:model
type SpeciesList struct {
	Species []*Species `json:"species"`
}

// swagger:model
type Value struct {
	Value string `json:"value"`
}

// swagger:model
type ErrorResponse struct {
	Error string `json:"error"`
}
