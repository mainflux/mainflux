// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux/commands"
	log "github.com/mainflux/mainflux/logger"
)

var _ commands.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    commands.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc commands.Service, logger log.Logger) commands.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateCommand(cmds ...commands.Command) (response string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method CreateCommands for cmds %s took %s to complete", cmds, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateCommand(cmds...)
}

func (lm *loggingMiddleware) ViewCommand(cmds ...commands.Command) (response string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method ViewCommand for secret %s took %s to complete", cmds, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewCommand(cmds...)
}

func (lm *loggingMiddleware) ListCommand(cmds ...commands.Command) (response string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method ListCommand for secret %s took %s to complete", cmds, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListCommand(cmds...)
}

func (lm *loggingMiddleware) UpdateCommand(cmds ...commands.Command) (response string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method UpdateCommand for secret %s took %s to complete", cmds, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateCommand(cmds...)
}

func (lm *loggingMiddleware) RemoveCommand(cmds ...commands.Command) (response string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method RemoveCommand for secret %s took %s to complete", cmds, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveCommand(cmds...)
}
