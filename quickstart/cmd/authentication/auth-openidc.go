/*
This is an example application to demonstrate querying the user info endpoint.
*/
package main

import (
	"encoding/json"
	oidc "github.com/coreos/go-oidc"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"log"
	"net/http"
)

func main() {

	ctx := context.Background()

	// get TenantID, ClientID and ClientSecret from Azure portal during app registration
	provider, err := oidc.NewProvider(ctx, "https://login.microsoftonline.com/yourTenantID/v2.0")
	if err != nil {
		log.Fatal(err)
	}

	config := oauth2.Config{
		ClientID:     "yourClientID",
		ClientSecret: "yourClientSecret",
		Endpoint:     provider.Endpoint(),
		RedirectURL:  "http://localhost:8080/auth/callback",
		// "openid" is a required scope for OpenID Connect flows
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	state := "foobar" // this is typically the page or tab a user was on before the sign-in

	// this handler initiates the sign-in process by redirecting to the provider authorization endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.AuthCodeURL(state), http.StatusFound)
	})

	// this handler validates the state, so it hasn't changed during the communication process,
	// then exchanges the authorization code for the access_token using clientId/clientSecret,
	// finally, extracts user info from the id_token and returns everything back to browser
	http.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state did not match", http.StatusBadRequest)
			return
		}

		oauth2Token, err := config.Exchange(ctx, r.URL.Query().Get("code"))
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
		if err != nil {
			http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
			return
		}

		resp := struct {
			OAuth2Token *oauth2.Token
			UserInfo    *oidc.UserInfo
		}{oauth2Token, userInfo}
		data, err := json.MarshalIndent(resp, "", "    ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(data)
	})

	log.Printf("listening on http://%s/", "127.0.0.1:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
