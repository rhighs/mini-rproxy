package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	jwtlib "github.com/tgym-digital/mini-rproxy/plugins/jwt"
)

type Command struct {
	Name        string
	Description string
	FlagSet     *flag.FlagSet
	Run         func(cmd *Command, args []string) error
}

var commands = map[string]*Command{}

func register(cmd *Command) {
	if cmd.FlagSet == nil {
		cmd.FlagSet = flag.NewFlagSet(cmd.Name, flag.ExitOnError)
	}
	commands[cmd.Name] = cmd
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if (strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`)) ||
			(strings.HasPrefix(val, `'`) && strings.HasSuffix(val, `'`)) {
			val = val[1 : len(val)-1]
		}
		_ = os.Setenv(key, val)
	}
	return sc.Err()
}

func main() {
	loadDotEnv(".env")
	if len(os.Args) < 2 {
		globalUsage()
		os.Exit(1)
	}
	name := os.Args[1]
	if name == "-h" || name == "--help" || name == "help" {
		if len(os.Args) == 2 {
			globalUsage()
			return
		}
		helpCommand(os.Args[2])
		return
	}

	initCommands()

	cmd, ok := commands[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", name)
		globalUsage()
		os.Exit(1)
	}
	// Parse its flags
	if err := cmd.FlagSet.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Argument parsing error for %s: %v\n", name, err)
		os.Exit(2)
	}
	if err := cmd.Run(cmd, cmd.FlagSet.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		os.Exit(1)
	}
}

func initCommands() {
	// parse
	parse := &Command{
		Name:        "parse",
		Description: "Parse a JWT payload and print indented JSON",
		Run: func(c *Command, _ []string) error {
			if *parseJWTFlag == "" {
				return fmt.Errorf("--jwt required")
			}
			payload, err := jwtlib.ParseJWTPayload(*parseJWTFlag)
			if err != nil {
				return fmt.Errorf("parse error: %w", err)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(payload)
		},
	}
	parse.FlagSet = flag.NewFlagSet(parse.Name, flag.ContinueOnError)
	parse.FlagSet.SetOutput(new(flagDiscard)) // silence default usage noise
	parseJWTFlag = parse.FlagSet.String("jwt", "", "JWT token to parse")
	register(parse)

	// legacy-from-jwt
	lfj := &Command{
		Name:        "legacy-from-jwt",
		Description: "Build a legacy token from a JWT (requires env: TGAUTH_HASH_KEY, TGAUTH_SIGN_SALT)",
		Run: func(c *Command, _ []string) error {
			if *legacyFromJWTFlag == "" {
				return fmt.Errorf("--jwt required")
			}
			payload, err := jwtlib.ParseJWTPayload(*legacyFromJWTFlag)
			if err != nil {
				return fmt.Errorf("parse error: %w", err)
			}
			tok := jwtlib.BuildLegacyToken(payload)
			if tok == "" {
				return fmt.Errorf("legacy token generation failed (check env vars / payload)")
			}
			fmt.Println(tok)
			return nil
		},
	}
	lfj.FlagSet = flag.NewFlagSet(lfj.Name, flag.ContinueOnError)
	lfj.FlagSet.SetOutput(new(flagDiscard))
	legacyFromJWTFlag = lfj.FlagSet.String("jwt", "", "JWT token input")
	register(lfj)

	// legacy-build
	lb := &Command{
		Name:        "legacy-build",
		Description: "Build a legacy token from key=value legacycompat fields",
		Run: func(c *Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("provide key=value pairs")
			}
			m, err := parseKeyValuePairs(args)
			if err != nil {
				return err
			}
			var legacy jwtlib.LegacyTokenPayload
			b, _ := json.Marshal(m)
			if err := json.Unmarshal(b, &legacy); err != nil {
				return fmt.Errorf("unmarshal legacy payload: %w", err)
			}
			payload := &jwtlib.JWTPayload{LegacyCompat: &legacy}
			tok := jwtlib.BuildLegacyToken(payload)
			if tok == "" {
				return fmt.Errorf("legacy token generation failed (missing env vars?)")
			}
			fmt.Println(tok)
			return nil
		},
	}
	lb.FlagSet = flag.NewFlagSet(lb.Name, flag.ContinueOnError)
	lb.FlagSet.SetOutput(new(flagDiscard))
	register(lb)

	// equipment-from-jwt
	efj := &Command{
		Name:        "equipment-from-jwt",
		Description: "Build an equipment token from JWT (or external equipment context) (env: TGAUTH_LEGACY_PKEY)",
		Run: func(c *Command, _ []string) error {
			if *equipmentFromJWTFlag == "" {
				return fmt.Errorf("--jwt required")
			}
			payload, err := jwtlib.ParseJWTPayload(*equipmentFromJWTFlag)
			if err != nil {
				return fmt.Errorf("parse error: %w", err)
			}
			token, err := jwtlib.EquipmentTokenFromContext(*equipmentFromJWTExtCtxFlag)
			if err != nil {
				token, err = jwtlib.EquipmentTokenFromPayload(payload)
				if err != nil {
					return err
				}
			}
			fmt.Println(token)
			return nil
		},
	}
	efj.FlagSet = flag.NewFlagSet(efj.Name, flag.ContinueOnError)
	efj.FlagSet.SetOutput(new(flagDiscard))
	equipmentFromJWTFlag = efj.FlagSet.String("jwt", "", "JWT token input")
	equipmentFromJWTExtCtxFlag = efj.FlagSet.String("equipment-context", "", "External base64url encoded equipment context JSON")
	register(efj)

	// equipment-build
	eb := &Command{
		Name:        "equipment-build",
		Description: "Build an equipment token from key=value equipmentContext fields",
		Run: func(c *Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("provide key=value pairs")
			}
			m, err := parseKeyValuePairs(args)
			if err != nil {
				return err
			}
			var eq jwtlib.EquipmentContextPayload
			b, _ := json.Marshal(m)
			if err := json.Unmarshal(b, &eq); err != nil {
				return fmt.Errorf("unmarshal equipment payload: %w", err)
			}
			payload := &jwtlib.JWTPayload{EquipmentContext: &eq}
			tok, err := jwtlib.EquipmentTokenFromPayload(payload)
			if err != nil {
				return err
			}
			fmt.Println(tok)
			return nil
		},
	}
	eb.FlagSet = flag.NewFlagSet(eb.Name, flag.ContinueOnError)
	eb.FlagSet.SetOutput(new(flagDiscard))
	register(eb)

	// decode-b64url
	db := &Command{
		Name:        "decode-b64url",
		Description: "Decode base64url (no padding) string to stdout",
		Run: func(c *Command, _ []string) error {
			if *decodeB64DataFlag == "" {
				return fmt.Errorf("--data required")
			}
			out, err := decodeBase64URL(*decodeB64DataFlag)
			if err != nil {
				return err
			}
			os.Stdout.Write(out)
			return nil
		},
	}
	db.FlagSet = flag.NewFlagSet(db.Name, flag.ContinueOnError)
	db.FlagSet.SetOutput(new(flagDiscard))
	decodeB64DataFlag = db.FlagSet.String("data", "", "base64url data")
	register(db)
}

func helpCommand(name string) {
	initCommands()
	if cmd, ok := commands[name]; ok {
		fmt.Printf("Command: %s\n\n", cmd.Name)
		fmt.Println(cmd.Description)
		fmt.Printf("\nUsage:\n  %s [flags] ", cmd.Name)
		switch cmd.Name {
		case "parse":
			fmt.Println("--jwt <token>")
		case "legacy-from-jwt":
			fmt.Println("--jwt <token>")
		case "equipment-from-jwt":
			fmt.Println("--jwt <token> [--equipment-context <b64url-json>]")
		case "decode-b64url":
			fmt.Println("--data <b64url>")
		default:
			fmt.Println("[key=value ...]")
		}
		fmt.Println()
		cmd.FlagSet.PrintDefaults()
		fmt.Println()
		return
	}
	fmt.Fprintf(os.Stderr, "No such command: %s\n\n", name)
	globalUsage()
}

func globalUsage() {
	initCommands()
	fmt.Println("Usage: <command> [flags] [args]")
	fmt.Println()
	fmt.Println("Commands:")
	names := make([]string, 0, len(commands))
	for k := range commands {
		names = append(names, k)
	}
	sort.Strings(names)
	max := 0
	for _, n := range names {
		if len(n) > max {
			max = len(n)
		}
	}
	for _, n := range names {
		c := commands[n]
		fmt.Printf("  %-*s  %s\n", max, n, c.Description)
	}
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  TGAUTH_HASH_KEY / TGAUTH_SIGN_SALT  Required for legacy token generation")
	fmt.Println("  TGAUTH_LEGACY_PKEY                  Required for equipment token generation")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  legacy-build tokenDateTime=2025-09-08T00:00:00Z credentialId=ABC userId=U1 tokenFor=0 domain=example.com")
	fmt.Println("  legacy-from-jwt --jwt <jwt>")
	fmt.Println("  equipment-build serial=SER1 facilityId=FAC1 deviceType=bike screenType=oled operatingSystem=linux isKiosk=true equipmentCode=EQ1 facilityUrl=https://f swVersion=1 platform=x mainAppVersion=2 domainId=0 lob=0")
	fmt.Println()
	fmt.Println("Use 'help <command>' for detailed help.")
}

// -------- Flag variables (kept global for clarity) ----------
var (
	parseJWTFlag               *string
	legacyFromJWTFlag          *string
	equipmentFromJWTFlag       *string
	equipmentFromJWTExtCtxFlag *string
	decodeB64DataFlag          *string
)

// -------- Helpers ----------

type flagDiscard struct{}

func (f *flagDiscard) Write(p []byte) (int, error) { return len(p), nil }

func parseKeyValuePairs(pairs []string) (map[string]any, error) {
	m := make(map[string]any, len(pairs))
	for _, kv := range pairs {
		if !strings.Contains(kv, "=") {
			return nil, fmt.Errorf("invalid pair %q (expected key=value)", kv)
		}
		parts := strings.SplitN(kv, "=", 2)
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if k == "" {
			return nil, fmt.Errorf("empty key in pair %q", kv)
		}
		m[k] = v
	}
	return m, nil
}

func decodeBase64URL(s string) ([]byte, error) {
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	return base64.StdEncoding.DecodeString(s)
}
