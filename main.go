package main

import (
	"flag"
	"fmt"

	"github.com/go-restream/stt/config"
	"github.com/go-restream/stt/internal/service"
	"github.com/go-restream/stt/internal/version"
	"github.com/go-restream/stt/pkg/health"
	"github.com/go-restream/stt/pkg/logger"

	"github.com/sirupsen/logrus"
)

var AppConfig *config.Config

func main() {
	versionFlag := flag.Bool("v", false, "Show version information")
	versionFullFlag := flag.Bool("version", false, "Show full version information")
	configPath := flag.String("c", "config.yaml", "Path to configuration file")
	flag.Parse()

		if *versionFlag {
		fmt.Println(version.Short())
		return
	}
	if *versionFullFlag {
		fmt.Println(version.Full())
		return
	}

	var err error
	AppConfig, err = config.LoadConfig(*configPath)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"component": "mont_srv_status",
			"action":        "health_check_status",
		}).Fatalf("✘ load config failed: %v", err)
	}

	if err := logger.InitLogger(AppConfig.Logging.Level, AppConfig.Logging.File); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "mont_srv_status",
			"action":        "health_check_status",
		}).Fatalf("✘ failed to initialize logger: %v", err)
	}

	if AppConfig.Logging.Format == "text" {
		logger.Logger.SetFormatter(&logger.CustomFormatterText{
			TimestampFormat: "2006-01-02 15:04:05.000",
			ForceColors:     true, 
		})
	} else if AppConfig.Logging.Format == "json" {
		logger.Logger.SetFormatter(&logger.CustomFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			ForceColors:     true, 
		})
	} else {
		logger.Logger.SetFormatter(&logger.CustomFormatter{
			TimestampFormat: "2006-01-02 15:04:05.000",
			ForceColors:     true,
		})
	}

	logger.WithFields(logrus.Fields{
			"component": "mont_srv_status",
			"action":        "health_check_status",
			"version":       version.Short(),
			"build_time":    version.GetBuildTime(),
			"git_commit":    version.GetGitCommit(),
		}).Infof("✔ Starting StreamASR %s with config: %s", version.Short(), *configPath)

	if err := checkASREngineHealth(); err != nil {
		logger.WithFields(logrus.Fields{
			"component": "mont_srv_status",
			"action":    "health_check_failed",
		}).Errorf("✘ ASR engine health check failed: %v", err)
		logger.WithFields(logrus.Fields{
			"component": "mont_srv_status",
			"action":    "health_check_warning",
		}).Warn("StreamASR will start, but ASR functionality may be limited")
	} else {
		logger.WithFields(logrus.Fields{
			"component": "mont_srv_status",
			"action":        "health_check_status",
		}).Info(" ✔ ASR engine health check passed")
	}

	service.WsServiceRun(AppConfig.ServicePort, *configPath)
}

func checkASREngineHealth() error {
	logger.WithFields(logrus.Fields{
		"component": "mont_srv_status",
		"action":    "health_check_start",
	}).Debug("Checking ASR engine health...")

	healthChecker := health.NewHealthChecker(
		AppConfig.ASR.BaseURL,
		AppConfig.ASR.APIKey,
		AppConfig.ASR.Model,
	)

	result := healthChecker.CheckASREngineHealth()
	logger.WithFields(logrus.Fields{
		"component": "sys_startup_main",
		"action":        "asr_health_check",
		"status":        result.Status,
		"asrEngineURL":  result.ASREngineURL,
		"totalChecks":   len(result.Checks),
	}).Debug("ASR engine health check completed")

	for _, check := range result.Checks {
		logger.WithFields(logrus.Fields{
			"component": "sys_startup_main",
			"endpoint":  check.Service,
			"status":    check.Status,
			"latency":   check.Latency.Milliseconds(),
			"error":     check.Error,
		}).Debugf("ASR %s endpoint check", check.Service)
	}

	if result.Status == "ok" {
		return nil
	}

	return fmt.Errorf("ASR engine health check failed: %s", result.Error)
}
