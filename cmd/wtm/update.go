package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Mlcarvalho1/wtm/internal/update"
	"github.com/Mlcarvalho1/wtm/internal/version"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print wtm's version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("wtm v%s\n", version.Version)
			return nil
		},
	}
}

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "update",
		Short:        "Check for a newer wtm release and reinstall it",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("current version: v%s\n", version.Version)
			fmt.Println("checking for a newer release...")

			latest, err := update.CheckLatest()
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}
			if latest == "" {
				fmt.Println("no releases published yet — you're already on the latest version.")
				return nil
			}
			if version.Compare(latest, version.Version) <= 0 {
				fmt.Printf("wtm is up to date (v%s)\n", version.Version)
				return nil
			}

			fmt.Printf("new version available: v%s (current: v%s)\n", latest, version.Version)
			fmt.Println("reinstalling...")
			if err := update.Install(latest); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			fmt.Printf("updated to v%s — restart wtm to use it.\n", latest)
			return nil
		},
	}
}
