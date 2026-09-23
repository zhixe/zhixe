import json
import os
import urllib.request
from collections import defaultdict
from datetime import datetime, timezone
from html import escape

USERNAME = os.environ.get("GITHUB_REPOSITORY_OWNER", "zhixe")
TOKEN = os.environ.get("METRICS_TOKEN") or os.environ.get("GITHUB_TOKEN")

OUTPUT = "arch-metrics.svg"


def graphql(query, variables=None):
    payload = json.dumps(
        {
            "query": query,
            "variables": variables or {},
        }
    ).encode("utf-8")

    request = urllib.request.Request(
        "https://api.github.com/graphql",
        data=payload,
        headers={
            "Authorization": f"Bearer {TOKEN}",
            "Content-Type": "application/json",
            "User-Agent": "zhixe-profile-metrics",
        },
        method="POST",
    )

    with urllib.request.urlopen(request) as response:
        result = json.loads(response.read().decode("utf-8"))

    if "errors" in result:
        raise RuntimeError(json.dumps(result["errors"], indent=2))

    return result["data"]


QUERY = """
query($login: String!) {
  user(login: $login) {
    login
    name
    createdAt
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
"""

data = graphql(QUERY, {"login": USERNAME})
user = data["user"]

name = user["name"] or USERNAME
repo_count = user["repositories"]["totalCount"]
followers = user["followers"]["totalCount"]
following = user["following"]["totalCount"]

contrib = user["contributionsCollection"]

year_contributions = contrib["contributionCalendar"]["totalContributions"]
commits = contrib["totalCommitContributions"]
issues = contrib["totalIssueContributions"]
pull_requests = contrib["totalPullRequestContributions"]
reviews = contrib["totalPullRequestReviewContributions"]

stars = sum(repo["stargazerCount"] for repo in user["repositories"]["nodes"])
forks = sum(repo["forkCount"] for repo in user["repositories"]["nodes"])

created_at = datetime.fromisoformat(
    user["createdAt"].replace("Z", "+00:00")
)

now = datetime.now(timezone.utc)

account_years = max(
    0,
    int((now - created_at).days / 365.25)
)

languages = defaultdict(int)
language_colors = {}

for repo in user["repositories"]["nodes"]:
    language_data = repo.get("languages")

    if not language_data:
        continue

    for edge in language_data["edges"]:
        language = edge["node"]["name"]
        size = edge["size"]

        languages[language] += size

        if edge["node"].get("color"):
            language_colors[language] = edge["node"]["color"]

sorted_languages = sorted(
    languages.items(),
    key=lambda item: item[1],
    reverse=True,
)[:8]

total_language_size = sum(size for _, size in sorted_languages) or 1


def language_percent(size):
    return round((size / total_language_size) * 100, 1)


ARCH_LOGO = [
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
]

WIDTH = 920
HEIGHT = 980

BG = "#0d1117"
TERMINAL_BG = "#111827"
BORDER = "#30363d"

ARCH_BLUE = "#1793d1"
WHITE = "#f0f6fc"
TEXT = "#c9d1d9"
MUTED = "#8b949e"
GREEN = "#3fb950"

FONT = "JetBrains Mono, Fira Code, Consolas, monospace"

svg = []

svg.append(
    f"""
<svg
    xmlns="http://www.w3.org/2000/svg"
    width="{WIDTH}"
    height="{HEIGHT}"
    viewBox="0 0 {WIDTH} {HEIGHT}"
>
"""
)

svg.append(
    f"""
<style>
    .mono {{
        font-family: {FONT};
    }}

    .title {{
        font-size: 20px;
        font-weight: 700;
        fill: {WHITE};
    }}

    .prompt {{
        font-size: 17px;
        font-weight: 700;
        fill: {ARCH_BLUE};
    }}

    .command {{
        font-size: 17px;
        fill: {WHITE};
    }}

    .text {{
        font-size: 16px;
        fill: {TEXT};
    }}

    .muted {{
        font-size: 14px;
        fill: {MUTED};
    }}

    .metric {{
        font-size: 16px;
        font-weight: 700;
        fill: {GREEN};
    }}

    .logo {{
        font-size: 12px;
        fill: {ARCH_BLUE};
    }}
</style>
"""
)

svg.append(
    f"""
<rect width="{WIDTH}" height="{HEIGHT}" rx="16" fill="{BG}"/>
<rect
    x="14"
    y="14"
    width="{WIDTH - 28}"
    height="{HEIGHT - 28}"
    rx="14"
    fill="{TERMINAL_BG}"
    stroke="{BORDER}"
/>
"""
)

