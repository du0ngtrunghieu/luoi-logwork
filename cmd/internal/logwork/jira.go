package logwork

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/du0ngtrunghieu/luoi-logwork/pkg/helper"
	"github.com/du0ngtrunghieu/luoi-logwork/pkg/types"
)

type Jira struct {
	endpoint string
	userName string
	apiToken string
	client   *jira.Client
}

func NewJira(endpoint string, userName string, apiToken string) *Jira {
	tp := jira.BasicAuthTransport{
		Username: userName,
		Password: apiToken,
	}

	client, err := jira.NewClient(tp.Client(), endpoint)

	if err != nil {
		log.Fatalf("Error creating JIRA client: %v", err)
	}

	return &Jira{
		endpoint: endpoint,
		userName: userName,
		apiToken: apiToken,
		client:   client,
	}
}

func (j *Jira) GetTicketToLog() ([]types.Ticket, error) {
	// JQL query to fetch your tickets. Customize this query as needed.
	fmt.Println("----------------Ticket able to log-------------------")
	jql := fmt.Sprintf(`assignee = "%s" AND status IN (Open, "In Progress", "PAUSED") AND type != Epic AND type != Bug ORDER BY created DESC`, j.userName)

	ticketList := []types.Ticket{}

	issues, _, err := j.client.Issue.SearchV2JQL(jql, &jira.SearchOptionsV2{
		MaxResults: 1000, // Adjust the number of results as needed
		Fields:     []string{"summary", "description", "issuetype", "status", "priority", "project", "timeoriginalestimate", "timespent"},
	})
	if err != nil {
		log.Fatalf("Error fetching JIRA issues: %v", err)
	}

	// Print the fetched issues
	for _, issue := range issues {
		fmt.Printf("Issue: %s, Summary %s, Est: %s, Status: %s\n", issue.Key, issue.Fields.Summary, helper.FormatEstimate(int64(issue.Fields.TimeOriginalEstimate)), issue.Fields.Status.Name)
		ticketList = append(ticketList, types.Ticket{
			ID:              issue.Key,
			Summary:         issue.Fields.Summary,
			Est:             int64(issue.Fields.TimeOriginalEstimate),
			EstimatedLogged: int64(issue.Fields.TimeSpent),
		})
	}
	return ticketList, nil
}

func (j *Jira) GetDayToLog() ([]types.LogWorkStatus, error) {
	// Calculate the start of the current week (Monday)
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	var startOfWeek time.Time

	// Sunday is 0 -> we need to handle this
	if now.Weekday().String() == "Sunday" {
		startOfWeek = start.AddDate(0, 0, -6) // Adjust according to your week's start day
	} else {
		startOfWeek = start.AddDate(0, 0, -int(now.Weekday())+1) // Adjust according to your week's start day
	}

	fmt.Println("----------------Your week worklog status-------------------")
	fmt.Println("Start of week: ", startOfWeek)

	logworkList := make([]types.LogWorkStatus, 7)

	// Create the correct date
	for i := range logworkList {
		if i == 0 {
			logworkList[i].Date = startOfWeek.AddDate(0, 0, 6)
		} else {
			logworkList[i].Date = startOfWeek.AddDate(0, 0, i-1)
		}
	}

	// JQL query to fetch issues assigned to you
	jql := fmt.Sprintf(`assignee = "%s" ORDER BY created DESC`, j.userName)

	issues, _, err := j.client.Issue.SearchV2JQL(jql, &jira.SearchOptionsV2{
		MaxResults: 1000, // Adjust the number of results as needed
		Fields:     []string{"summary", "description", "issuetype", "status", "priority", "project"},
	})
	if err != nil {
		log.Fatalf("Error fetching JIRA issues: %v", err)
	}

	fmt.Println("Work logs for the current week:")

	for _, issue := range issues {
		worklogs, _, err := j.client.Issue.GetWorklogs(issue.Key)
		if err != nil {
			log.Printf("Error fetching worklogs for issue %s: %v", issue.Key, err)
			continue
		}

		for _, worklog := range worklogs.Worklogs {
			worklogTimeStarted, _ := worklog.Started.MarshalJSON()
			worklogTime, err := time.Parse("\"2006-01-02T15:04:05.999-0700\"", string(worklogTimeStarted))
			if err != nil {
				log.Printf("Error parsing worklog time for issue %s: %v", issue.Key, err)
				continue
			}

			if worklogTime.After(startOfWeek) {
				logworkList[worklogTime.Weekday()].Add(int64(worklog.TimeSpentSeconds))
			}
		}
	}

	for i := range logworkList {
		fmt.Printf("%s: Time Spent: %d Hours\n", time.Weekday(i), logworkList[i].TimeSpent/3600)
	}

	return logworkList, nil
}

