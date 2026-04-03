package doctor

import "testing"

func TestDetectDBTarget(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		dialect string
		driver  string
		dsn     string
	}{
		{"postgres://u:p@localhost:5432/marchat?sslmode=disable", "postgres", "pgx", "postgres://u:p@localhost:5432/marchat?sslmode=disable"},
		{"mysql://u:p@tcp(localhost:3306)/marchat?parseTime=true", "mysql", "mysql", "u:p@tcp(localhost:3306)/marchat?parseTime=true"},
		{"mysql:u:p@tcp(localhost:3306)/marchat?parseTime=true", "mysql", "mysql", "u:p@tcp(localhost:3306)/marchat?parseTime=true"},
		{"./config/marchat.db", "sqlite", "sqlite", "./config/marchat.db"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got := detectDBTarget(tc.in)
			if got.dialect != tc.dialect || got.driver != tc.driver || got.dsn != tc.dsn {
				t.Fatalf("detectDBTarget(%q) = %+v", tc.in, got)
			}
		})
	}
}

func TestValidateConnectionString(t *testing.T) {
	t.Parallel()
	valid := []dbTarget{
		{dialect: "sqlite", dsn: "./config/marchat.db"},
		{dialect: "postgres", dsn: "postgres://u:p@localhost:5432/marchat?sslmode=disable"},
		{dialect: "mysql", dsn: "u:p@tcp(localhost:3306)/marchat?parseTime=true"},
	}
	for _, tc := range valid {
		tc := tc
		t.Run(tc.dialect+"_valid", func(t *testing.T) {
			t.Parallel()
			if err := validateConnectionString(tc); err != nil {
				t.Fatalf("expected valid connection string, got error: %v", err)
			}
		})
	}

	invalid := []dbTarget{
		{dialect: "sqlite", dsn: "   "},
		{dialect: "postgres", dsn: "postgres://%"},
		{dialect: "mysql", dsn: "not-a-dsn"},
	}
	for _, tc := range invalid {
		tc := tc
		t.Run(tc.dialect+"_invalid", func(t *testing.T) {
			t.Parallel()
			if err := validateConnectionString(tc); err == nil {
				t.Fatal("expected invalid connection string error")
			}
		})
	}
}
