// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	authmocks "github.com/mainflux/mainflux/auth/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/internal/groups"
	gmocks "github.com/mainflux/mainflux/internal/groups/mocks"
	"github.com/mainflux/mainflux/internal/testsutil"
	mflog "github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/api"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	id         = generateUUID(&testing.T{})
	validToken = "token"
)

func newClientServer() (*httptest.Server, *mocks.Repository, *gmocks.Repository, *authmocks.Service) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)

	auth := new(authmocks.Service)
	csvc := users.NewService(cRepo, auth, emailer, phasher, idProvider, passRegex)
	gsvc := groups.NewService(gRepo, idProvider, auth)

	logger := mflog.NewMock()
	mux := chi.NewRouter()
	api.MakeHandler(csvc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), cRepo, gRepo, auth
}

func TestCreateClient(t *testing.T) {
	ts, cRepo, _, _ := newClientServer()
	defer ts.Close()

	user := sdk.User{
		Credentials: sdk.Credentials{Identity: "admin@example.com", Secret: "secret"},
		Status:      mfclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "register new user",
			client:   user,
			response: user,
			token:    token,
			err:      nil,
		},
		{
			desc:     "register existing user",
			client:   user,
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedCreation), http.StatusInternalServerError),
		},
		{
			desc:     "register empty user",
			client:   sdk.User{},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity), http.StatusBadRequest),
		},
		{
			desc: "register a user that can't be marshalled",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
		{
			desc: "register user with invalid identity",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: mocks.WrongID,
					Secret:   "password",
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity), http.StatusBadRequest),
		},
		{
			desc: "register user with empty secret",
			client: sdk.User{
				Name: "emptysecret",
				Credentials: sdk.Credentials{
					Secret: "",
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity), http.StatusBadRequest),
		},
		{
			desc: "register user with empty identity",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "",
					Secret:   secret,
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity), http.StatusBadRequest),
		},
		{
			desc:     "register empty user",
			client:   sdk.User{},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity), http.StatusBadRequest),
		},
		{
			desc: "register user with every field defined",
			client: sdk.User{
				ID:          id,
				Name:        "name",
				Tags:        []string{"tag1", "tag2"},
				Owner:       id,
				Credentials: user.Credentials,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mfclients.EnabledStatus.String(),
			},
			response: sdk.User{
				ID:          id,
				Name:        "name",
				Tags:        []string{"tag1", "tag2"},
				Owner:       id,
				Credentials: user.Credentials,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mfclients.EnabledStatus.String(),
			},
			token: token,
			err:   nil,
		},
	}
	for _, tc := range cases {
		repoCall := cRepo.On("Save", mock.Anything, mock.Anything).Return(tc.response, tc.err)
		rClient, err := mfsdk.CreateUser(tc.client, tc.token)
		tc.response.ID = rClient.ID
		tc.response.Owner = rClient.Owner
		tc.response.CreatedAt = rClient.CreatedAt
		tc.response.UpdatedAt = rClient.UpdatedAt
		rClient.Credentials.Secret = tc.response.Credentials.Secret
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestListClients(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	var cls []sdk.User
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		cl := sdk.User{
			ID:   generateUUID(t),
			Name: fmt.Sprintf("client_%d", i),
			Credentials: sdk.Credentials{
				Identity: fmt.Sprintf("identity_%d", i),
				Secret:   fmt.Sprintf("password_%d", i),
			},
			Metadata: sdk.Metadata{"name": fmt.Sprintf("client_%d", i)},
			Status:   mfclients.EnabledStatus.String(),
		}
		if i == 50 {
			cl.Owner = "clientowner"
			cl.Status = mfclients.DisabledStatus.String()
			cl.Tags = []string{"tag1", "tag2"}
		}
		cls = append(cls, cl)
	}

	cases := []struct {
		desc       string
		token      string
		status     string
		total      uint64
		offset     uint64
		limit      uint64
		name       string
		identifier string
		ownerID    string
		tag        string
		metadata   sdk.Metadata
		err        errors.SDKError
		response   []sdk.User
	}{
		{
			desc:     "get a list of users",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: cls[offset:limit],
		},
		{
			desc:     "get a list of users with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
			response: nil,
		},
		{
			desc:     "get a list of users with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
			response: nil,
		},
		{
			desc:     "get a list of users with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusInternalServerError),
			response: nil,
		},
		{
			desc:     "get a list of users with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusInternalServerError),
			response: []sdk.User(nil),
		},
		{
			desc:       "get a list of users with same identity",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			metadata:   sdk.Metadata{},
			response:   []sdk.User{cls[89]},
		},
		{
			desc:       "get a list of users with same identity and metadata",
			token:      token,
			offset:     0,
			limit:      1,
			err:        nil,
			identifier: Identity,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.User{cls[89]},
		},
		{
			desc:   "list users with given metadata",
			token:  validToken,
			offset: 0,
			limit:  1,
			metadata: sdk.Metadata{
				"name": "client99",
			},
			response: []sdk.User{cls[89]},
			err:      nil,
		},
		{
			desc:     "list users with given name",
			token:    validToken,
			offset:   0,
			limit:    1,
			name:     "client10",
			response: []sdk.User{cls[0]},
			err:      nil,
		},
		{
			desc:     "list users with given owner",
			token:    validToken,
			offset:   0,
			limit:    1,
			ownerID:  "clientowner",
			response: []sdk.User{cls[50]},
			err:      nil,
		},
		{
			desc:     "list users with given status",
			token:    validToken,
			offset:   0,
			limit:    1,
			status:   mfclients.DisabledStatus.String(),
			response: []sdk.User{cls[50]},
			err:      nil,
		},
		{
			desc:     "list users with given tag",
			token:    validToken,
			offset:   0,
			limit:    1,
			tag:      "tag1",
			response: []sdk.User{cls[50]},
			err:      nil,
		},
	}

	for _, tc := range cases {
		pm := sdk.PageMetadata{
			Status:   tc.status,
			Total:    total,
			Offset:   tc.offset,
			Limit:    tc.limit,
			Name:     tc.name,
			OwnerID:  tc.ownerID,
			Metadata: tc.metadata,
			Tag:      tc.tag,
		}

		repoCall1 := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(errors.ErrAuthorization)
		repoCall2 := cRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mfclients.ClientsPage{Page: convertClientPage(pm), Clients: convertClients(tc.response)}, tc.err)
		page, err := mfsdk.Users(pm, validToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Users, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestClient(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	user = sdk.User{
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		clientID string
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "view client successfully",
			response: user,
			token:    validToken,
			clientID: generateUUID(t),
			err:      nil,
		},
		{
			desc:     "view client with an invalid token",
			response: sdk.User{},
			token:    invalidToken,
			clientID: generateUUID(t),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
		{
			desc:     "view client with valid token and invalid client id",
			response: sdk.User{},
			token:    validToken,
			clientID: mocks.WrongID,
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "view client with an invalid token and invalid client id",
			response: sdk.User{},
			token:    invalidToken,
			clientID: mocks.WrongID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		repoCall1 := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall2 := cRepo.On("RetrieveByID", mock.Anything, tc.clientID).Return(convertClient(tc.response), tc.err)
		rClient, err := mfsdk.User(tc.clientID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.clientID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall2.Unset()
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestProfile(t *testing.T) {
	ts, cRepo, _, _ := newClientServer()
	defer ts.Close()

	user = sdk.User{
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}
	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc     string
		token    string
		response sdk.User
		err      errors.SDKError
	}{
		{
			desc:     "view client successfully",
			response: user,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "view client with an invalid token",
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		rClient, err := mfsdk.UserProfile(tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		tc.response.Credentials.Secret = ""
		assert.Equal(t, tc.response, rClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, rClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestUpdateClient(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}

	client1 := user
	client1.Name = "Updated client"

	client2 := user
	client2.Metadata = sdk.Metadata{"role": "test"}
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "update client name with valid token",
			client:   client1,
			response: client1,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   client1,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("Update", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mfsdk.UpdateUser(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientTags(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}

	client1 := user
	client1.Tags = []string{"updatedTag1", "updatedTag2"}

	client2 := user
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      error
	}{
		{
			desc:     "update client name with valid token",
			client:   user,
			response: client1,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   client1,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				ID: generateUUID(t),
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("UpdateTags", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mfsdk.UpdateUserTags(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "UpdateTags", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateTags was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientIdentity(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "updatedclientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
	}

	client2 := user
	client2.Metadata = sdk.Metadata{"role": "test"}
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "update client name with valid token",
			client:   user,
			response: user,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   user,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				ID: generateUUID(t),
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, fmt.Errorf("json: unsupported type: chan int")), http.StatusInternalServerError),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("UpdateIdentity", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mfsdk.UpdateUserIdentity(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "UpdateIdentity", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateIdentity was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateClientSecret(t *testing.T) {
	ts, cRepo, _, _ := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	user.ID = generateUUID(t)
	rclient := user
	rclient.Credentials.Secret, _ = phasher.Hash(user.Credentials.Secret)

	cases := []struct {
		desc      string
		oldSecret string
		newSecret string
		token     string
		response  sdk.User
		err       error
		repoErr   error
	}{
		{
			desc:      "update client secret with valid token",
			oldSecret: user.Credentials.Secret,
			newSecret: "newSecret",
			token:     validToken,
			response:  rclient,
			repoErr:   nil,
			err:       nil,
		},
		{
			desc:      "update client secret with invalid token",
			oldSecret: user.Credentials.Secret,
			newSecret: "newPassword",
			token:     "non-existent",
			response:  sdk.User{},
			repoErr:   errors.ErrAuthentication,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
		{
			desc:      "update client secret with wrong old secret",
			oldSecret: "oldSecret",
			newSecret: "newSecret",
			token:     validToken,
			response:  sdk.User{},
			repoErr:   apiutil.ErrInvalidSecret,
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrInvalidSecret, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := cRepo.On("RetrieveByID", mock.Anything, user.ID).Return(convertClient(tc.response), tc.repoErr)
		repoCall1 := cRepo.On("RetrieveByIdentity", mock.Anything, user.Credentials.Identity).Return(convertClient(tc.response), tc.repoErr)
		repoCall2 := cRepo.On("UpdateSecret", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.repoErr)
		uClient, err := mfsdk.UpdatePassword(tc.oldSecret, tc.newSecret, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, user.ID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByIdentity", mock.Anything, user.Credentials.Identity)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIdentity was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "UpdateSecret", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateSecret was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateClientOwner(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	user = sdk.User{
		ID:          generateUUID(t),
		Name:        "clientname",
		Tags:        []string{"tag1", "tag2"},
		Credentials: sdk.Credentials{Identity: "clientidentity", Secret: secret},
		Metadata:    validMetadata,
		Status:      mfclients.EnabledStatus.String(),
		Owner:       "owner",
	}

	client2 := user
	client2.ID = invalidIdentity

	cases := []struct {
		desc     string
		client   sdk.User
		response sdk.User
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "update client name with valid token",
			client:   user,
			response: user,
			token:    validToken,
			err:      nil,
		},
		{
			desc:     "update client name with invalid token",
			client:   client2,
			response: sdk.User{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
		},
		{
			desc:     "update client name with invalid id",
			client:   client2,
			response: sdk.User{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedUpdate), http.StatusInternalServerError),
		},
		{
			desc: "update a user that can't be marshalled",
			client: sdk.User{
				Credentials: sdk.Credentials{
					Identity: "user@example.com",
					Secret:   "12345678",
				},
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.User{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("UpdateOwner", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.err)
		uClient, err := mfsdk.UpdateUserOwner(tc.client, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, uClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, uClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "UpdateOwner", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("UpdateOwner was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableClient(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	enabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mfclients.EnabledStatus.String()}
	disabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mfclients.DisabledStatus.String()}
	endisabledClient1 := disabledClient1
	endisabledClient1.Status = mfclients.EnabledStatus.String()
	endisabledClient1.ID = testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		id       string
		token    string
		client   sdk.User
		response sdk.User
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "enable disabled client",
			id:       disabledClient1.ID,
			token:    validToken,
			client:   disabledClient1,
			response: endisabledClient1,
			repoErr:  nil,
			err:      nil,
		},
		{
			desc:     "enable enabled client",
			id:       enabledClient1.ID,
			token:    validToken,
			client:   enabledClient1,
			response: sdk.User{},
			repoErr:  sdk.ErrFailedEnable,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedEnable, sdk.ErrFailedEnable), http.StatusInternalServerError),
		},
		{
			desc:     "enable non-existing client",
			id:       mocks.WrongID,
			token:    validToken,
			client:   sdk.User{},
			response: sdk.User{},
			repoErr:  sdk.ErrFailedEnable,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedEnable, errors.ErrNotFound), http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertClient(tc.client), tc.repoErr)
		repoCall2 := cRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.repoErr)
		eClient, err := mfsdk.EnableUser(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, eClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, eClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}

	cases2 := []struct {
		desc     string
		token    string
		status   string
		metadata sdk.Metadata
		response sdk.UsersPage
		size     uint64
	}{
		{
			desc:   "list enabled clients",
			status: mfclients.EnabledStatus.String(),
			size:   2,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, endisabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mfclients.DisabledStatus.String(),
			size:   1,
			response: sdk.UsersPage{
				Users: []sdk.User{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mfclients.AllStatus.String(),
			size:   3,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, disabledClient1, endisabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := sdk.PageMetadata{
			Total:  100,
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertClientsPage(tc.response), nil)
		clientsPage, err := mfsdk.Users(pm, validToken)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(clientsPage.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDisableClient(t *testing.T) {
	ts, cRepo, _, auth := newClientServer()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	enabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client1@example.com", Secret: "password"}, Status: mfclients.EnabledStatus.String()}
	disabledClient1 := sdk.User{ID: testsutil.GenerateUUID(t), Credentials: sdk.Credentials{Identity: "client3@example.com", Secret: "password"}, Status: mfclients.DisabledStatus.String()}
	disenabledClient1 := enabledClient1
	disenabledClient1.Status = mfclients.DisabledStatus.String()
	disenabledClient1.ID = testsutil.GenerateUUID(t)

	cases := []struct {
		desc     string
		id       string
		token    string
		client   sdk.User
		response sdk.User
		repoErr  error
		err      errors.SDKError
	}{
		{
			desc:     "disable enabled client",
			id:       enabledClient1.ID,
			token:    validToken,
			client:   enabledClient1,
			response: disenabledClient1,
			err:      nil,
			repoErr:  nil,
		},
		{
			desc:     "disable disabled client",
			id:       disabledClient1.ID,
			token:    validToken,
			client:   disabledClient1,
			response: sdk.User{},
			repoErr:  sdk.ErrFailedDisable,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedDisable, sdk.ErrFailedDisable), http.StatusInternalServerError),
		},
		{
			desc:     "disable non-existing client",
			id:       mocks.WrongID,
			client:   sdk.User{},
			token:    validToken,
			response: sdk.User{},
			repoErr:  sdk.ErrFailedDisable,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(sdk.ErrFailedDisable, errors.ErrNotFound), http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveByID", mock.Anything, tc.id).Return(convertClient(tc.client), tc.repoErr)
		repoCall2 := cRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(convertClient(tc.response), tc.repoErr)
		dClient, err := mfsdk.DisableUser(tc.id, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, dClient, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, dClient))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.id)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
			ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("ChangeStatus was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}

	cases2 := []struct {
		desc     string
		token    string
		status   string
		metadata sdk.Metadata
		response sdk.UsersPage
		size     uint64
	}{
		{
			desc:   "list enabled clients",
			status: mfclients.EnabledStatus.String(),
			size:   2,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, disenabledClient1},
			},
		},
		{
			desc:   "list disabled clients",
			status: mfclients.DisabledStatus.String(),
			size:   1,
			response: sdk.UsersPage{
				Users: []sdk.User{disabledClient1},
			},
		},
		{
			desc:   "list enabled and disabled clients",
			status: mfclients.AllStatus.String(),
			size:   3,
			response: sdk.UsersPage{
				Users: []sdk.User{enabledClient1, disabledClient1, disenabledClient1},
			},
		},
	}

	for _, tc := range cases2 {
		pm := sdk.PageMetadata{
			Total:  100,
			Offset: 0,
			Limit:  100,
			Status: tc.status,
		}
		repoCall := auth.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
		repoCall1 := cRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertClientsPage(tc.response), nil)
		page, err := mfsdk.Users(pm, validToken)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		size := uint64(len(page.Users))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		repoCall.Unset()
		repoCall1.Unset()
	}
}
