// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2023, Unikraft GmbH and The KraftKit Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package remove

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	kraftcloud "sdk.kraft.cloud"

	"kraftkit.sh/cmdfactory"
	"kraftkit.sh/config"
	"kraftkit.sh/internal/cli/kraft/cloud/utils"
	"kraftkit.sh/iostreams"
)

type RemoveOptions struct {
	metro string
	token string
}

// Remove a KraftCloud persistent volume.
func Remove(ctx context.Context, opts *RemoveOptions, args ...string) error {
	if opts == nil {
		opts = &RemoveOptions{}
	}

	return opts.Run(ctx, args)
}

func NewCmd() *cobra.Command {
	cmd, err := cmdfactory.New(&RemoveOptions{}, cobra.Command{
		Short:   "Permanently delete a persistent volume",
		Use:     "remove UUID [UUID [...]]",
		Args:    cobra.MinimumNArgs(1),
		Aliases: []string{"rm"},
		Long: heredoc.Doc(`
			Permanently delete a persistent volume.
		`),
		Example: heredoc.Doc(`
			# Delete three persistent volumes
			$ kraft cloud volume rm UUID1 UUID2 UUID3
		`),
		Annotations: map[string]string{
			cmdfactory.AnnotationHelpGroup: "kraftcloud-vol",
		},
	})
	if err != nil {
		panic(err)
	}

	return cmd
}

func (opts *RemoveOptions) Pre(cmd *cobra.Command, _ []string) error {
	err := utils.PopulateMetroToken(cmd, &opts.metro, &opts.token)
	if err != nil {
		return fmt.Errorf("could not populate metro and token: %w", err)
	}

	return nil
}

func (opts *RemoveOptions) Run(ctx context.Context, args []string) error {
	auth, err := config.GetKraftCloudAuthConfig(ctx, opts.token)
	if err != nil {
		return fmt.Errorf("could not retrieve credentials: %w", err)
	}

	client := kraftcloud.NewVolumesClient(
		kraftcloud.WithToken(config.GetKraftCloudTokenAuthConfig(*auth)),
	)

	for _, arg := range args {
		if utils.IsUUID(arg) {
			_, err = client.WithMetro(opts.metro).DeleteByUUID(ctx, arg)
		} else {
			_, err = client.WithMetro(opts.metro).DeleteByName(ctx, arg)
		}
		if err != nil {
			return fmt.Errorf("could not delete volume: %w", err)
		}

		_, err = fmt.Fprintln(iostreams.G(ctx).Out, arg)
		if err != nil {
			return fmt.Errorf("could not write volume UUID: %w", err)
		}
	}

	return nil
}
