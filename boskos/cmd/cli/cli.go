/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/test-infra/boskos/client"
	"k8s.io/test-infra/boskos/common"
)

type options struct {
	// common, used to create the client
	serverURL string
	ownerName string

	c *client.Client

	acquire acquireOptions
}

type acquireOptions struct {
	requestedType  string
	requestedState string
	targetState    string
	timeout        time.Duration
}

// for test mocking
var exit func(int)

func command() *cobra.Command {
	options := options{}

	root := &cobra.Command{
		Use:   "boskosctl",
		Short: "Boskos command-line client for resource leasing",
		Long: `Boskos provides a flexible resource leasing server.

The boskosctl is a command-line client for this server,
allowing for a user to acquire and release leases from
scripts with a simple interface.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// the root command does nothing, so just print help
			return cmd.Help()
		},
		Args: cobra.NoArgs,
	}
	root.PersistentFlags().StringVar(&options.serverURL, "server-url", "", "URL of the Boskos server")
	root.PersistentFlags().StringVar(&options.ownerName, "owner-name", "", "Name identifying the user of this client")
	for _, flag := range []string{"server-url", "owner-name"} {
		if err := root.MarkPersistentFlagRequired(flag); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	acquire := &cobra.Command{
		Use:   "acquire",
		Short: "Acquire resource leases",
		Long: `Acquire a resource lease, either best-effort or blocking.

Resources can be leased by identifying which type of resource is needed
and what state the resource should be in when leased. Resources will also
transition to a new state upon being leased.

On a successful lease acquisition, the leased resource will be printed in
JSON format for downstream consumption.

Examples:

  # Acquire one clean "my-thing" and mark it dirty when leasing
  $ boskosctl acquire --type my-thing --state clean --target-state dirty

  # Acquire one new "my-thing" and mark it old when leasing, block until successfully leased
  $ boskosctl acquire --type my-thing --state new --target-state old --timeout 30s`,
		Run: func(cmd *cobra.Command, args []string) {
			options.c = client.NewClient(options.ownerName, options.serverURL)
			acquireFunc := options.c.Acquire
			if options.acquire.timeout != 0*time.Second {
				acquireFunc = func(rtype, state, dest string) (resource *common.Resource, e error) {
					ctx := context.Background()
					ctx, cancel := context.WithTimeout(ctx, options.acquire.timeout)

					sig := make(chan os.Signal, 1)
					signal.Notify(sig, os.Interrupt)
					go func() {
						<-sig
						cancel()
					}()

					return options.c.AcquireWait(ctx, rtype, state, dest)
				}
			}
			resource, err := acquireFunc(options.acquire.requestedType, options.acquire.requestedState, options.acquire.targetState)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "failed to acquire a resource: %v\n", err)
				exit(1)
				return
			}
			raw, err := json.Marshal(resource)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "failed to marshal acquired resource: %v\n", err)
				exit(1)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(raw))
		},
		Args: cobra.NoArgs,
	}
	acquire.Flags().StringVar(&options.acquire.requestedType, "type", "", "Type of resource to acquire")
	acquire.Flags().StringVar(&options.acquire.requestedState, "state", "", "State to acquire the resource in")
	acquire.Flags().StringVar(&options.acquire.targetState, "target-state", "", "Move resource to this state after acquiring")
	for _, flag := range []string{"type", "state", "target-state"} {
		if err := acquire.MarkFlagRequired(flag); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	acquire.Flags().DurationVar(&options.acquire.timeout, "timeout", 0*time.Second, "If set, retry this long until the resource has been acquired")
	root.AddCommand(acquire)

	return root
}

func main() {
	exit = os.Exit
	if err := command().Execute(); err != nil {
		fmt.Println(err)
		exit(1)
	}
}
