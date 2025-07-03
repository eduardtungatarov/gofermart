package config

import (
	"flag"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Сохраняем оригинальные аргументы и окружение
	oldArgs := os.Args
	oldEnv := os.Environ()
	defer func() {
		os.Args = oldArgs
		os.Clearenv()
		for _, env := range oldEnv {
			keyVal := strings.SplitN(env, "=", 2)
			os.Setenv(keyVal[0], keyVal[1])
		}
	}()

	tests := []struct {
		name     string
		args     []string
		env      map[string]string
		expected Config
	}{
		{
			name: "default_values",
			args: []string{"cmd"},
			expected: Config{
				RunADDR:     DefaultRunADDR,
				AccrualADDR: DefaultAccrualSystemADRR,
				Database: Database{
					DSN:     DefaultDatabaseURI,
					Timeout: time.Second * 1,
				},
			},
		},
		{
			name: "command_line_flags",
			args: []string{"cmd", "-a=:9090", "-d=postgres://user:pass@localhost:5432/db", "-r=http://accrual:8080"},
			expected: Config{
				RunADDR:     ":9090",
				AccrualADDR: "http://accrual:8080",
				Database: Database{
					DSN:     "postgres://user:pass@localhost:5432/db",
					Timeout: time.Second * 1,
				},
			},
		},
		{
			name: "environment_variables",
			args: []string{"cmd"},
			env: map[string]string{
				"RUN_ADDRESS":             ":9090",
				"DATABASE_URI":            "postgres://user:pass@localhost:5432/db",
				"ACCRUAL_SYSTEM_ADDRESS":  "http://accrual:8080",
			},
			expected: Config{
				RunADDR:     ":9090",
				AccrualADDR: "http://accrual:8080",
				Database: Database{
					DSN:     "postgres://user:pass@localhost:5432/db",
					Timeout: time.Second * 1,
				},
			},
		},
		{
			name: "env_override_flags",
			args: []string{"cmd", "-a=:9091", "-d=postgres://flag:flag@flag:5432/flag", "-r=http://flag:8081"},
			env: map[string]string{
				"RUN_ADDRESS":             ":9090",
				"DATABASE_URI":            "postgres://env:env@env:5432/env",
				"ACCRUAL_SYSTEM_ADDRESS":  "http://env:8080",
			},
			expected: Config{
				RunADDR:     ":9090",
				AccrualADDR: "http://env:8080",
				Database: Database{
					DSN:     "postgres://env:env@env:5432/env",
					Timeout: time.Second * 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Подготовка аргументов командной строки
			os.Args = tt.args

			// Подготовка переменных окружения
			os.Clearenv()
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			// Сброс флагов перед каждым тестом
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			got := Load()

			if got.RunADDR != tt.expected.RunADDR {
				t.Errorf("RunADDR got = %v, want %v", got.RunADDR, tt.expected.RunADDR)
			}
			if got.AccrualADDR != tt.expected.AccrualADDR {
				t.Errorf("AccrualADDR got = %v, want %v", got.AccrualADDR, tt.expected.AccrualADDR)
			}
			if got.Database.DSN != tt.expected.Database.DSN {
				t.Errorf("Database.DSN got = %v, want %v", got.Database.DSN, tt.expected.Database.DSN)
			}
			if got.Database.Timeout != tt.expected.Database.Timeout {
				t.Errorf("Database.Timeout got = %v, want %v", got.Database.Timeout, tt.expected.Database.Timeout)
			}
		})
	}
}
