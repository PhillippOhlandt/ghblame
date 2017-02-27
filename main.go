package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"math"
	"os"
	"strconv"
)

var (
	owner string
	repo  string
)

func main() {

	token := ""
	amount := 100

	if len(os.Args) > 1 {
		token = os.Args[1]
	}

	if len(os.Args) > 2 {
		owner = os.Args[2]
	}

	if len(os.Args) > 3 {
		repo = os.Args[3]
	}

	if len(os.Args) > 4 {
		i, err := strconv.Atoi(os.Args[4])
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		amount = i
	}

	if token == "" || owner == "" || repo == "" {
		fmt.Println("You need to provide your github token, repo owner name and repo name!")
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	issues, _ := getAllIssues(client, ctx, amount)

	fmt.Printf("For %v Issues: \n", len(issues))

	var issuesTimeUntilClose []float64
	var issuesTimeUntilFirstComment []float64

	for _, issue := range issues {
		comments, _ := getComments(client, ctx, *issue.Number)

		timeUntilClose, timeUntilFirstComment := getStatisticsForIssue(issue, comments)

		if timeUntilClose > 0.0 {
			issuesTimeUntilClose = append(issuesTimeUntilClose, timeUntilClose)
		}

		if timeUntilFirstComment > 0.0 {
			issuesTimeUntilFirstComment = append(issuesTimeUntilFirstComment, timeUntilFirstComment)
		}
	}

	fmt.Printf("Average time until first comment: %v mins\nAverage time until close: %v mins",
		round(average(issuesTimeUntilFirstComment)),
		round(average(issuesTimeUntilClose)),
	)

	fmt.Println("")
	fmt.Println("")

	prs, _ := getAllPullRequests(client, ctx, amount)

	fmt.Printf("For %v Pull Requests: \n", len(prs))

	var prsTimeUntilClose []float64
	var prsTimeUntilMerge []float64
	var prsTimeUntilFirstComment []float64

	for _, pr := range prs {
		comments, _ := getComments(client, ctx, *pr.Number)
		//fmt.Printf("#%v  %v, Comments: %v %v\n", *pr.Number, *pr.Title, len(comments), *pr.State)
		timeUntilClose, timeUntilMerge, timeUntilFirstComment := getStatisticsForPullRequest(pr, comments)

		if timeUntilClose > 0.0 {
			prsTimeUntilClose = append(prsTimeUntilClose, timeUntilClose)
		}

		if timeUntilMerge > 0.0 {
			prsTimeUntilMerge = append(prsTimeUntilMerge, timeUntilMerge)
		}

		if timeUntilFirstComment > 0.0 {
			prsTimeUntilFirstComment = append(prsTimeUntilFirstComment, timeUntilFirstComment)
		}
	}

	fmt.Printf("Average time until first comment: %v mins\nAverage time until merge: %v mins \nAverage time until close: %v mins\n",
		round(average(prsTimeUntilFirstComment)),
		round(average(prsTimeUntilMerge)),
		round(average(prsTimeUntilClose)),
	)
}

func getAllIssues(client *github.Client, ctx context.Context, amount int) ([]*github.Issue, error) {
	options := &github.IssueListByRepoOptions{
		State:       "closed",
		Sort:        "created",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, options)
		if err != nil {
			return nil, err
		}
		allIssues = append(allIssues, getOnlyIssues(issues)...)
		if resp.NextPage == 0 || len(allIssues) >= amount {
			break
		}
		options.ListOptions.Page = resp.NextPage
	}

	var wantedIssues []*github.Issue

	for index, issue := range allIssues {
		if index < amount {
			wantedIssues = append(wantedIssues, issue)
		}
	}

	return wantedIssues, nil
}

func getOnlyIssues(issues []*github.Issue) []*github.Issue {
	var onlyIssues []*github.Issue

	for _, issue := range issues {
		if issue.PullRequestLinks == nil {
			onlyIssues = append(onlyIssues, issue)
		}
	}

	return onlyIssues
}

func getAllPullRequests(client *github.Client, ctx context.Context, amount int) ([]*github.PullRequest, error) {
	options := &github.PullRequestListOptions{
		State:       "closed",
		Sort:        "created",
		Direction:   "desc",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allPrs []*github.PullRequest
	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repo, options)
		if err != nil {
			return nil, err
		}
		allPrs = append(allPrs, prs...)
		if resp.NextPage == 0 || len(allPrs) >= amount {
			break
		}
		options.ListOptions.Page = resp.NextPage
	}

	var wantedPrs []*github.PullRequest

	for index, pr := range allPrs {
		if index < amount {
			wantedPrs = append(wantedPrs, pr)
		}
	}

	return wantedPrs, nil
}

func getComments(client *github.Client, ctx context.Context, number int) ([]*github.IssueComment, error) {
	options := &github.IssueListCommentsOptions{
		Sort:        "created",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allComments []*github.IssueComment
	for {
		comments, resp, err := client.Issues.ListComments(ctx, owner, repo, number, options)
		if err != nil {
			return nil, err
		}
		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		options.ListOptions.Page = resp.NextPage
	}

	return allComments, nil
}

func getStatisticsForIssue(issue *github.Issue, comments []*github.IssueComment) (float64, float64) {

	timeUntilClose := 0.0
	timeUntilFirstComment := 0.0

	if issue.ClosedAt != nil {
		delta := issue.ClosedAt.Sub(*issue.CreatedAt)
		timeUntilClose = round(delta.Minutes())
	}

	if len(comments) > 0 {
		for _, comment := range comments {
			if *issue.User.ID != *comment.User.ID {
				delta := comment.CreatedAt.Sub(*issue.CreatedAt)
				timeUntilFirstComment = round(delta.Minutes())
				break
			}
		}
	}

	return timeUntilClose, timeUntilFirstComment
}

func getStatisticsForPullRequest(pr *github.PullRequest, comments []*github.IssueComment) (float64, float64, float64) {

	timeUntilClose := 0.0
	timeUntilMerge := 0.0
	timeUntilFirstComment := 0.0

	if pr.ClosedAt != nil {
		delta := pr.ClosedAt.Sub(*pr.CreatedAt)
		timeUntilClose = round(delta.Minutes())
	}

	if pr.MergedAt != nil {
		delta := pr.MergedAt.Sub(*pr.CreatedAt)
		timeUntilMerge = round(delta.Minutes())
	}

	if len(comments) > 0 {
		for _, comment := range comments {
			if *pr.User.ID != *comment.User.ID {
				delta := comment.CreatedAt.Sub(*pr.CreatedAt)
				timeUntilFirstComment = round(delta.Minutes())
				break
			}
		}
	}

	return timeUntilClose, timeUntilMerge, timeUntilFirstComment
}

func round(f float64) float64 {
	return math.Floor(f + .5)
}

func average(v []float64) float64 {
	var total float64 = 0
	for _, value := range v {
		total += value
	}
	return total / float64(len(v))
}
