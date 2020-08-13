package config

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/multierr"
)

const (
	envPrefix = "msgbot"
)

// Application config part
type Application struct {
	Env           string
	Addr          string
	Port          string
	Secret        string
	LogLevel      string
	TelegramToken string
	LogFormat     string
}

func (a *Application) IsProduction() bool {
	return a.Env == "production"
}

func (a *Application) validate() error {
	//if a.Addr == "" {
	//	return errors.New("empty address provided for an http server to start on")
	//}
	//if a.Secret == "" {
	//	return errors.New("empty secret provided")
	//}
	return nil
}

// Database config part
type Database struct {
	Host     string
	User     string
	Password string
	Port     int
	Db       string
}

func (d *Database) validate() error {
	if d.Host == "" {
		return errors.New("empty db host provided")
	}
	if d.Port == 0 {
		return errors.New("empty db port provided")
	}
	if d.User == "" {
		return errors.New("empty db user provided")
	}
	if d.Password == "" {
		return errors.New("empty db password provided")
	}
	if d.Db == "" {
		return errors.New("empty db name provided")
	}
	return nil
}

// Broker config part
type Broker struct {
	UserURL         string
	UserCredits     string
	ExchangePrefix  string
	ExchangePostfix string
}

func (b *Broker) validate() error {
	if b.UserURL == "" {
		return errors.New("empty broker url provided")
	}
	if b.UserCredits == "" {
		return errors.New("empty broker credentials provided")
	}
	return nil
}

//DialogFlow config part
type DialogFlow struct {
	ProjectID    string
	JSONFilePath string
	Lang         string
	Timezone     string
}

func (b *DialogFlow) validate() error {
	if b.ProjectID == "" {
		return errors.New("empty DialogFlow project ID")
	}
	if b.JSONFilePath == "" {
		return errors.New("empty Google Service Account JSON file path")
	}
	return nil
}

//Setting config for use inside app
type Settings struct {
	CRMURL      string
	ClientURL   string
	ChatApi     ChatApi
	Preferences Preferences
}

func (st *Settings) validate() error {
	return nil
}

type Preferences struct {
	DefaultServiceUUID string
}

type ChatApi struct {
	URL1   string
	Token1 string
	URL2   string
	Token2 string
	URL3   string
	Token3 string
}

type Config struct {
	Application Application
	Database    Database
	Broker      Broker
	DialogFlow  DialogFlow
	Settings    Settings
	ChatApi     ChatApi
}

func (c *Config) validate() error {
	return multierr.Combine(
		c.Application.validate(),
		c.Database.validate(),
		c.Broker.validate(),
		c.DialogFlow.validate(),
		c.Settings.validate(),
	)
}

// Parse will parse the configuration from the environment variables and a file with the specified path.
// Environment variables have more priority than ones specified in the file.
func Parse(filepath string) (*Config, error) {
	setDefaults()

	// Parse the file
	viper.SetConfigFile(filepath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "failed to read the config file")
	}

	bindEnvVars() // remember to parse the environment variables

	// Unmarshal the config
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the configuration")
	}

	// Validate the provided configuration
	if err := cfg.validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate the config")
	}
	return &cfg, nil
}

func (c *Config) Print() {
	if c.Application.IsProduction() {
		return
	}
	inspected := *c // get a copy of an actual object
	// Hide sensitive data
	inspected.Application.Secret = ""
	inspected.Database.User = ""
	inspected.Database.Password = ""
	inspected.Broker.UserCredits = ""
	fmt.Printf("%+v\n", inspected)
}

// TODO: set the default values here
func setDefaults() {
	viper.SetDefault("Application.env", "production")
	viper.SetDefault("Application.loglevel", "debug")
	viper.SetDefault("Application.port", "8080")
	viper.SetDefault("Application.telegramtoken", "tokenhere")
	viper.SetDefault("Application.LogFormat", "text")

	viper.SetDefault("Database.Host", "")
	viper.SetDefault("Database.Port", 0)
	viper.SetDefault("Database.User", "")
	viper.SetDefault("Database.Password", "")
	viper.SetDefault("Database.Db", "")

	viper.SetDefault("Broker.UserURL", "")
	viper.SetDefault("Broker.UserCredits", "")

	viper.SetDefault("Chatapi.URL1", "url here")
	viper.SetDefault("Chatapi.Token1", "token here")
	viper.SetDefault("Chatapi.URL2", "url here")
	viper.SetDefault("Chatapi.Token2", "token here")
	viper.SetDefault("Chatapi.URL3", "url here")
	viper.SetDefault("Chatapi.Token3", "token here")

	viper.SetDefault("Dialogflow.ProjectID", "")
	viper.SetDefault("Dialogflow.JSONFilePath", "")
	viper.SetDefault("Dialogflow.Lang", "ru")
	viper.SetDefault("Dialogflow.Timezone", "Europe/Moscow")

	viper.SetDefault("Settings.CRMURL", "http://faem-backend-crm.faem.svc.cluster.local/api/v2")
	viper.SetDefault("Settings.ClientURL", "http://faem-backend-client.faem.svc.cluster.local/api/v2")
	viper.SetDefault("Settings.Preferences.DefaultServiceUUID", "b65d4d24-6df0-4630-a87e-e296447b04c5")
}

func bindEnvVars() {
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}
