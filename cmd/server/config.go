package main

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/energiegedrag-nl/needforheat-manual-server/parser"
	"golang.org/x/text/language"
)

const (
	SourceEnvDefault string = "./source"
)

var (
	ErrFallbackLangEnvNotSet = errors.New("environment variable NFH_FALLBACK_LANG was not set")
)

// Config contains the configuration for the server.
type Config struct {
	// Source is where the manuals are pulled from.
	//
	// Set by environment variable NFH_MANUAL_SOURCE.
	//
	// This can be a local directory or a git repository.
	// A local directory has to be a regular path (e.g. 'source' or './source').
	// A git repository has to start with https:// and end with .git (e.g. 'https://github.com/energiegedrag-nl/needforheat-presence-detector-firmware.git').
	//
	// The default branch is used, unless you set NFH_MANUAL_SOURCE_BRANCH to a branch name.
	//
	// OpenSource fetches the source: a git repository is cloned again on every call (start-up and each reload).
	OpenSource func() (fs.FS, error)

	// SourceName and SourceBranch describe the source, for the admin status.
	SourceName   string
	SourceBranch string

	// FallbackLanguage sets the fallback language for when
	// a client's Accept-Language header does not contain any available language.
	//
	// Set by environment variable NFH_FALLBACK_LANG.
	//
	// This must be a valid language code (e.g. nl-NL or en-US).
	FallbackLanguage language.Tag

	// AdminAddr is the address of the admin routes (GET /admin/status, POST /admin/reload), e.g. ":8081".
	//
	// Set by environment variable NFH_ADMIN_ADDR. Empty (the default) means no admin routes.
	// Use a port that is only reachable from inside the server, not the public one.
	AdminAddr string

	// AdminToken, when set, must be sent as "Authorization: Bearer <token>" to the admin routes.
	//
	// Set by environment variable NFH_ADMIN_TOKEN.
	AdminToken string
}

// Get configuration from environment variables.
// An error is returned if an environment variable is not set
// and there is no default setting for it or if a setting was invalid.
func getConfig() (*Config, error) {
	openSource, name, branch := parseSourceEnv()

	fallbackLang, err := parseFallbackLangEnv()
	if err != nil {
		return nil, err
	}

	return &Config{
		OpenSource:       openSource,
		SourceName:       name,
		SourceBranch:     branch,
		FallbackLanguage: fallbackLang,
		AdminAddr:        os.Getenv("NFH_ADMIN_ADDR"),
		AdminToken:       os.Getenv("NFH_ADMIN_TOKEN"),
	}, nil
}

func parseSourceEnv() (func() (fs.FS, error), string, string) {
	sourceEnv, ok := os.LookupEnv("NFH_MANUAL_SOURCE")
	if !ok {
		sourceEnv = SourceEnvDefault
	}

	sourceBranchEnv := os.Getenv("NFH_MANUAL_SOURCE_BRANCH")

	if strings.Contains(sourceEnv, "https://") {
		log.Println("using git repository", sourceEnv, "as manual source")
		if sourceBranchEnv != "" {
			log.Println("using branch", sourceBranchEnv)
		}
		return func() (fs.FS, error) { return parser.NewLabRepoSource(sourceEnv, sourceBranchEnv, nil) }, sourceEnv, sourceBranchEnv
	}
	log.Println("using local directory", sourceEnv, "as manual source")
	return func() (fs.FS, error) { return parser.NewLabDirSource(sourceEnv) }, sourceEnv, ""
}

func parseFallbackLangEnv() (language.Tag, error) {
	fallbackLangEnv, ok := os.LookupEnv("NFH_FALLBACK_LANG")
	if !ok {
		return language.Tag{}, ErrFallbackLangEnvNotSet
	}

	lang, err := language.Parse(fallbackLangEnv)
	if err != nil {
		return language.Tag{}, err
	}

	log.Println("using", fallbackLangEnv, "as fallback language")
	return lang, nil
}