svg.append(
    f"""
<rect
    x="14"
    y="14"
    width="{WIDTH - 28}"
    height="44"
    rx="14"
    fill="#161b22"
/>

<circle cx="42" cy="36" r="7" fill="#ff5f56"/>
<circle cx="65" cy="36" r="7" fill="#ffbd2e"/>
<circle cx="88" cy="36" r="7" fill="#27c93f"/>

<text
    x="{WIDTH / 2}"
    y="42"
    text-anchor="middle"
    class="mono muted"
>
    zhixe@arch: ~
</text>
"""
)

logo_x = 45
logo_y = 92

for i, line in enumerate(ARCH_LOGO):
    svg.append(
        f"""
<text x="{logo_x}" y="{logo_y + i * 14}" class="mono logo">{escape(line)}</text>
"""
    )

info_x = 390
info_y = 105

info_lines = [
    ("user", USERNAME),
    ("name", name),
    ("os", "Arch Linux"),
    ("role", "Software + Data Engineer"),
    ("shell", "zsh"),
    ("location", "Malaysia"),
    ("github", f"github.com/{USERNAME}"),
]

for i, (key, value) in enumerate(info_lines):
    y = info_y + i * 28

    svg.append(
        f"""
<text x="{info_x}" y="{y}" class="mono prompt">{escape(key)}</text>
<text x="{info_x + 110}" y="{y}" class="mono text">: {escape(value)}</text>
"""
    )

section_y = 390

svg.append(
    f"""
<text x="45" y="{section_y}" class="mono prompt">zhixe@arch ~ $</text>
<text x="210" y="{section_y}" class="mono command">github-status</text>
"""
)

metrics = [
    ("Repositories", repo_count),
    ("Contributions", year_contributions),
    ("Commits", commits),
    ("Pull requests", pull_requests),
    ("Reviews", reviews),
    ("Issues", issues),
    ("Followers", followers),
    ("Following", following),
    ("Stars", stars),
    ("Forks", forks),
]

left_metrics = metrics[:5]
right_metrics = metrics[5:]

metric_y = section_y + 34

for i, (label, value) in enumerate(left_metrics):
    y = metric_y + i * 28

    svg.append(
        f"""
<text x="65" y="{y}" class="mono muted">{escape(label)}</text>
<text x="250" y="{y}" class="mono metric">{value}</text>
"""
    )

for i, (label, value) in enumerate(right_metrics):
    y = metric_y + i * 28

    svg.append(
        f"""
<text x="430" y="{y}" class="mono muted">{escape(label)}</text>
<text x="620" y="{y}" class="mono metric">{value}</text>
"""
    )

lang_header_y = 580

svg.append(
    f"""
<text x="45" y="{lang_header_y}" class="mono prompt">zhixe@arch ~ $</text>
<text x="210" y="{lang_header_y}" class="mono command">languages --top</text>
"""
)

lang_y = lang_header_y + 36

BAR_X = 280
BAR_WIDTH = 420
BAR_HEIGHT = 12

for i, (language, size) in enumerate(sorted_languages):
    percent = language_percent(size)
    y = lang_y + i * 34
    color = language_colors.get(language, ARCH_BLUE)

    svg.append(
        f"""
<text x="65" y="{y + 10}" class="mono text">{escape(language)}</text>

<rect
    x="{BAR_X}"
    y="{y}"
    width="{BAR_WIDTH}"
    height="{BAR_HEIGHT}"
    rx="6"
    fill="#21262d"
/>

<rect
    x="{BAR_X}"
    y="{y}"
    width="{BAR_WIDTH * percent / 100:.1f}"
    height="{BAR_HEIGHT}"
    rx="6"
    fill="{color}"
/>

<text x="{BAR_X + BAR_WIDTH + 18}" y="{y + 10}" class="mono muted">{percent}%</text>
"""
    )

footer_y = HEIGHT - 55

svg.append(
    f"""
<text x="45" y="{footer_y}" class="mono prompt">zhixe@arch ~ $</text>

<rect
    x="210"
    y="{footer_y - 14}"
    width="10"
    height="18"
    fill="{ARCH_BLUE}"
>
    <animate
        attributeName="opacity"
        values="1;0;1"
        dur="1.1s"
        repeatCount="indefinite"
    />
</rect>
"""
)

svg.append("</svg>")

with open(OUTPUT, "w", encoding="utf-8") as file:
    file.write("".join(svg))

print(f"Generated {OUTPUT}")
