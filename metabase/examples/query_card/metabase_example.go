package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/grokify/oauth2more/metabase"
	hum "github.com/grokify/simplego/net/httputilmore"
	"github.com/grokify/simplego/net/urlutil"
	"github.com/joho/godotenv"
)

func loadEnv() error {
	if len(os.Getenv("ENV_PATH")) > 0 {
		return godotenv.Load(os.Getenv("ENV_PATH"))
	}
	return godotenv.Load()
}

func main() {
	err := loadEnv()
	if err != nil {
		log.Fatal(err)
	}

	cardId := 1

	metabase.TLSInsecureSkipVerify = true

	baseUrl := os.Getenv("METABASE_BASE_URL")

	client, _, err := metabase.NewClientPassword(
		baseUrl,
		os.Getenv("METABASE_USERNAME"),
		os.Getenv("METABASE_PASSWORD"),
		metabase.TLSInsecureSkipVerify)
	if err != nil {
		log.Fatal(err)
	}

	userUrl := urlutil.JoinAbsolute(baseUrl, "api/user/current")

	req, err := http.NewRequest("GET", userUrl, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	hum.PrintResponse(resp, true)

	cardUrl := fmt.Sprintf("api/card/%v/query/%s", cardId, "json")
	cardUrl = urlutil.JoinAbsolute(baseUrl, cardUrl)

	fmt.Println(cardUrl)

	req, err = http.NewRequest(http.MethodPost, cardUrl, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	hum.PrintResponse(resp, true)

	fmt.Println("DONE")
}
