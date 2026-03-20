// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	supersetclient "terraform-provider-superset/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
)

const (
	testAccSupersetEndpointEnv    = "SUPERSET_ENDPOINT"
	testAccSupersetUsernameEnv    = "SUPERSET_USERNAME"
	testAccSupersetPasswordEnv    = "SUPERSET_PASSWORD"
	testAccSupersetAccessTokenEnv = "SUPERSET_ACCESS_TOKEN"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during acceptance testing.
// The factory function is called for each Terraform CLI command to create a provider
// server that the CLI can connect to and interact with.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"superset": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccProtoV6ProviderFactoriesWithEcho includes the echo provider alongside the superset provider.
// It allows for testing assertions on data returned by an ephemeral resource during Open.
// The echoprovider is used to arrange tests by echoing ephemeral data into the Terraform state.
// This lets the data be referenced in test assertions with state checks.
var testAccProtoV6ProviderFactoriesWithEcho = map[string]func() (tfprotov6.ProviderServer, error){
	"superset": providerserver.NewProtocol6WithError(New("test")()),
	"echo":     echoprovider.NewProviderServer(),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv(testAccSupersetEndpointEnv) == "" {
		t.Fatalf("%s must be set for acceptance tests", testAccSupersetEndpointEnv)
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
	endpoint := os.Getenv(testAccSupersetEndpointEnv)
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
	endpoint := os.Getenv(testAccSupersetEndpointEnv)
	accessToken := os.Getenv(testAccSupersetAccessTokenEnv)

	if accessToken != "" {
		return fmt.Sprintf(`
provider "superset" {
  endpoint     = %q
  access_token = %q
}
`, endpoint, accessToken)
	}

	return fmt.Sprintf(`
provider "superset" {
  endpoint = %q
  username = %q
  password = %q
}
`, endpoint, os.Getenv(testAccSupersetUsernameEnv), os.Getenv(testAccSupersetPasswordEnv))
}

func testAccExpectedDatabaseEngine() string {
	return "postgresql"
}
