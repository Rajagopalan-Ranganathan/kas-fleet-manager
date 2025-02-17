package integration

import (
	"github.com/antihax/optional"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"net/http"
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"github.com/golang-jwt/jwt/v4"

	"github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
)

func TestServiceAccounts_GetByClientID(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	opts := public.GetServiceAccountsOpts{
		ClientId: optional.NewString("srvc-acct-12345678-1234-1234-1234-123456789012"),
	}

	list, resp, err := client.SecurityApi.GetServiceAccounts(ctx, &opts)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(list.Items).To(HaveLen(0))

	// Create one Service Account
	r := public.ServiceAccountRequest{
		Name:        "managed-service-integration-test-account",
		Description: "created by the managed service integration tests",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))

	// Find service account by clientid
	opts.ClientId = optional.NewString(sa.ClientId)

	list, resp, err = client.SecurityApi.GetServiceAccounts(ctx, &opts)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(list.Items).To(HaveLen(1))
	Expect(list.Items[0].ClientId == sa.ClientId)
	Expect(list.Items[0].Id == sa.Id)
	Expect(list.Items[0].Name == sa.Name)
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	Expect(err).ShouldNot(HaveOccurred())
}

func TestServiceAccounts_Success(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	//verify list
	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx, nil)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	currTime := time.Now().Format(time.RFC3339)
	createdAt, _ := time.Parse(time.RFC3339, currTime)
	//verify create
	r := public.ServiceAccountRequest{
		Name:        "managed-service-integration-test-account",
		Description: "created by the managed service integration tests",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.CreatedBy).Should(Equal(account.Username()))
	Expect(sa.Id).NotTo(BeEmpty())
	Expect(sa.CreatedAt).Should(BeTemporally(">=", createdAt))

	// verify get by id
	id := sa.Id
	sa, resp, err = client.SecurityApi.GetServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(err).ShouldNot(HaveOccurred())
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.CreatedBy).NotTo(BeEmpty())
	Expect(sa.CreatedBy).Should(Equal(account.Username()))
	Expect(sa.Id).NotTo(BeEmpty())
	Expect(sa.CreatedAt).Should(BeTemporally(">=", createdAt))

	//verify reset
	oldSecret := sa.ClientSecret
	sa, _, err = client.SecurityApi.ResetServiceAccountCreds(ctx, id)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(Equal(oldSecret))
	Expect(sa.CreatedBy).Should(Equal(account.Username()))
	Expect(sa.CreatedBy).NotTo(BeEmpty())
	Expect(sa.CreatedAt).Should(BeTemporally(">=", createdAt))

	//verify delete
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, id)
	Expect(err).ShouldNot(HaveOccurred())

	// verify deletion of non-existent service account throws http status code 404
	_, resp, _ = client.SecurityApi.DeleteServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	f := false
	accounts, _, _ := client.SecurityApi.GetServiceAccounts(ctx, nil)
	for _, a := range accounts.Items {
		if a.Id == id {
			f = true
		}
	}
	Expect(f).To(BeFalse())
}

func TestServiceAccounts_IncorrectOCMIssuer_AuthzFailure(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      "invalidiss",
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx, nil)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
}

func TestServiceAccounts_CorrectTokenIssuer_AuthzSuccess(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	claims := jwt.MapClaims{
		"iss":      test.TestServices.ServerConfig.TokenIssuerURL,
		"org_id":   account.Organization().ExternalID(),
		"username": account.Username(),
	}

	ctx := h.NewAuthenticatedContext(account, claims)

	_, resp, err := client.SecurityApi.GetServiceAccounts(ctx, nil)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
}

