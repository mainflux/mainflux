// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import "github.com/go-redis/redis/v8"

// Connect create new RedisDB client and connect to RedisDB server.
func Connect(url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	return redis.NewClient(opts), nil
}
