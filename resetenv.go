package postgres

import "os"

// resetEnv removes certain env variables that would otherwise be used by
// lib/pq to connect to postgres. I don't like surprises (or panics).
// see https://github.com/lib/pq/blob/v1.2.0/conn.go#L1838
//
// go.mod should pin lib/pq to version above.
func resetEnv() {
	v := []string{
		"PGHOST",
		"PGHOSTADDR",
		"PGPORT",
		"PGDATABASE",
		"PGUSER",
		"PGPASSWORD",
		"PGSERVICE", "PGSERVICEFILE", "PGREALM",
		"PGOPTIONS",
		"PGAPPNAME",
		"PGSSLMODE",
		"PGSSLCERT",
		"PGSSLKEY",
		"PGSSLROOTCERT",
		"PGREQUIRESSL", "PGSSLCRL",
		"PGREQUIREPEER",
		"PGKRBSRVNAME", "PGGSSLIB",
		"PGCONNECT_TIMEOUT",
		"PGCLIENTENCODING",
		"PGDATESTYLE",
		"PGTZ",
		"PGGEQO",
		"PGSYSCONFDIR", "PGLOCALEDIR",
	}

	for _, x := range v {
		if err := os.Unsetenv(x); err != nil {
			panic(err)
		}
	}
}
