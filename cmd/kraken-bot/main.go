package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"

	"kraken-bot/internal/bot"
	"kraken-bot/internal/config"
	"kraken-bot/internal/kraken"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	if len(args) == 0 {
		return runBot(output)
	}
	switch args[0] {
	case "funding-methods":
		return runFundingMethods(args[1:], output)
	case "funding-addresses":
		return runFundingAddresses(args[1:], output)
	case "help", "-h", "--help":
		printUsage(output)
		return nil
	default:
		return fmt.Errorf("unknown command %q; run kraken-bot help", args[0])
	}
}

func runBot(output io.Writer) error {
	cfg, err := config.Load(".env")
	if err != nil {
		return err
	}

	client, err := newClient(cfg.KrakenConfig)
	if err != nil {
		return err
	}

	ctx, stop := commandContext()
	defer stop()

	runner := bot.Runner{
		API:                      client,
		Output:                   output,
		MinimumWithdrawal:        cfg.MinimumWithdrawal,
		MinimumWithdrawalDisplay: cfg.MinimumWithdrawalDisplay,
		USDFundingMethodID:       cfg.USDFundingMethodID,
		USDFundingAddressID:      cfg.USDFundingAddressID,
		SettlementDelay:          cfg.SettlementDelay,
	}
	return runner.Run(ctx)
}

func runFundingMethods(args []string, output io.Writer) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: kraken-bot funding-methods [deposit|withdraw]")
	}
	directions := []string{"deposit", "withdraw"}
	if len(args) == 1 {
		if args[0] != "deposit" && args[0] != "withdraw" {
			return fmt.Errorf("funding direction must be deposit or withdraw")
		}
		directions = []string{args[0]}
	}

	client, err := loadDiscoveryClient()
	if err != nil {
		return err
	}
	ctx, stop := commandContext()
	defer stop()

	type row struct {
		direction string
		method    kraken.FundingMethod
	}
	var rows []row
	for _, direction := range directions {
		methods, err := client.ListFundingMethods(ctx, direction)
		if err != nil {
			return fmt.Errorf("list %s funding methods: %w", direction, err)
		}
		for _, method := range methods {
			rows = append(rows, row{direction: direction, method: method})
		}
	}
	sort.Slice(rows, func(left, right int) bool {
		leftKey := rows[left].direction + rows[left].method.Asset.Name + rows[left].method.MethodName + rows[left].method.MethodID
		rightKey := rows[right].direction + rows[right].method.Asset.Name + rows[right].method.MethodName + rows[right].method.MethodID
		return leftKey < rightKey
	})

	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "DIRECTION\tASSET\tCLASS\tMETHOD\tNETWORK\tMINIMUM\tMETHOD_ID")
	for _, row := range rows {
		network := "-"
		if row.method.Network != nil && row.method.Network.NetworkName != "" {
			network = row.method.Network.NetworkName
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			row.direction,
			cell(row.method.Asset.Name),
			cell(row.method.Asset.Class),
			cell(row.method.MethodName),
			cell(network),
			cell(row.method.MinimumAmount),
			cell(row.method.MethodID),
		)
	}
	return writer.Flush()
}

func runFundingAddresses(args []string, output io.Writer) error {
	if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("usage: kraken-bot funding-addresses METHOD_ID")
	}
	client, err := loadDiscoveryClient()
	if err != nil {
		return err
	}
	ctx, stop := commandContext()
	defer stop()
	addresses, err := client.ListFundingAddresses(ctx, args[0])
	if err != nil {
		return fmt.Errorf("list funding addresses: %w", err)
	}
	sort.Slice(addresses, func(left, right int) bool {
		return addresses[left].Name+addresses[left].AddressID < addresses[right].Name+addresses[right].AddressID
	})

	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "VERIFIED\tNAME\tDESCRIPTION\tSCOPE\tADDRESS_ID")
	for _, address := range addresses {
		fmt.Fprintf(writer, "%t\t%s\t%s\t%s\t%s\n",
			address.Verified,
			cell(address.Name),
			cell(address.Description),
			cell(formatScope(address.Scope)),
			cell(address.AddressID),
		)
	}
	return writer.Flush()
}

func loadDiscoveryClient() (*kraken.Client, error) {
	cfg, err := config.LoadKraken(".env")
	if err != nil {
		return nil, err
	}
	return newClient(cfg)
}

func newClient(cfg config.KrakenConfig) (*kraken.Client, error) {
	client, err := kraken.NewClient(kraken.ClientConfig{
		APIKey:  cfg.APIKey,
		Secret:  cfg.Secret,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("configure Kraken client: %w", err)
	}
	return client, nil
}

func commandContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func formatScope(scope kraken.FundingAddressScope) string {
	switch {
	case scope.MethodID != "":
		return "method:" + scope.MethodID
	case scope.NetworkID != "":
		return "network:" + scope.NetworkID
	case scope.NetworkGroupID != "":
		return "network-group:" + scope.NetworkGroupID
	default:
		return "-"
	}
}

func cell(value string) string {
	return strings.NewReplacer("\t", " ", "\r", " ", "\n", " ").Replace(value)
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  kraken-bot                         Run one trading and withdrawal cycle")
	fmt.Fprintln(output, "  kraken-bot funding-methods        List all deposit and withdrawal methods")
	fmt.Fprintln(output, "  kraken-bot funding-methods TYPE   List only deposit or withdraw methods")
	fmt.Fprintln(output, "  kraken-bot funding-addresses ID   List addresses compatible with a method")
}
