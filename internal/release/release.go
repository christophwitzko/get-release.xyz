package release

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/google/go-github/v30/github"
	"golang.org/x/oauth2"
)

var osarchRegexp = regexp.MustCompile("(?i)(android|darwin|dragonfly|freebsd|linux|nacl|netbsd|openbsd|plan9|solaris|windows)(_|-)(i?386|amd64p32|amd64|arm64|arm|mips64le|mips64|mipsle|mips|ppc64le|ppc64|s390x|x86_64)")

type Asset struct {
	FileName string
	URL      string
	OS       string
	Arch     string
}

type Assets []*Asset

func (a Assets) FindURLByOsArch(os, arch string) string {
	os = strings.ToLower(os)
	arch = strings.ToLower(arch)
	for _, asset := range a {
		if asset.OS == os && asset.Arch == arch {
			return asset.URL
		}
	}
	return ""
}

type Release struct {
	Version    *semver.Version
	Draft      bool
	Prerelease bool
	Assets     Assets
}

type Releases []*Release

func (r Releases) Len() int {
	return len(r)
}

func (r Releases) Less(i, j int) bool {
	return r[j].Version.LessThan(r[i].Version)
}

func (r Releases) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Releases) WithoutDraftsOrPrereleases() Releases {
	ret := make(Releases, 0)
	for _, re := range r {
		if re.Draft || re.Prerelease {
			continue
		}
		ret = append(ret, re)
	}
	return ret
}

func (r Releases) FindSatisfying(constraint *semver.Constraints) *Release {
	for _, re := range r {
		if constraint.Check(re.Version) {
			return re
		}
	}
	return nil
}

type GithubClient struct {
	Client *github.Client
	lock   chan struct{}
}

func NewClient(token string) *GithubClient {
	return &GithubClient{
		Client: github.NewClient(oauth2.NewClient(context.TODO(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))),
		lock:   make(chan struct{}, 1),
	}
}

func parseRelease(release *github.RepositoryRelease) *Release {
	version, _ := semver.NewVersion(release.GetTagName())
	ret := &Release{
		Version:    version,
		Draft:      release.GetDraft(),
		Prerelease: release.GetPrerelease(),
		Assets:     make(Assets, len(release.Assets)),
	}
	for i, ra := range release.Assets {
		ret.Assets[i] = &Asset{
			FileName: ra.GetName(),
			URL:      ra.GetBrowserDownloadURL(),
		}
		osarch := osarchRegexp.FindAllStringSubmatch(ret.Assets[i].FileName, -1)
		if len(osarch) < 1 || len(osarch[0]) < 4 {
			continue
		}
		ret.Assets[i].OS = strings.ToLower(osarch[0][1])
		ret.Assets[i].Arch = strings.ToLower(osarch[0][3])
	}
	return ret
}

func (c *GithubClient) GetAllReleases(ctx context.Context, owner, repo string) (Releases, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.lock <- struct{}{}:
	}
	defer func() {
		<-c.lock
	}()
	releases, _, err := c.Client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return nil, err
	}
	ret := make(Releases, len(releases))
	for i, re := range releases {
		ret[i] = parseRelease(re)
	}
	sort.Sort(ret)
	return ret, nil
}

func (c *GithubClient) GetLatestRelease(ctx context.Context, owner, repo string) (*Release, error) {
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
	return parseRelease(release), nil
}

func (c *GithubClient) GetLatestDownloadUrl(ctx context.Context, owner, repo, os, arch string) (string, error) {
	release, err := c.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	return release.Assets.FindURLByOsArch(os, arch), nil
}

func (c *GithubClient) GetMatchingDownloadUrl(ctx context.Context, owner, repo, os, arch, constraint string) (string, error) {
	releases, err := c.GetAllReleases(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	cnst, cerr := semver.NewConstraint(constraint)
	if cerr != nil {
		return "", cerr
	}
	found := releases.FindSatisfying(cnst)
	if found == nil {
		return "", nil
	}
	return found.Assets.FindURLByOsArch(os, arch), nil
}

func (c *GithubClient) GetAllVersions(ctx context.Context, prefix, owner, repo string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c.lock <- struct{}{}:
	}
	defer func() {
		<-c.lock
	}()
	allRefs := make([]string, 0)
	opts := &github.ReferenceListOptions{
		Type:        "tags/" + prefix,
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		refs, resp, err := c.Client.Git.ListRefs(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		for _, r := range refs {
			allRefs = append(allRefs, strings.TrimPrefix(r.GetRef(), "refs/tags/"))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allRefs, nil
}

func (c *GithubClient) GetGoVersions(ctx context.Context) ([]string, error) {
	refs, err := c.GetAllVersions(ctx, "go", "golang", "go")
	if err != nil {
		return nil, err
	}
	goVers := make([]string, 0)
	for _, ver := range refs {
		ver = strings.TrimPrefix(ver, "go")
		ver = strings.Replace(ver, "beta", "-beta", 1)
		ver = strings.Replace(ver, "rc", "-rc", 1)
		v, err := semver.NewVersion(ver)
		if err != nil {
			continue
		}
		goVers = append(goVers, v.String())
	}
	return goVers, nil
}
