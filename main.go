package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/cloudfoundry/cli/plugin"
)

/*
*	This is the struct implementing the interface defined by the core CLI. It can
*	be found at  "github.com/cloudfoundry/cli/plugin/plugin.go"
*
 */
type TreelineCli struct{}

/*
*	This function must be implemented by any plugin because it is part of the
*	plugin interface defined by the core CLI.
*
*	Run(....) is the entry point when the core CLI is invoking a command defined
*	by a plugin. The first parameter, plugin.CliConnection, is a struct that can
*	be used to invoke cli commands. The second paramter, args, is a slice of
*	strings. args[0] will be the name of the command, and will be followed by
*	any additional arguments a cli user typed in.
*
*	Any error handling should be handled with the plugin itself (this means printing
*	user facing errors). The CLI will exit 0 if the plugin exits 0 and will exit
*	1 should the plugin exits nonzero.
 */
func (c *TreelineCli) Run(cliConnection plugin.CliConnection, args []string) {
	// Ensure that we called the command treeline
	if args[0] == "treeline" {
		_, err := exec.LookPath("treeline")
		if err != nil {
			fmt.Println("Please install treeline using 'npm install -g treeline'")
			os.Exit(1)
		}

		if args[1] == "config-pws" {
			writeDevelopmentConfig()
			if _, err := os.Stat(".cfignore"); os.IsNotExist(err) {
				err := os.Symlink(".gitignore", ".cfignore")
				if err != nil {
					fmt.Println("Could not link .cfignore to .gitignore", err)
					os.Exit(1)
				}
			}
			npmInstalls()
			os.Exit(0)
		}

		if args[1] == "deploy" {
			_, err = cliConnection.CliCommand("push", "hackday-nc", "--no-start")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			_, err = cliConnection.CliCommand("set-env", "hackday-nc", "NODE_ENV", "development")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			createServices(cliConnection)

			_, err = cliConnection.CliCommand("start", "hackday-nc")
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			os.Exit(0)
		}

		cmd := exec.Command("treeline", args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		err = cmd.Start()
		if err != nil {
			fmt.Println("Error starting command", err)
			os.Exit(1)
		}
		err = cmd.Wait()
		if err != nil {
			fmt.Println("Error running command", err)
			os.Exit(1)
		}
	}
}

/*
*	This function must be implemented as part of the	plugin interface
*	defined by the core CLI.
*
*	GetMetadata() returns a PluginMetadata struct. The first field, Name,
*	determines the name of the plugin which should generally be without spaces.
*	If there are spaces in the name a user will need to properly quote the name
*	during uninstall otherwise the name will be treated as seperate arguments.
*	The second value is a slice of Command structs. Our slice only contains one
*	Command Struct, but could contain any number of them. The first field Name
*	defines the command `cf treeline` once installed into the CLI. The
*	second field, HelpText, is used by the core CLI to display help information
*	to the user in the core commands `cf help`, `cf`, or `cf -h`.
 */
func (c *TreelineCli) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "TreelineCli",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 0,
			Build: 0,
		},
		MinCliVersion: plugin.VersionType{
			Major: 6,
			Minor: 7,
			Build: 0,
		},
		Commands: []plugin.Command{
			plugin.Command{
				Name:     "treeline",
				HelpText: "Basic plugin command's help text",

				// UsageDetails is optional
				// It is used to show help of usage of each command
				UsageDetails: plugin.Usage{
					Usage: "treeline\n   cf treeline",
				},
			},
		},
	}
}

/*
* Unlike most Go programs, the `Main()` function will not be used to run all of the
* commands provided in your plugin. Main will be used to initialize the plugin
* process, as well as any dependencies you might require for your
* plugin.
 */
func main() {
	// Any initialization for your plugin can be handled here
	//
	// Note: to run the plugin.Start method, we pass in a pointer to the struct
	// implementing the interface defined at "github.com/cloudfoundry/cli/plugin/plugin.go"
	//
	// Note: The plugin's main() method is invoked at install time to collect
	// metadata. The plugin will exit 0 and the Run([]string) method will not be
	// invoked.
	plugin.Start(new(TreelineCli))
	// Plugin code should be written in the Run([]string) method,
	// ensuring the plugin environment is bootstrapped.
}

