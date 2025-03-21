package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
)

type TicketAnalysis struct {
	IssueType   string
	Count       int
	TotalMana   float64
	AverageMana float64
}

// getManaPoints converts the Mana Spent select value to story points
func getManaPoints(manaValue interface{}) float64 {
	if manaValue == nil {
		return 0
	}

	// The select field value might come as a string or map with "value" key
	var strValue string
	switch v := manaValue.(type) {
	case string:
		strValue = v
	case map[string]interface{}:
		if val, ok := v["value"].(string); ok {
			strValue = val
		}
	default:
		return 0
	}

	// Map the select values to story points
	switch strings.TrimSpace(strValue) {
	case "None (zero time spent)":
		return 0
	case "Small (2 hours or less)":
		return 2
	case "Medium (~half day)":
		return 4
	case "Large (~1 day)":
		return 8
	case "X-Large (~2-3 days)":
		return 20
	case "XX-Large (~1 week)":
		return 40
	default:
		return 0
	}
}

func main() {
	// Command line flags for dates only
	startDate := flag.String("start", "", "Start date (YYYY-MM-DD)")
	endDate := flag.String("end", "", "End date (YYYY-MM-DD)")
	flag.Parse()

	// Validate date flags
	if *startDate == "" || *endDate == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Get JIRA credentials from environment variables
	jiraURL := os.Getenv("JIRA_URL")
	username := os.Getenv("JIRA_USERNAME")
	apiToken := os.Getenv("JIRA_TOKEN")

	// Validate environment variables
	if jiraURL == "" || username == "" || apiToken == "" {
		log.Fatal("Missing required environment variables. Please set JIRA_URL, JIRA_USERNAME, and JIRA_TOKEN")
	}

	// Create JIRA client
	tp := jira.BasicAuthTransport{
		Username: username,
		Password: apiToken,
	}
	client, err := jira.NewClient(tp.Client(), jiraURL)
	if err != nil {
		log.Fatalf("Error creating JIRA client: %v", err)
	}

	// Parse dates
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Invalid start date format: %v", err)
	}
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("Invalid end date format: %v", err)
	}

	// Create JQL query
	jql := fmt.Sprintf(`status in (Resolved, Closed) AND
		resolutiondate >= "%s" AND
		resolutiondate <= "%s" AND
		"Mana Spent" is not EMPTY AND
		issuetype not in (Epic, Initiative)
		ORDER BY created DESC`,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"))

	// Initialize analysis map
	analysis := make(map[string]*TicketAnalysis)

	// Search issues with pagination
	var startAt int
	for {
		searchOpts := &jira.SearchOptions{
			StartAt:    startAt,
			MaxResults: 50,
			Fields:     []string{"issuetype", "customfield_11267"}, // Mana Spent field ID
		}

		issues, resp, err := client.Issue.Search(jql, searchOpts)
		if err != nil {
			log.Fatalf("Error searching issues: %v", err)
		}

		if len(issues) == 0 {
			break
		}

		// Process issues
		for _, issue := range issues {
			issueType := issue.Fields.Type.Name
			manaField := issue.Fields.Unknowns["customfield_10000"]
			manaSpent := getManaPoints(manaField)

			if _, exists := analysis[issueType]; !exists {
				analysis[issueType] = &TicketAnalysis{
					IssueType: issueType,
				}
			}

			analysis[issueType].Count++
			analysis[issueType].TotalMana += manaSpent
		}

		startAt += len(issues)
		if startAt >= resp.Total {
			break
		}
	}

	// Calculate averages and prepare for sorting
	var results []TicketAnalysis
	for _, a := range analysis {
		if a.Count > 0 {
			a.AverageMana = a.TotalMana / float64(a.Count)
		}
		results = append(results, *a)
	}

	// Sort by total mana spent
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalMana > results[j].TotalMana
	})

	// Print results
	fmt.Printf("\nAnalysis Period: %s to %s\n\n", *startDate, *endDate)
	fmt.Printf("%-20s %-10s %-15s %-15s\n", "Issue Type", "Count", "Total Mana", "Avg Mana")
	fmt.Println(strings.Repeat("-", 65))

	for _, r := range results {
		fmt.Printf("%-20s %-10d %-15.2f %-15.2f\n",
			r.IssueType,
			r.Count,
			r.TotalMana,
			r.AverageMana)
	}
}
