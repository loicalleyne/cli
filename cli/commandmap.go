package cli

import "github.com/loicalleyne/cli/command"

func CreateCommandMap(cli *Cli) map[string]func(args []string) {
	return commandsToMap(cli.Commands)
}

func commandsToMap(commands []command.Command) map[string]func(args []string) {
	commandMap := make(map[string]func(args []string))
	for _, command := range commands {
		commandMap[command.Name] = func(args []string) {
			command.Func(args)
		}
		if len(command.SubCommands) > 0 {
			nestedCommandMap := commandsToMap(command.SubCommands)
			commandMap = mergeMaps(commandMap, nestedCommandMap)
		}
	}
	return commandMap
}

func mergeMaps(map1, map2 map[string]func(args []string)) map[string]func(args []string) {
	result := make(map[string]func(args []string))
	for k, v := range map1 {
		result[k] = v
	}
	for k, v := range map2 {
		result[k] = v
	}
	return result
}
