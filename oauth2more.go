package oauth2more

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/grokify/oauth2more/scim"
	"github.com/grokify/simplego/net/httputilmore"
	hum "github.com/grokify/simplego/net/httputilmore"
	"github.com/grokify/simplego/time/timeutil"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/facebook"
)

const (
	VERSION                    = "0.2.0"
	PATH                       = "github.com/grokify/oauth2more"
	TokenBasic                 = "Basic"
	TokenBearer                = "Bearer"
	GrantTypeAuthorizationCode = "code"
)

type AuthorizationType int

const (
	Anonymous AuthorizationType = iota
	Basic
	Bearer
	Digest
	NTLM
	Negotiate
	OAuth
)

var authorizationTypes = [...]string{
	"Anonymous",
	"Basic",
	"Bearer",
	"Digest",
	"NTLM",
	"Negotiate",
	"OAuth",
}

// String returns the English name of the authorizationTypes ("Basic", "Bearer", ...).
func (a AuthorizationType) String() string {
	if Basic <= a && a <= OAuth {
		return authorizationTypes[a]
	}
	buf := make([]byte, 20)
	n := fmtInt(buf, uint64(a))
	return "%!AuthorizationType(" + string(buf[n:]) + ")"
}

// fmtInt formats v into the tail of buf.
// It returns the index where the output begins.
func fmtInt(buf []byte, v uint64) int {
	w := len(buf)
	if v == 0 {
		w--
		buf[w] = '0'
	} else {
		for v > 0 {
			w--
			buf[w] = byte(v%10) + '0'
			v /= 10
		}
	}
	return w
}

func PathVersion() string {
	return fmt.Sprintf("%v-v%v", PATH, VERSION)
}

type ServiceType int

const (
	Google ServiceType = iota
	Facebook
	RingCentral
	Aha
)

// ApplicationCredentials represents information for an app.
type ApplicationCredentials struct {
	ServerURL    string
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint
}

type AppCredentials struct {
	Service      string   `json:"service,omitempty"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURIs []string `json:"redirect_uris"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	Scopes       []string `json:"scopes"`
}

func (ac *AppCredentials) Defaultify() {
	switch ac.Service {
	case "facebook":
		if len(ac.AuthURI) == 0 || len(ac.TokenURI) == 0 {
			endpoint := facebook.Endpoint
			if len(ac.AuthURI) == 0 {
				ac.AuthURI = endpoint.AuthURL
			}
			if len(ac.TokenURI) == 0 {
				ac.TokenURI = endpoint.TokenURL
			}
		}
	}
}

type AppCredentialsWrapper struct {
	Web       *AppCredentials `json:"web"`
	Installed *AppCredentials `json:"installed"`
}

func (w *AppCredentialsWrapper) Config() (*oauth2.Config, error) {
	var c *AppCredentials
	if w.Web != nil {
		c = w.Web
	} else if w.Installed != nil {
		c = w.Installed
	} else {
		return nil, errors.New("No OAuth2 config info")
	}
	c.Defaultify()
	return c.Config(), nil
}

func NewAppCredentialsWrapperFromBytes(data []byte) (AppCredentialsWrapper, error) {
	var acw AppCredentialsWrapper
	err := json.Unmarshal(data, &acw)
	if err != nil {
		panic(err)
	}
	return acw, err
}

func (c *AppCredentials) Config() *oauth2.Config {
	cfg := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       c.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  c.AuthURI,
			TokenURL: c.TokenURI}}

	if len(c.RedirectURIs) > 0 {
		cfg.RedirectURL = c.RedirectURIs[0]
	}
	return cfg
}

// UserCredentials represents a user's credentials.
type UserCredentials struct {
	Username string
	Password string
}

type OAuth2Util interface {
	SetClient(*http.Client)
	GetSCIMUser() (scim.User, error)
}