func (j *Jira) FillEstimate(ticket []types.Ticket) error {
	return nil
}

func (j *Jira) LogWork(ticket []types.Ticket, logworkList []types.LogWorkStatus) error {
	logActionList, _ := defaultLogWorkAlgorithm(ticket, logworkList)

	fmt.Println("----------------Ticket to log-------------------")
	for i := range logActionList {
		fmt.Printf("Ticket ID: %s\tTiket Summary: %s\t\tTime to log: %dh\tDate to log: %s\n", logActionList[i].TicketToLog.ID, logActionList[i].TicketToLog.Summary, logActionList[i].TimeToLog/3600, logActionList[i].DateToLog)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("You sure to start logging work? [y/n]: ")
	status, _ := reader.ReadString('\n')
	status = status[:len(status)-1]

	status = strings.ReplaceAll(status, "\r", "")

	if status == "n" {
		return nil
	} else if status != "y" {
		log.Println("Invalid input")
		return errors.New("Invalid input, valid input are y/n")
	}

	for i := range logActionList {
		worklog := &jira.WorklogRecord{
			Started:          (*jira.Time)(&logActionList[i].DateToLog),
			TimeSpentSeconds: int(logActionList[i].TimeToLog),
		}

		// Log work to the Jira issue
		_, response, err := j.client.Issue.AddWorklogRecord(logActionList[i].TicketToLog.ID, worklog)
		if err != nil {
			log.Fatalf("Failed to log work: %v", err)
		}
		defer response.Body.Close()

		fmt.Printf("Work logged to issue %s: %s successfully.\n", logActionList[i].TicketToLog.ID, logActionList[i].TicketToLog.Summary)

		action := logActionList[i]
		// -----------------------------
		// Check status and transition
		// -----------------------------
		issue, _, err := j.client.Issue.Get(action.TicketToLog.ID, nil)
		if err != nil {
			log.Printf("⚠️  Cannot fetch issue %s details: %v\n", action.TicketToLog.ID, err)
			continue
		}

		if strings.EqualFold(issue.Fields.Status.Name, "Open") {
			transitions, _, err := j.client.Issue.GetTransitions(action.TicketToLog.ID)
			if err != nil {
				log.Printf("⚠️  Cannot get transitions for %s: %v\n", action.TicketToLog.ID, err)
				continue
			}

			var pauseTransitionID string
			for _, t := range transitions {
				if strings.EqualFold(t.Name, "PAUSE") {
					pauseTransitionID = t.ID
					break
				}
			}

			if pauseTransitionID == "" {
				log.Printf("⚠️  No 'Pause' transition found for issue %s\n", action.TicketToLog.ID)
				continue
			}

			_, err = j.client.Issue.DoTransition(action.TicketToLog.ID, pauseTransitionID)
			if err != nil {
				log.Printf("❌ Failed to move issue %s to Pause: %v\n", action.TicketToLog.ID, err)
			} else {
				fmt.Printf("🟡 Issue %s transitioned to 'Pause'\n", action.TicketToLog.ID)
			}
		}
	}

	return nil
}

// GetTicketToEst fetches tickets assigned to the current user (Open / In Progress / PAUSED),
// then for any Open ticket with Est == 0 it searches the whole JIRA for similar summaries
// that have timeoriginalestimate > 0 and uses the best match (score >= 0.8) to fill Est.
func (j *Jira) GetTicketToEst() ([]types.Ticket, error) {
	fmt.Println("----------------Ticket need to estimate (searching whole Jira)-------------------")

	// 1) Lấy các ticket của user để xử lý (các ticket bạn muốn fill)
	jqlForUser := fmt.Sprintf(`assignee = "%s" AND status IN (Open, "In Progress", "PAUSED", "DONE") AND type != Epic AND type != Bug ORDER BY created DESC`, j.userName)

	issues, _, err := j.client.Issue.SearchV2JQL(jqlForUser, &jira.SearchOptionsV2{
		MaxResults: 1000,
		Fields:     []string{"summary", "status", "timeoriginalestimate", "timespent"},
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching user issues: %v", err)
	}

	ticketList := []types.Ticket{}
	for _, issue := range issues {
		ticketList = append(ticketList, types.Ticket{
			ID:              issue.Key,
			Summary:         issue.Fields.Summary,
			Status:          issue.Fields.Status.Name,
			Est:             int64(issue.Fields.TimeOriginalEstimate),
			EstimatedLogged: int64(issue.Fields.TimeSpent),
		})
	}

	fmt.Printf("Fetched %d tickets assigned to %s\n", len(ticketList), j.userName)

	// 2) Với mỗi ticket cần fill (Open và Est == 0) -> search toàn Jira để tìm candidate có Est > 0
	fmt.Println("\n----------------Auto-fill estimate by searching-------------------")

	for idx := range ticketList {
		t := &ticketList[idx]
		// chỉ quan tâm Open + chưa có estimate
		if !strings.EqualFold(t.Status, "Open") || t.Est > 0 {
			continue
		}

		fmt.Printf("Searching matches for: %s (%s)\n", t.ID, t.Summary)

		// tạo keywords từ summary (loại các từ ngắn <= 2 ký tự)
		keywords := helper.ExtractKeywords(t.Summary, 3)
		if len(keywords) == 0 {
			fmt.Printf(" ⚠️  No useful keywords found for %s, skipping\n", t.ID)
			continue
		}

		// build JQL: tìm các issue trong toàn Jira có summary chứa 1 trong các keyword và có estimate
		// (giới hạn kết quả để tránh quá nhiều fetch)
		jqlSearch := helper.BuildJQLForKeywords(keywords)
		// thêm điều kiện có timeoriginalestimate > 0
		jqlSearch = fmt.Sprintf("(%s) AND timeoriginalestimate IS NOT EMPTY ORDER BY created DESC", jqlSearch)

		candidates, _, err := j.client.Issue.SearchV2JQL(jqlSearch, &jira.SearchOptionsV2{
			MaxResults: 500,
			Fields:     []string{"summary", "timeoriginalestimate", "status"},
		})
		if err != nil {
			log.Printf(" ⚠️  Error searching Jira for %s: %v\n", t.ID, err)
			continue
		}

		if len(candidates) == 0 {
			fmt.Printf(" ❌  No candidates found in Jira for %s\n", t.ID)
			continue
		}

		// evaluate similarity over candidates and pick best
		bestScore := 0.0
		bestEst := int64(0)
		bestSummary := ""
		for _, c := range candidates {
			cSummary := c.Fields.Summary
			cEst := int64(c.Fields.TimeOriginalEstimate)
			if cEst <= 0 {
				continue
			}
			score := helper.StringSimilarity(t.Summary, cSummary)
			if score > bestScore {
				bestScore = score
				bestEst = cEst
				bestSummary = cSummary
			}
		}

		if bestScore >= 0.8 && bestEst > 0 {
			t.Est = bestEst
			fmt.Printf(" ✅ Auto-filled %s => %s (matched with \"%s\", score=%.2f)\n", t.ID, helper.FormatEstimate(t.Est), bestSummary, bestScore)
		} else {
			fmt.Printf(" ❌  No sufficiently similar candidate for %s (best score %.2f)\n", t.ID, bestScore)
		}
	}

	return ticketList, nil
}

func (j *Jira) AddEstForTicket(ticketList []types.Ticket) error {
	fmt.Println("\n----------------Updating estimate to Jira-------------------")

	for _, t := range ticketList {
		// chỉ update cho task open và có estimate hợp lệ
		if !strings.EqualFold(t.Status, "Open") || t.Est <= 0 {
			continue
		}

		// lấy thông tin issue hiện tại để kiểm tra có Est chưa
		issue, _, err := j.client.Issue.Get(t.ID, nil)
		if err != nil {
			fmt.Printf(" ⚠️  Cannot fetch issue %s: %v\n", t.ID, err)
			continue
		}

		if issue.Fields.TimeOriginalEstimate > 0 {
			fmt.Printf("⏭️ %s đã có estimate (%s), bỏ qua\n", t.ID, helper.FormatEstimate(int64(issue.Fields.TimeOriginalEstimate)))
			continue
		}

		// cập nhật estimate lên Jira
		update := map[string]interface{}{
			"fields": map[string]interface{}{
				"timetracking": map[string]interface{}{
					"originalEstimate": helper.SecondsToJiraString(t.Est),
				},
			},
		}
		_, err = j.client.Issue.UpdateIssue(t.ID, update)
		if err != nil {
			fmt.Printf("❌Update fail %s (%s): %v\n", t.ID, t.Summary, err)
			continue
		}

		fmt.Printf("✅ Updated estimate %s -> %s\n", t.ID, helper.FormatEstimate(t.Est))
	}

	return nil
}
