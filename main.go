package main

import (
	"context"
	_ "embed"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/httpserver"
	"github.com/disgoorg/snowflake/v2"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
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
		return func(e *handler.InteractionEvent) error {
			customID := ""
			interactionType := "unknown"
			switch i := e.Interaction.(type) {
			case discord.ApplicationCommandInteraction:
				interactionType = "command"
			case discord.AutocompleteInteraction:
				interactionType = "autocomplete"
			case discord.ComponentInteraction:
				interactionType = "component"
				customID = i.Data.CustomID()
			case discord.ModalSubmitInteraction:
				interactionType = "modal"
				customID = i.Data.CustomID
			case discord.PingInteraction:
				interactionType = "ping"
			}

			slog.Info("handling interaction",
				slog.Int64("interaction_id", int64(e.Interaction.ID())),
				slog.Any("interaction", e.Interaction.Type()),
				slog.String("interaction_type", interactionType),
				slog.String("custom_id", customID),
			)
			return next(e)
		}
	})
	r.Command("/ping", pingHandler)
	r.Command("/create-report-button", createReportButtonHandler)
	r.Command("/help", helpHandler)
	r.Component("/report-button/{role}", reportButtonHandler)
	r.Component("/v2/report-button/{role}/{channel}", reportButtonHandler)
	r.Modal("/report-modal/{role}", reportModalHandler)
	r.Modal("/v2/report-modal/{role}/{channel}", reportModalHandler)

	client, err := disgo.New(
		BOT_TOKEN,
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
		address := fmt.Sprintf(":%d", viper.GetInt("http_server.port"))
		openHTTPServer(client, PUB_KEY, "/interactions", address)
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

//go:embed welcome.html
var welcomePage string

func openHTTPServer(client bot.Client, publicKey, webhookPath, address string) {
	r := echo.New()
	pubKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		panic(err)
	}

	handlerFunc := httpserver.HandleInteraction(
		pubKeyBytes,
		slog.Default(),
		client.EventManager().HandleHTTPEvent,
	)

	r.POST(webhookPath, func(c echo.Context) error {
		handlerFunc.ServeHTTP(c.Response().Writer, c.Request())
		return nil
	})

	r.GET("/", func(c echo.Context) error {
		return c.HTML(200, welcomePage)
	})

	slog.Info("HTTP server is listening", "address", address)

	r.HideBanner = true
	go r.Start(address)
}
