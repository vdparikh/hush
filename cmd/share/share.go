package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

const (
	vaultSecretsPath = "secrets/data/shared"
	tokenTTL         = "1h"
	tokenUses        = 2
)

func main() {
	// Load configuration
	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if slackAppToken == "" || slackBotToken == "" || vaultAddr == "" || vaultToken == "" {
		log.Fatalf("Missing required environment variables: SLACK_APP_TOKEN, SLACK_BOT_TOKEN, VAULT_ADDR, VAULT_TOKEN")
	}

	// Initialize clients
	slackClient := slack.New(
		slackBotToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack: ", log.Lshortfile)),
		slack.OptionAppLevelToken(slackAppToken),
	)
	socketClient := socketmode.New(slackClient)

	vaultClient, err := newVaultClient(vaultAddr, vaultToken)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Start event listener
	go handleSocketMode(socketClient, vaultClient)
	log.Println("Slack Bot and Vault integration is running...")

	socketClient.Run()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

func newVaultClient(addr, token string) (*api.Client, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	return client, nil
}

func handleSocketMode(client *socketmode.Client, vaultClient *api.Client) {
	for evt := range client.Events {
		switch evt.Type {
		case socketmode.EventTypeSlashCommand:
			cmd, ok := evt.Data.(slack.SlashCommand)
			if !ok {
				log.Println("Ignored unsupported slash command")
				continue
			}

			client.Ack(*evt.Request)
			log.Printf("Event received: %s, Data: %+v", evt.Type, evt.Data)

			switch cmd.Command {
			case "/share":
				handleShareCommand(client, vaultClient, cmd)
			default:
				log.Printf("Unsupported command: %s", cmd.Command)
			}
		default:
			log.Printf("Ignored unsupported event type: %s", evt.Type)
		}
	}
}

func handleShareCommand(client *socketmode.Client, vaultClient *api.Client, cmd slack.SlashCommand) {
	secret := cmd.Text
	if secret == "" {
		sendSlackResponse(client, cmd.ResponseURL, "Please provide a secret to share. Usage: `/share <secret>`")
		return
	}

	secretID := fmt.Sprintf("secret-%d", time.Now().UnixNano())
	secretPath := fmt.Sprintf("%s/%s", vaultSecretsPath, secretID)

	// Store secret in Vault
	if err := storeSecret(vaultClient, secretPath, secret); err != nil {
		log.Printf("Failed to store secret in Vault: %v", err)
		sendSlackResponse(client, cmd.ResponseURL, "Failed to store the secret. Please try again.")
		return
	}

	// Create short-lived token
	token, err := createVaultToken(vaultClient, secretID)
	if err != nil {
		log.Printf("Failed to create short-lived token: %v", err)
		sendSlackResponse(client, cmd.ResponseURL, "Failed to create a secure access token. Please try again.")
		return
	}

	// Generate Vault URL
	vaultURL := fmt.Sprintf("%s/v1/%s/%s?token=%s", vaultClient.Address(), vaultSecretsPath, secretID, token)
	response := fmt.Sprintf("Your secret has been securely shared and is valid for 1 hour: \n\n```curl --header \"X-Vault-Token: %s\" --request GET %s```", token, vaultURL)
	sendSlackResponse(client, cmd.ResponseURL, response)
}

func storeSecret(client *api.Client, path, secret string) error {
	data := map[string]interface{}{
		"data": map[string]string{
			"secret": secret,
		},
	}
	_, err := client.Logical().Write(path, data)
	return err
}

func createVaultToken(client *api.Client, secretID string) (string, error) {
	var notRenewable bool
	tokenRequest := &api.TokenCreateRequest{
		DisplayName: "Secret Share",
		Policies:    []string{"shared-secrets"},
		Metadata: map[string]string{
			"secret_id": secretID,
		},
		TTL:       tokenTTL,
		NumUses:   tokenUses,
		Renewable: &notRenewable,
		NoParent:  true,
	}

	token, err := client.Auth().Token().Create(tokenRequest)
	if err != nil {
		return "", err
	}
	return token.Auth.ClientToken, nil
}

func sendSlackResponse(client *socketmode.Client, responseURL, message string) {
	_, _, err := client.Client.PostMessage(
		"",
		slack.MsgOptionResponseURL(responseURL, slack.ResponseTypeEphemeral),
		slack.MsgOptionText(message, false),
	)
	if err != nil {
		log.Printf("Failed to send response to Slack: %v", err)
	}
}
