package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/asiffer/puzzle"
	"github.com/asiffer/puzzle/pflagset"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
)

const ENV_PREFIX = "XDPROBE_"

var (
	nicName  string = "lo"
	addr     string = ":8080"
	geoipdb  string
	tick     time.Duration = time.Second
	username string        = "admin"
	password string        = "password"
	insecure bool          = false
)

var log zerolog.Logger

var k = puzzle.NewConfig()

func init() {
	err := errors.Join(
		puzzle.DefineVar(
			k,
			"interface",
			&nicName,
			puzzle.WithShortFlagName("i"),
			puzzle.WithDescription("Network interface to listens to"),
			puzzle.WithEnvName(ENV_PREFIX+"NIC"),
		),
		puzzle.DefineVar(
			k,
			"addr",
			&addr,
			puzzle.WithShortFlagName("a"),
			puzzle.WithDescription("HTTP server address"),
			puzzle.WithEnvName(ENV_PREFIX+"ADDR"),
		),
		puzzle.DefineVar(
			k,
			"geoip",
			&geoipdb,
			puzzle.WithShortFlagName("g"),
			puzzle.WithDescription("GeoIP database file"),
			puzzle.WithEnvName(ENV_PREFIX+"GEOIP"),
		),
		puzzle.DefineVar(
			k,
			"tick",
			&tick,
			puzzle.WithShortFlagName("t"),
			puzzle.WithDescription("Tick duration"),
			puzzle.WithEnvName(ENV_PREFIX+"TICK"),
		),
		puzzle.DefineVar(
			k,
			"username",
			&username,
			puzzle.WithShortFlagName("u"),
			puzzle.WithDescription("Username for authentication"),
			puzzle.WithEnvName(ENV_PREFIX+"USERNAME"),
		),
		puzzle.DefineVar(
			k,
			"password",
			&password,
			puzzle.WithShortFlagName("p"),
			puzzle.WithDescription("Password for authentication"),
			puzzle.WithEnvName(ENV_PREFIX+"PASSWORD"),
		),
		puzzle.DefineVar(
			k,
			"insecure",
			&insecure,
			puzzle.WithShortFlagName("k"),
			puzzle.WithDescription("Disable authentication"),
			puzzle.WithEnvName(ENV_PREFIX+"INSECURE"),
		),
	)
	if err != nil {
		panic(err)
	}

}

func init() {
	// logging
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen}
	output.FormatLevel = func(i interface{}) string {
		s := strings.ToUpper(fmt.Sprintf("%s", i))
		switch s {
		case "DEBUG":
			return fmt.Sprintf("\033[36m%-6s\033[0m", s) // cyan
		case "INFO":
			return fmt.Sprintf("\033[32m%-6s\033[0m", s) // green
		case "WARN":
			return fmt.Sprintf("\033[33m%-6s\033[0m", s) // yellow
		case "ERROR":
			return fmt.Sprintf("\033[31m%-6s\033[0m", s) // red
		default:
			return fmt.Sprintf("\033[37m%-6s\033[0m", s) // white
		}
	}
	output.FormatMessage = func(i interface{}) string {
		return fmt.Sprintf("%s", i)
	}
	output.FormatFieldName = func(i interface{}) string {
		return fmt.Sprintf("\033[2m%s=\033[0m", i)
	}
	output.FormatFieldValue = func(i interface{}) string {
		return fmt.Sprintf("\033[2m%s\033[0m", i)
	}

	log = zerolog.New(output).With().Timestamp().Logger()
}

func ReadEnvAndFlags() error {
	if err := puzzle.ReadEnv(k); err != nil {
		return err
	}
	fs, err := pflagset.Build(k, "xdprobe", pflag.ContinueOnError)
	if err != nil {
		return err
	}
	return fs.Parse(os.Args[1:])
}

const BANNER = "XDProbe - XDP-based ingress network traffic monitor\n\033[3mWho the hell hits my server?\033[0m\n"
