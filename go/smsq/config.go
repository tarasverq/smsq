package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"
)

type config struct {
	ListenAddress  string            `json:"listen_address"`  // the address to listen to incoming Telegram messages
	APIDomain      string            `json:"api_domain"`      // the domain name for API
	WebhookDomain  string            `json:"webhook_domain"`  // the domain name for the webhook
	BotToken       string            `json:"bot_token"`       // your Telegram bot token
	TimeoutSeconds int               `json:"timeout_seconds"` // HTTP timeout
	AdminID        int64             `json:"admin_id"`        // admin Telegram ID
	DBPath         string            `json:"db_path"`         // path to the database
	Debug          bool              `json:"debug"`           // debug mode
	PrivateKey     string            `json:"private_key"`     // private key
	ReceivedLimit  int               `json:"received_limit"`  // received messages limit
	DeliveredLimit int               `json:"delivered_limit"` // delivered messages limit
	Challenges     map[string]string `json:"challenges"`      // validation challenges

	privateKey *keyset.Handle
}

func readConfig(path string) *config {
	file, err := os.Open(filepath.Clean(path))
	checkErr(err)
	defer func() { checkErr(file.Close()) }()
	return parseConfig(file)
}

func parseConfig(r io.Reader) *config {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	cfg := &config{}
	err := decoder.Decode(cfg)
	parseEnv(cfg)
	checkErr(err)
	checkErr(checkConfig(cfg))
	privateKey, err := parsePrivateKey(cfg.PrivateKey)
	checkErr(err)
	cfg.privateKey = privateKey
	return cfg
}

func parseEnv(config *config) {
	envVar, ok := os.LookupEnv("DOMAIN")
	if ok == true {
		config.WebhookDomain = envVar
		config.APIDomain = envVar
	}

	envVar, ok = os.LookupEnv("BOT_TOKEN")
	if ok == true {
		config.BotToken = envVar
	}

	envVar, ok = os.LookupEnv("ADMIN_ID")
	if ok == true {
		id, _ := strconv.ParseInt(envVar, 10, 64)
		config.AdminID = id
	}
}

func checkConfig(cfg *config) error {
	if cfg.ListenAddress == "" {
		return errors.New("configure listen_address")
	}
	if cfg.BotToken == "" {
		return errors.New("configure bot_token")
	}
	if cfg.TimeoutSeconds == 0 {
		return errors.New("configure timeout_seconds")
	}
	if cfg.AdminID == 0 {
		return errors.New("configure admin_id")
	}
	if cfg.DBPath == "" {
		return errors.New("configure db_path")
	}
	if cfg.APIDomain == "" {
		return errors.New("configure api_domain")
	}
	if cfg.WebhookDomain == "" {
		return errors.New("configure webhook_domain")
	}
	if cfg.PrivateKey == "" {
		return errors.New("configure private_key")
	}
	if cfg.ReceivedLimit == 0 {
		return errors.New("configure received_limit")
	}
	if cfg.DeliveredLimit == 0 {
		return errors.New("configure delivered_limit")
	}
	return nil
}

func parsePrivateKey(file string) (*keyset.Handle, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	kh, err := insecurecleartextkeyset.Read(keyset.NewJSONReader(bytes.NewReader(data)))
	if err != nil {
		return nil, err
	}
	return kh, nil
}
