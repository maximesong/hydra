package server

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/ory/hydra/client"
	"github.com/ory/hydra/config"
	"github.com/ory/hydra/pkg"
	"github.com/ory/ladon"
)

func (h *Handler) createRootIfNewInstall(c *config.Config) {
	ctx := c.Context()

	clients, err := h.Clients.Manager.GetClients()
	pkg.Must(err, "Could not fetch client list: %s", err)
	if len(clients) != 0 {
		return
	}

	rs, err := pkg.GenerateSecret(16)
	pkg.Must(err, "Could notgenerate secret because %s", err)
	secret := string(rs)

	id := ""
	forceRoot := os.Getenv("FORCE_ROOT_CLIENT_CREDENTIALS")
	if forceRoot != "" {
		credentials := strings.Split(forceRoot, ":")
		if len(credentials) == 2 {
			id = credentials[0]
			secret = credentials[1]
		} else {
			logrus.Warnln("You passed malformed root client credentials, falling back to random values.")
		}
	}

	logrus.Warn("No clients were found. Creating a temporary root client...")
	root := &client.Client{
		ID:            id,
		Name:          "This temporary client is generated by hydra and is granted all of hydra's administrative privileges. It must be removed when everything is set up.",
		ResponseTypes: []string{"id_token", "code", "token"},
		GrantTypes:    []string{"implicit", "refresh_token", "authorization_code", "password", "client_credentials"},
		Scope:         "hydra openid offline",
		RedirectURIs:  []string{"http://localhost:4445/callback"},
		Secret:        secret,
	}

	err = h.Clients.Manager.CreateClient(root)
	pkg.Must(err, "Could not create temporary root because %s", err)
	err = ctx.LadonManager.Create(&ladon.DefaultPolicy{
		Description: "This is a policy created by hydra and issued to the first client. It grants all of hydra's administrative privileges to the client and enables the client_credentials response type.",
		Subjects:    []string{root.GetID()},
		Effect:      ladon.AllowAccess,
		Resources:   []string{"rn:hydra:<.*>"},
		Actions:     []string{"<.*>"},
	})
	pkg.Must(err, "Could not create admin policy because %s", err)

	c.ClientID = root.ID
	c.ClientSecret = string(secret)

	logrus.Infoln("Temporary root client created.")
	if forceRoot == "" {
		logrus.Infof("client_id: %s", root.GetID())
		logrus.Infof("client_secret: %s", string(secret))
		logrus.Warn("WARNING: YOU MUST delete this client once in production, as credentials may have been leaked in your logfiles.")
	}
}
