package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Masterminds/semver"
	"github.com/christophwitzko/github-release-download/internal/release"
	"github.com/julienschmidt/httprouter"
)

var bindAddress = ":5000"
var client *release.GithubClient

func init() {
	if ba := os.Getenv("BIND_ADDRESS"); ba != "" {
		bindAddress = ba
	} else if port := os.Getenv("PORT"); port != "" {
		bindAddress = ":" + port
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
	// w.Header().Set("Cache-Control", "public, max-age=7200")
	http.Redirect(w, r, url, 302)
}

func getLatestDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if len(ps) == 4 {
		if _, err := semver.NewConstraint(ps[3].Value); err != nil {
			url, err := client.GetLatestDownloadUrl(ctx, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value)
			doRedirect(w, r, url, err)
			return
		}
		url, err := client.GetMatchingDownloadUrl(ctx, "go-"+ps[0].Value, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value)
		doRedirect(w, r, url, err)
		return
	}
	url, err := client.GetLatestDownloadUrl(ctx, "go-"+ps[0].Value, ps[0].Value, ps[1].Value, ps[2].Value)
	doRedirect(w, r, url, err)
}

func getMatchingDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	url, err := client.GetMatchingDownloadUrl(ctx, ps[0].Value, ps[1].Value, ps[2].Value, ps[3].Value, ps[4].Value)
	doRedirect(w, r, url, err)
}

func getVersions(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if ps[0].Value == "_" && ps[1].Value == "go" {
		versions, err := client.GetGoVersions(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(w).Encode(versions)
		if err != nil {
			log.Println(err)
		}
		return
	}
	http.NotFound(w, r)
}

func usage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if ps[0].Value != "_usage" {
		http.NotFound(w, r)
		return
	}
	rl, _, err := client.Client.RateLimits(r.Context())
	if err != nil {
		log.Println(err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	err = json.NewEncoder(w).Encode(rl.Core)
	if err != nil {
		log.Println(err)
	}
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, _ = fmt.Fprintln(w, "https://get-release.xyz/:owner/:repo/:os/:arch{/:constraint}")
}

func main() {
	router := httprouter.New()
	router.GET("/", index)
	router.GET("/:owner", usage)
	router.GET("/:owner/:repo", getVersions)
	router.GET("/:owner/:repo/:os", getLatestDownload)
	router.GET("/:owner/:repo/:os/:arch", getLatestDownload)
	router.GET("/:owner/:repo/:os/:arch/:constraint", getMatchingDownload)
	server := http.Server{
		Addr:    bindAddress,
		Handler: logger(router),
	}
	go func() {
		log.Printf("starting server on port %s...", bindAddress)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	if err := server.Shutdown(context.TODO()); err != nil {
		log.Println(err)
	}
	log.Println("server stopped")
}
