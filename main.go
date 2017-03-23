package main

import (
	"context"
	"fmt"
	"github.com/christophwitzko/github-release-download/release"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"time"
)

var Client *release.GithubClient

func init() {
	Client = release.NewClient(os.Getenv("GITHUB_TOKEN"))
}

func logger(router http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s - %s %s", r.RemoteAddr, r.Method, r.URL.EscapedPath())
		router.ServeHTTP(w, r)
	})
}

func doRedirect(w http.ResponseWriter, r *http.Request, url string, err error) {
	if err != nil {
		log.Println(err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	if url == "" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, url, 302)
}

func GetLatestDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url, err := Client.GetLatestDownloadUrl(ctx, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value)
	doRedirect(w, r, url, err)
}

func GetMatchingDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url, err := Client.GetMatchingDownloadUrl(ctx, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value, ps[4].Value)
	doRedirect(w, r, url, err)
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintln(w, "https://get-release.xyz/:owner/:repo/:os/:arch{/:constraint}")
}

func main() {
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/:owner/:repo/:os/:arch", GetLatestDownload)
	router.GET("/:owner/:repo/:os/:arch/:constraint", GetMatchingDownload)
	log.Println("starting server on port 5000...")
	log.Fatal(http.ListenAndServe(":5000", logger(router)))
}