func npmInstalls() {
	packages := []string{"connect-redis@1.4.5", "sails-postgresql", "socket.io-redis"}
	for _, value := range packages {
		npmSetup := exec.Command("npm", "install", value, "--save", "--save-exact")
		npmSetup.Stdout = os.Stdout
		err := npmSetup.Run()
		if err != nil {
			fmt.Println("Error installing npm packages", err)
		}
	}
}

func createServices(cliConnection plugin.CliConnection) {
	services, err := cliConnection.GetServices()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	redisFound, redisBound, sqlFound, sqlBound := false, false, false, false
	for _, service := range services {
		if service.Name == "hackday-rediscloud" {
			redisFound = true
			for _, app := range service.ApplicationNames {
				if app == "hackday-nc" {
					redisBound = true
				}
			}
		}
		if service.Name == "hackday-elephantsql" {
			sqlFound = true
			for _, app := range service.ApplicationNames {
				if app == "hackday-nc" {
					sqlBound = true
				}
			}
		}
	}
	if !redisFound {
		_, err = cliConnection.CliCommand("cs", "rediscloud", "30mb", "hackday-rediscloud")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	if !redisBound {
		_, err = cliConnection.CliCommand("bs", "hackday-nc", "hackday-rediscloud")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	if !sqlFound {
		_, err = cliConnection.CliCommand("cs", "elephantsql", "turtle", "hackday-elephantsql")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	if !sqlBound {
		_, err = cliConnection.CliCommand("bs", "hackday-nc", "hackday-elephantsql")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func writeDevelopmentConfig() {
	developmentConfig := []byte(`
/**
 * Development environment settings
 */

if (process.env.VCAP_SERVICES) {
  vcapServices = JSON.parse(process.env.VCAP_SERVICES);

  module.exports = {

    /***************************************************************************
     * Set the default database connection for models in the development       *
     * environment (see config/connections.js and config/models.js )           *
     ***************************************************************************/

    models: {
      connection: 'sailsPsql',
      migrate: 'drop'
    },
    connections: {
      sailsPsql: {
        adapter: 'sails-postgresql',
        url: vcapServices.elephantsql[0].credentials.uri
      }
    },

    /***************************************************************************
     * Session configuration                                                   *
     ***************************************************************************/

    session: {
      adapter: 'redis',
      host: vcapServices.rediscloud[0].credentials.hostname,
      port: vcapServices.rediscloud[0].credentials.port,
      pass: vcapServices.rediscloud[0].credentials.password,
      prefix: 'sess:',
      // ttl: <redis session TTL in seconds>,
      // db: 0,
    },

    /***************************************************************************
     * WebSocket Configuration                                                 *
     ***************************************************************************/

    sockets: {
      adapter: 'socket.io-redis',
      host: vcapServices.rediscloud[0].credentials.hostname,
      port: vcapServices.rediscloud[0].credentials.port,
      pass: vcapServices.rediscloud[0].credentials.password,
      // db: 'sails',
    },

    /***************************************************************************
     * Set the port in the development environment to 80                       *
     ***************************************************************************/

    port: process.env.PORT,

    /***************************************************************************
     * Set the log level in development environment to "silent"                *
     ***************************************************************************/

    log: {
       level: "verbose"
    }

  };
}
`)
	err := ioutil.WriteFile("config/env/development.js", developmentConfig, 0644)
	if err != nil {
		fmt.Println("Error writing configuration", err)
		os.Exit(1)
	}
	fmt.Println("Updated config/env/development.js")

	localConfig := []byte(`
/**
 * Local environment settings
 */

module.exports = {

  /***************************************************************************
   * Set the default database connection for models in the local             *
   * environment (see config/connections.js and config/models.js )           *
   ***************************************************************************/

  models: {
    connection: 'localDiskDb',
  },
  connections: {
    localDiskDb: {
      adapter: 'sails-disk',
    }
  },

  /***************************************************************************
   * Session configuration                                                   *
   ***************************************************************************/

  session: {
  },

  /***************************************************************************
   * WebSocket Configuration                                                 *
   ***************************************************************************/

  sockets: {
  },

  /***************************************************************************
   * Set the port in the development environment to 80                       *
   ***************************************************************************/

  port: process.env.PORT || 1337,

  /***************************************************************************
   * Set the log level in development environment to "silent"                *
   ***************************************************************************/

  log: {
     level: "verbose"
  }

};
`)
	err = ioutil.WriteFile("config/local.js", localConfig, 0644)
	if err != nil {
		fmt.Println("Error writing configuration", err)
		os.Exit(1)
	}
	fmt.Println("Updated config/local.js")
}
