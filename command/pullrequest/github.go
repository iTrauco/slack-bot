package pullrequest

import (
	"context"
	"github.com/google/go-github/github"
	"github.com/innogames/slack-bot/bot"
	"github.com/innogames/slack-bot/bot/config"
	"github.com/innogames/slack-bot/bot/matcher"
	"github.com/innogames/slack-bot/client"
	"golang.org/x/oauth2"
	"text/template"
)

type githubFetcher struct {
	client *github.Client
}

func newGithubCommand(slackClient client.SlackClient, cfg config.Config) bot.Command {
	if cfg.Github.AccessToken == "" {
		return nil
	}

	ctx := context.Background()

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: cfg.Github.AccessToken},
	))

	githubClient := github.NewClient(client)

	return &command{
		slackClient,
		&githubFetcher{githubClient},
		"(?s).*https://github.com/(?P<project>.+)/(?P<repo>.+)/pull/(?P<number>\\d+).*",
	}
}

func (c *githubFetcher) getPullRequest(match matcher.Result) (pullRequest, error) {
	var pr pullRequest

	project := match.GetString("project")
	repo := match.GetString("repo")
	prNumber := match.GetInt("number")

	ctx := context.Background()
	rawPullRequest, _, err := c.client.PullRequests.Get(ctx, project, repo, prNumber)
	if err != nil {
		return pr, err
	}

	reviews, _, err := c.client.PullRequests.ListReviews(ctx, project, repo, prNumber, &github.ListOptions{})
	if err != nil {
		return pr, err
	}

	approved := false
	inReview := false

	for _, review := range reviews {
		state := review.GetState()
		if state == "COMMENTED" {
			continue
		}
		inReview = true

		if state == "APPROVED" {
			approved = true
		}
	}

	pr = pullRequest{
		name:     rawPullRequest.GetTitle(),
		merged:   rawPullRequest.GetMerged(),
		declined: false,
		approved: approved,
		inReview: inReview,
	}

	return pr, nil
}

func (c *githubFetcher) GetTemplateFunction() template.FuncMap {
	return template.FuncMap{
		"githubPullRequest": func(project string, repo string, number string) (pullRequest, error) {
			return c.getPullRequest(matcher.MapResult{
				"project": project,
				"repo":    repo,
				"number":  number,
			})
		},
	}
}

func (c *githubFetcher) getHelp() []bot.Help {
	return []bot.Help{
		{
			"github pull request",
			"tracks the state of github pull requests",
			[]string{
				"https://github.com/home-assistant/home-assistant/pull/13958",
			},
		},
	}
}