func NewClientPassword(conf oauth2.Config, ctx context.Context, username, password string) (*http.Client, error) {
	token, err := BasicAuthToken(username, password)
	if err != nil {
		return nil, err
	}
	return conf.Client(ctx, token), nil
}

func NewClientPasswordConf(conf oauth2.Config, username, password string) (*http.Client, error) {
	token, err := conf.PasswordCredentialsToken(oauth2.NoContext, username, password)
	if err != nil {
		return &http.Client{}, err
	}

	return conf.Client(oauth2.NoContext, token), nil
}

func NewClientAuthCode(conf oauth2.Config, authCode string) (*http.Client, error) {
	token, err := conf.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		return &http.Client{}, err
	}
	return conf.Client(oauth2.NoContext, token), nil
}

func NewClientTokenJSON(ctx context.Context, tokenJSON []byte) (*http.Client, error) {
	token := &oauth2.Token{}
	err := json.Unmarshal(tokenJSON, token)
	if err != nil {
		return nil, err
	}

	oAuthConfig := &oauth2.Config{}

	return oAuthConfig.Client(ctx, token), nil
}

func NewClientHeaders(headersMap map[string]string, tlsInsecureSkipVerify bool) *http.Client {
	client := &http.Client{}
	header := httputilmore.NewHeadersMSS(headersMap)

	if tlsInsecureSkipVerify {
		client = ClientTLSInsecureSkipVerify(client)
	}

	client.Transport = hum.TransportWithHeaders{
		Header:    header,
		Transport: client.Transport}

	return client
}

func NewClientToken(tokenType, tokenValue string, tlsInsecureSkipVerify bool) *http.Client {
	client := &http.Client{}

	header := http.Header{}
	header.Add(httputilmore.HeaderAuthorization, tokenType+" "+tokenValue)

	if tlsInsecureSkipVerify {
		client = ClientTLSInsecureSkipVerify(client)
	}

	client.Transport = hum.TransportWithHeaders{
		Header:    header,
		Transport: client.Transport}

	return client
}

func NewClientTokenBase64Encode(tokenType, tokenValue string, tlsInsecureSkipVerify bool) *http.Client {
	return NewClientToken(
		tokenType,
		base64.StdEncoding.EncodeToString([]byte(tokenValue)),
		tlsInsecureSkipVerify)
}

// NewClientAuthzTokenSimple returns a *http.Client given a token type and token string.
func NewClientAuthzTokenSimple(tokenType, accessToken string) *http.Client {
	token := &oauth2.Token{
		AccessToken: strings.TrimSpace(accessToken),
		TokenType:   strings.TrimSpace(tokenType),
		Expiry:      timeutil.TimeZeroRFC3339()}

	oAuthConfig := &oauth2.Config{}

	return oAuthConfig.Client(oauth2.NoContext, token)
}

func NewClientTokenOAuth2(token *oauth2.Token) *http.Client {
	oAuthConfig := &oauth2.Config{}
	return oAuthConfig.Client(oauth2.NoContext, token)
}

func NewClientBearerTokenSimpleOrJson(ctx context.Context, tokenOrJson []byte) (*http.Client, error) {
	tokenOrJsonString := strings.TrimSpace(string(tokenOrJson))
	if len(tokenOrJsonString) == 0 {
		return nil, fmt.Errorf("No token [%v]", string(tokenOrJson))
	} else if strings.Index(tokenOrJsonString, "{") == 0 {
		return NewClientTokenJSON(ctx, tokenOrJson)
	} else {
		return NewClientAuthzTokenSimple(TokenBearer, tokenOrJsonString), nil
	}
}

func NewClientTLSToken(ctx context.Context, tlsConfig *tls.Config, token *oauth2.Token) *http.Client {
	tlsClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig}}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, tlsClient)

	cfg := &oauth2.Config{}

	return cfg.Client(ctx, token)
}

func ClientTLSInsecureSkipVerify(client *http.Client) *http.Client {
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true}}
	return client
}
