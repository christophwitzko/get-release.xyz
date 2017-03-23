package release

import (
	"context"
	"encoding/json"
	"github.com/google/go-github/github"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func createReleaseAsset(name, url string) github.ReleaseAsset {
	return github.ReleaseAsset{Name: &name, BrowserDownloadURL: &url}
}

var GITHUB_DEFAULT_TAG = "v1.0.0"
var GITHUB_REPO = github.RepositoryRelease{TagName: &GITHUB_DEFAULT_TAG, Assets: []github.ReleaseAsset{
	createReleaseAsset("test_darwin_amd64", "url/a"),
	createReleaseAsset("test_windows_386.exe", "url/b"),
	createReleaseAsset("testfile", "url/c"),
	createReleaseAsset("test_nacl_amd64p32.zip", "url/d"),
}}

func githubHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer token" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method == "GET" && r.URL.Path == "/repos/owner/repo/releases/latest" {
		json.NewEncoder(w).Encode(GITHUB_REPO)
		return
	}
	http.Error(w, "invalid route", http.StatusNotImplemented)
}

func getNewTestClient(t *testing.T) (*GithubClient, *httptest.Server) {
	client := NewClient("token")
	ts := httptest.NewServer(http.HandlerFunc(githubHandler))
	client.Client.BaseURL, _ = url.Parse(ts.URL)
	return client, ts
}

func TestClient(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	assets, err := client.GetLatestReleaseAssets(context.TODO(), "owner", "invalid")
	if assets != nil || err == nil {
		t.Fatal("invalid client")
	}
}

func checkAsset(t *testing.T, a *Asset, os, arch string) {
	if a.Arch != arch || a.OS != os {
		t.Fatal("invalid asset")
	}
}

func TestGetLatestReleaseAssets(t *testing.T) {
	client, ts := getNewTestClient(t)
	defer ts.Close()
	assets, err := client.GetLatestReleaseAssets(context.TODO(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	checkAsset(t, assets[0], "darwin", "amd64")
	checkAsset(t, assets[1], "windows", "386")
	checkAsset(t, assets[2], "", "")
	checkAsset(t, assets[3], "nacl", "amd64p32")
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
