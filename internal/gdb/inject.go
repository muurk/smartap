package gdb

import (
	"context"
	"fmt"

	"github.com/muurk/smartap/internal/gdb/scripts"
	"go.uber.org/zap"
)

// InjectOptions holds options for certificate injection.
type InjectOptions struct {
	// Executor is the GDB executor to use
	Executor *Executor

	// CertManager is the certificate manager
	CertManager *CertManager

	// CertName is the name of an embedded certificate to inject
	// (e.g., "root_ca"). Mutually exclusive with CertPath.
	CertName string

	// CertPath is the path to a custom certificate file to inject.
	// Mutually exclusive with CertName.
	CertPath string

	// TargetFile is the destination path on the device.
	// Default: "/cert/129.der"
	TargetFile string

	// FirmwareVersion is the firmware version to use.
	// If empty, firmware will be auto-detected.
	FirmwareVersion string

	// SkipDetection skips firmware detection and uses FirmwareVersion directly.
	SkipDetection bool

	// OnProgress is a callback for progress updates.
	// Called for each step with the step info.
	OnProgress func(step scripts.Step)
}

// InjectCertificate performs the complete certificate injection workflow.
// This is the high-level function that orchestrates:
//  1. Prerequisite validation
//  2. Certificate loading (embedded or custom)
//  3. Firmware detection (if not provided)
//  4. Script execution
//  5. Result parsing
func InjectCertificate(ctx context.Context, opts InjectOptions) (*scripts.Result, error) {
	logger := opts.Executor.logger

	logger.Info("starting certificate injection workflow",
		zap.String("cert_name", opts.CertName),
		zap.String("cert_path", opts.CertPath),
		zap.String("target_file", opts.TargetFile),
		zap.String("firmware_version", opts.FirmwareVersion),
		zap.Bool("skip_detection", opts.SkipDetection),
	)

	// Step 1: Validate prerequisites
	logger.Debug("validating prerequisites")
	if err := opts.Executor.ValidateConfig(ctx); err != nil {
		return nil, fmt.Errorf("prerequisite validation failed: %w", err)
	}

	// Step 2: Load certificate
	logger.Debug("loading certificate")
	var certData []byte
	var err error

	if opts.CertPath != "" {
		// Load custom certificate
		logger.Info("loading custom certificate", zap.String("path", opts.CertPath))
		certData, err = opts.CertManager.LoadCustom(opts.CertPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load custom certificate: %w", err)
		}
	} else {
		// Use embedded certificate
		certName := opts.CertName
		if certName == "" {
			certName = "root_ca"
		}
		logger.Info("using embedded certificate", zap.String("name", certName))
		certData, err = opts.CertManager.GetRootCA("der")
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded certificate: %w", err)
		}
	}

	logger.Info("certificate loaded",
		zap.Int("size", len(certData)),
	)

	// Step 3: Detect or validate firmware version
	var firmware *Firmware
	firmwareVersion := opts.FirmwareVersion

	if !opts.SkipDetection && firmwareVersion == "" {
		// Auto-detect firmware
		logger.Info("detecting firmware version")

		// Load firmware catalog first to pass to detection script
		db, err := LoadFirmwares()
		if err != nil {
			return nil, fmt.Errorf("failed to load firmware catalog: %w", err)
		}

		detectScript := scripts.NewDetectFirmwareScript(
			opts.Executor.config.OpenOCDHost,
			opts.Executor.config.OpenOCDPort,
			db.List(),
		)

		detectResult, err := opts.Executor.Execute(ctx, detectScript)
		if err != nil {
			return nil, fmt.Errorf("firmware detection failed: %w", err)
		}

		// Validate confidence score
		confidence := detectResult.GetDataInt("confidence")
		if confidence < 100 {
			matches := detectResult.GetDataInt("matches")
			total := detectResult.GetDataInt("total")
			version := detectResult.GetDataString("version")

			logger.Error("firmware detection confidence too low",
				zap.Int("confidence", confidence),
				zap.Int("matches", matches),
				zap.Int("total", total),
				zap.String("version", version),
			)

			return nil, &FirmwareConfidenceError{
				Version:    version,
				Confidence: confidence,
				Matches:    matches,
				Total:      total,
			}
		}

		firmwareVersion = detectResult.GetDataString("version")
		logger.Info("firmware detected",
			zap.String("version", firmwareVersion),
			zap.Int("confidence", confidence),
		)
	}

	// Load firmware from catalog
	logger.Debug("loading firmware from catalog", zap.String("version", firmwareVersion))
	db, err := LoadFirmwares()
	if err != nil {
		return nil, fmt.Errorf("failed to load firmware catalog: %w", err)
	}

	firmware, ok := db.Get(firmwareVersion)
	if !ok {
		return nil, HandleUnknownFirmware(firmwareVersion)
	}

	logger.Info("firmware loaded from catalog",
		zap.String("version", firmware.Version),
		zap.String("name", firmware.Name),
		zap.Bool("verified", firmware.Verified),
	)

	// Step 4: Create and execute injection script
	targetFile := opts.TargetFile
	if targetFile == "" {
		targetFile = "/cert/129.der"
	}

	logger.Info("preparing certificate injection",
		zap.String("target_file", targetFile),
		zap.Int("cert_size", len(certData)),
	)

	injectScript := scripts.NewInjectCertsScript(
		firmware,
		certData,
		targetFile,
		opts.Executor.config.OpenOCDHost,
		opts.Executor.config.OpenOCDPort,
	)

	result, err := opts.Executor.Execute(ctx, injectScript)
	if err != nil {
		return nil, fmt.Errorf("certificate injection failed: %w", err)
	}

	// Call progress callback for each step
	if opts.OnProgress != nil {
		for _, step := range result.Steps {
			opts.OnProgress(step)
		}
	}

	if !result.Success {
		logger.Error("certificate injection failed",
			zap.Error(result.Error),
			zap.Int("steps", result.TotalSteps()),
			zap.Int("success_steps", result.SuccessSteps()),
			zap.Int("failed_steps", result.FailedSteps()),
		)
		return result, fmt.Errorf("certificate injection failed: %w", result.Error)
	}

	logger.Info("certificate injection completed successfully",
		zap.Duration("duration", result.Duration),
		zap.Int("bytes_written", result.BytesWritten),
		zap.String("target_file", targetFile),
	)

	return result, nil
}
