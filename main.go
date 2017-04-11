package main

import (
	"context"
	"fmt"
	"github.com/christophwitzko/github-release-download/release"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var bindAddress string = ":5000"
var client *release.GithubClient

func init() {
	if ba := os.Getenv("BIND_ADDRESS"); ba != "" {
		bindAddress = ba
	}
	client = release.NewClient(os.Getenv("GITHUB_TOKEN"))
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

func getLatestDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	url, err := client.GetLatestDownloadUrl(ctx, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value)
	doRedirect(w, r, url, err)
}

func getMatchingDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	url, err := client.GetMatchingDownloadUrl(ctx, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value, ps[4].Value)
	doRedirect(w, r, url, err)
}

func getVersions(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.NotFound(w, r)
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintln(w, "https://get-release.xyz/:owner/:repo/:os/:arch{/:constraint}")
}

func main() {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/:owner/:repo", getVersions)
	router.GET("/:owner/:repo/:os/:arch", getLatestDownload)
	router.GET("/:owner/:repo/:os/:arch/:constraint", getMatchingDownload)
	server := http.Server{
		Addr:    bindAddress,
		Handler: logger(router),
	}
	go func() {
		log.Printf("starting server on port %s...", bindAddress)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
	defer server.Shutdown(nil)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	<-c
	log.Println("server stopped")
}
