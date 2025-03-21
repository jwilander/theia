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
	MedianMana  float64
	ManaValues  []float64 // Store individual mana values for median calculation
}

type MonthlyAnalysis struct {
	Month    time.Time
	Analysis map[string]*TicketAnalysis
}

type TeamAnalysis struct {
	Team     string
	Analysis map[string]*TicketAnalysis
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

// normalizeIssueType converts Task and Sub-task types to Story
func normalizeIssueType(issueType string) string {
	switch issueType {
	case "Task", "Sub-task":
		return "Story"
	default:
		return issueType
	}
}

// calculateMedian returns the median value from a slice of float64
func calculateMedian(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	// If odd number of values
	if len(sorted)%2 == 1 {
		return sorted[len(sorted)/2]
	}

	// If even number of values
	mid := len(sorted) / 2
	return (sorted[mid-1] + sorted[mid]) / 2
}

// printAnalysisTable prints the analysis results in a formatted table
func printAnalysisTable(results []TicketAnalysis, period string) {
	// Calculate totals
	var totalCount int
	var totalMana float64
	var allManaValues []float64
	for _, r := range results {
		totalCount += r.Count
		totalMana += r.TotalMana
		allManaValues = append(allManaValues, r.ManaValues...)
	}
	overallAvgMana := 0.0
	if totalCount > 0 {
		overallAvgMana = totalMana / float64(totalCount)
	}
	overallMedianMana := calculateMedian(allManaValues)

	// Print header
	if period != "" {
		fmt.Printf("\n%s\n", period)
	}
	fmt.Printf("%-20s %-10s %-15s %-15s %-15s\n",
		"Issue Type",
		"Count",
		"Total Mana",
		"Avg Mana",
		"Median Mana")
	fmt.Println(strings.Repeat("-", 80))

	// Print results
	for _, r := range results {
		fmt.Printf("%-20s %-10d %-15.2f %-15.2f %-15.2f\n",
			r.IssueType,
			r.Count,
			r.TotalMana,
			r.AverageMana,
			r.MedianMana)
	}

	// Print totals
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-20s %-10d %-15.2f %-15.2f %-15.2f\n",
		"TOTAL",
		totalCount,
		totalMana,
		overallAvgMana,
		overallMedianMana)
}

