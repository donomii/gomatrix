package gomatrix

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadStreamsEscapedMedia(t *testing.T) {
	finishBody := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.EscapedPath() != "/_matrix/media/r0/download/media.example/file%23one%3Ftwo" {
			t.Errorf("request path %q did not preserve the media identifier", req.URL.EscapedPath())
		}
		if req.Header.Get("Authorization") != "Bearer download-token" || req.URL.Query().Get("access_token") != "" {
			t.Error("download did not use bearer-header authentication without a query token")
		}
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''report%20one.txt")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-finishBody
		io.WriteString(w, "file contents")
	}))
	defer server.Close()
	defer close(finishBody)
	client, err := NewClient(server.URL, "", "download-token")
	if err != nil {
		t.Fatal(err)
	}
	filename, body, err := client.Download("mxc://media.example/file%23one%3Ftwo")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if filename != "report one.txt" {
		t.Fatalf("filename = %q; expected report one.txt", filename)
	}
}

func TestDownloadWithoutFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "file contents")
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	filename, body, err := client.Download("mxc://media.example/file")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	contents, err := io.ReadAll(body)
	if err != nil || string(contents) != "file contents" || filename != "" {
		t.Fatalf("filename %q, contents %q, error %v; expected empty filename and complete contents", filename, contents, err)
	}
}

func TestDownloadRejectsInvalidMediaURLs(t *testing.T) {
	client, err := NewClient("http://127.0.0.1:1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, mediaURL := range []string{
		"https://media.example/file", "mxc:///file", "mxc://user@media.example/file",
		"mxc://media.example/", "mxc://media.example/..", "mxc://media.example/a/b",
		"mxc://media.example/a%2Fb", "mxc://media.example/file?query", "mxc://media.example/file#fragment",
		"mxc://media.example/file?", "mxc://media.example/file#",
		"mxc://media.example/%",
	} {
		_, body, err := client.Download(mediaURL)
		if err == nil || body != nil || !strings.Contains(err.Error(), "invalid MXC URL") {
			t.Errorf("Download(%q) body %v, error %v; expected invalid MXC URL rejection", mediaURL, body, err)
		}
	}
}

func TestDownloadReportsUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, `{"errcode":"M_NOT_FOUND","error":"missing media"}`, http.StatusNotFound)
	}))
	defer server.Close()
	client, err := NewClient(server.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := client.Download("mxc://media.example/missing")
	upstream, ok := err.(HTTPError)
	if !ok || upstream.Code != http.StatusNotFound || body != nil || !strings.Contains(upstream.Message, "M_NOT_FOUND") {
		t.Fatalf("body %v, error %v; expected HTTP 404 with upstream error code", body, err)
	}
}
