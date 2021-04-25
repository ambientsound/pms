package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ambientsound/pms/console"
	"github.com/ambientsound/pms/pms"
	"github.com/ambientsound/pms/version"
	"github.com/ambientsound/pms/xdg"

	"github.com/jessevdk/go-flags"
)

var buildVersion = "undefined"

type cliOptions struct {
	Version     bool   `short:"v" long:"version" description:"Print program version"`
	Debug       string `short:"d" long:"debug" description:"Write debugging info to file"`
	MpdHost     string `short:"h" long:"host" description:"MPD host" default-mask:"MPD_HOST environment variable or localhost"`
	MpdPort     string `short:"p" long:"port" description:"MPD port" default-mask:"MPD_PORT environment variable or 6600"`
	MpdPassword string `long:"password" description:"MPD password"`
}

// mpdEnvironmentVariables reads the host, port, and password parameters to MPD
// from the MPD_HOST and MPD_PORT environment variables, then returns them to the
// user. In case there is a password in MPD_HOST, it is parsed out.
func mpdEnvironmentVariables(host, port, password string) (string, string, string) {
	if len(host) == 0 {
		env, ok := os.LookupEnv("MPD_HOST")
		if ok {
			// If MPD_HOST is found, try to parse out the password.
			tokens := strings.SplitN(env, "@", 2)
			switch len(tokens) {
			case 0:
				// Empty string, default to localhost
				host = "localhost"
			case 1:
				// No '@' sign, use host as-is
				host = env
			case 2:
				// password@host
				host = tokens[1]
				password = tokens[0]
			}
		} else {
			host = "localhost"
		}
	}
	if len(port) == 0 {
		env, ok := os.LookupEnv("MPD_PORT")
		if ok {
			port = env
		} else {
			port = "6600"
		}
	}

	return host, port, password
}

func main() {
	var opts cliOptions

	version.SetVersion(buildVersion)
	fmt.Printf("%s %s\n", version.LongName(), version.Version())

	remainder, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}
	if len(remainder) > 0 {
		trailing := strings.Join(remainder, " ")
		fmt.Printf("error: trailing characters: %s\n", trailing)
		os.Exit(1)
	}

	if len(opts.Debug) > 0 {
		err := console.Open(opts.Debug)
		if err != nil {
			fmt.Printf("Error while opening log file: %s", err)
			os.Exit(1)
		}
	}

	if opts.Version {
		os.Exit(0)
	}

	console.Log("Starting Practical Music Search.")

	p, err := pms.New()
	if err != nil {
		fmt.Printf("Error starting up: %s", err)
		os.Exit(1)
	}

	defer func() {
		p.QuitSignal <- 0
	}()

	// Source default configuration.
	p.Message("Applying default configuration.")
	if err := p.SourceDefaultConfig(); err != nil {
		panic(fmt.Sprintf("BUG in default config: %s\n", err))
	}

	// Source configuration files from all XDG standard directories.
	configDirs := xdg.ConfigDirectories()
	for _, dir := range configDirs {
		path := path.Join(dir, "pms.conf")
		p.Message("Reading configuration file '%s'.", path)
		err = p.SourceConfigFile(path)
		if err != nil {
			p.Error("Error while reading configuration file '%s': %s", path, err)
		}
	}

	// If host, port and password is not set by the command-line flags, try to
	// read them from the environment variables.
	host, port, password := mpdEnvironmentVariables(opts.MpdHost, opts.MpdPort, opts.MpdPassword)

	// Set up the self-healing connection.
	p.Connection = pms.NewConnection(p.EventMessage)
	p.Connection.Open(host, port, password)
	go p.Connection.Run()

	// Every second counts
	go p.RunTicker()

	p.Main()
	p.Wait()

	console.Log("Exiting normally.")
}
