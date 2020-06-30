package commands

import (
	"context"

	"github.com/spf13/cobra"
)

type Command interface {
	Build(context.Context) *cobra.Command
	Stop(context.Context)
}
