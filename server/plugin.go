package main

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type Plugin struct {
	plugin.MattermostPlugin
	configuration *configuration
}

type configuration struct {
	RestrictedChannelName string
	RejectionMessage      string
	parsedNames           map[string]struct{}
}

func (p *Plugin) OnConfigurationChange() error {
	var config configuration
	if err := p.API.LoadPluginConfiguration(&config); err != nil {
		return err
	}

	// Extract channel names from string
	names := strings.Split(config.RestrictedChannelName, ",")
	parsed := make(map[string]struct{})
	for _, name := range names {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed != "" {
			parsed[trimmed] = struct{}{}
		}
	}
	config.parsedNames = parsed

	p.configuration = &config
	return nil
}
func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {

	// Get channel details
	channel, appErr := p.API.GetChannel(post.ChannelId)
	if appErr != nil {
		p.API.LogError("Failed to get channel", "error", appErr.Error())
		return post, ""
	}

	// Check if the channel is in the map
	if _, ok := p.configuration.parsedNames[strings.ToLower(channel.Name)]; !ok {
		// The channel is not restricted
		return post, ""
	}

	// Get user data
	user, appErr := p.API.GetUser(post.UserId)
	if appErr != nil {
		p.API.LogError("Failed to get user", "error", appErr.Error())
		return post, ""
	}

	// Get the users status in the channel
	member, appErr := p.API.GetChannelMember(post.ChannelId, post.UserId)
	if appErr != nil {
		p.API.LogError("Failed to get channel member", "error", appErr.Error())
		return nil, ""
	}

	isChannelAdmin := strings.Contains(member.Roles, "channel_admin")

	if !isChannelAdmin {
		// Let the user know their message was rejected
		p.API.SendEphemeralPost(post.UserId, &model.Post{
			ChannelId: post.ChannelId,
			Message:   p.configuration.RejectionMessage,
		})
		p.API.LogInfo("Blocked non-channel admin post", "user", user.Username)
		return nil, p.configuration.RestrictedChannelName
	}

	// The user is a channel admin - post the message
	return post, ""
}
