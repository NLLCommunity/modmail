package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/json"
	"github.com/disgoorg/snowflake/v2"
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
	Contexts:                 []discord.InteractionContextType{discord.InteractionContextTypeGuild},
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
		discord.ApplicationCommandOptionChannel{
			Name:        "channel",
			Description: "The channel to send the report to",
			Required:    false,
			ChannelTypes: []discord.ChannelType{
				discord.ChannelTypeGuildText,
			},
		},
		discord.ApplicationCommandOptionInt{
			Name:        "max-active-reports",
			Description: "The maximum number of active reports a user can have. (0 or unspecified = no limit)",
			Required:    false,
			MinValue:    ref(0),
			MaxValue:    ref(100),
		},
		discord.ApplicationCommandOptionString{
			Name:        "slow-mode-time",
			Description: "Enable slow mode for the report thread in format '1h5m10s' (0s = disabled)",
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
	channel := data.Channel("channel")
	maxActive := data.Int("max-active-reports")
	slowModeStr := data.String("slow-mode-time")
	if color == "" {
		color = "blue"
	}

	slowMode, err := time.ParseDuration(slowModeStr)
	if err != nil {
		slog.Info("Failed to parse slow mode duration", "err", err, "slow_mode", slowModeStr)
		return ev.CreateMessage(discord.NewMessageCreateBuilder().
			SetContentf("Could not parse duration: `%s`", slowModeStr).
			SetEphemeral(true).
			Build())
	}

	if slowMode.Hours() > 6 {
		return ev.CreateMessage(discord.NewMessageCreateBuilder().
			SetContentf("Slow mode duration is too long: `%s`. Max is 6 hours.", slowModeStr).
			SetEphemeral(true).
			Build())
	}

	return ev.CreateMessage(discord.MessageCreate{
		Components: []discord.ContainerComponent{
			discord.ActionRowComponent{
				discord.NewButton(
					stringToButtonStyle[color],
					label,
					fmt.Sprintf("/v4/report-button/%d/%d/%d/%.0f", uint64(role.ID), uint64(channel.ID), maxActive, slowMode.Seconds()),
					"",
					0,
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
		"custom_id", ev.Data.CustomID(),
	)
	role := ev.Vars["role"]
	channel := ev.Vars["channel"]

	maxActive, ok := ev.Vars["max_active"]
	if !ok {
		maxActive = "0"
	}
	slowModeStr, ok := ev.Vars["slow_mode"]
	if !ok {
		slowModeStr = "0"
	}

	customID := fmt.Sprintf("/v4/report-modal/%s/%s/%s/%s", role, channel, maxActive, slowModeStr)

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

	role := ev.Vars["role"]
	channel := ev.Vars["channel"]
	maxActiveStr := ev.Vars["max_active"]
	slowModeStr := ev.Vars["slow_mode"]

	maxActive, err := strconv.Atoi(maxActiveStr)
	if err != nil {
		slog.Error("Failed to parse max active", "err", err, "max_active", maxActiveStr)
		_, err := ev.CreateFollowupMessage(discord.NewMessageCreateBuilder().
			SetContent("Failed to parse report config.").
			SetEphemeral(true).
			Build())
		return err
	}

	slowMode, err := strconv.Atoi(slowModeStr)
	if err != nil {
		slog.Error("Failed to parse slow mode", "err", err, "slow_mode", slowModeStr)
		_, err := ev.CreateFollowupMessage(discord.NewMessageCreateBuilder().
			SetContent("Failed to parse report config.").
			SetEphemeral(true).
			Build())
		return err
	}

	canSubmit, err := isBelowMaxActive(*ev, maxActive)
	if err != nil {
		slog.Error("Failed to check if user can submit report", "err", err, "max_active", maxActiveStr)
		_, err := ev.CreateFollowupMessage(discord.NewMessageCreateBuilder().
			SetContent("Failed to check if user can submit report.").
			SetEphemeral(true).
			Build())
		return err
	}

	if !canSubmit {
		_, err := ev.CreateFollowupMessage(discord.NewMessageCreateBuilder().
			SetContent("You have reached the maximum number of active reports.").
			SetEphemeral(true).
			Build())
		return err
	}

	title := ev.Data.Text("title")
	description := ev.Data.Text("description")

	thread, err := ev.Client().Rest().CreateThread(
		ev.Channel().ID(),
		discord.GuildPrivateThreadCreate{
			Name:                title,
			AutoArchiveDuration: 10080,
			Invitable:           ref(false),
		})
	if err != nil {
		return err
	}

	if slowMode > 0 {
		_, err = ev.Client().Rest().UpdateChannel(thread.ID(), discord.GuildThreadUpdate{
			RateLimitPerUser: ref(slowMode),
		})
		if err != nil {
			slog.Warn("Failed to update thread rate limit", "err", err, "channel", thread.ID())
		}
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

	if channel != "" && channel != "0" {
		channelSf, err := snowflake.Parse(channel)
		if err == nil {
			_, err = ev.Client().Rest().CreateMessage(
				channelSf,
				discord.NewMessageCreateBuilder().
					SetContentf("## New Modmail thread in <#%d>", ev.Channel().ID()).
					AddEmbeds(embed).
					AddActionRow(
						discord.NewLinkButton("Go to thread", message.JumpURL()),
					).
					Build(),
			)

			if err != nil {
				slog.Error("Failed to send message to modmail channel", "err", err, "channel", channel)
			}
		}
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

func isBelowMaxActive(e handler.ModalEvent, maxActive int) (bool, error) {
	if maxActive == 0 {
		return true, nil
	}

	if e.GuildID() == nil {
		slog.Error("Not in guild")
		return false, nil
	}
	guildID := *e.GuildID()

	activeThreads, err := e.Client().Rest().GetActiveGuildThreads(guildID)
	if err != nil {
		slog.Error("Failed to list active threads", "err", err)
		return false, err
	}

	userThreadsCount := 0
	for _, thread := range activeThreads.Threads {
		if *thread.ParentID() != e.Channel().ID() {
			continue
		}
		members, err := e.Client().Rest().GetThreadMembers(thread.ID())
		if err != nil {
			slog.Error("Failed to get thread members", "err", err)
			return false, err
		}
		for _, member := range members {
			if member.UserID == e.User().ID {
				userThreadsCount++
			}
			if userThreadsCount >= maxActive {
				return false, nil
			}
		}
	}
	if userThreadsCount >= maxActive {
		return false, nil
	}

	return true, nil
}
