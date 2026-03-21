package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

func runAssetsDelete(args []string) error {
	fs := newFlagSet("assets-delete")
	owner := fs.String("owner", "", "GitHub repository owner (required)")
	repo := fs.String("repo", "", "GitHub repository name (required)")
	tag := fs.String("tag", "", "Release tag (required)")
	token := fs.String("token", os.Getenv(EnvGitHubToken), "GitHub token (or set GITHUB_TOKEN env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *owner == "" || *repo == "" || *tag == "" || *token == "" {
		return fmt.Errorf("owner, repo, tag, and token are required")
	}
	return deleteReleaseAssetsByTag(*owner, *repo, *tag, *token)
}

func deleteReleaseAssetsByTag(owner, repo, tag, token string) error {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	release, _, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, tag)
	if err != nil {
		return fmt.Errorf("failed to get release by tag: %w", err)
	}

	assets, _, err := client.Repositories.ListReleaseAssets(ctx, owner, repo, release.GetID(), nil)
	if err != nil {
		return fmt.Errorf("failed to list release assets: %w", err)
	}

	for _, asset := range assets {
		_, err := client.Repositories.DeleteReleaseAsset(ctx, owner, repo, asset.GetID())
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to delete asset %s: %v\n", asset.GetName(), err)
		} else {
			fmt.Printf("🗑️  Deleted asset: %s\n", asset.GetName())
		}
	}
	return nil
}
