package main

import (
	"context"
	"fmt"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/httpserver"
	"github.com/disgoorg/snowflake/v2"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	BOT_TOKEN := viper.GetString("discord.token")
	PUB_KEY := viper.GetString("discord.pub_key")

	missing := false
	if BOT_TOKEN == "" {
		missing = true
		slog.Error("Missing discord.token")
	}
	if PUB_KEY == "" {
		missing = true
		slog.Error("Missing discord.pub_key")
	}
	if missing {
		os.Exit(1)
	}

	r := handler.New()
	r.Use(func(next handler.Handler) handler.Handler {
		return func(e *events.InteractionCreate) error {
			slog.Info("handling interaction",
				slog.Int64("interaction_id", int64(e.Interaction.ID())),
				slog.Int("interaction_type", int(e.Interaction.Type())),
			)
			return next(e)
		}
	})
	r.Command("/ping", pingHandler)
	r.Command("/create-report-button", createReportButtonHandler)
	r.Component("/report-button/{role}", reportButtonHandler)
	r.Modal("/report-modal/{role}", reportModalHandler)

	client, err := disgo.New(
		BOT_TOKEN,
		bot.WithHTTPServerConfigOpts(
			PUB_KEY,
			httpserver.WithURL("/interactions"),
			httpserver.WithAddress(fmt.Sprintf(":%d", viper.GetUint("http_server.port"))),
		),
		bot.WithDefaultGateway(),
		bot.WithEventListeners(r),
		bot.WithEventListenerFunc(func(ev *events.Ready) {
			slog.Info("Bot is ready")
		}),
	)

	if err != nil {
		panic(err)
	}

	commands := []discord.ApplicationCommandCreate{
		pingCommand,
		createReportButtonCommand,
		helpCommand,
	}

	if viper.GetBool("dev_mode.enabled") {
		var cmds []discord.ApplicationCommand
		cmds, err = client.Rest().SetGuildCommands(
			client.ApplicationID(),
			snowflake.ID(viper.GetUint64("dev_mode.guild")),
			commands,
		)
		for _, cmd := range cmds {
			slog.Info("Registered guild command", "command_id", cmd.ID(), "command_name", cmd.Name())
		}
	} else {
		var cmds []discord.ApplicationCommand
		cmds, err = client.Rest().SetGlobalCommands(client.ApplicationID(), commands)
		for _, cmd := range cmds {
			slog.Info("Registered global command", "command_id", cmd.ID(), "command_name", cmd.Name())
		}
	}
	if err != nil {
		panic(err)
	}

	if viper.GetBool("http_server.enabled") {
		err = client.OpenHTTPServer()
		slog.Info("HTTP server is listening", "port", viper.GetInt("http_server.port"))
	} else {
		err = client.OpenGateway(context.Background())
	}
	if err != nil {
		panic(err)
	}

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)
	<-s
}
