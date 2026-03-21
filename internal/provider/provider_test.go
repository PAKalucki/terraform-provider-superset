// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	testAccSupersetEndpointEnv    = providerEndpointEnv
	testAccSupersetURLEnv         = providerURLEnv
	testAccSupersetUsernameEnv    = providerUsernameEnv
	testAccSupersetPasswordEnv    = providerPasswordEnv
	testAccSupersetAccessTokenEnv = providerAccessTokenEnv
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"superset": providerserver.NewProtocol6WithError(New("test")()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if testAccSupersetEndpoint() == "" {
		t.Fatalf("%s or %s must be set for acceptance tests", testAccSupersetEndpointEnv, testAccSupersetURLEnv)
	}
	client, err := testAccSupersetClient()
	if err != nil {
		t.Fatalf("failed to configure Superset test client: %v", err)
	}

	var availableDatabasesResponse map[string]any
	if err := client.Get(context.Background(), "/api/v1/database/available/", &availableDatabasesResponse); err != nil {
		t.Fatalf("failed to reach Superset acceptance environment: %v", err)
	}
}

func testAccSupersetClient() (*supersetclient.Client, error) {
	endpoint := testAccSupersetEndpoint()
	accessToken := os.Getenv(testAccSupersetAccessTokenEnv)
	username := os.Getenv(testAccSupersetUsernameEnv)
	password := os.Getenv(testAccSupersetPasswordEnv)

	if accessToken == "" && (username == "" || password == "") {
		return nil, fmt.Errorf(
			"set %s or both %s and %s for acceptance tests",
			testAccSupersetAccessTokenEnv,
			testAccSupersetUsernameEnv,
			testAccSupersetPasswordEnv,
		)
	}

	return supersetclient.New(supersetclient.Config{
		Endpoint:    endpoint,
		Username:    username,
		Password:    password,
		AccessToken: accessToken,
	})
}

func testAccProviderConfig() string {
	return `
provider "superset" {}
`
}

func testAccSupersetEndpoint() string {
	endpoint := strings.TrimSpace(os.Getenv(testAccSupersetEndpointEnv))
	if endpoint != "" {
		return endpoint
	}

	return strings.TrimSpace(os.Getenv(testAccSupersetURLEnv))
}
