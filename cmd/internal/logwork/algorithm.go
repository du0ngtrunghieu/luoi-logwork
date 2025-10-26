package logwork

import (
	"slices"
	"time"

	"github.com/du0ngtrunghieu/luoi-logwork/pkg/types"
)

func defaultLogWorkAlgorithm(ticket []types.Ticket, logworkList []types.LogWorkStatus) ([]types.LogAction, error) {
	const defaultShiftTime = 7.5 // giờ
	const shiftSeconds = int64(defaultShiftTime * 3600)
	workingDay := []int{1, 2, 3, 4, 5}
	startShiftHour := 7*time.Hour + 30*time.Minute // 7h30 sáng

	logActionList := []types.LogAction{}

	for i := range logworkList {
		day := logworkList[i]
		if !slices.Contains(workingDay, int(day.Date.Weekday())) {
			continue
		}

		// còn lại trong ca hôm đó
		remainingShift := shiftSeconds - day.TimeSpent
		if remainingShift <= 0 {
			continue
		}

		// log đến khi hết ca hoặc hết ticket khả thi
		for remainingShift > 0 && len(ticket) > 0 {
			assigned := false
			for tIdx := range ticket {
				t := &ticket[tIdx]
				remainingEst := t.Est - t.EstimatedLogged
				if remainingEst <= 0 {
					continue
				}

				// thời gian có thể log vào ticket này
				timeToLog := remainingEst
				if timeToLog > remainingShift {
					timeToLog = remainingShift
				}

				if timeToLog <= 0 {
					continue
				}

				// thêm log action
				logActionList = append(logActionList, types.LogAction{
					TimeToLog:   timeToLog,
					TicketToLog: *t,
					DateToLog:   day.Date.Add(startShiftHour),
				})

				// cập nhật lại estimate và shift còn lại
				t.EstimatedLogged += timeToLog
				remainingShift -= timeToLog
				assigned = true

				// nếu hết giờ trong ca -> dừng
				if remainingShift <= 0 {
					break
				}
			}

			// nếu không còn ticket khả thi thì dừng
			if !assigned {
				break
			}
		}
	}

	return logActionList, nil
}
