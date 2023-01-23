# ssl-checker

This is `ssl-checker`, a Go CLI program that I created as a playground project. It is a fast and simple solution to check the SSL certificates of a large number of HTTPS endpoints.

## Demo

<p align="center">
    <img width="700" src="demo.gif" />
</p>

## Usage

```
ssl-checker is a tool to _quickly_ check certificate details of multiple https targets.

Usage:
  ssl-checker [flags] [file-targets <files>|domain-targets <domains>]
  ssl-checker [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  domains     Run test against provided list of domains
  files       Run test against provided list of files
  help        Help about any command
  version     Show the current version

Flags:
  -c, --config string         Configuration file location (default "$HOME/.config/ssl-checker/config.yaml")
  -d, --debug                 Enable debug log, out will be saved in ./ssl-checker.log
  -e, --environments string   Comma delimited string specifying the environments to check
  -h, --help                  help for ssl-checker
  -s, --silent                disable ui
  -t, --timeout uint16        Set timeout for SSL check queries (default 10)
  -v, --version               version for ssl-checker

Use "ssl-checker [command] --help" for more information about a command.
```

## Running queries

The tool can be used through a configuration file or command line options. 
The recommended way to use the tool is through a configuration file, located at $HOME/.config/ssl-checker/config.yaml. 
This file allows you to define "queries" which are organized by environment. The queries can be set in two formats:

  - A text file with one endpoint DNS per line
  - A list of endpoint DNS.

For example, the following is a sample configuration file:
```yaml
timeout: 5 # default to 10s
queries:
  EnvA: "$HOME/domains_projectA.txt"
  EnvB: "$HOME/domains_projectB.txt"
  qa:
    - www.foo.com
    - www.bar.com
  poc:
    - www.mypoc.com
```

In this configuration file, the timeout is set to 5 seconds, and there are four different environments: EnvA, EnvB, qa and poc, each one with its own set of queries.

It's important to notice that you can use either a file with a list of DNS or directly put them in the configuration file, depending on your needs.

If you don't have a config file or are in a hurry, you can still use the tool by specifying the targets directly on the command line. To run a query against targets defined in files, use the command `ssl-checker files file1,file2`. To specify the targets directly, use the command `ssl-checker domains www.domainA.com,www.domainB.com`.

Additionally, you can generate a markdown report of the results by using the E key or the -s option. This report will provide a detailed summary of the SSL certificate information for each endpoint. It's useful for sending the results to your team members or for storing it for future reference.

# Credits

I'd like to extend a special thank you to the creators of the following libraries that have made this project possible:

- The team at [charm.sh](https://charm.sh/) for their awesome libraries and the incredible [vhs](https://github.com/charmbracelet/vhs) tool used in the demo.
- The developers behind [cobra](https://github.com/spf13/cobra) and [viper](https://github.com/spf13/viper) for simplifying the process of handling CLI flags and configuration files.
- The creator of [zerolog](https://github.com/rs/zerolog) for providing such a powerful logging solution.

