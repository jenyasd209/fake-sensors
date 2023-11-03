package response

type Groups struct {
	Groups []string `json:"groups"`
}

type Average struct {
	Average string `json:"average"`
}

type Species struct {
	Name  string `json:"name"`
	Count string `json:"count"`
}

type SpeciesList struct {
	Species []*Species `json:"species"`
}

type Value struct {
	Value string `json:"value"`
}
