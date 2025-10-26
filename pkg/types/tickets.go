package types

type Ticket struct {
	ID              string
	Summary         string
	Est             int64
	EstimatedLogged int64
	Status          string
}
