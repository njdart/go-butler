// A mumble bot based on the Gumble libary
// https://github.com/layeh/gumble/
package main

import (
  "fmt"
  "net"
  "regexp"
  "github.com/layeh/gumble/gumble"
  "github.com/layeh/gumble/gumbleutil"
  "github.com/Sirupsen/logrus"
  "github.com/njdart/go-butler/configuration"
)

var (
  log *logrus.Logger
  Steamconnect *regexp.Regexp
  ChatCommand *regexp.Regexp
)

//TODO attach this map
var comands = map[string]string{
  "help": "Display this message.",
}

func init() {
  steamconnect, err := regexp.Compile(`connect ([A-Za-z0-9.:]+); *password (.*)`) //connect <ip>; <password>
  if err != nil {
    panic(err)
  }
  chatCmd, err := regexp.Compile(`!(\w+) *(.*)`) //separates args. first age must have a '!' e.g. !help arg1 arg2
  if err != nil {
    panic(err)
  }
  Steamconnect = steamconnect
  ChatCommand = chatCmd
}


//parse steam connect strings and provide a html button to the channel
func parseSteamConnect(e *gumble.TextMessageEvent) {
  log.Info("steam link match ip: %d pass: %d", result[1], result[2])
  button := fmt.Sprintf("<br />IP: %s <br /> PASS: %s <br /><strong><a href='steam://connect/%s/%s'>CLICK TO CONNECT TO SERVER</a></strong><br />",
    result[1], result[2], result[1], result[2])
  log.Debug(button)
  e.Client.Self.Channel.Send(button, true)
}

func HandleMessage(e *gumble.TextMessageEvent) {
  result := Steamconnect.FindStringSubmatch(e.Message)
  if result != nil {
    parseSteamConnect(e)
    return nil
  }
  result = ChatCommand.FindStringSubmatch(e.Message)
  if result != nil {
    //TODO COLLECT AND MATCH AVAILABLE COMMANDS WITH COMPORTING SUBROUTINES
    //TODO else retrun a nice message or/and cmds printout to user that sent the command
    return nil
  }
}

func main() {
  configuration, err := configuration.LoadConfiguration()
  if err != nil {
    panic(err)
  }
  log = configuration.GetLogger()
  log.Info("go-butler has sucessfully started!")
  tlsConfig, gumbleConfig := configuration.ExplodeConfiguration()

  keepAlive := make(chan bool)

  gumbleConfig.Attach(gumbleutil.Listener{
    UserChange: func(e *gumble.UserChangeEvent) {
      if e.Type.Has(gumble.UserChangeConnected) {
        e.User.Send("Welcome to the server, " + e.User.Name + "!")
      }
    },
    TextMessage: func(e *gumble.TextMessageEvent) {
      log.Infof("Received text message: %s\n", e.Message)
      HandleMessage(e)
    },
    //kill the program if we are disconnected
    Disconnect: func(e *gumble.DisconnectEvent) {
      log.Info("gumble was Disconnected from the server")
      keepAlive <- true
    },
  })

  log.Info("connecting to" + configuration.GetUri())
  client, err := gumble.DialWithDialer(new(net.Dialer), configuration.GetUri(), &gumbleConfig, &tlsConfig)
  if err != nil {
    log.Panic(err)
  } else {
    log.Info("connected!")
    log.Debug(client.State())
  }

  <-keepAlive
}
