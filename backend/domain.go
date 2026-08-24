package main

type Sample struct {
	ID       string `json:"id"`
	Material string `json:"material"`
	Batch    string `json:"batch"`
	Status   string `json:"status"`
}

var sampleTransitions = map[string]map[string]bool{
	"received":  {"in_review": true, "rejected": true},
	"in_review": {"released": true, "rejected": true},
	"released":  {},
	"rejected":  {},
}
