package release

import (
	"context"
	"encoding/json"
	"github.com/Masterminds/semver"
	"github.com/google/go-github/v30/github"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func createReleaseAsset(name, url string) *github.ReleaseAsset {
	return &github.ReleaseAsset{Name: &name, BrowserDownloadURL: &url}
}

func createRelease(version string, draft bool) *github.RepositoryRelease {
	return &github.RepositoryRelease{TagName: &version, Draft: &draft, Assets: []*github.ReleaseAsset{
		createReleaseAsset("test_darwin_amd64", "url/a"),
		createReleaseAsset("test_windows_386.exe", "url/b"),
		createReleaseAsset("testfile", "url/c"),
		createReleaseAsset("test_nacl_amd64p32.zip", "url/d"),
	}}
}

var GITHUB_LATEST_RELEASE = createRelease("v1.0.0", false)
var GITHUB_RELEASES = []*github.RepositoryRelease{
	createRelease("v1.0.0", false),
	createRelease("v1.0.1", false),
	createRelease("v1.1.0", false),
	createRelease("v2.0.0", false),
	createRelease("v2.1.0", true),
}

func createRef(ref string) *github.Reference {
	return &github.Reference{Ref: &ref}
}

var GITHUB_TAGS = []*github.Reference{
	createRef("refs/tags/go1.0"),
	createRef("refs/tags/go1.8.1"),
	createRef("refs/tags/go1.9rc1"),
	createRef("refs/tags/goinvalid"),
}

func githubHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer token" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method == "GET" && r.URL.Path == "/repos/owner/repo/releases/latest" {
		json.NewEncoder(w).Encode(GITHUB_LATEST_RELEASE)
		return
	}
	if r.Method == "GET" && r.URL.Path == "/repos/owner/repo/releases" {
		json.NewEncoder(w).Encode(GITHUB_RELEASES)
		return
	}
	if r.Method == "GET" && r.URL.Path == "/repos/golang/go/git/refs/tags/go" {
		json.NewEncoder(w).Encode(GITHUB_TAGS)
		return
	}
	http.Error(w, "invalid route", http.StatusNotImplemented)
}

func getNewTestClient(t *testing.T) (*GithubClient, *httptest.Server) {
	client := NewClient("token")
	ts := httptest.NewServer(http.HandlerFunc(githubHandler))
	client.Client.BaseURL, _ = url.Parse(ts.URL + "/")
	return client, ts
}

func TestClient(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	release, err := client.GetLatestRelease(context.TODO(), "owner", "invalid")
	if release != nil || err == nil {
		t.Fatal("invalid client")
	}
}

func checkAsset(t *testing.T, a *Asset, os, arch string) {
	if a.Arch != arch || a.OS != os {
		t.Fatal("invalid asset")
	}
}

func TestGetLatestRelease(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	release, err := client.GetLatestRelease(context.TODO(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	checkAsset(t, release.Assets[0], "darwin", "amd64")
	checkAsset(t, release.Assets[1], "windows", "386")
	checkAsset(t, release.Assets[2], "", "")
	checkAsset(t, release.Assets[3], "nacl", "amd64p32")
}

func TestGetLatestDownloadUrl(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	url, err := client.GetLatestDownloadUrl(context.TODO(), "owner", "repo", "darwin", "amd64")
	if err != nil || url != "url/a" {
		t.Fail()
	}
	url, err = client.GetLatestDownloadUrl(context.TODO(), "owner", "repo", "darwin", "arm")
	if err != nil || url != "" {
		t.Fail()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	url, err = client.GetLatestDownloadUrl(ctx, "owner", "repo", "darwin", "amd64")
	if err == nil || err.Error() != "context deadline exceeded" {
		t.Fail()
	}
}

func TestGetAllReleases(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	releases, err := client.GetAllReleases(context.TODO(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	c1, _ := semver.NewConstraint("^1.0.0")
	c2, _ := semver.NewConstraint(">1")
	c3, _ := semver.NewConstraint(">3")
	if releases.FindSatisfying(c1).Version.String() != "1.1.0" ||
		releases.FindSatisfying(c2).Version.String() != "2.1.0" ||
		releases.WithoutDraftsOrPrereleases().FindSatisfying(c2).Version.String() != "2.0.0" ||
		releases.FindSatisfying(c3) != nil {
		t.Fatal("found invalid versions")
	}
}

func TestGetMatchingDownloadUrl(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	url, err := client.GetMatchingDownloadUrl(context.TODO(), "owner", "repo", "darwin", "amd64", "1.0.x")
	if err != nil || url != "url/a" {
		t.Fail()
	}
	url, err = client.GetMatchingDownloadUrl(context.TODO(), "owner", "repo", "darwin", "amd64", "invalid")
	if err == nil || url != "" {
		t.Fail()
	}
	url, err = client.GetMatchingDownloadUrl(context.TODO(), "owner", "repo", "darwin", "amd64", ">3")
	if err != nil || url != "" {
		t.Fail()
	}
}

func TestGetGoVersions(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	versions, err := client.GetGoVersions(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	if versions[0] != "1.0.0" ||
		versions[1] != "1.8.1" ||
		versions[2] != "1.9.0-rc1" {
		t.Fail()
	}
}
