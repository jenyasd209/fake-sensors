package api

type GroupsResponse struct {
	Groups []string `json:"groups"`
}

type AvgResponse struct {
	Average string `json:"average"`
}

type Species struct {
	Name  string `json:"name"`
	Count string `json:"count"`
}

type SpeciesResponse struct {
	Species []*Species `json:"species"`
}

type ValueResponse struct {
	Value string `json:"value"`
}
