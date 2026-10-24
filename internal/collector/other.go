//go:build !windows && !linux && !darwin

package collector

import "context"

func collectHardware(_ context.Context) (hardwareInfo, error) { return hardwareInfo{}, nil }

func enrichNetwork(_ context.Context, _ *NetworkInfo) error { return nil }

func collectPrinting(ctx context.Context) (PrintingInfo, error) { return collectCUPS(ctx) }
