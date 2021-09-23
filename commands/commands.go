package commands

import (
	"context"
	"time"
)

// const (
// 	maxLocalLen  = 64
// 	maxDomainLen = 255
// 	maxTLDLen    = 24 // longest TLD currently in existence

// 	atSeparator  = "@"
// 	dotSeparator = "."
// )

// Metadata to be used for mainflux thing or channel for customized
// describing of particular thing or channel.
type Metadata map[string]interface{}

// User represents a Mainflux user account. Each user is identified given its
// email and password.
type Command struct {
	ID          string
	Owner       string
	Name        string
	Command     string
	ChannelID   string
	ExecuteTime time.Time
	Metadata    Metadata
}

// UserRepository specifies an account persistence API.
type CommandRepository interface {
	// Save persists the user account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, c Command) (string, error)

	// Update updates the user metadata.
	Update(ctx context.Context, u Command) error

	// RetrieveByID retrieves user by its unique identifier ID.
	RetrieveByID(ctx context.Context, id string) (Command, error)

	// RetrieveAll retrieves all users for given array of userIDs.
	// RetrieveAll(ctx context.Context, offset, limit uint64, commandIDs []string, email string, m Metadata) (CommandPage, error)

	// Remove removes the thing having the provided identifier, that is owned
	// by the specified user.
	Remove(ctx context.Context, owner, id string) error
}