func TestServiceAccounts_InputValidation(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	//length check
	r := public.ServiceAccountRequest{
		Name:        "length-more-than-50-is-not-allowed-managed-service-integration-test",
		Description: "created by the managed service integration",
	}
	_, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//xss prevention
	r = public.ServiceAccountRequest{
		Name:        "<script>alert(\"TEST\");</script>",
		Description: "created by the managed service integration",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//description length can not be more than 255
	r = public.ServiceAccountRequest{
		Name:        "test-svc-1",
		Description: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//min length required for name
	r = public.ServiceAccountRequest{
		Name:        "",
		Description: "test",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	//min length required is not required for desc
	r = public.ServiceAccountRequest{
		Name:        "test",
		Description: "",
	}
	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Id).NotTo(BeEmpty())

	//verify delete
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	Expect(err).ShouldNot(HaveOccurred())

	// certain characters are allowed in the description
	r = public.ServiceAccountRequest{
		Name:        "test",
		Description: "Created by the managed-services integration tests.,",
	}
	sa, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(sa.ClientId).NotTo(BeEmpty())
	Expect(sa.ClientSecret).NotTo(BeEmpty())
	Expect(sa.Id).NotTo(BeEmpty())

	//verify delete
	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
	Expect(err).ShouldNot(HaveOccurred())

	//xss prevention
	r = public.ServiceAccountRequest{
		Name:        "service-account-1",
		Description: "created by the managed service integration #$@#$#@$#@$$#",
	}
	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
	Expect(err).Should(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

	// verify malformed  id
	id := faker.ID
	_, resp, err = client.SecurityApi.GetServiceAccountById(ctx, id)
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
	Expect(err).Should(HaveOccurred())
}

func TestServiceAccounts_SsoProvider_MAS_SSO(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(c *keycloak.KeycloakConfig) {
		c.SelectSSOProvider = keycloak.MAS_SSO
	})
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)
	sp, resp, err := client.SecurityApi.GetSsoProviders(ctx)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(sp.BaseUrl).To(Equal(test.TestServices.KeycloakConfig.BaseURL))
	Expect(sp.TokenUrl).To(Equal(test.TestServices.KeycloakConfig.BaseURL + "/auth/realms/rhoas/protocol/openid-connect/token"))
}

func TestServiceAccounts_SsoProvider_SSO(t *testing.T) {
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	h, client, teardown := test.NewKafkaHelperWithHooks(t, ocmServer, func(c *keycloak.KeycloakConfig) {
		c.SelectSSOProvider = keycloak.REDHAT_SSO
	})
	defer teardown()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account, nil)

	sp, resp, err := client.SecurityApi.GetSsoProviders(ctx)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(sp.BaseUrl).To(Equal("https://sso.redhat.com"))
	Expect(sp.TokenUrl).To(Equal("https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/token"))
}

//Todo Temporary commenting out the test
//func TestServiceAccount_CreationLimits(t *testing.T) {
//	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
//	defer ocmServer.Close()
//
//	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
//	// ocm
//	h, client, teardown := test.NewKafkaHelper(t, ocmServer)
//	defer teardown()
//
//	account := h.NewRandAccount()
//	ctx := h.NewAuthenticatedContext(account, nil)
//
//	r := public.ServiceAccountRequest{
//		Name:        "test-account-acc-1",
//		Description: "created by the managed service integration tests",
//	}
//
//	sa, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
//	Expect(err).ShouldNot(HaveOccurred())
//	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
//	Expect(sa.ClientId).NotTo(BeEmpty())
//	Expect(sa.ClientSecret).NotTo(BeEmpty())
//	Expect(sa.CreatedBy).Should(Equal(account.Username()))
//	Expect(sa.Id).NotTo(BeEmpty())
//
//	r = public.ServiceAccountRequest{
//		Name:        "test-account-acc-2",
//		Description: "created by the managed service integration tests",
//	}
//	sa2, resp, err := client.SecurityApi.CreateServiceAccount(ctx, r)
//	Expect(err).ShouldNot(HaveOccurred())
//	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
//	Expect(sa2.ClientId).NotTo(BeEmpty())
//	Expect(sa2.ClientSecret).NotTo(BeEmpty())
//	Expect(sa2.CreatedBy).Should(Equal(account.Username()))
//	Expect(sa2.Id).NotTo(BeEmpty())
//
//	// limit has reached for 2 service accounts
//	r = public.ServiceAccountRequest{
//		Name:        "test-account-acc-3",
//		Description: "created by the managed service integration tests",
//	}
//	_, resp, err = client.SecurityApi.CreateServiceAccount(ctx, r)
//	Expect(err).Should(HaveOccurred())
//	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
//
//	//cleanup
//	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa.Id)
//	Expect(err).ShouldNot(HaveOccurred())
//
//	_, _, err = client.SecurityApi.DeleteServiceAccountById(ctx, sa2.Id)
//	Expect(err).ShouldNot(HaveOccurred())
//}
