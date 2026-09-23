package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

const outputFile = "arch-metrics.svg"

const (
	width = 920

	bgColor       = "#0d1117"
	terminalColor = "#111827"
	borderColor   = "#30363d"

	archBlue = "#1793d1"
	white    = "#f0f6fc"
	text     = "#c9d1d9"
	muted    = "#8b949e"
	green    = "#3fb950"

	fontFamily = "JetBrains Mono, Fira Code, Consolas, monospace"
)

var archLogo = []string{
	"                   -`",
	"                  .o+`",
	"                 `ooo/",
	"                `+oooo:",
	"               `+oooooo:",
	"               -+oooooo+:",
	"              `/:-:++oooo+:",
	"             `/++++/+++++++:",
	"            `/++++++++++++++:",
	"           `/+++ooooooooooooo/`",
	"          ./ooosssso++osssssso+`",
	"         .oossssso-````/ossssss+`",
	"        -osssssso.      :ssssssso.",
	"       :osssssss/        osssso+++.",
	"      /ossssssss/        +ssssooo/-",
	"    `/ossssso+/:-        -:/+osssso+-",
	"   `+sso+:-`                 `.-/+oso:",
	"  `++:.                           `-/+/",
	"  .`                                 `/",
}

const query = `
query($login: String!) {
  user(login: $login) {
    login
    name
    followers {
      totalCount
    }
    following {
      totalCount
    }
    repositories(
      first: 100
      ownerAffiliations: OWNER
      isFork: false
      orderBy: {field: UPDATED_AT, direction: DESC}
    ) {
      totalCount
      nodes {
        name
        stargazerCount
        forkCount
        languages(first: 10, orderBy: {field: SIZE, direction: DESC}) {
          edges {
            size
            node {
              name
              color
            }
          }
        }
      }
    }
    contributionsCollection {
      totalCommitContributions
      totalIssueContributions
      totalPullRequestContributions
      totalPullRequestReviewContributions
      contributionCalendar {
        totalContributions
      }
    }
  }
}
`

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLResponse struct {
	Data struct {
		User userData `json:"user"`
	} `json:"data"`

	Errors []graphQLError `json:"errors"`
}

type userData struct {
	Login string `json:"login"`
	Name  string `json:"name"`

	Followers struct {
		TotalCount int `json:"totalCount"`
	} `json:"followers"`

	Following struct {
		TotalCount int `json:"totalCount"`
	} `json:"following"`

	Repositories struct {
		TotalCount int          `json:"totalCount"`
		Nodes      []repository `json:"nodes"`
	} `json:"repositories"`

	ContributionsCollection struct {
		TotalCommitContributions            int `json:"totalCommitContributions"`
		TotalIssueContributions             int `json:"totalIssueContributions"`
		TotalPullRequestContributions       int `json:"totalPullRequestContributions"`
		TotalPullRequestReviewContributions int `json:"totalPullRequestReviewContributions"`

		ContributionCalendar struct {
			TotalContributions int `json:"totalContributions"`
		} `json:"contributionCalendar"`
	} `json:"contributionsCollection"`
}

type repository struct {
	Name            string `json:"name"`
	StargazerCount  int    `json:"stargazerCount"`
	ForkCount       int    `json:"forkCount"`
	Languages       languageConnection `json:"languages"`
}

type languageConnection struct {
	Edges []languageEdge `json:"edges"`
}

