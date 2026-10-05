package main

type Budget struct {
	Category string
	Limit    float64
	Period   string
}

var budgets = map[string]Budget{
	"Food": Budget{
		Category: "Food",
		Limit:    1000.0,
		Period:   "2026-10",
	},
	"Transport": Budget{
		Category: "Transport",
		Limit:    500.0,
		Period:   "2026-10",
	},
}
