package build

import "time"

// Comentários em pt-BR: informações de build e uptime, para health endpoints

var (
	Version   = "dev"     // pode ser setado via -ldflags "-X suno-wallets/src/shared/build.Version=1.0.0"
	Commit    = "unknown" // -ldflags "-X suno-wallets/src/shared/build.Commit=$(GIT_SHA)"
	BuildDate = "unknown" // -ldflags "-X suno-wallets/src/shared/build.BuildDate=$(DATE)"
	startedAt = time.Now().UTC()
)

func StartedAt() time.Time { return startedAt }
func UptimeSeconds() int64 { return int64(time.Since(startedAt).Seconds()) }
