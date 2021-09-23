// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/commands"
)

func createCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		cmd := commands.Command{
			Command:   req.Command,
			ChannelID: req.ChannelID,
			// ExecuteTime: req.ExecuteTime,
		}
		cid, err := svc.CreateCommand(cmd)
		if err != nil {
			return nil, err
		}
		res := createCommandRes{
			ID:      cid,
			created: true,
		}
		return res, nil
	}
}

func viewCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		cid, err := svc.ViewCommand()
		if err != nil {
			return nil, err
		}

		res := viewCommandRes{
			ID: cid,
		}
		return res, nil
	}
}

func listCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListCommands(ctx)
		if err != nil {
			return nil, err
		}

		res := commandsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
				Order:  page.Order,
				Dir:    page.Dir,
			},
			Commands: []viewCommandRes{},
		}
		for _, command := range page.Commands {
			view := viewCommandRes{
				ID:       command.ID,
				Metadata: command.Metadata,
			}
			res.Commands = append(res.Commands, view)
		}
		return res, nil
	}
}

func updateCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		greeting, err := svc.UpdateCommand(req.Secret)
		if err != nil {
			return nil, err
		}

		res := updateCommandRes{
			Greeting: greeting,
		}
		return res, nil
	}
}

func removeCommandEndpoint(svc commands.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeCommandReq)

		err := req.validate()
		if err == commands.ErrNotFound {
			return removeCommandRes{}, nil
		}

		if err != nil {
			return nil, err
		}

		if err := svc.RemoveCommand(commands.Command{}, req.id); err != nil {
			return nil, err
		}
		return removeCommandRes{}, nil
	}
}
