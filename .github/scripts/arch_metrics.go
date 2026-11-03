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
	"time"
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
query($login: String!, $after: String) {
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
      after: $after
      ownerAffiliations: OWNER
      isFork: false
      orderBy: {field: UPDATED_AT, direction: DESC}
    ) {
      totalCount
      pageInfo {
        hasNextPage
        endCursor
      }
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
