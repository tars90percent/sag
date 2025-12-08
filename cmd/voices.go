package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/steipete/sag/internal/elevenlabs"

	"github.com/spf13/cobra"
)

type voicesOptions struct {
	search string
	limit  int
}

func init() {
	opts := voicesOptions{
		limit: 100,
	}

	cmd := &cobra.Command{
		Use:   "voices",
		Short: "List available ElevenLabs voices",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return ensureAPIKey()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client := elevenlabs.NewClient(cfg.APIKey, cfg.BaseURL)
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			voices, err := client.ListVoices(ctx, opts.search)
			if err != nil {
				return err
			}

			if opts.limit > 0 && len(voices) > opts.limit {
				voices = voices[:opts.limit]
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "VOICE ID\tNAME\tCATEGORY\n")
			for _, v := range voices {
				fmt.Fprintf(w, "%s\t%s\t%s\n", v.VoiceID, v.Name, v.Category)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&opts.search, "search", "", "Filter voices by name (server-side when supported)")
	cmd.Flags().IntVar(&opts.limit, "limit", opts.limit, "Maximum rows to display (0 = all)")
	rootCmd.AddCommand(cmd)
}
