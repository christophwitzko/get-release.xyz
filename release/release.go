package release

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"regexp"
	"strings"
)

var osarchRegexp = regexp.MustCompile("(?i)(android|darwin|dragonfly|freebsd|linux|nacl|netbsd|openbsd|plan9|solaris|windows)(_|-)(i?386|amd64p32|amd64|arm64|arm|mips64le|mips64|mipsle|mips|ppc64le|ppc64|s390x|x86_64)")

type Asset struct {
	Version  string
	OS       string
	Arch     string
	FileName string
	URL      string
}

type GithubClient struct {
	Client *github.Client
	lock   chan struct{}
}

func NewClient(token string) *GithubClient {
	return &GithubClient{github.NewClient(oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	))), make(chan struct{}, 1)}
}

func (c *GithubClient) GetLatestReleaseAssets(ctx context.Context, owner, repo string) ([]*Asset, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.lock <- struct{}{}:
	}
	release, _, err := c.Client.Repositories.GetLatestRelease(ctx, owner, repo)
	<-c.lock
	if err != nil {
		return nil, err
	}
	ret := make([]*Asset, len(release.Assets))
	for i, ra := range release.Assets {
		ret[i] = &Asset{
			Version:  release.GetTagName(),
			FileName: ra.GetName(),
			URL:      ra.GetBrowserDownloadURL(),
		}
		osarch := osarchRegexp.FindAllStringSubmatch(ret[i].FileName, -1)
		if len(osarch) < 1 || len(osarch[0]) < 4 {
			continue
		}
		ret[i].OS = strings.ToLower(osarch[0][1])
		ret[i].Arch = strings.ToLower(osarch[0][3])
	}
	return ret, nil
}

func (c *GithubClient) GetLatestDownloadUrl(ctx context.Context, owner, repo, os, arch string) (string, error) {
	assets, err := c.GetLatestReleaseAssets(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	os = strings.ToLower(os)
	arch = strings.ToLower(arch)
	for _, asset := range assets {
		if asset.OS == os && asset.Arch == arch {
			return asset.URL, nil
		}
	}
	return "", nil
}
