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

// Configuration
const vaultSecretsPath = "secrets/data/shared"

func main() {
	// Load Slack App Token and Vault Address
	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if slackAppToken == "" || slackBotToken == "" || vaultAddr == "" || vaultToken == "" {
		log.Fatalf("Missing required environment variables: SLACK_APP_TOKEN, SLACK_BOT_TOKEN, VAULT_ADDR, VAULT_TOKEN")
	}

	// Initialize Slack and Vault Clients
	api := slack.New(
		slackBotToken,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "slack: ", log.Lshortfile)),
		slack.OptionAppLevelToken(slackAppToken),
	)
	client := socketmode.New(api)

	vaultClient, err := newVaultClient(vaultAddr, vaultToken)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Start listening to Slack events
	go handleSocketMode(client, vaultClient)
	log.Println("Slack Bot and Vault integration is running...")
	client.Run()
	// Wait for exit signal
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan
}

// Initialize Vault Client
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

func testSecretStore(vaultClient *api.Client) {
	secretID := fmt.Sprintf("secret-%d", time.Now().UnixNano())

	// Store the secret in Vault
	secretPath := fmt.Sprintf("%s/%s", vaultSecretsPath, secretID)
	data := map[string]interface{}{
		"data": map[string]string{
			"secret": "vishal",
		},
	}

	_, err := vaultClient.Logical().Write(secretPath, data)
	if err != nil {
		log.Printf("Vault error: %v", err)
	}
}

// Handle Socket Mode Events
func handleSocketMode(client *socketmode.Client, vaultClient *api.Client) {
	fmt.Println("handleSocketMode")

	policies, err := vaultClient.Sys().ListPolicies()
	fmt.Println(policies, err)

	for evt := range client.Events {
		switch evt.Type {
		case socketmode.EventTypeInteractive:
			log.Println("Interactive Event Received")
		case socketmode.EventTypeSlashCommand:
			cmd, ok := evt.Data.(slack.SlashCommand)
			if !ok {
				log.Println("Ignored unsupported slash command")
				continue
			}

			client.Ack(*evt.Request)
			log.Printf("Event received: %s, Data: %+v", evt.Type, evt.Data)
			if cmd.Command == "/share" {
				handleShareCommand(client, vaultClient, cmd)
			} else {
				log.Printf("Unsupported command: %s", cmd.Command)
			}
		default:
			log.Printf("Ignored unsupported event type: %s", evt.Type)
		}
	}
}

// Handle the /share command
func handleShareCommand(client *socketmode.Client, vaultClient *api.Client, cmd slack.SlashCommand) {
	secret := cmd.Text
	if secret == "" {
		sendSlackResponse(client, cmd.ResponseURL, "Please provide a secret to share. Usage: `/share <secret>`")
		return
	}

	// Generate a unique identifier for the secret
	secretID := fmt.Sprintf("secret-%d", time.Now().UnixNano())

	// Store the secret in Vault
	secretPath := fmt.Sprintf("%s/%s", vaultSecretsPath, secretID)
	data := map[string]interface{}{
		"data": map[string]string{
			"secret": secret,
		},
	}

	_, err := vaultClient.Logical().Write(secretPath, data)
	if err != nil {
		log.Printf("Failed to store secret in Vault: %v", err)
		sendSlackResponse(client, cmd.ResponseURL, "Failed to store the secret. Please try again.")
		return
	}

	// Generate a short-lived token for secret access
	var notRenewable bool
	tokenRequest := &api.TokenCreateRequest{
		DisplayName: "Secret Share",
		Policies:    []string{"shared-secrets"},
		Metadata: map[string]string{
			"secret_id": secretID,
		},
		TTL:            "1h", // Token validity
		ExplicitMaxTTL: "1h",
		NumUses:        2, //1 to create 2 to get
		Renewable:      &notRenewable,
		NoParent:       true,
	}

	token, err := vaultClient.Auth().Token().Create(tokenRequest)
	if err != nil {
		log.Printf("Failed to create short-lived token: %v", err)
		sendSlackResponse(client, cmd.ResponseURL, "Failed to create a secure access token. Please try again.")
		return
	}

	// wrappedToken, err := Wrap(vaultClient, token.Auth.ClientToken)
	// if err != nil {
	// 	log.Printf("Failed to wrap token: %v", err)
	// 	// ... (your error handling) ...
	// 	return
	// }

	// Generate a Vault URL for secret retrieval
	vaultURL := fmt.Sprintf("%s/v1/%s/%s?token=%s", vaultClient.Address(), vaultSecretsPath, secretID, token.Auth.ClientToken)

	// Respond with the Vault URL
	fmt.Printf("Your secret has been securely shared: %s (valid for 1 hour)\n", vaultURL)

	// Respond with the Vault URL
	response := fmt.Sprintf(`curl \
--header "X-Vault-Token: %s" \
--request GET \
%s`, token.Auth.ClientToken, vaultURL)
	// sendSlackResponse(client, cmd.ResponseURL, fmt.Sprintf("Your secret has been securely shared: %s (valid for 1 hour)", vaultURL))
	sendSlackResponse(client, cmd.ResponseURL, fmt.Sprintf("Your secret has been securely shared and is valid for 1 hour: \n\n```%s```", response))

}

func Wrap(client *api.Client, plaintext string) (string, error) {
	wrapData := map[string]interface{}{
		"data": plaintext,
	}

	wrappedSecret, err := client.Logical().Write("sys/wrapping/wrap", wrapData)
	if err != nil {
		return "", err
	}

	return wrappedSecret.WrapInfo.Token, nil
}

// Send response to Slack
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
