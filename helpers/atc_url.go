package helpers

import (
	"net/http"
	"os"
)

const defaultATCURL = "http://127.0.0.1:8080"

var storedATCURL string

func AtcURL() string {
	if storedATCURL != "" {
		return storedATCURL
	}

	atcURL := os.Getenv("ATC_URL")
	if atcURL == "" {
		response, err := http.Get(defaultATCURL + "/api/v1/info")
		if err != nil || response.StatusCode != http.StatusOK {
			panic("must set $ATC_URL")
		}

		atcURL = defaultATCURL
	}

	storedATCURL = atcURL

	return atcURL
}

func AtcUsername() string {
	username := os.Getenv("ATC_USERNAME")

	if username == "" {
		panic("must set $ATC_USERNAME")
	}

	return username
}

func AtcPassword() string {
	password := os.Getenv("ATC_PASSWORD")

	if password == "" {
		panic("must set $ATC_PASSWORD")
	}

	return password
}
