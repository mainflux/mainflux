// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/auth"
)

type MockSubjectSet struct {
	Object   string
	Relation string
}

type policyAgentMock struct {
	authzDB map[string][]MockSubjectSet
}

// NewKetoMock returns a mock service for Keto.
// This mock is not implemented yet.
func NewKetoMock(db map[string][]MockSubjectSet) auth.PolicyAgent {
	return &policyAgentMock{db}
}

func (k *policyAgentMock) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	ssList := k.authzDB[pr.Subject]
	for _, ss := range ssList {
		if ss.Object == pr.Object && ss.Relation == pr.Relation {
			return nil
		}
	}
	return auth.ErrAuthorization
}

func (k *policyAgentMock) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	k.authzDB[pr.Subject] = append(k.authzDB[pr.Subject], MockSubjectSet{Object: pr.Object, Relation: pr.Relation})
	return nil
}

func (k *policyAgentMock) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	k.authzDB[pr.Subject] = append(k.authzDB[pr.Subject], MockSubjectSet{Object: pr.Object, Relation: pr.Relation})
	return nil
}
