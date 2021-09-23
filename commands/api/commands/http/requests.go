// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/mainflux/mainflux/commands"
)

type apiReq interface {
	validate() error
}

type createCommandReq struct {
	Command     string `json:"command"`
	Name        string `josn:"name"`
	ChannelID   string `json:"channel_id"`
	ExecuteTime string `json:"execute_time"`
}

func (req createCommandReq) validate() error {
	if req.Command == "" {
		return commands.ErrMalformedEntity
	}
	return nil
}

type listCommandReq struct {
	Secret string `json:"secret"`
}

func (req listCommandReq) validate() error {
	if req.Secret == "" {
		return commands.ErrMalformedEntity
	}

	return nil
}

type updateCommandReq struct {
	ID       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateCommandReq) validate() error {
	if req.ID == "" {
		return commands.ErrMalformedEntity
	}

	return nil
}

type removeCommandReq struct {
	ID string
}

func (req removeCommandReq) validate() error {
	if req.ID == "" {
		return commands.ErrMalformedEntity
	}

	return nil
}
