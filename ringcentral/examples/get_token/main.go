package main

import (
	"fmt"

	"github.com/grokify/oauth2more/ringcentral"
	"github.com/grokify/simplego/fmt/fmtutil"
	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"
)

type Options struct {
	CredsPath string `short:"c" long:"credspath" description:"Environment File Path"`
	Account   string `short:"a" long:"account" description:"Environment Variable Name"`
	Token     string `short:"t" long:"token" description:"Token"`
}

func main() {
	opts := Options{}
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatal().Err(err)
	}
	fmtutil.PrintJSON(opts)

	cset, err := ringcentral.ReadFileCredentialsSet(opts.CredsPath)
	if err != nil {
		log.Fatal().Err(err).
			Str("credentials_filepath", opts.CredsPath).
			Msg("cannot read credentials file")
	}

	credentials, err := cset.Get(opts.Account)
	if err != nil {
		log.Fatal().Err(err).
			Str("credentials_account", opts.Account).
			Msg("cannot find credentials account")
	}

	token, err := credentials.NewToken()
	if err != nil {
		log.Fatal().Err(err)
	}

	token.Expiry = token.Expiry.UTC()

	fmtutil.PrintJSON(token)

	fmt.Println("DONE")
}
