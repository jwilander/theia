# JIRA Mana Analysis Tool

This tool analyzes JIRA tickets based on their resolution time and a custom field called "Mana Spent". It helps identify which types of tickets consume the most resources over a specified time period. Epic and Initiative ticket types are excluded from the analysis.

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
go run main.go -start "2024-01-01" -end "2024-03-21"
```

### Command Line Arguments

- `-start`: Start date in YYYY-MM-DD format
- `-end`: End date in YYYY-MM-DD format

## Important Note

The tool assumes you have a custom field called "Mana Spent" in your JIRA instance. You'll need to modify the `customfield_10000` ID in the code to match your actual custom field ID. To find your custom field ID:

1. Open any issue in your JIRA instance
2. View the page source
3. Search for "Mana Spent"
4. Look for the corresponding `customfield_XXXXX` ID

## Output

The tool will output a table showing:
- Issue Type
- Count of issues
- Total Mana spent
- Average Mana per issue type

Results are sorted by total Mana spent in descending order.
