// SPDX-License-Identifier: BSD-3-Clause
// Copyright (c) 2023, Unikraft GmbH and The KraftKit Authors.
// Licensed under the BSD-3-Clause License (the "License").
// You may not use this file except in compliance with the License.

package stop

import (
	"context"
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"

	kraftcloud "sdk.kraft.cloud"

	"kraftkit.sh/cmdfactory"
	"kraftkit.sh/config"
	"kraftkit.sh/internal/cli/kraft/cloud/utils"
	"kraftkit.sh/log"
)

type StopOptions struct {
	DrainTimeout time.Duration `local:"true" long:"drain-timeout" short:"d" usage:"Timeout for the instance to stop (ms/s/m/h)"`
	Output       string        `long:"output" short:"o" usage:"Set output format. Options: table,yaml,json,list" default:"table"`
	All          bool          `long:"all" usage:"Stop all instances"`
	Metro        string        `noattribute:"true"`
	Token        string        `noattribute:"true"`
}

// Stop a KraftCloud instance.
func Stop(ctx context.Context, opts *StopOptions, args ...string) error {
	if opts == nil {
		opts = &StopOptions{}
	}

	return opts.Run(ctx, args)
}

func NewCmd() *cobra.Command {
	cmd, err := cmdfactory.New(&StopOptions{}, cobra.Command{
		Short:   "Stop an instance",
		Use:     "stop [FLAGS] [UUID|NAME [UUID|NAME]...]",
		Args:    cobra.ArbitraryArgs,
		Aliases: []string{"st"},
		Example: heredoc.Doc(`
			# Stop a KraftCloud instance by UUID
			$ kraft cloud instance stop 77d0316a-fbbe-488d-8618-5bf7a612477a

			# Stop a KraftCloud instance by name
			$ kraft cloud instance stop my-instance-431342

			# Stop multiple KraftCloud instances
			$ kraft cloud instance stop my-instance-431342 my-instance-other-2313

			# Stop all KraftCloud instances
			$ kraft cloud instance stop --all
		`),
		Long: heredoc.Doc(`
			Stop a KraftCloud instance.
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

func (opts *StopOptions) Pre(cmd *cobra.Command, args []string) error {
	if !opts.All && len(args) == 0 {
		return fmt.Errorf("either specify an instance UUID or --all flag")
	}

	err := utils.PopulateMetroToken(cmd, &opts.Metro, &opts.Token)
	if err != nil {
		return fmt.Errorf("could not populate metro and token: %w", err)
	}

	return nil
}

func (opts *StopOptions) Run(ctx context.Context, args []string) error {
	auth, err := config.GetKraftCloudAuthConfig(ctx, opts.Token)
	if err != nil {
		return fmt.Errorf("could not retrieve credentials: %w", err)
	}

	client := kraftcloud.NewInstancesClient(
		kraftcloud.WithToken(config.GetKraftCloudTokenAuthConfig(*auth)),
	)

	if opts.DrainTimeout < time.Millisecond {
		return fmt.Errorf("drain timeout must be at least 1ms")
	}

	timeout := int(opts.DrainTimeout / time.Millisecond)

	if opts.All {
		instListResp, err := client.WithMetro(opts.Metro).List(ctx)
		if err != nil {
			return fmt.Errorf("could not list instances: %w", err)
		}

		log.G(ctx).Infof("Stopping %d instance(s)", len(instListResp))

		uuids := make([]string, 0, len(instListResp))
		for _, instItem := range instListResp {
			uuids = append(uuids, instItem.UUID)
		}

		if _, err = client.WithMetro(opts.Metro).StopByUUIDs(ctx, timeout, uuids...); err != nil {
			log.G(ctx).Error("could not stop instance: %w", err)
		}

		return nil
	}

	log.G(ctx).Infof("Stopping %d instance(s)", len(args))

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
		if _, err := client.WithMetro(opts.Metro).StopByUUIDs(ctx, timeout, args...); err != nil {
			return fmt.Errorf("stopping %d instance(s): %w", len(args), err)
		}
	case allNames:
		if _, err := client.WithMetro(opts.Metro).StopByNames(ctx, timeout, args...); err != nil {
			return fmt.Errorf("stopping %d instance(s): %w", len(args), err)
		}
	default:
		for _, arg := range args {
			log.G(ctx).Infof("Stopping instance %s", arg)

			if utils.IsUUID(arg) {
				_, err = client.WithMetro(opts.Metro).StopByUUIDs(ctx, timeout, arg)
			} else {
				_, err = client.WithMetro(opts.Metro).StopByNames(ctx, timeout, arg)
			}

			if err != nil {
				return fmt.Errorf("could not stop instance %s: %w", arg, err)
			}
		}
	}

	return nil
}
