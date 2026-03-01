# Authelia OAuth2 Secrets

## Populate Secrets for a New App

Use the following script to generate and store OAuth2 secrets for a new application:

```fish
# Set the application name
set APP AUTOBRR_TEST

# Generate a client ID
set CLIENT_ID (podman run --rm docker.io/authelia/authelia:latest authelia crypto rand --length 72 --charset rfc3986 | awk '{print $3}')

# Generate client secret and hash
set CLIENT_SECRET_OUTPUT (podman run docker.io/authelia/authelia:latest authelia crypto hash generate argon2 --random --random.length 64 --random.charset alphanumeric)

# Extract CLIENT_SECRET and CLIENT_SECRET_HASH
set CLIENT_SECRET (echo $CLIENT_SECRET_OUTPUT | grep "Random Password" | cut -d ' ' -f 3)
set CLIENT_SECRET_HASH (echo $CLIENT_SECRET_OUTPUT | awk -F'Digest: ' '/Digest/ {print $2}' | xargs)

# Store credentials in 1Password
op item edit authelia "$APP"_OAUTH_CLIENT_ID="$CLIENT_ID"
op item edit authelia "$APP"_OAUTH_CLIENT_SECRET="$CLIENT_SECRET"
op item edit authelia "$APP"_OAUTH_CLIENT_SECRET_HASH="$CLIENT_SECRET_HASH"
```
