package google

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/grokify/simplego/type/stringsutil"
	"github.com/pkg/errors"
	json "github.com/pquerna/ffjson/ffjson"
	"golang.org/x/oauth2"
	o2g "golang.org/x/oauth2/google"
)

func ConfigFromFile(file string, scopes []string) (*oauth2.Config, error) {
	b, err := ioutil.ReadFile(file) // Google client_secret.json
	if err != nil {
		return &oauth2.Config{},
			errors.Wrap(err, fmt.Sprintf("Unable to read client secret file: %v", err))
	}
	return o2g.ConfigFromJSON(b, scopes...)
}

func ConfigFromEnv(envVar string, scopes []string) (*oauth2.Config, error) {
	envVar = strings.TrimSpace(envVar)
	if len(envVar) == 0 {
		envVar = EnvGoogleAppCredentials
	}
	if len(scopes) == 0 {
		scopesString := os.Getenv(EnvGoogleAppScopes)
		scopes = stringsutil.SplitCondenseSpace(scopesString, ",")
	}
	return o2g.ConfigFromJSON([]byte(os.Getenv(envVar)), scopes...)
}

// ConfigFromBytes returns an *oauth2.Config given a byte array
// containing the Google client_secret.json data.
func ConfigFromBytes(configJson []byte, scopes []string) (*oauth2.Config, error) {
	if len(strings.TrimSpace(string(configJson))) == 0 {
		return nil, errors.Wrap(errors.New("No Credentials Provided"), "oauth2more/google.ConfigFromBytes()")
	}

	if len(scopes) == 0 {
		cc := CredentialsContainer{}
		err := json.Unmarshal(configJson, &cc)
		if err != nil {
			return nil, errors.Wrap(err, "ConfigFromBytes")
		}
		if len(cc.Scopes) > 0 {
			scopes = append(scopes, cc.Scopes...)
		}
	}

	return o2g.ConfigFromJSON(configJson, scopes...)
}
