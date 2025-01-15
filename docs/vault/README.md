# Setup Hashicorp Vault on your Mac

Using homebrew, setup your Vault
```sh
brew install vault
```

```sh
vault --version
Vault v1.16.2 (c6e4c2d4dc3b0d57791881b087c026e2f75a87cb), built 2024-04-22T16:25:54Z
```

```sh
# Start Vault
vault server -dev
```

Make a note of the root token 
Unseal Key: bUr6kBLnoR/qxB1DzIb/c6g0x80EwST5o6L/qz3OqDE=
Root Token: hvs.3RCmLjLTvvj5dz8Sv6ctn4p1


You can check the status of your vault using
```
vault status
Key             Value
---             -----
Seal Type       shamir
Initialized     true
Sealed          false
Total Shares    1
Threshold       1
Version         1.16.2
Build Date      2024-04-22T16:25:54Z
Storage Type    inmem
Cluster Name    vault-cluster-9f5d848c
Cluster ID      ee2cc93d-5d13-e1ed-5365-82c88d7bc0c9
HA Enabled      false
```

Enable KV Engine
```
vault secrets enable -path=secrets kv-v2
vault secrets list
```


Create a custom policy so that we can create a token for tempoary use. 
Instead of modifying the default policy, create and use a specific policy for this purpose.

```sh
vault policy write shared-secrets shared-secrets.hcl
```

