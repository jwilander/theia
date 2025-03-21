# JIRA Mana Analysis Tool

This tool analyzes JIRA tickets based on their resolution time and a custom field called "Mana Spent". It helps identify which types of tickets consume the most resources over a specified time period. Epic and Initiative ticket types are excluded from the analysis, as well as tickets resolved as "Won't Do", "Invalid", or "Duplicate". Task and Sub-task issue types are counted as Story tickets in the analysis.

## Mana Spent Values

The tool maps the "Mana Spent" select field values to hours as follows:

| Select Value | Hours Mapped |
|-------------|-------------|
| None (zero time spent) | 0 |
| Small (2 hours or less) | 2 |
| Medium (~half day) | 4 |
| Large (~1 day) | 8 |
| X-Large (~2-3 days) | 20 |
| XX-Large (~1 week) | 40 |

## Prerequisites

- Go 1.16 or higher
- Access to a JIRA Cloud instance
- JIRA API token (can be generated from your Atlassian account settings)

## Installation

```bash
go get github.com/jwilander/theia
```

## Configuration

Set the following environment variables:

```bash
export JIRA_URL="https://your-domain.atlassian.net"
export JIRA_USERNAME="your-email@domain.com"
export JIRA_TOKEN="your-api-token"
```

## Usage

```bash
# For overall analysis
go run main.go -start "2024-01-01" -end "2024-03-21" -project "PROJ"

# For monthly breakdown
go run main.go -start "2024-01-01" -end "2024-03-21" -project "PROJ" -monthly

# For specific teams
go run main.go -start "2024-01-01" -end "2024-03-21" -project "PROJ" -teams "Team A,Team B"
```

### Command Line Arguments

- `-project`: JIRA project key (e.g., "PROJ", "TEAM", etc.)
- `-start`: Start date in YYYY-MM-DD format
- `-end`: End date in YYYY-MM-DD format
- `-monthly`: Optional flag to show month-by-month breakdown
- `-teams`: Optional comma-separated list of team names to filter by (e.g., "Team A,Team B"). Uses the JIRA Team field.

## Output

The tool will output:
1. Analysis period and project details
2. The JQL query used to fetch issues
3. If `-monthly` flag is used:
   - A breakdown table for each month in the date range
   - An overall summary table at the end
4. Each table shows:
   - Issue Type
   - Count of issues
   - Total Mana spent
   - Average Mana per issue type
   - Median Mana per issue type (useful for identifying typical effort without outlier impact)
5. A totals row showing:
   - Total count of all issues
   - Total Mana across all types
   - Overall average Mana per issue
   - Overall median Mana across all issues

Results in each table are sorted by total Mana spent in descending order.
