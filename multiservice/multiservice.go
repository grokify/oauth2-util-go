package multiservice

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/grokify/simplego/os/osutil"

	"github.com/grokify/oauth2more"
	"github.com/grokify/oauth2more/aha"
	"github.com/grokify/oauth2more/facebook"
	"github.com/grokify/oauth2more/google"
	"github.com/grokify/oauth2more/multiservice/tokens"
	"github.com/grokify/oauth2more/multiservice/tokens/tokensetmemory"
	"github.com/grokify/oauth2more/ringcentral"
)

type OAuth2Manager struct {
	ConfigSet *ConfigMoreSet
	TokenSet  tokens.TokenSet
}

func NewOAuth2Manager() *OAuth2Manager {
	return &OAuth2Manager{
		ConfigSet: NewConfigMoreSet(),
		TokenSet:  tokensetmemory.NewTokenSet(),
	}
}

func (cb *OAuth2Manager) GetClient(ctx context.Context, serviceKey string) (*http.Client, error) {
	if cb.ConfigSet == nil {
		return nil, fmt.Errorf("OAuth2Manager.ConfigSet == nil")
	}
	if cb.TokenSet == nil {
		return nil, fmt.Errorf("OAuth2Manager.TokenSet == nil")
	}

	cfgMore, err := cb.ConfigSet.Get(serviceKey)
	if err != nil {
		return nil, err
	}
	cfg := cfgMore.Config()
	tok, err := cb.TokenSet.GetToken(serviceKey)
	if err != nil {
		return nil, err
	}
	return cfg.Client(ctx, tok), nil
}

type AppURLs struct {
	AuthURL     string `json:"authUrl,omitempty"`
	TokenURL    string `json:"tokenUrl,omitempty"`
	RedirectURL string `json:"redirectUrl,omitempty"`
}

// EnvOAuth2ConfigMap returns a map of *oauth2.Config from environment
// variables in AppCredentialsWrapper format.
func EnvOAuth2ConfigMap(env []osutil.EnvVar, prefix string) (*ConfigMoreSet, error) {
	cfgs := NewConfigMoreSet()

	rx, err := regexp.Compile(fmt.Sprintf(`^%v(.*)`, prefix))
	if err != nil {
		return nil, err
	}

	for _, pair := range env {
		key := strings.TrimSpace(pair.Key)
		val := pair.Value
		m := rx.FindStringSubmatch(key)
		if len(m) > 0 {
			key := m[1]
			err := cfgs.AddConfigMoreJson(key, []byte(val))
			if err != nil {
				return nil, err
			}
		}
	}
	return cfgs, nil
}

func NewClientUtilForProviderType(providerType OAuth2Provider) (oauth2more.OAuth2Util, error) {
	switch providerType {
	case Aha:
		return &aha.ClientUtil{}, nil
	case Facebook:
		return &facebook.ClientUtil{}, nil
	case Google:
		return &google.ClientUtil{}, nil
	case RingCentral:
		return &ringcentral.ClientUtil{}, nil
	default:
		return nil, fmt.Errorf("Cannot find ClientUtil for provider type [%s]", providerType)
	}
}

func NewClientUtilForProviderTypeString(providerTypeString string) (oauth2more.OAuth2Util, error) {
	providerType, err := ProviderStringToConst(providerTypeString)
	if err != nil {
		return &ringcentral.ClientUtil{}, nil
	}
	return NewClientUtilForProviderType(providerType)
}
