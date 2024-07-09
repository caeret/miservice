package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/caeret/miservice"
	cli "github.com/urfave/cli/v2"
)

type ctxKey string

const (
	ctxKeyToken ctxKey = "token"
)

func main() {
	app := cli.NewApp()
	app.Usage = "cli for xiao mi devices"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "username",
			Usage:   "xiaomi account's `username`",
			EnvVars: []string{"MI_USER"},
		},
		&cli.StringFlag{
			Name:    "password",
			Usage:   "xiaomi account's `password`",
			EnvVars: []string{"MI_PASS"},
		},
	}
	app.Before = func(cctx *cli.Context) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return fmt.Errorf("get config path: %w", err)
		}

		token := miservice.NewToken()

		b, err := os.ReadFile(cfgPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("read token file: %w", err)
			}
		} else {
			err = json.Unmarshal(b, token)
			if err != nil {
				return fmt.Errorf("unmarshal token: %w", err)
			}
		}

		cctx.Context = context.WithValue(cctx.Context, ctxKeyToken, token)

		return nil
	}
	app.After = func(cctx *cli.Context) error {
		cfgPath, err := getConfigPath()
		if err != nil {
			return fmt.Errorf("get config path: %w", err)
		}
		token := cctx.Context.Value(ctxKeyToken)
		if token == nil {
			return nil
		}
		b, err := json.Marshal(token)
		if err != nil {
			return fmt.Errorf("marshal token: %w", err)
		}
		err = os.WriteFile(cfgPath, b, 0o644)
		if err != nil {
			return fmt.Errorf("write token file: %w", err)
		}
		return nil
	}
	app.Commands = []*cli.Command{
		actionCmd,
		listDevicesCmd,
		getSpecCmd,
	}
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var listDevicesCmd = &cli.Command{
	Name:  "devices",
	Usage: "list devices",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "region",
			Value: "cn",
		},
		&cli.BoolFlag{
			Name: "virtual-model",
		},
		&cli.IntFlag{
			Name: "huami-devices",
		},
	},
	Action: func(cctx *cli.Context) error {
		account, err := miservice.NewAccount(cctx.String("username"), cctx.String("password"), cctx.Context.Value(ctxKeyToken).(*miservice.Token))
		if err != nil {
			return err
		}
		miio := miservice.NewMiIO(account, cctx.String("region"))
		devices, err := miio.ListDevices(cctx.Context, cctx.Bool("virtual-model"), cctx.Int("huami-devices"))
		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")

		return enc.Encode(devices)
	},
}

var actionCmd = &cli.Command{
	Name:      "action",
	Usage:     "send action to specified device",
	ArgsUsage: "[did] [iid] [args...]",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() < 3 {
			return cli.ShowSubcommandHelp(cctx)
		}
		account, err := miservice.NewAccount(cctx.String("username"), cctx.String("password"), cctx.Context.Value(ctxKeyToken).(*miservice.Token))
		if err != nil {
			return err
		}

		iid := miservice.T2(0, 0)
		strs := strings.Split(cctx.Args().Get(1), "-")
		switch len(strs) {
		case 2:
		case 1:
			strs = append(strs, "1")
		default:
			return fmt.Errorf("invalid iid: %s", cctx.Args().Get(1))
		}

		for i := range strs {
			v, err := strconv.Atoi(strs[i])
			if err != nil {
				return fmt.Errorf("invalid iid: %w", err)
			}
			if i == 0 {
				iid.A = v
			} else {
				iid.B = v
			}
		}

		var args []any
		for _, v := range cctx.Args().Slice()[2:] {
			args = append(args, v)
		}

		miio := miservice.NewMiIO(account, cctx.String("region"))
		err = miio.SendAction(cctx.Context, cctx.Args().First(), iid, args...)
		if err != nil {
			return err
		}

		fmt.Println("OK")

		return nil
	},
}

var getSpecCmd = &cli.Command{
	Name:      "spec",
	Usage:     "get spec for specified device type",
	ArgsUsage: "[device_type]",
	Action: func(cctx *cli.Context) error {
		if cctx.Args().Len() == 0 {
			return cli.ShowSubcommandHelp(cctx)
		}

		spec, err := miservice.GetSpec(cctx.Context, cctx.Args().First())
		if err != nil {
			return err
		}

		buf := bytes.NewBuffer(nil)
		err = json.Indent(buf, spec, "", "  ")
		if err != nil {
			return err
		}

		_, _ = io.Copy(os.Stdout, buf)
		fmt.Println()

		return nil
	},
}

func getConfigPath() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}
	cfgDir := filepath.Join(dir, ".config", "micli")
	err = os.MkdirAll(cfgDir, 0o755)
	if err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	return filepath.Join(cfgDir, "token.json"), nil
}
