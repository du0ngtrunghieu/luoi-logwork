package types

import "github.com/andygrunwald/go-jira"

type Ticket struct {
	ID              string
	Summary         string
	Est             int64
	EstimatedLogged int64
	Status          string
	Type            string
	Project         string
	Labels          []string
	Parent          string
	Created         jira.Time
}
