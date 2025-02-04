// Package cli is a simple package to help implement interactive command line interfaces in golang.
// One of the main reasons behind generate it is that there is a lack of subcommand support in other packages.
package cli

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/loicalleyne/cli/command"
	"github.com/loicalleyne/cli/vault"
)

// Cli structure contains configuration and commands
type Cli struct {
	Commands        []command.Command
	ReadlineConfig  *readline.Config
	LastInteraction time.Time
	Scanner         *readline.Instance
	Vault           vault.Database
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

var completer = readline.NewPrefixCompleter()

// NewCli creates a new instance of Cli
// It returns a pointer to the Cli object
func NewCli() *Cli {
	c := &Cli{}

	l, err := readline.NewEx(&readline.Config{
		Prompt:          ">>> ",
		HistoryFile:     "/tmp/readline.tmp",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		// TODO some weird version error broke this
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	c.Scanner = l

	return c
}

// AddCommand is a method on Cli takes Command as input
// This appends to the current command list to search through for input
func (cli *Cli) AddCommand(c command.Command) {
	cli.Commands = append(cli.Commands, c)

	// recusively add command names to completer
	pc := readline.PcItem(c.Name)
	cli.recurseCompletion(c.SubCommands, pc, 0)
	completer.Children = append(completer.Children, pc)
}

func (cli *Cli) peakChildren(c []command.Command, name string) *command.Command {
	for _, cmd := range c {
		if cmd.Name == name {
			return &cmd
		}
	}
	return nil
}

func (cli *Cli) recurseCompletion(c []command.Command, pc *readline.PrefixCompleter, i int) error {
	for _, cmd := range c {
		p := readline.PcItem(cmd.Name)
		pc.Children = append(pc.Children, p)

		if len(cmd.SubCommands) > 0 {
			cli.recurseCompletion(cmd.SubCommands, p, i+1)
		}
	}
	return nil
}

func (cli *Cli) recurseHelp(c []command.Command, rootCommands []string, offset int) {
	for _, cmd := range c {
		if cmd.Name == "." {
			continue
		}
		for i := 0; i < offset; i++ {
			fmt.Printf("\t")
		}
		for _, n := range rootCommands {
			if strings.Compare(n, cmd.Name) == 0 {
				offset = 0
			}
		}
		fmt.Printf("[%s]: %s\n", cmd.Name, cmd.Help)
		if len(cmd.SubCommands) > 0 {
			cli.recurseHelp(cmd.SubCommands, rootCommands, offset+1)
		}
	}
}

func (cli *Cli) recurseManPage(c []command.Command, rootCommands []string, offset int) {
	for _, cmd := range c {
		for i := 0; i < offset; i++ {
			fmt.Printf("\t")
		}
		for _, n := range rootCommands {
			if strings.Compare(n, cmd.Name) == 0 {
				offset = 0
			}
		}
		fmt.Printf("[%s]: %s\n", cmd.Name, cmd.ManPage)
		if len(cmd.SubCommands) > 0 {
			cli.recurseManPage(cmd.SubCommands, rootCommands, offset+1)
		}
	}
}

func (cli *Cli) parseSystemCommands(input []string) error {
	if input[0] == "exit" {
		fmt.Println("Bye")
		os.Exit(0)
	}
	if input[0] == "clear" {
		print("\033[H\033[2J")
	}
	if input[0] == "help" {
		switch i := len(input); i {
		case 1:
			var rootCommands []string
			for _, r := range cli.Commands {
				rootCommands = append(rootCommands, r.Name)
			}
			cli.recurseHelp(cli.Commands, rootCommands, 0)
		default:
			var rootCommandsNames []string
			var rootCommands []command.Command
			for _, cmd := range input[1:] {
				for _, r := range cli.Commands {
					if cmd == r.Name {
						rootCommandsNames = append(rootCommandsNames, r.Name)
						rootCommands = append(rootCommands, r)
					}
				}
			}
			if len(rootCommandsNames) > 0 {
				cli.recurseHelp(rootCommands, rootCommandsNames, 0)
			} else {
				fmt.Println("command(s) not found")
			}
		}
	}
	if input[0] == "man" {
		switch i := len(input); i {
		case 1:
			var rootCommandsNames []string
			var rootCommands []command.Command
			for _, r := range cli.Commands {
				if r.ManPage != "" {
					rootCommandsNames = append(rootCommandsNames, r.Name)
					rootCommands = append(rootCommands, r)
				}
			}
			if len(rootCommandsNames) > 0 {
				cli.recurseManPage(rootCommands, rootCommandsNames, 0)
			} else {
				fmt.Println("no manpages found")
			}

		default:
			var rootCommandsNames []string
			var rootCommands []command.Command
			var unfound []string
			for _, cmd := range input[1:] {
				for _, r := range cli.Commands {
					if cmd == r.Name && r.ManPage != "" {
						rootCommandsNames = append(rootCommandsNames, r.Name)
						rootCommands = append(rootCommands, r)
						break
					}
				}
				if len(rootCommandsNames) > 0 && rootCommandsNames[len(rootCommandsNames)-1] != cmd {
					unfound = append(unfound, cmd)
				}
			}
			if len(unfound) > 0 {
				fmt.Println("manpage not found for command(s) %v", unfound)
			}
			if len(rootCommandsNames) > 0 {
				cli.recurseManPage(rootCommands, rootCommandsNames, 0)
			} else {
				fmt.Println("no manpages found")
			}
		}
	}

	return nil
}

func (cli *Cli) recurse(c []command.Command, args []string, i int) error {
	for _, cmd := range c {
		if i > len(args) {
			return nil
		}
		if cmd.Name == args[i] {
			if len(args) > i+1 {
				if child := cli.peakChildren(cmd.SubCommands, args[i+1]); child != nil {
					cli.recurse(cmd.SubCommands, args, i+1)
				} else {
					cmd.Func(args[i+1:])
					fmt.Printf("\n")
				}
			} else {
				cmd.Func(args[i+1:])
				fmt.Printf("\n")
			}
		}
	}
	return nil
}

func (cli *Cli) findCommand(input string) error {
	parsed := strings.Fields(input)
	if len(parsed) == 0 {
		fmt.Println("No input detected")
		return nil
	}
	if systemCmd := cli.parseSystemCommands(parsed); systemCmd != nil {
		return nil
	}
	currentCommands := cli.Commands
	error := cli.recurse(currentCommands, parsed, 0)
	if error != nil {
		return error
	}
	return nil
}

func (cli *Cli) readline() string {
	text, _ := cli.Scanner.Readline()
	cli.Scanner.SaveHistory(text)
	return text
}

// Run is the primary entrypoint to start blocking and reading user input
func (cli *Cli) Run() {
	if len(os.Args) > 1 && os.Args[1] == "unattended" {
		err := cli.findCommand(strings.Join(os.Args[2:], " "))
		if err != nil {
			color.Red(err.Error())
		}
		os.Exit(0)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			fmt.Printf("\nBye\n")
			os.Exit(0)
		}
	}()

	for {
		// Get user input
		fmt.Print(cli.Scanner.Config.Prompt)

		text := cli.readline()
		cli.LastInteraction = time.Now()

		err := cli.findCommand(text)
		if err != nil {
			color.Red(err.Error())
		}
	}
}

func (cli *Cli) Suspend() {
	cli.ReadlineConfig = cli.Scanner.Config.Clone()
	cli.Scanner.Config.Stdin.Close()
	cli.Scanner.Operation.Close()
	cli.Scanner.Terminal.Close()
	t := new(time.Time)
	cli.LastInteraction = *t
}

func (cli *Cli) Resume() {
	l, err := readline.NewEx(cli.ReadlineConfig)
	if err != nil {
		panic(err)
	}
	cli.Scanner = l
	cli.LastInteraction = time.Now()
	for {
		// Get user input
		fmt.Print(cli.Scanner.Config.Prompt)

		text := cli.readline()
		cli.LastInteraction = time.Now()

		err := cli.findCommand(text)
		if err != nil {
			color.Red(err.Error())
		}
	}
}
