// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package commands

import (
	"errors"
	"fmt"
)

var (
	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ViewCommands compares a given string with secret
	CreateCommand(commands ...Command) (string, error)
	ViewCommand(commands ...Command) (string, error)
	ListCommand(commands ...Command) (string, error)
	UpdateCommand(commands ...Command) (string, error)
	RemoveCommand(commands ...Command) error
}

type commandsService struct {
}

var _ Service = (*commandsService)(nil)

// New instantiates the commands service implementation.
func New(secret string) Service {
	return &commandsService{}
}
func (ks *commandsService) CreateCommand(commands ...Command) (string, error) {
	fmt.Println("proba")
	return "Create Command", nil
}

func (ks *commandsService) ViewCommand(commands ...Command) (string, error) {
	return "View Command", nil
}

func (ks *commandsService) ListCommand(commands ...Command) (string, error) {
	return "Command list", nil
}

func (ks *commandsService) UpdateCommand(commands ...Command) (string, error) {
	return "Command updated", nil
}

func (ks *commandsService) RemoveCommand(commands ...Command) (string, error) {
	return "Command removed", nil
}
