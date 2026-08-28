package config

import (
	"os"
	"testing"
)

func clearIdpforgeEnv(t *testing.T) {
	t.Helper()
	for _, e := range os.Environ() {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				key := e[:i]
				if len(key) >= 9 && key[:9] == "IDPFORGE_" {
					old, existed := os.LookupEnv(key)
					os.Unsetenv(key)
					if existed {
						t.Cleanup(func() { os.Setenv(key, old) })
					}
				}
				break
			}
		}
	}
}

func TestLoadRequiresDSN(t *testing.T) {
	clearIdpforgeEnv(t)
	if _, err := Load(); err == nil {
		t.Error("expected Load() to fail without IDPFORGE_DB_DSN set")
	}
}

func TestLoadRejectsUnknownDriver(t *testing.T) {
	clearIdpforgeEnv(t)
	os.Setenv("IDPFORGE_DB_DSN", "postgres://localhost/x")
	os.Setenv("IDPFORGE_DB_DRIVER", "oracle")
	t.Cleanup(func() {
		os.Unsetenv("IDPFORGE_DB_DSN")
		os.Unsetenv("IDPFORGE_DB_DRIVER")
	})

	if _, err := Load(); err == nil {
		t.Error("expected Load() to reject an unsupported driver")
	}
}

func TestLoadDefaults(t *testing.T) {
	clearIdpforgeEnv(t)
	os.Setenv("IDPFORGE_DB_DSN", "postgres://localhost/x")
	t.Cleanup(func() { os.Unsetenv("IDPFORGE_DB_DSN") })

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.DB.Driver != DBPostgres {
		t.Errorf("default driver = %v, want postgres", cfg.DB.Driver)
	}
	if cfg.HTTP.ListenAddr != ":8080" {
		t.Errorf("default listen addr = %v, want :8080", cfg.HTTP.ListenAddr)
	}
	if !cfg.RateLimit.Enabled {
		t.Error("expected rate limiting to default to enabled")
	}
	if cfg.Storage.Backend != "local" {
		t.Errorf("default storage backend = %v, want local", cfg.Storage.Backend)
	}
}

func TestMaskedDSNRedactsPassword(t *testing.T) {
	cfg := DBConfig{DSN: "postgres://idpforge:supersecret@db.example.com:5432/idpforge?sslmode=disable"}
	masked := cfg.MaskedDSN()

	if masked == cfg.DSN {
		t.Error("MaskedDSN() returned the DSN unchanged")
	}
	for i := 0; i+11 <= len(masked); i++ {
		if masked[i:i+11] == "supersecret" {
			t.Errorf("MaskedDSN() leaked the password: %s", masked)
		}
	}
}
