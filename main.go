// A mumble bot based on the Gumble libary
package main

import (
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"layeh.com/gumble/gumble"
	"layeh.com/gumble/gumbleutil"
	"net"
	"os"
	"regexp"
	"strings"
)

var (
	log          *logrus.Logger
	Steamconnect *regexp.Regexp
	ChatCommand  *regexp.Regexp
)

//taken from cmd/go/main as it can not be imported
// A Command is an implementation of command
type Command struct {
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string, sender *gumble.TextMessageEvent) string

	// UsageLine is the one-line usage message.
	// The first word in the line is taken to be the command name.
	UsageLine string

	// Short is the short description shown in the '!help' output.
	Short string

	// Long is the long message shown in the !help <command> output.
	Long string

	//if the cmd output will be sent back to the whole channel
	// or to a user in a private message
	PublicResponse bool
}

// Name returns the command's name: the first word in the usage line.
func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(c.Long))
	os.Exit(2)
}

var commands = []*Command{
	status,
	connect,
	joinme,
}

//this is generated at init
var HelpString string

//The name you call to get help
const HelpCmdName string = "help"

var help = &Command{
	PublicResponse: false,
	UsageLine:      "help [command]",
	Short:          "cmds list as well as cmd deail",
	Long:           ``,
}

func FormatHelpString(commands []*Command) string {
	commands = append(commands, help)
	out := HtmlNewLine + "Available comands:"
	for _, cmd := range commands {
		out += fmt.Sprintf("%s <strong>%s</strong> - %s", HtmlNewLine, cmd.Name(), cmd.Short)
	}
	return out
}

func FormatCmdHelp(cmd *Command) string {
	return fmt.Sprintln(HtmlNewLine, cmd.UsageLine, HtmlNewLine, cmd.Long)
}

//called with special case case in HandleMessage
func Help(args []string) string {
	if args[2] == "" { //just help
		return HelpString
	}
	//look for other cmds passed as args
	// and show there UsageLine + Long discription
	for _, cmd := range commands {
		if cmd.Name() == args[2] {
			return FormatCmdHelp(cmd)
		}
	}
	return CommandNotFound(args[2])
}

const HtmlNewLine string = `<br />`

//make to string to tell the user we haven't a clue
func CommandNotFound(usrInput string) string {
	return fmt.Sprintf("%s No command '%s' found! %s", HtmlNewLine, usrInput, HelpString)
}

func init() {
	HelpString = FormatHelpString(commands)

	steamconnect, err := regexp.Compile(`^connect ([A-Za-z0-9.:]+); *password ([^<\n]*)`) //connect <ip>; <password> (ignore <html> and newline)
	if err != nil {
		panic(err)
	}
	chatCmd, err := regexp.Compile(`!(\w+) *(.*)`) //separates 1st arg from the rest. first age must have a '!' e.g. !help arg1 arg2
	if err != nil {
		panic(err)
	}
	Steamconnect = steamconnect
	ChatCommand = chatCmd
}

// takes a gumble.TextMessageEvent and cmd
// runs the cmd if it exits and return it's output
// else return a canned response
func HandleCmd(args []string, event *gumble.TextMessageEvent) (string, bool) {
	for _, cmd := range commands {
		if cmd.Name() == args[1] {
			return cmd.Run(cmd, args, event), cmd.PublicResponse
		}
	}
	return CommandNotFound(args[1]), false
}

//parse steam connect strings and provide a html button to the channel
func HandleMessage(e *gumble.TextMessageEvent, config *ButlerConfiguration) {
	message := gumbleutil.PlainText(&e.TextMessage)
	result := Steamconnect.FindStringSubmatch(message) //check for steam connect cmds
	if result != nil {
		log.Infof("steam link match ip: %s pass: %s", result[1], result[2])
		currentConnect = &SourceConnect{
			hostname: result[1],
			password: result[2],
			UserID:   e.Sender.UserID,
		}
		currentConnectHTML, currentConnectString = currentConnect.GenConnectString()

		e.Client.Self.Channel.Send(currentConnectHTML, config.Bot.RecursiveChannelMessages)
	} else {
		//check for bot commands
		result = ChatCommand.FindStringSubmatch(message)
		if result != nil {
			if result[1] == HelpCmdName {
				//special case to avoid initialization loop
				log.Infof("User %s (ID:%d) called '!%s'", e.Sender.Name, e.Sender.UserID, HelpCmdName)
				e.Sender.Send(Help(result))
			} else {
				log.Infof("User %s (ID:%d) called '%s'", e.Sender.Name, e.Sender.UserID, result[0])

				responce, PublicResponse := HandleCmd(result, e)
				if PublicResponse { //send to channel
					e.Client.Self.Channel.Send(responce, config.Bot.RecursiveChannelMessages)
				} else { //send to usr
					e.Sender.Send(responce)
				}
				log.Debugf("Returning to user username:'%s' id:%d PublicResponse: %t with the message:\n%s", e.Sender.Name, e.Sender.UserID, PublicResponse, responce)
			}
		}
	}
}

func Greeter(e *gumble.UserChangeEvent, config *ButlerConfiguration) {
	if config.Greeter.WelcomeUsers && e.Type.Has(gumble.UserChangeConnected) {
		e.User.Send("Welcome to the server, " + e.User.Name + "!")
	}
	if config.Greeter.PassConnectOnChannelJoin && e.Type.Has(gumble.UserChangeChannel) && (e.User.Channel.ID == e.Client.Self.Channel.ID) {
		e.Client.Self.Channel.Send(currentConnectHTML, false)
	}
}

func main() {
	configPath := flag.String("c", os.Getenv("BUTLERCONFIG"), "Path to config file")
	hostname := flag.String("h", "", "Override server address, implies you are allso overriding port (port flag can be ignored for mumble default)")
	port := flag.Int("p", 64738, "Override port")
	console := flag.Bool("console", false, "Force logging to stdout")
	flag.Parse()

	config, err := LoadConfiguration(*configPath)
	if err != nil {
		panic(err)
	}
	if *console {
		config.Log.File = ""
	}
	if *hostname != "" {
		config.Server.Host = *hostname
		config.Server.Port = *port
	}

	log = config.GetLogger()
	log.Infof("Given config parsed:\n%+v", config)
	log.Info("go-butler has sucessfully started!")
	tlsConfig, gumbleConfig := config.GetGumbleConfig()

	keepAlive := make(chan bool)

	gumbleConfig.Attach(gumbleutil.Listener{
		UserChange: func(e *gumble.UserChangeEvent) {
			Greeter(e, &config)
		},
		TextMessage: func(e *gumble.TextMessageEvent) {
			HandleMessage(e, &config)
		},
		//kill the program if we are disconnected
		Disconnect: func(e *gumble.DisconnectEvent) {
			log.Info("gobuter was Disconnected from the server")
			keepAlive <- true
		},
	})

	log.Info("connecting to " + config.GetUri())
	client, err := gumble.DialWithDialer(new(net.Dialer), config.GetUri(), &gumbleConfig, &tlsConfig)
	if err != nil {
		log.Panic(err)
	} else {
		log.Info("connected!")
		if config.Bot.DefaultChannel != "" {
			defaultChannel := client.Channels.Find(config.Bot.DefaultChannel)
			if defaultChannel != nil {
				log.Infof("Moving to default channel '%s'", defaultChannel.Name)
				client.Self.Move(defaultChannel)
			}
		}
	}

	<-keepAlive
}
