package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/browser"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

func openConsole(stdout bool, duration time.Duration, profile string) {
	var sess *session.Session
	if profile != "" {
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			Profile: profile,
		}))
	} else {
		sess = session.Must(session.NewSession())
	}

	creds, err := sess.Config.Credentials.Get()
	if err != nil {
		panic("Failed to get AWS credentials: " + err.Error())
	}

	urlCredentials := map[string]string{
		"sessionId":    creds.AccessKeyID,
		"sessionKey":   creds.SecretAccessKey,
		"sessionToken": creds.SessionToken,
	}

	credentialsJSON, _ := json.Marshal(urlCredentials)
	requestParameters := fmt.Sprintf("?Action=getSigninToken&DurationSeconds=%d&Session=%s", duration/time.Second, url.QueryEscape(string(credentialsJSON)))
	requestURL := "https://signin.aws.amazon.com/federation" + requestParameters

	print(requestURL)
	response, err := http.Get(requestURL)
	if err != nil {
		log.Fatalf("Failed to get Signin Token from AWS: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Failed to decode error message: %v", err)
		}
		log.Fatalf("Failed to get federation token. HTTP %d: %s", response.StatusCode, string(bodyBytes))
	}

	var tokenResponse struct {
		SigninToken string `json:"SigninToken"`
	}
	err = json.NewDecoder(response.Body).Decode(&tokenResponse)
	if err != nil {
		log.Fatalf("Failed to decode federation token response: %v", err)
	}

	requestParameters = "?Action=login&Destination=" + url.QueryEscape("https://console.aws.amazon.com/") +
		"&SigninToken=" + tokenResponse.SigninToken + "&Issuer=" + url.QueryEscape("https://example.com")
	requestURL = "https://us-east-1.signin.aws.amazon.com/federation" + requestParameters

	federateURL := "https://us-east-1.signin.aws.amazon.com/oauth?Action=logout&redirect_uri=" + url.QueryEscape(requestURL)

	if stdout {
		fmt.Println(federateURL)
	} else {
		err = browser.OpenURL(federateURL)
		if err != nil {
			log.Fatalf("Failed to open browser: %v", err)
		}
	}
}

func main() {
	profileFlag := flag.String("profile", "", "AWS profile to use, if not specified the default credentials will be used")
	stdoutFlag := flag.Bool("stdout", false, "don't open the browser, but print the sign in URL to stdout")
	durationFlag := flag.Duration("session-duration", 2*time.Hour, "the max duration of the console session, defaults to 2h")
	flag.Parse()
	openConsole(*stdoutFlag, *durationFlag, *profileFlag)
}
