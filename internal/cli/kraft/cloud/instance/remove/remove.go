// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2022, Unikraft GmbH and The KraftKit Authors.
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
	"kraftkit.sh/log"
)

type RemoveOptions struct {
	Output string `long:"output" short:"o" usage:"Set output format. Options: table,yaml,json,list" default:"table"`
	All    bool   `long:"all" usage:"Remove all instances"`

	metro string
	token string
}

// Remove a KraftCloud instance.
func Remove(ctx context.Context, opts *RemoveOptions, args ...string) error {
	if opts == nil {
		opts = &RemoveOptions{}
	}

	return opts.Run(ctx, args)
}

func NewCmd() *cobra.Command {
	cmd, err := cmdfactory.New(&RemoveOptions{}, cobra.Command{
		Short:   "Remove an instance",
		Use:     "remove [FLAGS] [UUID|NAME [UUID|NAME]...]",
		Aliases: []string{"del", "delete", "rm"},
		Args:    cobra.ArbitraryArgs,
		Example: heredoc.Doc(`
			# Remove a KraftCloud instance by UUID
			$ kraft cloud instance remove fd1684ea-7970-4994-92d6-61dcc7905f2b

			# Remove a KraftCloud instance by name
			$ kraft cloud instance remove my-instance-431342

			# Remove multiple KraftCloud instances
			$ kraft cloud instance remove my-instance-431342 my-instance-other-2313

			# Remove all KraftCloud instances
			$ kraft cloud instance remove --all
		`),
		Long: heredoc.Doc(`
			Remove a KraftCloud instance.
		`),
		Annotations: map[string]string{
			cmdfactory.AnnotationHelpGroup: "kraftcloud-instance",
		},
	})
	if err != nil {
		panic(err)
	}

	return cmd
}

func (opts *RemoveOptions) Pre(cmd *cobra.Command, args []string) error {
	if !opts.All && len(args) == 0 {
		return fmt.Errorf("either specify an instance name or UUID, or use the --all flag")
	}

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

	client := kraftcloud.NewInstancesClient(
		kraftcloud.WithToken(config.GetKraftCloudTokenAuthConfig(*auth)),
	)

	if opts.All {
		instListResp, err := client.WithMetro(opts.metro).List(ctx)
		if err != nil {
			return fmt.Errorf("could not list instances: %w", err)
		}

		log.G(ctx).Infof("Removing %d instance(s)", len(instListResp))

		uuids := make([]string, 0, len(instListResp))
		for _, instItem := range instListResp {
			uuids = append(uuids, instItem.UUID)
		}

		if _, err := client.WithMetro(opts.metro).DeleteByUUIDs(ctx, uuids...); err != nil {
			return fmt.Errorf("removing %d instance(s): %w", len(instListResp), err)
		}
		return nil
	}

	log.G(ctx).Infof("Removing %d instance(s)", len(args))

	allUUIDs := true
	allNames := true
	for _, arg := range args {
		if utils.IsUUID(arg) {
			allNames = false
		} else {
			allUUIDs = false
		}
		if !(allUUIDs || allNames) {
			break
		}
	}

	switch {
	case allUUIDs:
		if _, err := client.WithMetro(opts.metro).DeleteByUUIDs(ctx, args...); err != nil {
			return fmt.Errorf("removing %d instance(s): %w", len(args), err)
		}
	case allNames:
		if _, err := client.WithMetro(opts.metro).DeleteByNames(ctx, args...); err != nil {
			return fmt.Errorf("removing %d instance(s): %w", len(args), err)
		}
	default:
		for _, arg := range args {
			log.G(ctx).Infof("Removing instance %s", arg)

			if utils.IsUUID(arg) {
				_, err = client.WithMetro(opts.metro).DeleteByUUIDs(ctx, arg)
			} else {
				_, err = client.WithMetro(opts.metro).DeleteByNames(ctx, arg)
			}

			if err != nil {
				return fmt.Errorf("could not remove instance %s: %w", arg, err)
			}
		}
	}

	return nil
}
