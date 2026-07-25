package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var AllowedHosts = map[string]struct{}{
	"loopbudget.com":     {},
	"www.loopbudget.com": {},
	"127.0.0.1":          {},
	"localhost":          {},
}

type Config struct {
	BaseURL          string
	APIKey           string
	PriceIn          float64
	PriceOut         float64
	StatePath        string
	CredentialsPath  string // empty if env-only
}

func HomeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".loopbudget")
	}
	return filepath.Join(h, ".loopbudget")
}

func CredentialsPath() string { return filepath.Join(HomeDir(), "credentials") }
func DefaultStatePath() string {
	return filepath.Join(HomeDir(), "claude-code-state.json")
}

func EnsureHomeDir() error {
	dir := HomeDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0o700)
	return nil
}

func ParseDotEnv(raw string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		eq := strings.IndexByte(trimmed, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		value := strings.TrimSpace(trimmed[eq+1:])
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		out[key] = value
	}
	return out
}

func AssertSafeBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid LOOPBUDGET_URL: %s", raw)
	}
	if u.User != nil {
		return "", fmt.Errorf("LOOPBUDGET_URL must not include credentials")
	}
	host := strings.ToLower(u.Hostname())
	if _, ok := AllowedHosts[host]; !ok {
		keys := make([]string, 0, len(AllowedHosts))
		for k := range AllowedHosts {
			keys = append(keys, k)
		}
		return "", fmt.Errorf("LOOPBUDGET_URL host %q is not allowlisted. Allowed: %s", host, strings.Join(keys, ", "))
	}
	loopback := host == "127.0.0.1" || host == "localhost"
	if !loopback && u.Scheme != "https" {
		return "", fmt.Errorf("LOOPBUDGET_URL must use https:// (http only allowed for localhost)")
	}
	if loopback && u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("LOOPBUDGET_URL must use http:// or https://")
	}
	return u.Scheme + "://" + u.Host, nil
}

func Load() (Config, error) {
	fileVars := map[string]string{}
	credPath := ""
	path := CredentialsPath()
	if st, err := os.Stat(path); err == nil {
		credPath = path
		if runtime.GOOS != "windows" {
			mode := st.Mode().Perm()
			if mode&0o077 != 0 {
				return Config{}, fmt.Errorf("%s mode is %o; fix with: chmod 600 %s", path, mode, path)
			}
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		fileVars = ParseDotEnv(string(raw))
	}

	getenv := func(k, fallback string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		if v, ok := fileVars[k]; ok && v != "" {
			return v
		}
		return fallback
	}

	baseRaw := getenv("LOOPBUDGET_URL", "https://loopbudget.com")
	apiKey := getenv("LOOPBUDGET_API_KEY", "")
	priceIn, _ := strconv.ParseFloat(getenv("PRICE_IN_PER_M", "3"), 64)
	priceOut, _ := strconv.ParseFloat(getenv("PRICE_OUT_PER_M", "15"), 64)
	statePath := getenv("STATE_PATH", DefaultStatePath())

	if apiKey == "" {
		return Config{}, fmt.Errorf("missing LOOPBUDGET_API_KEY. Run: loopbudget-claude-code init\n(writes %s)", CredentialsPath())
	}
	base, err := AssertSafeBaseURL(baseRaw)
	if err != nil {
		return Config{}, err
	}
	return Config{
		BaseURL:         base,
		APIKey:          apiKey,
		PriceIn:         priceIn,
		PriceOut:        priceOut,
		StatePath:       statePath,
		CredentialsPath: credPath,
	}, nil
}

func WriteCredentials(baseURL, apiKey string) (string, error) {
	safe, err := AssertSafeBaseURL(baseURL)
	if err != nil {
		return "", err
	}
	if err := EnsureHomeDir(); err != nil {
		return "", err
	}
	path := CredentialsPath()
	body := "# LoopBudget Claude Code hook — keep mode 600. Do not commit.\n" +
		"LOOPBUDGET_URL=" + safe + "\n" +
		"LOOPBUDGET_API_KEY=" + apiKey + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", err
	}
	_ = os.Chmod(path, 0o600)
	return path, nil
}
