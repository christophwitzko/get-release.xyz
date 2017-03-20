package main

import (
	"github.com/christophwitzko/github-release-download/release"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
)

var Client *release.GithubClient

func init() {
	Client = release.NewClient(os.Getenv("GITHUB_TOKEN"))
}

func FindDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Printf("%s - %s %s", r.RemoteAddr, r.Method, r.URL.EscapedPath())
	url, err := Client.GetLatestDownloadUrl(ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value)
	if err != nil || url == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, 302)
}

func main() {
	router := httprouter.New()
	router.GET("/:owner/:repo/:os/:arch", FindDownload)

	log.Println("starting server on port 5000...")
	log.Fatal(http.ListenAndServe(":5000", router))
}
