package main

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/json"
	"log/slog"
)

var pingCommand = discord.SlashCommandCreate{
	Name:                     "ping",
	Description:              "Ping...... pong!",
	DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionManageGuild),
}

func pingHandler(ev *handler.CommandEvent) error {
	return ev.CreateMessage(discord.MessageCreate{
		Content: "Pong!",
		Flags:   discord.MessageFlagEphemeral,
	})
}

var createReportButtonCommand = discord.SlashCommandCreate{
	Name:                     "create-report-button",
	Description:              "Create a button",
	DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionManageGuild),
	DMPermission:             ref(false),
	Options: []discord.ApplicationCommandOption{
		discord.ApplicationCommandOptionString{
			Name:        "label",
			Description: "The label on the button to create",
			Required:    true,
		},
		discord.ApplicationCommandOptionString{
			Name:        "button-color",
			Description: "The color of the button",
			Required:    false,
			Choices: []discord.ApplicationCommandOptionChoiceString{
				{
					Name:  "red",
					Value: "red",
				},
				{
					Name:  "green",
					Value: "green",
				},
				{
					Name:  "blue",
					Value: "blue",
				},
				{
					Name:  "grey",
					Value: "grey",
				},
			},
		},
		discord.ApplicationCommandOptionRole{
			Name:        "role",
			Description: "The role that should be tagged when submitting a report",
			Required:    false,
		},
	},
}

var stringToButtonStyle = map[string]discord.ButtonStyle{
	"red":   discord.ButtonStyleDanger,
	"green": discord.ButtonStyleSuccess,
	"blue":  discord.ButtonStylePrimary,
	"grey":  discord.ButtonStyleSecondary,
}

func createReportButtonHandler(ev *handler.CommandEvent) error {
	data := ev.SlashCommandInteractionData()
	label := data.String("label")
	color := data.String("button-color")
	role := data.Role("role")
	if color == "" {
		color = "blue"
	}

	return ev.CreateMessage(discord.MessageCreate{
		Components: []discord.ContainerComponent{
			discord.ActionRowComponent{
				discord.NewButton(
					stringToButtonStyle[color],
					label,
					fmt.Sprintf("/report-button/%d", uint64(role.ID)),
					"",
				),
			},
		},
	})
}

func reportButtonHandler(ev *handler.ComponentEvent) error {
	slog.Info("Received report button interaction",
		"user_id", ev.User().ID,
		"user_name", ev.User().Username,
		"guild_id", *ev.GuildID(),
		"channel_id", ev.Channel().ID(),
	)
	role := ev.Variables["role"]
	customID := fmt.Sprintf("/report-modal/%s", role)
	slog.Info("Sending modal", "custom_id", customID)
	modal := discord.NewModalCreateBuilder().
		SetCustomID(customID).
		SetTitle("Report").
		AddActionRow(
			discord.NewShortTextInput("title", "Subject").
				WithPlaceholder("Subject or topic of the report").
				WithRequired(true).
				WithMinLength(5).
				WithMaxLength(72)).
		AddActionRow(
			discord.NewParagraphTextInput("description", "Description").
				WithPlaceholder(
					"Report information\n\n" +
						"Markdown is supported.\n" +
						"More details, images, etc. can be submitted afterwards.").
				WithRequired(true).
				WithMinLength(24),
		).
		Build()

	err := ev.Modal(modal)
	if err != nil {
		slog.Error("Failed to send modal", "err", err)

		var restErr rest.Error
		if errors.As(err, &restErr) {
			println(string(restErr.RsBody))
		}
	} else {
		slog.Info("Sent modal")
	}
	return err
}

func reportModalHandler(ev *handler.ModalEvent) error {
	_ = ev.DeferCreateMessage(true)
	role := ev.Variables["role"]
	title := ev.Data.Text("title")
	description := ev.Data.Text("description")

	thread, err := ev.Client().Rest().CreateThread(
		ev.Channel().ID(),
		discord.GuildPrivateThreadCreate{
			Name:                title,
			AutoArchiveDuration: 10080,
			Invitable:           false,
		})
	if err != nil {
		return err
	}

	user := ev.User()
	avatarUrl := ""
	if url := user.AvatarURL(); url != nil {
		avatarUrl = *url
	} else if url := user.DefaultAvatarURL(); url != "" {
		avatarUrl = url
	}
	embed := discord.NewEmbedBuilder().
		SetTitle(title).
		SetDescription(description).
		SetColor(0x4848FF).
		SetAuthor(user.Username, "", avatarUrl).
		Build()

	message, err := ev.Client().Rest().CreateMessage(
		thread.ID(),
		discord.MessageCreate{
			Content: fmt.Sprintf("%s%s",
				iif(role != "0", fmt.Sprintf("<@&%s> ", role), ""),
				user.Mention(),
			),
			Embeds: []discord.Embed{embed},
		})
	if err != nil {
		return err
	}

	_, err = ev.CreateFollowupMessage(discord.MessageCreate{
		Content: "Report created!",
		Flags:   discord.MessageFlagEphemeral,
		Components: []discord.ContainerComponent{
			discord.ActionRowComponent{
				discord.NewLinkButton("View", message.JumpURL()),
			},
		},
	})
	return err
}

var helpCommand = discord.SlashCommandCreate{
	Name:                     "help",
	Description:              "Show help for setting up the bot",
	DefaultMemberPermissions: json.NewNullablePtr(discord.PermissionManageGuild),
	DMPermission:             ref(true),
}

//go:embed help.md
var helpText string

func helpHandler(ev *handler.CommandEvent) error {
	isInGuild := true
	if ev.GuildID() == nil {
		isInGuild = false
	}

	return ev.CreateMessage(discord.MessageCreate{
		Content: "",
		Embeds: []discord.Embed{
			discord.NewEmbedBuilder().
				SetTitle("Modmail Help").
				SetDescription(helpText).
				SetColor(0x20FF20).
				Build(),
		},
		Flags: iif(isInGuild, discord.MessageFlagEphemeral, 0),
	})
}

func iif[T any](cond bool, ifTrue, ifFalse T) T {
	if cond {
		return ifTrue
	}
	return ifFalse
}

func ref[T any](v T) *T {
	return &v
}
