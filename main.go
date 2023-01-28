package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/repository"
)

func main() {
	if err := cli(); err != nil {
		fmt.Fprintf(os.Stderr, "gh-comment-update failed: %s\n", err.Error())
		os.Exit(1)
	}
}

func cli() error {
	issueFlag := flag.Int("issue", 0, "Issue or PR number")
	regexpFlag := flag.String("regexp", "", "Search first matched comment by regexp")
	bodyFlag := flag.String("body", "", "Update body text")
	repoOverride := flag.String(
		"repo", "", "Specify a repository. If omitted, uses current repository")
	flag.Parse()

	if *issueFlag == 0 || *regexpFlag == "" || *bodyFlag == "" {
		return fmt.Errorf("issue, regexp and body flags are required")
	}

	var repo repository.Repository
	var err error

	if *repoOverride == "" {
		repo, err = gh.CurrentRepository()
	} else {
		repo, err = repository.Parse(*repoOverride)
	}
	if err != nil {
		return fmt.Errorf("could not determine what repo to use: %w", err.Error())
	}

	client, err := gh.GQLClient(nil)
	if err != nil {
		return fmt.Errorf("could not create a graphql client: %w", err)
	}
	query := fmt.Sprintf(`{
		repository(owner: "%s", name: "%s") {
			id
			issueOrPullRequest(number: %d) {
				... on Issue {
					id
					comments(first: 100) {
						edges {
							node {
								id
								body
								viewerDidAuthor
							}
						}
					}
				}
				... on PullRequest {
					id
					comments(first: 100) {
						edges {
							node {
								id
								body
								viewerDidAuthor
							}
						}
					}
				}
			}
		}
	}`, repo.Owner(), repo.Name(), *issueFlag)

	type Comment struct {
		Id              string
		Body            string
		ViewerDidAuthor bool
	}

	response := struct {
		Repository struct {
			IssueOrPullRequest struct {
				Comments struct {
					Edges []struct {
						Node Comment
					}
				}
			}
		}
	}{}

	err = client.Do(query, nil, &response)
	if err != nil {
		return fmt.Errorf("failed to talk to the GitHub API: %w", err)
	}
	var match *Comment

	r := regexp.MustCompile(*regexpFlag)

	for _, edge := range response.Repository.IssueOrPullRequest.Comments.Edges {
		if edge.Node.ViewerDidAuthor && r.MatchString(edge.Node.Body) {
			match = &edge.Node
			break
		}
	}
	if match == nil {
		fmt.Fprintln(os.Stderr, "No matching comment found.")
		return nil
	}

	// mutation

	var m struct {
		UpdateIssueComment struct {
			ClientMutationId string
		} `graphql:"updateIssueComment(input: $input)"`
	}
	type UpdateIssueCommentInput struct {
		ID   string `json:"id"`
		Body string `json:"body"`
	}
	variables := map[string]interface{}{
		"input": UpdateIssueCommentInput{
			ID:   string(match.Id),
			Body: string(*bodyFlag),
		},
	}

	err = client.Mutate("gh_comment_update", &m, variables)
	if err != nil {
		return fmt.Errorf("failed to mutate comment: %w", err)
	}

	return nil
}
