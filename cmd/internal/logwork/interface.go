package logwork

import (
	"github.com/du0ngtrunghieu/luoi-logwork/pkg/types"
)

type ProjectTracking interface {
	GetTicketToLog() ([]types.Ticket, error)
	GetTicketToEst() ([]types.Ticket, error)
	GetDayToLog() ([]types.LogWorkStatus, error)
	LogWork(ticket []types.Ticket, logworkList []types.LogWorkStatus) error
	FillEstimate(ticket []types.Ticket) error
	AddEstForTicket(tickets []types.Ticket) error
}
