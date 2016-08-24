package moderation

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/yagpdb/common"
	"github.com/jonas747/yagpdb/web"
	"goji.io/pat"
	"golang.org/x/net/context"
	"html/template"
	"log"
	"net/http"
	"unicode/utf8"
)

func (p *Plugin) InitWeb() {
	web.Templates = template.Must(web.Templates.ParseFiles("templates/plugins/moderation_commands.html"))

	web.CPMux.HandleC(pat.Get("/cp/:server/commands/moderation"), web.RenderHandler(HandleModeration, "cp_moderation_commands"))
	web.CPMux.HandleC(pat.Get("/cp/:server/commands/moderation/"), web.RenderHandler(HandleModeration, "cp_moderation_commands"))
	web.CPMux.HandleC(pat.Post("/cp/:server/commands/moderation"), web.RenderHandler(HandlePostModeration, "cp_moderation_commands"))
}

// The moderation page itself
func HandleModeration(ctx context.Context, w http.ResponseWriter, r *http.Request) interface{} {
	client, activeGuild, templateData := web.GetBaseCPContextData(ctx)
	templateData["current_page"] = "moderation"
	config, err := GetConfig(client, activeGuild.ID)

	if err != nil {
		templateData.AddAlerts(web.ErrorAlert("Failed retrieving config", err))
		log.Println("Failed retrieving config", err)
	}

	templateData["current_config"] = config

	return templateData
}

// Update the settings
func HandlePostModeration(ctx context.Context, w http.ResponseWriter, r *http.Request) interface{} {
	client, activeGuild, templateData := web.GetBaseCPContextData(ctx)
	templateData["current_page"] = "moderation"

	newConfig := &Config{
		BanEnabled:    r.FormValue("ban_enabled") == "on",
		KickEnabled:   r.FormValue("kick_enabled") == "on",
		ReportEnabled: r.FormValue("report_enabled") == "on",
		CleanEnabled:  r.FormValue("clean_enabled") == "on",
		KickMessage:   r.FormValue("kick_message"),
		BanMessage:    r.FormValue("ban_message"),
	}

	// Validate the messages
	if newConfig.KickMessage != "" {
		_, err := common.ParseExecuteTemplate(newConfig.KickMessage, nil)
		if err != nil {
			templateData.AddAlerts(web.ErrorAlert("Failed parsing/executing template for kick message:", err))
			newConfig.KickMessage = ""
		}

		if utf8.RuneCountInString(newConfig.KickMessage) > 2000 {
			templateData.AddAlerts(web.ErrorAlert("Kick message is too large (max 2k)"))
			newConfig.KickMessage = ""
		}
	}

	if newConfig.BanMessage != "" {
		_, err := common.ParseExecuteTemplate(newConfig.BanMessage, nil)
		if err != nil {
			templateData.AddAlerts(web.ErrorAlert("Failed parsing/executing template for ban message:", err))
			newConfig.BanMessage = ""
		}

		if utf8.RuneCountInString(newConfig.BanMessage) > 2000 {
			templateData.AddAlerts(web.ErrorAlert("ban message is too large (max 2k)"))
			newConfig.BanMessage = ""
		}
	}

	channels, err := common.GetGuildChannels(client, activeGuild.ID)
	if err != nil {
		templateData.AddAlerts(web.ErrorAlert("Failed updating channels", err))
		log.Println("Failed updating channels", err)
	} else {
		// Make sure the channel is on the desired guild
		for _, c := range channels {
			if c.ID == r.FormValue("report_channel") {
				newConfig.ReportChannel = c.ID
			}
			if c.ID == r.FormValue("action_channel") {
				newConfig.ActionChannel = c.ID
			}
		}
	}

	err = newConfig.Save(client, activeGuild.ID)

	if err != nil {
		templateData.AddAlerts(web.ErrorAlert("Failed saving config", err))
	} else {
		templateData.AddAlerts(web.SucessAlert("Sucessfully saved config! :o"))
	}

	templateData["current_config"] = newConfig

	user := ctx.Value(web.ContextKeyUser).(*discordgo.User)
	common.AddCPLogEntry(client, activeGuild.ID, fmt.Sprintf("%s(%s) Updated moderation settings", user.Username, user.ID))

	return templateData
}
