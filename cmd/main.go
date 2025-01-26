package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/urfave/cli/v2"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"sort"
)

var (
	version       = "DEV"
	date          = "unknown"
	versionString = fmt.Sprintf("%s (built %s)", version, date)
)

type SignInToken struct {
	SignInToken string `json:"SigninToken"`
}

func main() {
	if buildInfo, available := debug.ReadBuildInfo(); available {
		versionString = fmt.Sprintf("%s (built %s with %s)", version, date, buildInfo.GoVersion)
	}
	app := &cli.App{
		Name:    "go-aws-console",
		Usage:   "CLI tool to open AWS Console URLs",
		Version: versionString,
		Suggest: true,
		Authors: []*cli.Author{
			{
				Name:  "Cyb3r-Jak3",
				Email: "git@cyberjake.xyz",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "profile",
				Usage:   "Specify the AWS profile to use",
				EnvVars: []string{"AWS_PROFILE"},
			},
			//&cli.BoolFlag{
			//	Name:  "headless",
			//	Usage: "Run in headless mode (no browser launch)",
			//},
		},
		Action: run,
	}
	sort.Sort(cli.FlagsByName(app.Flags))
	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("Error running app: %s\n", err)
		os.Exit(1)
	}
}

func run(ctx *cli.Context) error {
	awsConfig, err := config.LoadDefaultConfig(ctx.Context, config.WithSharedConfigProfile(ctx.String("profile")))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	awsCreds, err := awsConfig.Credentials.Retrieve(ctx.Context)
	if err != nil {
		return fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}
	signinToken, err := getSigninToken(awsCreds)
	if err != nil {
		return fmt.Errorf("failed to get signin token: %w", err)
	}
	consoleRequestURL := fmt.Sprintf("https://signin.aws.amazon.com/federation?Action=login&Issuer=go-aws-console&Destination=https://console.aws.amazon.com/&SigninToken=%s", signinToken)
	fmt.Println(consoleRequestURL)
	return nil
}

func getSigninToken(credentials aws.Credentials) (string, error) {
	newRequestURL, err := url.Parse("https://signin.aws.amazon.com/federation")
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	urlValues := url.Values{}
	urlValues.Set("Action", "getSigninToken")
	urlValues.Set("DurationSeconds", "3600")
	urlValues.Set("Session", fmt.Sprintf(`{"sessionId":"%s","sessionKey":"%s","sessionToken":"%s"}`, credentials.AccessKeyID, credentials.SecretAccessKey, credentials.SessionToken))
	newRequestURL.RawQuery = urlValues.Encode()
	tokenResponse, err := http.Get(newRequestURL.String())
	if err != nil {
		return "", fmt.Errorf("failed to get signin token: %w", err)
	}
	if tokenResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", tokenResponse.StatusCode)
	}
	var SignInToken SignInToken
	err = json.NewDecoder(tokenResponse.Body).Decode(&SignInToken)
	if err != nil {
		return "", fmt.Errorf("failed to decode response body: %w", err)
	}
	return SignInToken.SignInToken, nil
}
