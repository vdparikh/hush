# Hush - Secret Sharing using Vault and Slack
This project allows users to securely share secrets via Slack commands. It integrates with HashiCorp Vault to store and retrieve secrets, generating unique URLs for users to access secrets in a secure manner.

### Features
- Slack Integration: Use a Slack command (/share) to store secrets securely in HashiCorp Vault.
- Unique URLs: After storing a secret, a unique URL is generated for the user to access the secret.
- Vault Token Management: Generates Vault tokens with specific TTL for secure access to secrets.

### Prerequisites
- HashiCorp Vault installed and running locally.
- Go installed.
- A Slack app with Socket Mode enabled and the appropriate permissions (use of slash commands).


### Setup
Check out the `docs/slack` and `docs/vault` to setup slack and vault. 


### Run 
Export the environment variables with required parameters
```
export SLACK_APP_TOKEN="xapp-..."
export SLACK_BOT_TOKEN="xoxb-..."
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="s.ZZZZZZZZZZZZZZZZ"
```

- SLACK_APP_TOKEN: Slack app-level token (required for socket mode).
- SLACK_BOT_TOKEN: Slack bot token for posting messages.
- VAULT_ADDR: URL of your Vault server (e.g., http://127.0.0.1:8200).
- VAULT_TOKEN: Root token or a token with appropriate permissions.
  
Execute `go run cmd/share/share.go` 

### Share Secret
- Go to slack and type `/share password123` in any chat window. 
- You will see a response like below. 

```
Your secret has been securely shared and is valid for 1 hour:
curl \
--header "X-Vault-Token: hvs.CAESIPmvODV50_xv33zHWK_R0EEhSDm6GzHKt9mrM2iWAoAiGh4KHGh2cy5tVkdjUzh1eU54YlpHU2VDQUcyYmlPc1Q" \
--request GET \
http://127.0.0.1:8200/v1/secrets/data/shared/secret-1736903751628627000?token=hvs.CAESIPmvODV50_xv33zHWK_R0EEhSDm6GzHKt9mrM2iWAoAiGh4KHGh2cy5tVkdjUzh1eU54YlpHU2VDQUcyYmlPc1Q
```

### View Secret
Run the CURL command and you should see a response like below. Please note that the secret is only one time use and a TTL of 1 hour (hard coded for now)

```json
{
  "request_id": "37e4c132-a7b9-f026-12d2-aa9483865cd7",
  "lease_id": "",
  "renewable": false,
  "lease_duration": 0,
  "data": {
    "data": {
      "secret": "password 123"
    },
    "metadata": {
      "created_time": "2025-01-15T01:16:53.095087Z",
      "custom_metadata": null,
      "deletion_time": "",
      "destroyed": false,
      "version": 1
    }
  },
  "wrap_info": null,
  "warnings": [
    "Endpoint ignored these unrecognized parameters: [token]"
  ],
  "auth": null,
  "mount_type": "kv"
}
```



## License
This project is licensed under the MIT License - see the LICENSE file for details.