type languageEdge struct {
	Size int `json:"size"`

	Node struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"node"`
}

type languageStat struct {
	Name       string
	Size       int
	Color      string
	Percentage float64
}

type metric struct {
	Label string
	Value int
}

type infoLine struct {
	Key   string
	Value string
}

func main() {
	username := os.Getenv("GITHUB_REPOSITORY_OWNER")
	if username == "" {
		username = "zhixe"
	}

	token := os.Getenv("METRICS_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	if token == "" {
		fatal("METRICS_TOKEN or GITHUB_TOKEN is required")
	}

	user, err := fetchGitHubData(username, token)
	if err != nil {
		fatal(err.Error())
	}

	if user.Name == "" {
		user.Name = username
	}

	languages := calculateLanguages(user.Repositories.Nodes)

	metrics := []metric{
		{"Repositories", user.Repositories.TotalCount},
		{
			"Contributions",
			user.ContributionsCollection.ContributionCalendar.TotalContributions,
		},
		{
			"Commits",
			user.ContributionsCollection.TotalCommitContributions,
		},
		{
			"Pull requests",
			user.ContributionsCollection.TotalPullRequestContributions,
		},
		{
			"Reviews",
			user.ContributionsCollection.TotalPullRequestReviewContributions,
		},
		{
			"Issues",
			user.ContributionsCollection.TotalIssueContributions,
		},
		{"Followers", user.Followers.TotalCount},
		{"Following", user.Following.TotalCount},
		{"Stars", totalStars(user.Repositories.Nodes)},
		{"Forks", totalForks(user.Repositories.Nodes)},
	}

	info := []infoLine{
		{"user", username},
		{"name", user.Name},
		{"os", "Arch Linux"},
		{"role", "Software + Data Engineer"},
		{"shell", "zsh"},
		{"location", "Malaysia"},
		{"github", "github.com/" + username},
	}

	svg := generateSVG(username, info, metrics, languages)

	if err := os.WriteFile(outputFile, []byte(svg), 0644); err != nil {
		fatal(fmt.Sprintf("failed to write SVG: %v", err))
	}

	fmt.Printf("Generated %s\n", outputFile)
}

func fetchGitHubData(username, token string) (userData, error) {
	body := graphQLRequest{
		Query: query,
		Variables: map[string]any{
			"login": username,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return userData{}, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://api.github.com/graphql",
		bytes.NewReader(payload),
	)
	if err != nil {
		return userData{}, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "zhixe-profile-metrics")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return userData{}, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return userData{}, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return userData{}, fmt.Errorf(
			"github API returned HTTP %d: %s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	var result graphQLResponse

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return userData{}, err
	}

	if len(result.Errors) > 0 {
		messages := make([]string, 0, len(result.Errors))

		for _, apiErr := range result.Errors {
			messages = append(messages, apiErr.Message)
		}

		return userData{}, fmt.Errorf(
			"GraphQL error: %s",
			strings.Join(messages, "; "),
		)
	}

	return result.Data.User, nil
}

func calculateLanguages(repositories []repository) []languageStat {
	sizes := map[string]int{}
	colors := map[string]string{}

	for _, repo := range repositories {
		for _, edge := range repo.Languages.Edges {
			if edge.Node.Name == "" {
				continue
			}

			sizes[edge.Node.Name] += edge.Size

			if edge.Node.Color != "" {
				colors[edge.Node.Name] = edge.Node.Color
			}
		}
	}

	languages := make([]languageStat, 0, len(sizes))

	for name, size := range sizes {
		color := colors[name]

		if color == "" {
			color = archBlue
		}

		languages = append(
			languages,
			languageStat{
				Name:  name,
				Size:  size,
				Color: color,
			},
		)
	}

	sort.Slice(
		languages,
		func(i, j int) bool {
			return languages[i].Size > languages[j].Size
		},
	)

	if len(languages) > 8 {
		languages = languages[:8]
	}

	total := 0

	for _, language := range languages {
		total += language.Size
	}

	if total == 0 {
		total = 1
	}

	for i := range languages {
		languages[i].Percentage =
			float64(languages[i].Size) /
				float64(total) *
				100
	}

	return languages
}

func totalStars(repositories []repository) int {
	total := 0

	for _, repo := range repositories {
		total += repo.StargazerCount
	}

	return total
}

func totalForks(repositories []repository) int {
	total := 0

	for _, repo := range repositories {
		total += repo.ForkCount
	}

	return total
}

func generateSVG(
	username string,
	info []infoLine,
	metrics []metric,
	languages []languageStat,
) string {

	/*
		Dynamic layout configuration.

		The positions below are starting points only.
		The lower sections are calculated automatically.
	*/

	logoX := 25.0
	logoY := 68.0
	logoLineHeight := 16.0
	logoScale := 1.35

	infoX := 470.0
	infoY := 110.0
	infoRowGap := 28.0

	metricRowGap := 28.0
	languageRowGap := 34.0

	barX := 280.0
	barWidth := 420.0
	barHeight := 12.0

	/*
		Calculate where the top area ends.
	*/

	logoBottom :=
		logoY +
			float64(len(archLogo)-1)*
				logoLineHeight*
				logoScale

	infoBottom :=
		infoY +
			float64(len(info)-1)*
				infoRowGap

	topBlockBottom := maxFloat(
		logoBottom,
		infoBottom,
	)

	/*
		GitHub status position is derived from
		the bottom of the logo and information block.
	*/

	sectionY := topBlockBottom + 45
	metricY := sectionY + 34

	leftMetrics := metrics[:5]
	rightMetrics := metrics[5:]

	metricRows := len(leftMetrics)

	if len(rightMetrics) > metricRows {
		metricRows = len(rightMetrics)
	}

	metricsBottom :=
		metricY +
			float64(metricRows-1)*
				metricRowGap

	/*
		Language section follows the metrics section.
	*/

	languageHeaderY := metricsBottom + 60
	languageY := languageHeaderY + 36

	languagesBottom := languageY

	if len(languages) > 0 {
		languagesBottom =
			languageY +
				float64(len(languages)-1)*
					languageRowGap
	}

	/*
		Footer follows however many languages exist.
	*/

	footerY := languagesBottom + 100

	height := int(footerY + 55)

	var svg strings.Builder

	fmt.Fprintf(
		&svg,
		`<svg
xmlns="http://www.w3.org/2000/svg"
width="%d"
height="%d"
viewBox="0 0 %d %d"
>
`,
		width,
		height,
		width,
		height,
	)

	fmt.Fprintf(
		&svg,
		`
<style>
	.mono {
		font-family: %s;
	}

	.prompt {
		font-size: 17px;
		font-weight: 700;
		fill: %s;
	}

	.command {
		font-size: 17px;
		font-weight: 700;
		fill: %s;
	}

	.text {
		font-size: 16px;
		fill: %s;
	}

	.muted {
		font-size: 14px;
		fill: %s;
	}

	.metric {
		font-size: 16px;
		font-weight: 700;
		fill: %s;
	}

	.logo {
		font-size: 13px;
		font-weight: 500;
		fill: %s;
		white-space: pre;
	}
</style>
`,
		fontFamily,
		archBlue,
		white,
		text,
		muted,
		green,
		archBlue,
	)

	fmt.Fprintf(
		&svg,
		`
<rect
	width="%d"
	height="%d"
	rx="16"
	fill="%s"
/>

<rect
	x="14"
	y="14"
	width="%d"
	height="%d"
	rx="14"
	fill="%s"
	stroke="%s"
/>
`,
		width,
		height,
		bgColor,
		width-28,
		height-28,
		terminalColor,
		borderColor,
	)

	/*
		Window header.
	*/

	fmt.Fprintf(
		&svg,
		`
<rect
	x="14"
	y="14"
	width="%d"
	height="44"
	rx="14"
	fill="#161b22"
/>

<circle cx="42" cy="36" r="7" fill="#ff5f56"/>
<circle cx="65" cy="36" r="7" fill="#ffbd2e"/>
<circle cx="88" cy="36" r="7" fill="#27c93f"/>

<text
	x="%d"
	y="42"
	text-anchor="middle"
	class="mono muted"
>
	%s@arch: ~
</text>
`,
		width-28,
		width/2,
		xmlEscape(username),
	)

	/*
		Arch ASCII logo.
	*/

	fmt.Fprintf(
		&svg,
		`
<g transform="translate(%.1f %.1f) scale(%.2f)">
`,
		logoX,
		logoY,
		logoScale,
	)

	for i, line := range archLogo {
		y := float64(i) * logoLineHeight

		preservedLine := strings.ReplaceAll(
			html.EscapeString(line),
			" ",
			"&#160;",
		)

		fmt.Fprintf(
			&svg,
			`
<text
	x="0"
	y="%.1f"
	class="mono logo"
	xml:space="preserve"
>%s</text>
`,
			y,
			preservedLine,
		)
	}

	svg.WriteString("</g>\n")

	/*
		System information.
	*/

	for i, line := range info {
		y := infoY + float64(i)*infoRowGap

		fmt.Fprintf(
			&svg,
			`
<text
	x="%.1f"
	y="%.1f"
	class="mono prompt"
>%s</text>

<text
	x="%.1f"
	y="%.1f"
	class="mono text"
>: %s</text>
`,
			infoX,
			y,
			xmlEscape(line.Key),
			infoX+110,
			y,
			xmlEscape(line.Value),
		)
	}

	/*
		GitHub status command.
	*/

	fmt.Fprintf(
		&svg,
		`
<text
	x="45"
	y="%.1f"
	class="mono prompt"
>%s@arch ~ $</text>

<text
	x="210"
	y="%.1f"
	class="mono command"
>github-status</text>
`,
		sectionY,
		xmlEscape(username),
		sectionY,
	)

	/*
		Metrics columns.
	*/

	for i, item := range leftMetrics {
		y := metricY + float64(i)*metricRowGap

		fmt.Fprintf(
			&svg,
			`
<text
	x="65"
	y="%.1f"
	class="mono muted"
>%s</text>

<text
	x="250"
	y="%.1f"
	class="mono metric"
>%d</text>
`,
			y,
			xmlEscape(item.Label),
			y,
			item.Value,
		)
	}

	for i, item := range rightMetrics {
		y := metricY + float64(i)*metricRowGap

		fmt.Fprintf(
			&svg,
			`
<text
	x="430"
	y="%.1f"
	class="mono muted"
>%s</text>

<text
	x="620"
	y="%.1f"
	class="mono metric"
>%d</text>
`,
			y,
			xmlEscape(item.Label),
			y,
			item.Value,
		)
	}

	/*
		Language command.
	*/

	fmt.Fprintf(
		&svg,
		`
<text
	x="45"
	y="%.1f"
	class="mono prompt"
>%s@arch ~ $</text>

<text
	x="210"
	y="%.1f"
	class="mono command"
>languages --top</text>
`,
		languageHeaderY,
		xmlEscape(username),
		languageHeaderY,
	)

	/*
		Language bars.
	*/

	for i, language := range languages {
		y := languageY +
			float64(i)*languageRowGap

		fillWidth :=
			barWidth *
				language.Percentage /
				100

		fmt.Fprintf(
			&svg,
			`
<text
	x="65"
	y="%.1f"
	class="mono text"
>%s</text>

<rect
	x="%.1f"
	y="%.1f"
	width="%.1f"
	height="%.1f"
	rx="6"
	fill="#21262d"
/>

<rect
	x="%.1f"
	y="%.1f"
	width="%.1f"
	height="%.1f"
	rx="6"
	fill="%s"
/>

<text
	x="%.1f"
	y="%.1f"
	class="mono muted"
>%.1f%%</text>
`,
			y+10,
			xmlEscape(language.Name),

			barX,
			y,
			barWidth,
			barHeight,

			barX,
			y,
			fillWidth,
			barHeight,
			xmlEscape(language.Color),

			barX+barWidth+18,
			y+10,
			language.Percentage,
		)
	}

	/*
		Final prompt with blinking cursor.
	*/

	fmt.Fprintf(
		&svg,
		`
<text
	x="45"
	y="%.1f"
	class="mono prompt"
>%s@arch ~ $</text>

<rect
	x="210"
	y="%.1f"
	width="10"
	height="18"
	fill="%s"
>
	<animate
		attributeName="opacity"
		values="1;0;1"
		dur="1.1s"
		repeatCount="indefinite"
	/>
</rect>
`,
		footerY,
		xmlEscape(username),
		footerY-14,
		archBlue,
	)

	svg.WriteString("</svg>\n")

	fmt.Printf(
		"Computed layout: HEIGHT=%d sectionY=%.0f languageHeaderY=%.0f footerY=%.0f\n",
		height,
		sectionY,
		languageHeaderY,
		footerY,
	)

	return svg.String()
}

func xmlEscape(value string) string {
	return html.EscapeString(value)
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "error:", message)
	os.Exit(1)
}
