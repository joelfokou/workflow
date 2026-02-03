package logger

import (
	"io"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	l    *zap.Logger
	once sync.Once
	logW io.WriteCloser
)

// Define logger behavior
type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json or console
	OutputFile string // empty = stdout
}

func Init(cfg Config) error {
	var err error
	once.Do(func() {
		var lvl zapcore.Level
		if err = lvl.UnmarshalText([]byte(cfg.Level)); err != nil {
			lvl = zapcore.InfoLevel
		}

		var enc zapcore.Encoder
		if cfg.Format == "json" {
			enc = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		} else {
			encCfg := zap.NewDevelopmentEncoderConfig()
			if cfg.OutputFile == "" {
				encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
			}
			enc = zapcore.NewConsoleEncoder(encCfg)
		}

		var ws zapcore.WriteSyncer

		if cfg.OutputFile == "" {
			ws = zapcore.AddSync(os.Stdout)
		} else {
			dir := filepath.Dir(cfg.OutputFile)
			if dir != "." {
				if err = os.MkdirAll(dir, 0755); err != nil {
					return
				}
			}
			logW, err = os.OpenFile(cfg.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			ws = zapcore.AddSync(logW)
		}

		core := zapcore.NewCore(enc, ws, lvl)
		l = zap.New(core)
		// l = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	})

	return err
}

func L() *zap.Logger {
	if l == nil {
		panic("logger not initialised. Call logger.Init() first")
	}
	return l
}

// Flush buffers
func Sync() {
	if l != nil {
		_ = l.Sync()
	}
	if logW != nil {
		_ = logW.Close()
	}
}
