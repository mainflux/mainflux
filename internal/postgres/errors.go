// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mainflux/mainflux/pkg/errors"
)

// Postgres error codes:
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	errDuplicate  = "23505" // unique_violation
	errTruncation = "22001" // string_data_right_truncation
	errFK         = "23503" // foreign_key_violation
	errInvalid    = "22P02" // invalid_text_representation
)

func HandleError(err, wrapper error) error {
	pqErr, ok := err.(*pgconn.PgError)
	if ok {
		switch pqErr.Code {
		case errDuplicate:
			return errors.Wrap(errors.ErrConflict, err)
		case errInvalid, errTruncation:
			return errors.Wrap(errors.ErrMalformedEntity, err)
		case errFK:
			return errors.Wrap(errors.ErrCreateEntity, err)
		}
	}

	return errors.Wrap(wrapper, err)
}