func main() {
	// Command line flags
	startDate := flag.String("start", "", "Start date (YYYY-MM-DD)")
	endDate := flag.String("end", "", "End date (YYYY-MM-DD)")
	projectKey := flag.String("project", "", "JIRA project key (e.g., PROJ)")
	monthly := flag.Bool("monthly", false, "Show monthly breakdown")
	teams := flag.Bool("teams", false, "Group results by team")
	flag.Parse()

	// Validate flags
	if *startDate == "" || *endDate == "" || *projectKey == "" {
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

	// Create base JQL query
	jql := fmt.Sprintf(`project = "%s" AND
		status in (Resolved, Closed) AND
		resolution not in ("Won't Do", "Invalid", "Duplicate") AND
		resolutiondate >= "%s" AND
		resolutiondate <= "%s" AND
		"Mana Spent" is not EMPTY AND
		issuetype not in (Epic, Initiative)
		ORDER BY created DESC`,
		*projectKey,
		start.Format("2006-01-02"),
		end.Format("2006-01-02"))

	// Initialize analysis maps
	analysis := make(map[string]*TicketAnalysis)
	var monthlyAnalyses []MonthlyAnalysis
	var teamAnalyses []TeamAnalysis

	if *teams {
		// We'll populate the teams as we find them
		teamAnalyses = make([]TeamAnalysis, 0)
	}

	if *monthly {
		// Create a map for each month in the date range
		current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
		endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, end.Location())

		for !current.After(endMonth) {
			monthlyAnalyses = append(monthlyAnalyses, MonthlyAnalysis{
				Month:    current,
				Analysis: make(map[string]*TicketAnalysis),
			})
			current = current.AddDate(0, 1, 0)
		}
	}

	// Print debug info about the search
	fmt.Printf("\nDebug Info:\n")
	fmt.Printf("JQL Query:\n%s\n\n", jql)

	// Search issues with pagination
	var startAt int
	for {
		searchOpts := &jira.SearchOptions{
			StartAt:    startAt,
			MaxResults: 50,
			Fields:     []string{"issuetype", "customfield_11267", "resolutiondate", "team"},
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
			issueType := normalizeIssueType(issue.Fields.Type.Name)
			manaField := issue.Fields.Unknowns["customfield_11267"]
			manaSpent := getManaPoints(manaField)

			// Update overall analysis
			if _, exists := analysis[issueType]; !exists {
				analysis[issueType] = &TicketAnalysis{
					IssueType:  issueType,
					ManaValues: make([]float64, 0),
				}
			}
			analysis[issueType].Count++
			analysis[issueType].TotalMana += manaSpent
			analysis[issueType].ManaValues = append(analysis[issueType].ManaValues, manaSpent)

			// Update team analysis if enabled
			if *teams {
				team := "No Team"
				if teamField := issue.Fields.Unknowns["team"]; teamField != nil {
					if teamName, ok := teamField.(string); ok && teamName != "" {
						team = teamName
					}
				}

				// Find or create team analysis
				var teamAnalysis *TeamAnalysis
				for i := range teamAnalyses {
					if teamAnalyses[i].Team == team {
						teamAnalysis = &teamAnalyses[i]
						break
					}
				}
				if teamAnalysis == nil {
					teamAnalyses = append(teamAnalyses, TeamAnalysis{
						Team:     team,
						Analysis: make(map[string]*TicketAnalysis),
					})
					teamAnalysis = &teamAnalyses[len(teamAnalyses)-1]
				}

				// Update team's issue type analysis
				if _, exists := teamAnalysis.Analysis[issueType]; !exists {
					teamAnalysis.Analysis[issueType] = &TicketAnalysis{
						IssueType:  issueType,
						ManaValues: make([]float64, 0),
					}
				}
				teamAnalysis.Analysis[issueType].Count++
				teamAnalysis.Analysis[issueType].TotalMana += manaSpent
				teamAnalysis.Analysis[issueType].ManaValues = append(teamAnalysis.Analysis[issueType].ManaValues, manaSpent)
			}

			// Update monthly analysis if enabled
			if *monthly {
				resolutionDate := time.Time(issue.Fields.Resolutiondate)

				for i := range monthlyAnalyses {
					maStart := monthlyAnalyses[i].Month
					maEnd := maStart.AddDate(0, 1, 0).Add(-time.Second)

					if (resolutionDate.After(maStart) || resolutionDate.Equal(maStart)) &&
						(resolutionDate.Before(maEnd) || resolutionDate.Equal(maEnd)) {
						if _, exists := monthlyAnalyses[i].Analysis[issueType]; !exists {
							monthlyAnalyses[i].Analysis[issueType] = &TicketAnalysis{
								IssueType:  issueType,
								ManaValues: make([]float64, 0),
							}
						}
						monthlyAnalyses[i].Analysis[issueType].Count++
						monthlyAnalyses[i].Analysis[issueType].TotalMana += manaSpent
						monthlyAnalyses[i].Analysis[issueType].ManaValues = append(monthlyAnalyses[i].Analysis[issueType].ManaValues, manaSpent)
						break
					}
				}
			}
		}

		startAt += len(issues)
		if startAt >= resp.Total {
			break
		}
	}

	// Calculate averages and medians for overall analysis
	var results []TicketAnalysis
	for _, a := range analysis {
		if a.Count > 0 {
			a.AverageMana = a.TotalMana / float64(a.Count)
			a.MedianMana = calculateMedian(a.ManaValues)
		}
		results = append(results, *a)
	}

	// Sort by total mana spent
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalMana > results[j].TotalMana
	})

	// Print header information
	fmt.Printf("\nAnalysis Period: %s to %s\n", *startDate, *endDate)
	fmt.Printf("Project: %s\n", *projectKey)
	fmt.Printf("\nJQL Query:\n%s\n", jql)

	if *teams {
		// Sort teams alphabetically
		sort.Slice(teamAnalyses, func(i, j int) bool {
			return teamAnalyses[i].Team < teamAnalyses[j].Team
		})

		// Print team breakdowns
		for _, ta := range teamAnalyses {
			var teamResults []TicketAnalysis
			for _, a := range ta.Analysis {
				if a.Count > 0 {
					a.AverageMana = a.TotalMana / float64(a.Count)
					a.MedianMana = calculateMedian(a.ManaValues)
				}
				teamResults = append(teamResults, *a)
			}
			sort.Slice(teamResults, func(i, j int) bool {
				return teamResults[i].TotalMana > teamResults[j].TotalMana
			})
			printAnalysisTable(teamResults, fmt.Sprintf("Team: %s", ta.Team))
		}

		// Print overall summary
		fmt.Printf("\nOVERALL SUMMARY:\n")
	} else if *monthly {
		// Print monthly breakdowns
		for _, ma := range monthlyAnalyses {
			var monthResults []TicketAnalysis
			for _, a := range ma.Analysis {
				if a.Count > 0 {
					a.AverageMana = a.TotalMana / float64(a.Count)
					a.MedianMana = calculateMedian(a.ManaValues)
				}
				monthResults = append(monthResults, *a)
			}
			sort.Slice(monthResults, func(i, j int) bool {
				return monthResults[i].TotalMana > monthResults[j].TotalMana
			})
			printAnalysisTable(monthResults, fmt.Sprintf("Month: %s", ma.Month.Format("January 2006")))
		}

		// Print overall summary
		fmt.Printf("\nOVERALL SUMMARY:\n")
	}

	printAnalysisTable(results, "")
}
